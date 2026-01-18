## K8S Lab Day_30

# Istio 與 Service Mesh Interface (SMI)：標準化 API 與跨 Mesh 整合

## 前言

昨天講到了 webhook 的機制，今天來介紹一下 service mesh 在過去也是百家爭鳴，可能沒有百 XD，因為不同工具間有不同的實作方法，後來未來減少移轉的麻煩性，社群提出了 Service Mesh Interface 來統一 api 的 interface

## 1. SMI 是什麼？

Service Mesh Interface (SMI) 是一個開放標準，由 Microsoft、Buoyant、HashiCorp、Solo.io 等多家廠商共同維護，主要是為 k8s 上的 Service Mesh 提供一致的 API 標準、允許在不同 Mesh 之間重用同一套 YAML 定義、降低特定 Mesh 與使用者之間的耦合，SMI 並非新的 Mesh，而是抽象的一層，它將「Service Mesh 的功能」定義成 k8s CRD，而各個 implement 只需實現對應的行為即可

## 2. SMI 的 Non-goals

SMI 並不打算取代任何現有的 service mesh，也不會限制其功能範圍，它的宗旨是提供一個通用的 Minimal Common Subset，讓開發者可以在不同 Mesh 間使用一致的 API 操作邏輯，任何廠商都可以在 SMI 之外實作更多自有功能，未來若被社群廣泛採用，就可以納入 SMI 標準

## 3. SMI 的核心能力

1. Traffic Policy 允許對不同服務之間的連線設定身份驗證、加密與策略

2. Traffic Access Control 可根據 client side 的身份（ServiceAccount、Pod Label）控制對特定 Pod 的訪問權限

3. Traffic Specs 基於 HTTP、gRPC 等的流量結構，與策略控制結合，以達到更細緻的流量治理

4. Traffic Telemetry 用於收集如錯誤率、延遲、請求量等關鍵監控指標，方便與 Prometheus、Grafana 整合

5. Traffic Metrics 為外部工具（如自動擴縮容 HPA）提供統一的流量度量 API

6. Traffic Management 定義如何在不同版本的服務之間切換流量，例如 Canary、A/B Testing

7. Traffic Split 允許用百分比調整流量分配，用於漸進式部署、金絲雀發布或灰階部署

## 4. Traffic Split：SMI 核心 API 實例

TrafficSplit 是 SMI 的核心資源之一，用於描述如何在多個服務版本之間按比例分配流量，這個規範允許使用者逐步調整不同後端版本的流量百分比，實現如金絲雀發布（Canary Deployment）與 A/B 測試等場景，SMI 並不直接控制流量遷移的節奏，而是由外部控制器（如 Flagger 或 Argo Rollouts）根據指標動態修改 TrafficSplit 配置，以完成自動化流量轉移

```yaml
kind: TrafficSplit
metadata:
  name: canary
spec:
  # root service 名稱，客戶端將透過此名稱訪問
  service: website
  # 定義多個後端服務與其流量權重
  backends:
    - service: website-v1
      weight: 90
    - service: website-v2
      weight: 10
```

這邊定義所有針對 website 的請求會依照比例分流，90% 送往舊版本 website-v1，10% 送往新版本 website-v2

## 5. 那 `TrafficSplit` 又跟 `VirtualService` 有什麼差異

VirtualService 是 Istio 自身的流量管理核心，根據 HTTP Path、Header、Cookie、Query Param 進行路由分配，高階可以處理到 Retries、Timeout、Fault Injection、Mirror

TrafficSplit 比較單純的就是跨 Mesh 標準化的簡化 API，目標是統一 API 介面，讓不同 Mesh（如 Istio、Linkerd、Consul）都能理解相同的 Canary / A/B testing 語意，它不定義重試、延遲、Header match 等細節，他只關心「要分流哪些服務」、「流量比例是多少」

總結來說就是應用場景會使用到多 Mesh，像是 Linkerd + Istio 混合使用，想用 SMI Controller 做 Canary 自動化才會使用到 TrafficSplit

## 總結

今天講到的 SMI 我自己感覺主要是處理一些 legacy 或是特殊情況下才會使用到，因為假如已經判斷好要用哪個工具來建立的話其實就可以避免掉多 mesh 的場景，我自己的想法啦，我功力還太淺，也許未來就會更有相關的應用，有興趣的可以去看一下 servicemeshinterface/smi-spec 是怎麼操作這些 interface 的，不過他也在 Oct 20, 2023 archived 起來了，感覺一下他的操作也是蠻有趣的，後來 CNCF 也把重心轉到 GAMMA (Gateway API for Mesh Management and Administration)，感覺又是一個大鍋 XD

## Reference

https://release-v1-0.docs.openservicemesh.io/docs/guides/traffic_management/traffic_split/

https://linkerd.io/2-edge/features/traffic-split/

https://skyao.net/post/201906-service-mesh-interface-detail/

https://github.com/servicemeshinterface/smi-spec/blob/main/apis/traffic-split/v1alpha4/traffic-split.md

https://cloud.google.com/service-mesh/v1.26/docs/gateway/overview?hl=zh-tw
