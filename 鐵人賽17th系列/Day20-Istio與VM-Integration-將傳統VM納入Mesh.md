## K8S Lab Day_22

# Istio 與 VM Integration：將傳統 VM 納入 Mesh

## 前言

昨天講到了假如在 multi-cluster 的狀況下需要怎麼設定流量的運作，今天會講到假如舊有的服務是 VM，今天很難做搬運，那假如我們又想要讓 VM 擁有 istio 帶來方便的能力，我們應該怎麼去設定～

## 為什麼要把 VM 納入 Mesh？

在安全性方面，透過 mTLS 與 Istio CA，確保 VM 與 Pod 的通訊加密且雙向驗證身份，不只是安全，我們也可以透過 vertualservice、destinationrule、authorizationpolicy 等等更好地控制進入 VM 的流量，接著前面也有提到 observability 更方便地去做統一的監控

1. WorkloadEntry

WorkloadEntry 用於將一個單一、非 k8s 註冊到 istio mesh 中，其實可以想像成一個 pod 但是用不同的方式來表示，流量的流程為

![WorkloadEntry](https://github.com/user-attachments/assets/649865e1-ba15-4ed6-98c7-87bd078576d7)

```
Pod → VirtualService (指向 WorkloadEntry) → Istio Sidecar → VM → VM Service
```

```yaml
apiVersion: networking.istio.io/v1beta1
kind: WorkloadEntry
metadata:
  name: details-svc
  namespace: default
spec:
  address: 10.20.30.40 # VM 的 IP
  labels:
    app: details-legacy
    instance-id: vm1
  ports:
    http: 9080
```

前面也有提到 serviceentry 這邊就需要把 VM 的 service 註冊進來讓內部服務的流量可以送進來

```yaml
apiVersion: networking.istio.io/v1
kind: ServiceEntry
metadata:
  name: details-svc
spec:
  hosts:
    - details.bookinfo.com
  location: MESH_INTERNAL
  ports:
    - number: 80
      name: http
      protocol: HTTP
      targetPort: 8080
  resolution: STATIC
  workloadSelector:
    labels:
      app: details-legacy
```

2. WorkloadGroup

那今天假如有多台 VM 呢？就可以使用 workloadgroup 來進行管理，其實可以想像成 deployment，把所有的 attribute 和基礎資料都放進來

```yaml
apiVersion: networking.istio.io/v1
kind: WorkloadGroup
metadata:
  name: reviews
  namespace: bookinfo
  labels:
    app: ratings-vm
spec:
  template:
    address: 2.2.2.2
    ports:
      grpc: 3550
      http: 8080
    labels:
      app: ratings-vm
      class: vm
    serviceAccount: bookinfo-ratings
  probe: # health check configuration: 定義了這些工作負載的 K8s 健康檢查探針
    initialDelaySeconds: 5 # 應用程式啟動後，延遲 5 秒才開始第一次檢查
    timeoutSeconds: 3 # 每次檢查的超時時間為 3 秒
    periodSeconds: 4 # 探針檢查的間隔時間為 4 秒
    successThreshold: 3 # 檢查連續成功 3 次才視為健康
    failureThreshold: 3 # 檢查連續失敗 3 次才視為不健康
    httpGet:
      path: /foo/bar # 探針檢查的 URI
      host: 127.0.0.1 # 探針檢查的 ip
      port: 3100 # 探針檢查的 port
      scheme: HTTPS
      httpHeaders:
        - name: Lit-Header
          value: Im-The-Best
---
apiVersion: networking.istio.io/v1
kind: WorkloadEntry
metadata:
  name: reviews-vm1
  namespace: bookinfo
spec:
  address: 2.2.2.2
  labels:
    class: vm
    app: ratings-vm
    version: v3
  serviceAccount: bookinfo-ratings
  network: vm-us-east
---
apiVersion: networking.istio.io/v1beta1
kind: Sidecar # 幫 VM 配置 envoy proxy ingress 和 egress 的流量
metadata:
  name: bookinfo-ratings-vm
  namespace: bookinfo
spec:
  egress:
    - bind: 127.0.0.2
      hosts:
        - ./*
  ingress:
    - defaultEndpoint: 127.0.0.1:8080
      port:
        name: http
        number: 8080
        protocol: HTTP
    - defaultEndpoint: 127.0.0.1:3550 # 轉發到 VM 上的 gRPC 端口
        port:
        name: grpc
        number: 3550
        protocol: TCP
  workloadSelector:
    labels:
      app: ratings-vm
      class: vm
```

## 總結

這些都是蠻實用配置的方法，我也是看完文件之後在 github 上面找有沒有類似的 repo 也有相同的配置去更改，這個方式蠻推薦的，因為現在 ai 有時候對於這些 yaml 的配置還不是相當的了解，假如配合現在有人使用過的 configuration 的話會是一個相當好的學習方式～

## Reference

https://istio.io/latest/docs/reference/config/networking/workload-entry/

https://istio.io/latest/docs/reference/config/networking/workload-group/

https://github.com/kiali/kiali/blob/master/tests/integration/assets/bookinfo-workload-groups.yaml
