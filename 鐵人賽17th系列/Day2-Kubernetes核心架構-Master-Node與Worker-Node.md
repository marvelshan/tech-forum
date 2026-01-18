## K8S Lab Day_4

### Kubernetes 核心架構：Master Node 與 Worker Node

今天要來探討 k8s 是由什麼組成的，k8s 的內部是由一個一個 node(節點)，來組成的，各個 node 負責自己的工作，並且這些 node 組合起來會形成 cluster，構成 k8s 的內部運作，其中就分為兩種主要的 node，一個是 master node，一個是 master node，負責整個系統的腦袋跟控制中心；另一個是 worker node，則是實際跑應用程式、做事情的地方。

那我們先從 master node 開始說。你可以把 master node 想像成是整個公司的老闆兼總經理。它本身不會直接去搬貨或處理產品，而是專門做「決策」跟「管理」。  
在 master node 裡面有幾個很重要的組件：

- API Server：就像公司的接待櫃檯，所有人（包含你這個管理員）要下指令，都要先經過它。
- Scheduler：排班組長，負責決定哪個 worker node 要去執行哪個工作。
- Controller Manager：比較像是專案經理，隨時確認東西有沒有照規劃運作，缺了什麼就去補上。
- etcd：這是一本超重要的記事本，所有的設定、狀態都會記錄在這裡。

接下來是 worker node。它們就像是一群認真工作的員工。老闆（master node）安排好任務之後，實際把應用程式跑起來、處理資料的，就是 worker node。  
在每個 worker node 裡，也有幾個重要角色：

- kubelet：它就像現場的班長，負責跟老闆回報狀況，並確保工作有被正確執行。
- kube-proxy：這個有點像網路交通警察，幫忙把外部的請求導流到正確的應用程式。
- Pod：這才是真正工作的員工，也就是實際跑應用程式的地方，Pod 裡面會包著一個或多個容器。  
  <img width="667" height="445" alt="截圖 2025-09-16 下午2 01 09" src="https://github.com/user-attachments/assets/005c3907-63ef-4d36-836e-63800c551199" />

假設我們現在要在 k8s 上面跑一個應用程式，整個流程大概會是這樣：

1. 使用者下指令  
   我們通常會用 `kubectl` 這個工具下指令，比如 `kubectl apply -f app.yaml`。這就像是我們把一份工作計畫書交給老闆，跟他說：「我想要跑一個服務，幫我安排一下。」

2. API Server 接收請求  
   指令送出去後，會先到 master node 的 API Server。這裡就像公司前台，負責接收所有請求，並檢查你講的話（指令）合不合法。

3. 存進 etcd  
   當 API Server 確認沒問題，會把這個「想要的狀態」記錄到 etcd 裡。etcd 就像一本超可靠的記事本，專門保存叢集的真實狀態。

4. Scheduler 安排工作  
   接著 Scheduler 會跳出來，看看目前有哪些 worker node 有空、有資源，然後決定要把這個應用程式丟到哪個 node 去跑。就像排班組長一樣，把人力資源安排好。

5. kubelet 接手任務  
   任務分派好之後，對應的 worker node 上的 kubelet（現場班長）就會接到通知：「欸，老闆要我們啟一個 Pod！」。kubelet 會根據指示，幫忙把 Pod 建起來，裡面再拉起對應的容器。

6. kube-proxy 負責網路流量  
   當 Pod 起來後，kube-proxy 會幫忙設定網路，把外部的流量導到這個 Pod 上，確保別人可以順利連進來使用這個服務。

7. 實際執行  
   最後，Pod 裡的容器開始運作，你的應用程式就正式跑起來了。這時候，你再用 `kubectl get pods` 查詢，就可以看到它在叢集裡活蹦亂跳。

總結一下：  
使用者下指令 → API Server 收到 → etcd 記錄狀態 → Scheduler 分配 → worker node 上的 kubelet 執行 → kube-proxy 處理網路 → Pod 真正跑起來。

## ![alt text](image.png)

### 實作環節

