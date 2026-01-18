## K8S Lab Day_9

# Day7: Service Mesh 維運戰術，EVA 駕駛員的網路同步率指南

## 前言

介紹完 K8S 和 Nix，我們要來介紹這次的主題 Service Mesh，這個不外乎就是網路，還是網路，再來就是網路，在微服務架構裡，每個服務就像是一個小積木，它們之間需要不停地交換資料，當服務數量還很少的時候，我們可以用最單純的方式：A 直接呼叫 B，B 再呼叫 C，但一旦規模變大，這些服務之間的溝通就會變得錯綜複雜，像是我要怎麼保證服務之間的流量是安全的？如果某個服務臨時壞掉了，其他服務要怎麼優雅地處理？我要怎麼追蹤一個請求從入口一路走到哪裡出錯？這些問題都不單純只是「呼叫 API」而已，它牽涉到流量控制、負載平衡、安全性、觀察性，而 Service Mesh 就是為了這些「服務與服務之間的網路問題」而生的解法

## 那麼 Service Mesh 是什麼？

Service Mesh 提供一個「透明的網路層」，讓開發者不用在業務程式裡寫一堆錯誤處理、重試、認證邏輯，而是交給 Mesh 來處理。這樣一來，開發者只要專心寫業務邏輯，網路相關的麻煩事，就交給 Mesh

換個角度想，在微服務世界裡，每個服務就像是一台 EVA 駕駛員要操縱的機體。本來我們以為只要駕駛員（開發者）跟 EVA（服務）就能對抗使徒（使用者需求 or 外部請求），但實際上情況超複雜，像是 EVA 出擊時需要同步率連結，這就像服務之間要能順利通訊、如果同步率太低，駕駛員可能無法控制 EVA，就像 API 呼叫 timeout 或失敗、當多台 EVA 一起作戰，誰先上、誰掩護、誰負責正面衝撞，就像微服務之間的流量路由和負載平衡

### 那 Service Mesh 又是啥？

Service Mesh 的核心概念是把服務間通訊的控制權抽離到一層基礎設施，而不是交給應用程式自己處理

1. Data Plane：就是 Sidecar Proxy，每個 Pod/Service 旁邊都會有一個 Proxy，專門負責接收、轉送流量，應用程式只管「寫業務邏輯」，不用管網路細節

2. Control Plane：負責統一管理這些 Proxy，設定路由規則、監控流量、開啟 mTLS 加密

### 那 Service Mesh 能做什麼？

1. 流量管理：Gray release，只把 10% 的流量導到新版本、藍綠部署、重試、超時、斷路器

2. 安全性：mTLS 加密、驗證與授權

3. 可觀察性：自動收集服務之間的指標、提供 tracing

---

> 那你可能會想：「Kubernetes 本身不是也有 Service、Ingress、NetworkPolicy 嗎？為什麼還需要 Service Mesh？」

對，但 K8S 無法幫你做每個服務的微流量控制、細粒度安全或自動監控，尤其是當服務很多、流量複雜時，每個服務要自己加重試、超時、斷路器邏輯，程式碼就會被「網路處理邏輯」淹沒，若想加密流量或驗證呼叫者身份，每個服務都要自己實作 mTLS 或 Token 驗證，想追蹤一次請求跑了哪些服務，必須每個服務自己打點、收集、整理，Service Mesh 就幫我們把這些「非業務邏輯」的事情統一抽離，開發者只要專心寫功能，網路問題交給 Mesh 處理

接著我們就要繼續提到 William Morgan，他在 2016 年寫了一篇文章或提出概念，將「微服務的通訊問題」系統化，並提出一個框架化的解法，這個想法正是後來 Service Mesh 的概念原型，我們來引用他說的話

> A Service Mesh is a dedicated infrastructure layer for handling service-to-service communication. It’s responsible for the reliable delivery of requests through the complex topology of services that comprise a modern, cloud native application. In practice, the Service Mesh is typically implemented as an array of lightweight network proxies that are deployed alongside application code, without the application needing to be aware.

他強調微服務架構中，每個服務不應該自行處理通訊細節，而應將通訊功能抽象到 mesh layer，由統一的 sidecar 代理管理

在實際運作上，Service Mesh 會用 Sidecar 來幫忙處理所有服務之間的通訊細節，以 Istio 為例，它會在每個 Pod 裡放一個 Envoy 當 sidecar 代理，當一個服務發出請求時，sidecar 會先判斷要把請求送到哪裡，是送到 production 環境、testing 環境還是 staging 環境？這些路由規則都是可以動態調整的，控制平面會統一下發配置，既可以是 Global Configuration，也可以針對某些服務單獨設定，sidecar 確認目的地後，會把流量送到 k8s 的 Service，再由 Service 將流量轉發給 pod，sidecar 會根據它觀測到最近的延遲時間，選則觸發最快的 pod，這樣能確保用戶請求的延遲最小，也減少服務壓力，請求送出去後，sidecar 會記錄觸發類型和延遲，如果目標容器掛掉、不回應或進程異常，sidecar 會自動把請求轉到其他可用 pod 重試，如果某個容器持續返回錯誤，sidecar 會把它從負載均衡池中暫時移除，過一段時間再重試，如果請求的 deadline 已過，sidecar 會主動標記這個請求為失敗，而不是一直重試增加負載，sidecar 會以 metric 和分布式追蹤的方式，捕捉這些操作的各個細節，並將資料發送到集中式的 metric 系統，這樣我們就能知道哪些服務慢、哪個實例不穩定、流量分布情況等，方便觀察和排錯

<img width="1010" height="386" alt="截圖 2025-09-20 晚上9 08 01" src="https://github.com/user-attachments/assets/1c3683f3-539c-4f5f-ac98-2415d18f6c34" />

## 總結

開發者只要像駕駛 EVA 專心打使徒，Service Mesh 幫你處理協作、指揮、後勤支援，保證每台 EVA 都能穩定、快速、安全地完成任務 XD

## Reference

https://ithelp.ithome.com.tw/articles/10217196

https://ithelp.ithome.com.tw/articles/10301328

https://ithelp.ithome.com.tw/articles/10289605

https://jimmysong.io/blog/what-is-a-service-mesh/
