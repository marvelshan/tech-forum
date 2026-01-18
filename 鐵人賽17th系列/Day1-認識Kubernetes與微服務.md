## K8S Lab Day_3

今天會是貼人賽的第一天，有鑒於現在 ai 盛行，我請他幫我生成了 30 天的計畫，能不能順利地執行完畢是一回事，但是盡力地跟上他的實作吧～
| 天數 | 題目 |
| ------ | ----------------------------------------- |
| Day 1 | 認識 Kubernetes 與微服務 |
| Day 2 | Master Node 與 Worker Node |
| Day 3 | Kubernetes 核心資源：服務、命名空間、儲存與配置管理 |
| Day 4 | 使用 Nix 打造可重現的 Kubespray Kubernetes 叢集 |
| Day 5 | Nix + Flake 你的開發環境 01 號機 |
| Day 6 | K8S 資源調度大作戰，從 QoS 防護網到 LimitRange 社區公約 |
| Day 7 | Service Mesh 維運戰術，EVA 駕駛員的網路同步率指南 |
| Day 8 | Istio 實戰開戰，從 Helm 部署到 Control Plane 的維運戰術 |
| Day 9 | Istio 安全防線實戰，mTLS Identity 與 Authorization Policy 的跨命名空間存取控制 |
| Day 10 | Istio 流量管理的戰術總覽與自動化 SSH 優化 |
| Day 11 | istio 的流量管理的戰術下集|
| Day 12 | Istio 流量戰術進階，使用 Envoy 實現速率限制與 Port Forward 快速測試 |
| Day 13 | Istio 可觀測性戰術解鎖，從 Mixer 到 Telemetry API 的深度觀察 |
| Day 14 | Istio 可觀測性進擊，自訂 Metrics 與 Prometheus 實戰 |
| Day 15 | Istio Sidecar 將 Access Logs 輸出到 stdout 實戰 |
| Day 16 | Istio Debug 實戰記錄 |
| Day 17 | Istio Gateway 戰術：Ingress / Egress Gateway 的配置與安全防護 |
| Day 18 | Istio Ingress Gateway 的進階流量管理：mTLS、TLS Termination 與 AuthorizationPolicy |
| Day 19 | Istio 與 Multi-Cluster Mesh：跨叢集通訊與聯邦架構 |
| Day 20 | Istio 與 VM Integration：將傳統 VM 納入 Mesh |
| Day 21 | Istio 的 Health Checking |
| Day 22 | Istio 與 CLI 整合：服務拓樸與流量觀察實戰 |
| Day 23 | Istio WASM Plugin 戰術：撰寫與掛載自訂過濾器 |
| Day 24 | Istio 與 Zero-Trust 架構：進階 AuthorizationPolicy 與 JWT 驗證 |
| Day 25 | Istio 與 Sidecar 資源調優：效能最佳化、Envoy Proxy Tuning |
| Day 26 | Istio 驗證錯誤與 Webhook 問題排查 |
| Day 27 | Istio 與 Kubernetes Admission Webhook：動態注入與驗證機制解析 |
| Day 28 | Istio 與 Service Mesh Interface (SMI)：標準化 API 與跨 Mesh 整合 |
| Day 29 | Istio 的漏洞案例觀察：Envoy CVE-2024-53270 |
| Day 30 | 為什麼你的 AI Workload 仍然需要 Service Mesh |

---

### 什麼是 K8S？

首先先來介紹什麼事 K8S 就是在 Kubernetes 中間有 8 個英文字母，未來會接觸到更多的單字，像是可觀測性 Observability 又可簡稱 o11y，那為何是 Kubernetes 這個單字呢？這個單字是來自希臘文，意思為「舵手，船長」，也是代表了這個工具管理了容器的部署，擴展等等的應用，就像是一位船長必須掌控著整艘船的運作正常。

那我們再回頭來講，會和需要管理這些容器，假如今天的應用場景是單容器，那我們簡單的使用 docker 去 build 一個 image，直接進行部署，這樣好像就不太會使用到，但今天假如服務慢慢地擴大，我們需要更多的容器，更加複雜的網路管理，這樣我們一個一個容器去做部署，好像就不那麼直覺，直接開機器開到死，當然這只是 k8s 其中一個優勢，還有他可以設定自動擴展，高可用的狀況等等。

那再回來繼續介紹 k8s，引用 document 裡面所說：

> Kubernetes is a portable, extensible, open source platform for managing containerized workloads and services, that facilitates both declarative configuration and automation. It has a large, rapidly growing ecosystem. Kubernetes services, support, and tools are widely available.

他是一個便於跨平台跨環境，可擴充的開源管理容器化服務的平台，他支援宣告式的配置，他是一個大且快速成長的生態系，並且有相當多的服務和支援的工具。

聽起來相當的不賴對吧，那接著要來說明為服務這個詞了。

### 為什麼適合微服務？

首先我們要先來認識微服務 microservices 這個詞，一個來看 document

