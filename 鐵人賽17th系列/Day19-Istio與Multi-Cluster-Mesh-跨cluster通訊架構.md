## K8S Lab Day_21

# Istio 與 Multi-Cluster Mesh：跨 cluster 通訊架構

## 前言

昨天回來講了一下 traffic 的細節部分，今天來講個 multi-cluster，是想一個狀況，今天的產品規模變大了，開始接觸了不同國家的客源，開始要將服務設立在其他的 Zone，或是供應的廠商覺得不能只設定在 AWS，必須要設立在 GCP 等等這些（無理）需求，multi-cluster 就變為更加的重要

## Istio Multi-Cluster 架構模式

比較常見的架構有分為兩種：

1. Single Control Plane, Multi-Cluster

一個叢集負責安裝 Istio control plane，其他的 cluster 只安裝 Data Plane，由統一的 control plane 去控制所有的 cluster 的 service mesh，使用場景比較像是在需要有 flat network

2. Multi Control Plane, Federated Mesh

每個 cluster 都有自己的 control plane，cluster 可以透過 gateway 來建立連線，主要場景就是有 multi cloud 或是跨區的部署

那接下來我們要怎麼將兩邊的流量打通呢？我們要先設定 gateway，這邊特別要注意 `tls.mode` 這裡的 AUTO_PASSTHROUGH 代表 gateway 不會終止 TLS，而是直接把流量轉給目標服務的 sidecar，確保跨 cluster 之間依然是 mTLS

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: eastwest-gateway
  namespace: istio-system
spec:
  selector:
    istio: eastwestgateway
  servers:
    - port:
        number: 15443
        name: tls-istio
        protocol: TLS
      tls:
        mode: AUTO_PASSTHROUGH
      hosts:
        - "*.local"
```

接著要告訴 cluster1 說我們這邊外面還有個服務喔，並用之前提到的 `ServiceEntry` 可以直接呼叫到 `reviews.default.svc.cluster2.local` 這樣 traffic 就可以直接經過 gateway 打到 cluster2 了

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: ServiceEntry
metadata:
  name: reviews-remote
  namespace: default
spec:
  hosts:
    - reviews.default.svc.cluster2.local
  location: MESH_INTERNAL
  ports:
    - number: 9080
      name: http
      protocol: HTTP
  resolution: DNS
```

![multi-cluster](https://github.com/user-attachments/assets/86ab969f-d900-4045-bc8f-5f2f301a7d7e)

## Reference

https://istio.io/latest/docs/ops/configuration/traffic-management/multicluster/

https://joehuang-pop.github.io/2020/08/23/Istio-%E5%A4%9A%E7%B5%84K8s%E5%8F%A2%E9%9B%86%EF%BC%8C%E5%AF%A6%E4%BD%9C%E7%B5%B1%E4%B8%80Istio-%E7%AE%A1%E7%90%86-Anthos-Shared-Control-Plane-in-Multi-Cluster/

https://istio.io/latest/docs/setup/install/multiple-controlplanes/

https://cloudnativecn.com/blog/istio-analysis-5/
