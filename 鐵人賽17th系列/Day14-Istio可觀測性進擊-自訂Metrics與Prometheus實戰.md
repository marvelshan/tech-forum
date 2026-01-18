## K8S Lab Day_16

# Day14: Istio 可觀測性進擊，自訂 Metrics 與 Prometheus 實戰

## 前言

昨天大概簡單介紹了 Istio Telemetry API 的基本概念，接著要來做一下 Metrics 的部分，在微服務架構中，Metrics 是觀察系統狀態最直觀的方式之一，能幫助我們快速發現延遲、錯誤率、請求量等異常行為，Istio 預設已經會收集一組標準的 Metrics，但有些情境下我們需要加入自訂的標籤（tag）或修改現有的度量，這就要透過 Telemetry API 來實作

## Customizing Istio Metrics

### 安裝 Prometheus

首先要來安裝 Prometheus，Istio 官方已經提供了現成的範例：

```bash
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.27/samples/addons/prometheus.yaml
```

這會在 `istio-system` namespace 裡安裝 Prometheus ，會透過它來驗證自訂 Metrics 是否有生效

### 新增自訂標籤到 Metrics

接著，我們要針對 `REQUEST_COUNT`（對應到 Prometheus 指標 `istio_requests_total`）新增兩個自訂的標籤：

- `request_host`：請求的 Host（例如 productpage.default.svc.cluster.local）
- `destination_port`：請求目的地的 Port

這樣可以讓我們更細緻地分析 Gateway 和 Sidecar 在 inbound / outbound 方向發出的流量～

建立 `custom_metrics.yaml`：

```bash
# 這邊是 heredoc 語法，把後續內容當成 stdin 傳給 cat，再寫入檔案
cat <<EOF > ./custom_metrics.yaml
apiVersion: telemetry.istio.io/v1
kind: Telemetry
metadata:
  name: namespace-metrics
spec:
  metrics:
  - providers:
    - name: prometheus
    overrides:
    - match:
        metric: REQUEST_COUNT
      tagOverrides:
        destination_port:
          value: "string(destination.port)"
        request_host:
          value: "request.host"
EOF
# 這邊使用到昨天說的 overrides 來修改現有指標，新增自訂標籤

kubectl apply -f custom_metrics.yaml
```

`istio_requests_total` 就會多出兩個新的標籤維度

---

### 建立測試流量

然後我們建立一個臨時通道，把流量導到本地端來測試

```bash
# 把外部流量導到 Ingress Gateway
kubectl port-forward -n istio-system svc/istio-ingressgateway 8080:80

# 同時也把 Prometheus 開到本地 9090
kubectl -n istio-system port-forward svc/prometheus 9090:9090
```

接著發送測試請求：

```bash
curl -I http://localhost:8080/productpage
```

### 驗證 Metrics

要確認剛剛新增的標籤是否生效，可以直接從 Envoy sidecar 拉 Prometheus 格式的 metrics：

```bash
kubectl exec "$(kubectl get pod -l app=productpage -o jsonpath='{.items[0].metadata.name}')" \
-c istio-proxy -- curl -sS 'localhost:15000/stats/prometheus' | grep istio_requests_total
```

如果設定正確，你會看到類似這樣的輸出：

```json
istio_requests_total{
    reporter="destination",
    source_workload="istio-ingressgateway",
    source_canonical_service="istio-ingressgateway",
    source_canonical_revision="latest",
    source_workload_namespace="istio-system",
    source_principal="spiffe://cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account",
    source_app="istio-ingressgateway",
    source_version="unknown",
    source_cluster="Kubernetes",
    destination_workload="productpage-v1",
    destination_workload_namespace="default",
    destination_principal="spiffe://cluster.local/ns/default/sa/bookinfo-productpage",
    destination_app="productpage",
    destination_version="v1",
    destination_service="productpage.default.svc.cluster.local",
    destination_canonical_service="productpage",
    destination_canonical_revision="v1",
    destination_service_name="productpage",
    destination_service_namespace="default",
    destination_cluster="Kubernetes",
    request_protocol="http",
    response_code="200",
    response_flags="-",
    connection_security_policy="mutual_tls",
    destination_port="9080",
    request_host="localhost:8080"
} 3
// 代表目前累積 3 次成功請求
```

這代表我們的 `request_host` 與 `destination_port` 標籤已經成功加入到 Metrics

可以再試試看把 yaml 刪了會發生什麼事喔

```bash
kubectl delete -f custom_metrics.yaml
```

### Istio Telemetry v2 的 metric override 表達式語法

在 v2 裡面少了 | operator，但它還是允許用類似三元運算子的方式處理值

```yaml
tagOverrides:
  request_host:
    value: 'has(request.host) ? request.host : "unknown"'
```

## 結論

這邊簡單介紹了 metrics 的用法，也嘗試了取得 metrics 的方法，那下一步呢？就是繼續獲得更多的資料啦～～

## Reference

https://istio.io/latest/docs/ops/integrations/prometheus/