> Microservices - also known as the microservice architecture - is an architectural style that structures an application as a collection of two or more services that are:
>
> - Independently deployable
> - Loosely coupled
>
> Services are typically organized around business capabilities. Each service is often owned by a single, small team.
> <img width="1019" height="443" alt="截圖 2025-09-15 下午1 33 34" src="https://github.com/user-attachments/assets/0d342f62-9ef1-4b76-904b-3b0e507c446c" />

是一種架構風格，它將應用程式設計為由兩個或更多服務所組成的集合，服務具有以下特性，可獨立部署，低耦合，服務通常是依照業務能力來劃分，每個服務通常由一個小型團隊負責。

看完文件，好像也沒比較好懂，會拿來比較的就會是單體式服務(Monolithic)，這種服務的架構設計相對單一，功能服務全部都在一個 service 裡面，不用思考他會跟其他的服務交互的運作，但缺點就是當服務越加龐大，scaling 的成本就會愈加困難，對於單一語言的依賴性就會更強。這時候就可以回來介紹微服務，各個功能會拆開不同的 service，像是一個售票系統，會員登入會是一個 service，結帳會是一個 service 等等，這樣未來我們再針對於服務的 bottleneck 時，我們就可以針對性的 scaling 單一服務，而不是整個服務一同 scaling，降低了擴展的成本，這個過程也產生相對應的缺點，整體的服務間相互溝通，當出錯時就會需要考慮多個服務的運作是哪裡出問題。

所以看完這些介紹，微服務就一定是最好的嗎？不一定，一切取決於當下的應用場景與未來的架構考量。

回到 k8s，有提到說當今天的大量的容器需要被開啟，那不是就跟微服務的架構不謀而合嗎～把不同的功能分為不同的微服務，並利用 k8s 的優勢去啟動和管理這些服務不是更加容易嗎～並且其中有很好的網路通信機制，讓我們能更佳容易的去管理這些微服務。

### 使用 `Kubespray`，啟動一個本地 K8s 叢集

接著就是開始實作的部分，我的環境是參考 Tico 大大的[文章](https://ithelp.ithome.com.tw/users/20112934/ironman/5640)去架設的，非常推薦觀看～
大部分都是按照大大的實作，我這邊就不多做說明了，因為在運行完以下指令可能會多花一點時間。

這邊介紹幾個指令：

```shell
test -f requirements-$ANSIBLE_VERSION.yml && \
ansible-galaxy role install -r requirements-$ANSIBLE_VERSION.yml && \
ansible-galaxy collection install -r requirements-$ANSIBLE_VERSION.yml
```

- 目的：Kubespray 是用 Ansible 部署 Kubernetes，所以需要依賴一些 Ansible roles 與 collections，最後確保部署環境有正確的依賴，避免 playbook 執行失敗。

- 步驟：

1. test -f requirements-$ANSIBLE_VERSION.yml：檢查該版本的 requirements.yml 是否存在，如果不存在就不執行後續指令。

2. ansible-galaxy role install -r requirements-$ANSIBLE_VERSION.yml：從 requirements.yml 安裝所有指定的 Ansible roles。

3. ansible-galaxy collection install -r requirements-$ANSIBLE_VERSION.yml：安裝 Ansible collections（可能包含 module、plugin 等）。

```shell
vi inventory/mycluster/inventory.ini
```

目的：指定你自己 Kubernetes cluster 的節點資訊和角色。

```yaml
[all]
k8s-m0 ansible_host=192.168.200.126 ansible_user=ubuntu
k8s-n0 ansible_host=192.168.200.54 ansible_user=ubuntu
k8s-n1 ansible_host=192.168.200.249 ansible_user=ubuntu
...
```

接著就是執行命令去安裝了！

```shell
ansible-playbook -i inventory/mycluster/inventory.ini --private-key=~/private.key --become --become-user=root cluster.yml
```

結果：

```
PLAY RECAP *********************************************************************
k8s-m0                     : ok=31   changed=3    unreachable=0    failed=0    skipped=58   rescued=0    ignored=0
k8s-n0                     : ok=26   changed=3    unreachable=0    failed=0    skipped=48   rescued=0    ignored=0
k8s-n1                     : ok=25   changed=3    unreachable=0    failed=1    skipped=48   rescued=0    ignored=0
localhost                  : ok=4    changed=0    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
```

因為我在開機器的時候因為 Quota 不夠，在 k8s-n1 這台機器只開啟了 d1.tiny 的機器，顯然 ram 只有 512MB 是不太夠的，這邊的問題是可以用以下指令去尋找說 `minimal_node_memory_mb` 這裡的最低 memory 是 1024MB 像我所設定的 512MB 顯然時不足夠的，這邊就要去調整 ram 的大小去符合他，調整完就可以順利啟動啦！

```
grep -R "minimal_node_memory_mb" inventory/mycluster/group_vars/ roles/kubernetes/preinstall/defaults/
roles/kubernetes/preinstall/defaults/main.yml:minimal_node_memory_mb: 1024
```
