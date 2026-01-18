## K8S Lab Day_19

# Day17: Istio Gateway 戰術：Ingress / Egress Gateway 的配置與安全防護

## 前言

在 istio 中 Gateway 作為流量的出入口，有分為 Ingress Gateway 和 Egress Gateway，下面會簡單實作試著控制流量的進出～

![Ingress Gateway 和 Egress Gateway](https://github.com/user-attachments/assets/e571cc3a-652d-41d4-9e76-9f0445d02926)

## Gateway

Istio 的 Gateway 本質上是一個 envoy proxy 的 Deployment + Service，透過 Gateway + VirtualService CRD 控制流量進出，Ingress Gateway 通常部署在 istio-system namespace，以 LoadBalancer / NodePort / ClusterIP + port-forward 方式暴露，Egress Gateway 需要 Explicit Configuration，讓內部流量強制經過 Egress，再進行外部存取

## 那跟 k8s service 相關服務又有什麼差異呢？

| 功能面向           | K8s Service             | Istio Ingress Gateway                     | Istio Egress Gateway            |
| ------------------ | ----------------------- | ----------------------------------------- | ------------------------------- |
| **流量路由**       | 只能做 L4 負載均衡      | L7 路由（HTTP header、path、host）        | L7 路由（外部 API domain、URI） |
| **安全防護**       | 無法檢查 request / user | 支援 TLS termination、AuthorizationPolicy | 可限制白名單                    |
| **進出 Mesh 控制** | 沒有                    | 控制外部 → Mesh                           | 控制 Mesh → 外部                |
| **Observability**  | 只能看 Service 狀態     | 內建 Telemetry、Tracing                   | 內建 Telemetry、Tracing         |

## Ingress Gateway 實戰

建立 Gateway

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: bookinfo-gateway
  namespace: default
spec:
  selector:
    istio: ingressgateway
  servers:
    - port:
        number: 80
        name: http
        protocol: HTTP
      hosts:
        - "*"
```

建立 VirtualService

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: bookinfo
spec:
  hosts:
    - "*"
  gateways:
    - bookinfo-gateway
  http:
    - match:
        - uri:
            prefix: /productpage
      route:
        - destination:
            host: productpage
            port:
              number: 9080
```

接下來就是執行啦！

```bash
kubectl apply -f gateway.yaml
kubectl apply -f virtualservice.yaml
kubectl get svc -n istio-ingress istio-ingressgateway
```

```bash
kubectl port-forward -n istio-ingress svc/istio-ingressgateway 8080:80
# 我發現要看一開始的設定，`helm install istio-ingressgateway istio/gateway -n istio-ingress` 要看當初建立在哪個 namespace 下面
curl -I http://localhost:8080/productpage
```

這樣就有順利的啟動了

```bash
HTTP/1.1 200 OK
server: istio-envoy
date: Wed, 01 Oct 2025 03:33:11 GMT
content-type: text/html; charset=utf-8
content-length: 7712
vary: Cookie
x-envoy-upstream-service-time: 95
```

## Egress Gateway 實戰

建立 ServiceEntry，那我們要先了解 ServiceEntry 又是什麼？他像是 Istio 的「補充通訊錄」，在 k8s 中 istio 會自己知道有哪些 service，那今天假如你有其他服務是需要連接到外部的 API 或是有其他的服務是在外部的，這樣就要透過 engress 的方法去將 ServiceEntry 加入，這樣 istio 就可以幫忙做到 traffic management 等等的機制

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: httpbin-ext
spec:
  hosts:
    - httpbin.org
  ports:
    - number: 80
      name: http
      protocol: HTTP
  resolution: DNS
```

> resolution 代表 Istio 在處理 ServiceEntry 的流量時，要怎麼決定 endpoint，通常有三個比較常使用：`NONE` 是沒有做 endpoint resolution，通常用於 direct IP 或一些不需要解析的情境或是一些 wildcards 的 hosts (\*.bar.com); `STATIC` 直接使用靜態配置的 endpoint IP; `DNS` 透過 DNS 來解析 host，取得實際的外部 IP，如果沒有指定的 endpoint，會 proxy 到指定的 DNS address，但 DNS 無法解析 Unix domain socket 的 endpoints (api.dropboxapi.com)

建立 Egress Gateway

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: istio-egressgateway
  namespace: istio-system
spec:
  selector:
    istio: egressgateway
  servers:
    - port:
        number: 80
        name: http
        protocol: HTTP
      hosts:
        - httpbin.org
```

建立 VirtualService，並強制流量經過 Egress Gateway

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: route-via-egressgateway
spec:
  hosts:
    - httpbin.org
  gateways:
    - mesh
    - istio-egressgateway
  http:
    - match:
        - port: 80
        gateways:
        - mesh
        route:
        - destination:
            host: istio-egressgateway.istio-system.svc.cluster.local
    - match:
        - uri:
            prefix: /get
      route:
        - destination:
            host: httpbin.org
            port:
              number: 80
```

然後就可以測試呼叫外部的 api（httpbin.org）

```bash
kubectl exec -it <some-pod> -c istio-proxy -- curl -s http://httpbin.org/get
```

```json
{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "Host": "httpbin.org",
    "User-Agent": "curl/8.5.0",
    "X-Amzn-Trace-Id": "Root=1-68dca294-0d54337825d956b766aff9bb"
  }, // AWS 內部做 request tracing
  "origin": "103.122.117.**", // origin 顯示的是出口 IP
  "url": "http://httpbin.org/get"
}
```

## 總結

今天就補充了前幾天沒講完的進出流量的控管，當然還有更多細節的管控可以去操作～

## Reference

https://istio.io/latest/docs/reference/config/networking/service-entry/
