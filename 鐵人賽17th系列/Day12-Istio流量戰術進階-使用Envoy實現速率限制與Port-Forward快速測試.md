## K8S Lab Day_14

# Day12: Istio 流量戰術進階，使用 Envoy 實現速率限制與 Port Forward 快速測試

## 前言

昨天介紹完了 traffic management 的各個描述，有沒有覺得更邁向 yaml 工程師邁進啊（誤，今天呢會繼續往下一步邁進！

## Enabling Rate Limits using Envoy

因為要來快速地來測試，所以今天會使用 port-forward 去繞過一些繁雜的設定去快速的測試，這個指令直接將本地的 k8s cluster 和 istio-ingressgateway 之劍建立一個安全的通道，直接將本地的 8080 導到 port 80，簡單來說我們搭建了一條臨時的代理通道，而不需要依賴外部 IP、LoadBalancer 或 NodePort

```bash
# 建立 istio 提供的 sample app bookinfo
kubectl apply -f samples/bookinfo/platform/kube/bookinfo.yaml
kubectl apply -f samples/bookinfo/networking/bookinfo-gateway.yaml
```

```bash
kubectl port-forward -n istio-system svc/istio-ingressgateway 8080:80
```

![port-forwarding](https://github.com/user-attachments/assets/94bdd4e6-798b-4d25-8c46-6078bb0d79a9)

接著嘗試看看是否有正確的執行

```bash
curl -I http://localhost:8080/productpage
```

```bash
HTTP/1.1 200 OK
server: istio-envoy
date: Wed, 24 Sep 2025 03:04:20 GMT
content-type: text/html; charset=utf-8
content-length: 15068
vary: Cookie
x-envoy-upstream-service-time: 171
```

### 實驗目標

在 Istio ingressgateway 上限制流量速率，每個 client IP 最多 2 requests/5s，超過的請求要回 429 的 Too Many Requests

![rate-limit](https://github.com/user-attachments/assets/f4ac7393-36ea-424c-a4ca-8bb6ce6bb687)

- 建立 RateLimit EnvoyFilter

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: ratelimit
  namespace: istio-system
spec:
  workloadSelector:
    labels:
      istio: ingressgateway
  configPatches:
    - applyTo: HTTP_FILTER
      match:
        context: GATEWAY
        listener:
          filterChain:
            filter:
              name: envoy.filters.network.http_connection_manager
              subFilter:
                name: envoy.filters.http.router
      patch:
        operation: INSERT_BEFORE
        value:
          name: envoy.filters.http.local_ratelimit
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit
            stat_prefix: http_local_rate_limiter
            token_bucket: # 定義 token bucket 限流策略
              max_tokens: 2 # 每個週期最多可用的 token 數
              tokens_per_fill: 2 # 每次 refill 新增多少 token
              fill_interval: 5s # token refill 的時間間隔
            filter_enabled: # 控制此 filter 是否啟用
              runtime_key: local_rate_limit_enabled
              default_value:
                numerator: 100 # 預設啟用（100%）
                denominator: HUNDRED
            filter_enforced: # 控制限流是否強制執行
              runtime_key: local_rate_limit_enforced
              default_value:
                numerator: 100 # 預設強制執行（100%）
                denominator: HUNDRED
            response_headers_to_add: # 回應中加上標頭，用於判斷是否被限流
              - append: false
                header:
                  key: x-ratelimited
                  value: "true"
```

然後我們就可以寫一個 bash 來去測試他的行為

```bash
for i in {1..3}; do
  curl -I http://localhost:8080/productpage
done

sleep 5

curl -I http://localhost:8080/productpage
```

我們就可以看到在第三個回覆的 request 資訊就會跳出

```bash
HTTP/1.1 429 Too Many Requests
x-ratelimited: true
content-length: 18
content-type: text/plain
date: Wed, 24 Sep 2025 03:29:00 GMT
server: istio-envoy
```

## 結論

這樣一個簡單使用 Envoy 去做 rate limit 的方式就完成啦！

## Reference

https://istio.io/latest/docs/tasks/policy-enforcement/rate-limit/
