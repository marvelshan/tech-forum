## K8S Lab Day_23

# Day21: Istio 的 Health Checking

## 前言

昨天介紹如何把 VM 加入到 istio 中，在其中有提到 probe，那天就來看看說 k8s 有 livenessProbe 與 readinessProbe 會跟 kubelet 報告他們的健康狀態，那 istio 呢？當 sidecar 成為進出口後，health check 的責任就不只是在 application 本身，還要考慮到 envoy proxy 和 application 整合的關係～

## Istio health check 會遇到的挑戰

當 Istio sidecar 注入後，k8s 的 httpGet probe 請求也會先經過 Envoy，那這樣 k8s 的 probe 有可能就無法正確的回送狀態給 kubelet 而被 Enovy 攔截; Envoy 的啟動會比 application 慢，這樣也會導致誤判為 unhealthy

## 1. 關閉 probe rewrite

Istio 預設會將 Pod 的 httpGet probe 自動重寫，讓探針請求走過 Envoy sidecar，但這有時會造成誤判，利用 annotation 的方法來關閉 probe rewrite，這樣 probe 能避免被 Envoy 影響

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: liveness-http
spec:
  selector:
    matchLabels:
      app: liveness-http
      version: v1
  template:
    metadata:
      labels:
        app: liveness-http
        version: v1
      annotations:
        sidecar.istio.io/rewriteAppHTTPProbers: "false"
    spec:
      containers:
        - name: liveness-http
          image: docker.io/istio/health:example
          ports:
            - containerPort: 8001
          livenessProbe:
            httpGet:
              path: /foo
              port: 8001
            initialDelaySeconds: 5
            periodSeconds: 5
```

## 2. 延遲 container 啟動時間

這個設定名詞取的蠻直覺的，他就是 hold 住，等 proxy 啟動，這樣的好處就是他可以確認 proxy 也正確的啟動，probe 也可以正確的監控 container 的狀態

```yaml
annotations:
  proxy.istio.io/config: '{"holdApplicationUntilProxyStarts": true}'
```

## 總結

以上的方法是不是就能更安全的啟動 istio 了吧～不然 health check 不到也會是個大問題，明明在沒有 istio 之前就好好的，加上 istio 之後卻一直重啟...?

## Reference

https://istio.io/latest/docs/ops/configuration/mesh/app-health-check/

https://hackmd.io/@vincent960/S1dmV4zTs
