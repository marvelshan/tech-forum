## K8S Lab Day_5

<img width="837" height="289" alt="截圖 2025-09-17 下午3 46 19" src="https://github.com/user-attachments/assets/78f296f6-ded4-4f70-ad0d-d87baf935429" />

### Service

Kubernetes 裡面，Service 是一種抽象化的資源，主要解決兩個問題：

- Pod IP 會隨著重新建立而改變，不好固定
- 我們需要一個穩定的方式去存取一組 Pod（通常是一個 Deployment 裡的多個 Pod）

所以 Service 就像是 Pod 前面的一個「門牌」，讓外界（或者 cluster 內部的其他 Pod）能夠透過固定的方式去連到它後面那一組 Pod

#### 1. ClusterIP（預設值）

- 運作方式：會建立一個 虛擬 IP（Cluster IP），這個 IP 只能在 cluster 內部使用
- 適用情境：服務只要給 cluster 內的 Pod 存取，比如後端 API、資料庫，或者內部微服務之間的互通
- 存取範圍：只能在 Kubernetes 內部網路用，外部的使用者（例如你的筆電或外部網頁）是無法直接連到的
- 例子：
  - 你建立了一個 `backend-service`，型態是 `ClusterIP`。
  - 這個 service 會有一個 cluster 內的 IP
  - cluster 裡的其他 Pod，可以透過 `http://backend-service:80` 或這個 IP 去連線。

→ 可以把它想成「公司內線分機號碼」，只能在公司內部打。

#### 2. NodePort

- 運作方式：在 cluster 裡每一個 node 的指定 port（30000–32767 範圍）開一個對外入口，然後把流量導到後面的 Pod。
- 適用情境：當你需要從 cluster 外部，直接存取某個服務，但還沒有設定 Ingress 或 LoadBalancer。
- 存取範圍：外部用戶可以透過 `http://<NodeIP>:<NodePort>` 存取服務。
- 例子：

  - 你建立了一個 `frontend-service`，型態是 `NodePort`。
  - 它會自動在 cluster 每個 node 的 30080 port 打開對應的入口。
  - 外部使用者只要知道其中一台 node 的 IP，就能用 `http://<NodeIP>:30080` 連到這個服務。

→ 可以把它想成「公司接待處的電話總機」，外面的人打進來會被導到對應的部門。

### Namespace

在 Kubernetes 裡，Namespace 就像是把一個大社區，切分成不同的「社區分區」，雖然大家都住在同一個 Kubernetes cluster 裡，但透過 namespace 可以做到 隔離與管理，來用個新世紀福音戰士來比擬：

- 碇真嗣的 namespace 裡有他的 EVA 初號機、同步率數據、情緒狀態。
- 綾波零的 namespace 裡有零號機、她的特殊同步數據、個人任務記錄。
- 明日香的 namespace 裡有二號機、她的攻擊模式、她的戰鬥日誌。

如果沒有 namespace 呢？ 所有駕駛員的「同步率」就會塞在一起，真嗣的同步率變成 40%，可能會直接覆蓋掉明日香的同步率，導致二號機出問題

而這邊我們也會想到 container 的 Cgroups，而這兩個是 container 的核心技術，namespace 是作為隔離資源的，而 Cgroups 是來限制資源的，但這邊就很好奇了啊，那為何在 k8s 中我們只會操作到 namespace 呢？因為這邊其實是底層的 container runtime 所完成的，像是 CRI 或 containerd，像是我們常常使用 `resources.limits` 和 `resources.requests` 在限制資源，他其實就會告訴底層的 runtime 要去完成 cgroup 的限制

#### namespace 可做到哪些：

- 資源管理： namespace 可以綁定配額，限制這個 namespace 底下的 Pod 最多用多少 CPU、Memory，防止某一組服務吃光 cluster 的資源

- RBAC 的存取控制： 設定某個團隊的帳號，只能操作特定 namespace

#### Kubernetes default 的 Namespace

- default → 沒特別指定 namespace 的資源，都會放在這裡
- kube-system → Kubernetes 自己用的系統元件（像 kube-dns、metrics-server）
- kube-public → 公開資訊，cluster 裡任何人都能讀取
- kube-node-lease → 管理 node 心跳用

### Storage

在 Kubernetes 裡，Pod 本身的生命週期是短暫的，當刪掉一個 Pod，它裡面的資料也會跟著消失，並且 Pod 重新建立後，IP、檔案系統可能都不一樣，這樣很不方便，特別是像資料庫（MySQL、PostgreSQL）、檔案上傳系統這種應用，如果沒有一個穩定的儲存空間，資料就會消失不見

- PersistentVolume (PV)：把底層的存儲設備抽象化，可能是本地磁碟（local storage）、NFS、雲端磁碟（AWS EBS、GCP PD、Ceph 等），像是「公司倉庫裡的實體儲物櫃」
- PersistentVolumeClaim (PVC)：Pod 不能直接去跟 PV 要空間，而是透過 PVC 這個「申請單」來要，Pod 會說「我需要一個 5GiB 的空間」，PVC 會幫它找對應的 PV，就像員工要去申請一個儲物櫃，透過表單（PVC）拿到櫃子（PV）
- StorageClass：不用事先準備好很多 PV，而是當 Pod 需要的時候，StorageClass 幫你動態產生，在 AWS 中，可以定義一個 StorageClass，規則是「建立 gp2 類型的 EBS Volume」，當 Pod 提出 PVC 時，就會自動建立一個新的 EBS，然後掛到 Pod 上

那實際的建立順序呢？ 會有兩種情況：

- 靜態配置：先建立 PV，再建立 PVC，StorageClass 可省略，因為 PV 是你手動先建好的
- 動態配置：先建立 StorageClass，建立 PVC，Kubernetes 自動根據 StorageClass 建立 PV 並綁定

### ConfigMap

除了要跑應用程式的 Pod 以外，我們常常還需要一些「設定檔」或「環境變數」，像是資料庫的連線 string，API 的 URL，應用程式的 env，而 ConfigMap 可以讓我們把「設定」和「程式碼」分開，這樣應用程式要改設定時，不需要重建 image，只要更新 ConfigMap 就好

### Secret

在 k8s 中，很常會需要存放敏感的資料，像是資料庫密碼，API Key，TLS certificate，如果直接硬寫在 YAML 裡，會很危險，Secret 預設會用 Base64 編碼存起來（如果未加密 etcd，Secret 的 Base64 編碼資料在 etcd 中仍然是可讀的），主要是避免在設定檔中明碼顯示，如果要提高安全性通常會搭配外部工具像是 HashiCorp Vault 或 KMS
