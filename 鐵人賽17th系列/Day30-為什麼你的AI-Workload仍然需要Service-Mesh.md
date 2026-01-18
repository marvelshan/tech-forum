## K8S Lab Day_32

# Day30: 為什麼你的 AI Workload 仍然需要 Service Mesh

## 前言

走到最後一天，說長不長、說短不短，今天想談的主題是當前最火熱的「AI」，或許你會想 Service Mesh 跟 AI 有什麼關聯？其實還真有，與其說這是追潮流，不如說是觀察技術如何演進與適應，AI 的應用雖然多半仍以 HTTP 協定為基礎，但它們對安全性（Security）、流量控制（Traffic Control）與可觀測性（Observability）的要求卻更高、更複雜，並不是「AI 一定需要 Service Mesh」，而是「使用 Service Mesh 可以讓 AI Workload 更容易被管理與調優」，Service Mesh 的核心價值在於提供一個可觀察、安全且高效能的 Data Plane，而這通常是透過 Envoy Proxy 以 Sidecar 形式實現的

不過，當這個網格開始承載 LLM 的請求流量時，傳統的 Sidecar 架構就會面臨新的挑戰，像是高頻率的 streaming response、long-lived connections、大流量 Token 傳輸與延遲敏感度問題，都在重新定義 Mesh 在 AI 場景中的角色

## Envoy 來打造智能的 Service Mesh Sidecar

為了讓 Service Mesh 成為部署和消費 AI 服務的最佳基礎設施，Envoy 必須進化，從一個單純的 L7 代理，變成一個能感知 Application Semantics 的智能 Data plane

### Model-Aware Routing

模型名稱通常隱藏在 HTTP 請求的 Payload 內，在傳統的 Envoy Sidecar 路由規則是基於 Header 或 Path 無法讀取 Payload，因此難以實現精準路由，因此利用 Envoy 的 X-Authz 外部授權擴充功能，將 Payload Parsing 的工作外部化，Sidecar 可以在請求進入後端服務前，調用外部服務解析 JSON Body，提取出目標模型名稱，並將其作為一個新的 Header 附加到請求中，這個的好處是可以透過不同的模型名稱來將不同的流量導到不同的 mesh 裡面，可以做到不同 mesh 之間的 A/B 測試或 Rate Limiting 等等的流量控制策略

### Inference-Optimized Load Balancing

對 LLM 來說，瓶頸是 KV Cache 的利用率和 Pending Request Queue 的大小，而非傳統的 CPU 或連線數，Google 引入了 Orca 機制，讓 LLM 服務能夠透過 HTTP Response Headers 等方式，將這些內部資源狀態的負載訊號主動回報給 Envoy Sidecar，Sidecar 獲得 Orca 訊號後，可以採用更智能的負載平衡演算法（例如客戶端加權 Round Robin），避免將新的請求導向已經接近飽和的 LLM 實例，從而顯著降低請求的 Tail Latency，並最大化昂貴的 GPU 資源利用率

## 總結

我其實也不是 AI master，看了 EnvoyCon 也開始了解對於 ai 和 service mesh 可應用的狀況，因為 service mesh 也不是萬能的，所以可以使用到 extension 的方式來去做到更多的應用

## 最後一天

非常開心這次能夠參與鐵人賽的這個活動，這個期間也去了蠻多台灣的城市，邊寫做邊做實驗和旅遊真的是一個很舒服的一件事，放鬆了好一段工作壓力的心情，也抱持的熱情持續的接觸這些技術，未來還是會想繼續參加吧，研究一門技術真的是一個很好玩的一件事，這三十天過後還是會繼續寫文章，但應該就會更新在我的 [github](https://github.com/marvelshan/K8sLab/) 上面，但是後面應該就會比較無釐頭，想寫什麼就寫什麼，更自由一些，感謝現在的自己啦！

## Reference

https://developers.redhat.com/articles/2025/06/16/how-use-service-mesh-improve-ai-model-security

https://blog.howardjohn.info/posts/ai-mesh/

https://www.youtube.com/watch?v=wWbxdaTqnA4