接著是繼續昨天實作的部分，昨天完成了 ansible 去啟動了 kubspray，後來到 vm 裡面發現沒辦法連到外網，發現了沒有開私有網段給 NAT 連出去，就解決了問題，成功設定成功，不然真的很躁 QQ

<img width="1135" height="723" alt="截圖 2025-09-16 下午2 02 26" src="https://github.com/user-attachments/assets/49eca180-6b25-4ffe-8f59-2a4ba7c25a85" />

等 Ansible 完成部署後，我們切換到 master node (m0) 做一些設定，讓本地的 kubectl 可以操作 cluster：

```
ubuntu@k8s-m0:~$ sudo cp /etc/kubernetes/admin.conf ~/
```

把 Kubernetes 初始化時生成的管理者設定檔 admin.conf 複製到家目錄，方便後續使用

```
ubuntu@k8s-m0:~$ sudo chown ubuntu:ubuntu ~/admin.conf
```

把檔案擁有權改成自己，避免每次操作都要用 sudo

```
ubuntu@k8s-m0:~$ mkdir -p .kube
```

在 local 目錄建立 .kube 資料夾，這是 kubectl 預設的設定資料夾

```
ubuntu@k8s-m0:~$ mv ~/admin.conf ~/.kube/config
```

把 admin.conf 移到 .kube/config，讓 kubectl 知道要用這個設定來管理 cluster

```
ubuntu@k8s-m0:~$ kubectl get node
```

最後，用 kubectl 查看叢集裡所有 node 的狀態，確認 master 與 worker node 都已經加入，且狀態是 Ready

但是，當我在 bastion 主機 上，啟動 kubectl 嘗試查看 cluster node 時：

```
(kubespray-venv) ubuntu@bastion-host:~/kubespray$ kubectl get nodes
The connection to the server 127.0.0.1:6443 was refused - did you specify the right host or port?
```

出現了錯誤，是 kubectl 嘗試連到 127.0.0.1:6443（也就是本地）去找 Kubernetes API Server，但在 bastion 上並沒有運行 Kubernetes，所以連不到

為了解決這個問題，我修改了 kubectl 的 config，把 API Server 的地址從本地改成 master node 的真實 IP：

```
(kubespray-venv) ubuntu@bastion-host:~/kubespray$ sed -i 's/127.0.0.1/192.168.200.126/' ~/.kube/config
```

這個指令會把 kubeconfig 裡的 127.0.0.1 全部替換成 master node 的 IP（192.168.200.126），這樣 kubectl 就會直接連到 master node 的 API Server

修改後，我用 netcat 確認 6443 port 可以連通：

```
(kubespray-venv) ubuntu@bastion-host:~/kubespray$ nc -zv 192.168.200.126 6443
Connection to 192.168.200.126 6443 port [tcp/*] succeeded!
```

結果顯示連線成功，表示 bastion 可以透過網路連到 Kubernetes API Server，接下來再執行 kubectl get nodes 就可以正常看到 cluster 裡的 node 了

<img width="931" height="180" alt="截圖 2025-09-16 下午2 01 44" src="https://github.com/user-attachments/assets/d3964c6d-e17d-4edf-9296-74877c62c642" />

#### 手動建立一個最簡單的 Pod

1. 建立 pod yaml

```
vi nginx-pod.yaml
```

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-pod
  labels:
    app: nginx
spec:
  containers:
    - name: nginx
      image: nginx:1.25
      ports:
        - containerPort: 80
```

2. 套用 Pod

```
ubuntu@bastion-host:~$ kubectl apply -f nginx-pod.yaml
pod/nginx-pod created
```

3. 使用 Port Forward 測試存取

```
ubuntu@bastion-host:~$ kubectl port-forward pod/nginx-pod 8080:80
Forwarding from 127.0.0.1:8080 -> 80
Forwarding from [::1]:8080 -> 80
```

這個指令會把本地機器（bastion host）的 8080 port，轉發到 Pod 裡面的 80 port

#### Reference

https://ithelp.ithome.com.tw/articles/10294526

#### 特別感謝

特別感謝 Tico，對於操作上遇到問題都很有耐心的回答！而且每個排錯的過程都觀察入微，小的學習到相當的多！
