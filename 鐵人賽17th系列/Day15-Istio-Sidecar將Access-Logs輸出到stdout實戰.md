## K8S Lab Day_17

# Day15: Istio Sidecar 將 Access Logs 輸出到 stdout 實戰

## 前言

昨天我們做了簡單的 metrics 查看，今天做一個 Istio sidecar 直接打 log 到 stdout

### 1. ProxyMetadata

Istio 的 sidecar 需要設定 `proxyMetadata.ACCESS_LOG_FILE`，才能把 log 打到指定位置，建立一個 mesh-wide 的 ProxyMetadata

```bash
cat <<EOF | kubectl apply -n istio-system -f -
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  name: mesh-log-stdout
spec:
  meshConfig:
    defaultConfig:
      proxyMetadata:
        ACCESS_LOG_FILE: /dev/stdout
EOF
```

### 2. 測試的 `httpbin` 和 `curl`

```bash
kubectl apply -f samples/httpbin/httpbin.yaml
kubectl apply -f samples/curl/curl.yaml
```

### 3. 使用 Telemetry 設定

建立一個 Telemetry yaml，啟動 mesh-wide logging

```bash
cat <<EOF | kubectl apply -n istio-system -f -
apiVersion: telemetry.istio.io/v1
kind: Telemetry
metadata:
  name: mesh-logging-stdout
spec:
  accessLogging:
  - providers:
    - name: envoy
EOF
```

### 4. 發送請求觸發 log

```bash
SOURCE_POD=$(kubectl get pod -l app=curl -o jsonpath={.items..metadata.name})
kubectl exec "$SOURCE_POD" -c curl -- curl -sS http://httpbin:8000/get
```

多發幾次：

```bash
kubectl exec "$SOURCE_POD" -c curl -- curl -sS http://httpbin:8000/status/404
```

### 5. 查看 Access Logs

到 httpbin 的 sidecar 看

```bash
HTTPBIN_POD=$(kubectl get pod -l app=httpbin -o jsonpath={.items..metadata.name})
kubectl logs "$HTTPBIN_POD" -c istio-proxy | grep "GET /"
```

我們就會看到

```bash
[2025-09-25T06:03:57.982Z] "GET /get HTTP/1.1" 200 - via_upstream - "-" 0 640 2 1 "-" "curl/8.16.0" "0baabc4d-de4e-46dc-8e4e-044863e8b779" "httpbin:8000" "10.233.118.148:8080" inbound|8080|| 127.0.0.6:55269 10.233.118.148:8080 10.233.97.222:50034 outbound_.8000_._.httpbin.default.svc.cluster.local default
[2025-09-25T06:04:03.320Z] "GET /status/404 HTTP/1.1" 404 - via_upstream - "-" 0 0 1 0 "-" "curl/8.16.0" "43418893-0770-4bb0-bcb4-0ad31e07d930" "httpbin:8000" "10.233.118.148:8080" inbound|8080|| 127.0.0.6:49379 10.233.118.148:8080 10.233.97.222:50036 outbound_.8000_._.httpbin.default.svc.cluster.local default
```

### 最後就是要清理

```bash
kubectl delete -f samples/httpbin/httpbin.yaml
kubectl delete -f samples/curl/curl.yaml
kubectl delete telemetry mesh-logging-stdout -n istio-system
kubectl delete IstioOperator mesh-log-stdout -n istio-system
```

## 總結

這大概就是簡單的 log 測試啦～接著，有點累了，明天再想想要來做什麼吧

## Reference

https://istio.io/latest/docs/tasks/observability/logs/telemetry-api/
