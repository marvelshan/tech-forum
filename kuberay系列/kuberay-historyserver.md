## 一、什麼是 History Server？

### 定義

**History Server** 是一個 **離線/事後分析工具**，用來：

- 查看 **已結束（dead）Ray Cluster** 的任務（Tasks）、演員（Actors）、節點（Nodes）等詳細資訊
- 查看 **正在運行（live）Ray Cluster** 的即時狀態（需配合 Collector）

它本質上是一個 **Web API 服務**，提供 RESTful endpoints（如 `/api/v0/tasks`），前端可接 Grafana、自訂 UI 或直接用 `curl` 查詢。

> 注意：它 **不負責執行 Ray Job**，只負責「讀取並展示」已經產生的資料。

---

## 二、為什麼需要 MinIO？

### MinIO 的角色：**持久化儲存後端**

Ray 程式在執行時會產生大量 **事件日誌（event logs）**，例如：

- Task 建立、排程、完成
- Actor 生命週期
- Node 上下線
- Job 提交與結束

這些日誌預設只存在 **Ray Head Pod 的記憶體或臨時磁碟**（`/tmp/ray`），一旦 Pod 被刪除（如 Step 7 刪除 RayCluster），資料就永久消失。

### 解決方案：Collector + MinIO

1. **Collector Sidecar**

   - 在 Ray Head Pod 中額外跑一個容器（`collector:v0.1.0`）
   - 持續監聽 Ray 事件，並將其 **序列化後上傳到 MinIO**

2. **MinIO**

   - 作為 S3 相容的物件儲存
   - 保存格式：`s3://ray-historyserver/<cluster-name>/<session-id>/events.log`

3. **History Server**
   - 啟動時連接 MinIO
   - 讀取對應 session 的 event log
   - 提供 API 讓使用者查詢（如 `curl /api/v0/tasks`）

> 所以 MinIO 是 **History Server 的資料來源**，沒有它，History Server 就沒東西可查！

---

## 三、執行這些步驟會產生什麼？

| 步驟 | 動作                                  | 產生的資源/資料                                                                         |
| ---- | ------------------------------------- | --------------------------------------------------------------------------------------- |
| 3    | `kubectl apply -f minio.yaml`         | MinIO Pod + Service (`minio-service.minio-dev:9000`)                                    |
| 5    | `kubectl apply -f raycluster.yaml`    | RayCluster → Head Pod (含 `ray-head` + `collector` sidecar)                             |
| 6    | `kubectl apply -f rayjob.yaml`        | RayJob → 觸發任務執行 → 產生事件                                                        |
| 7    | `kubectl delete raycluster.yaml`      | **觸發 Collector 上傳日誌到 MinIO**（關鍵！）<br>→ MinIO 中出現 `session_2026-...` 目錄 |
| 8    | `kubectl apply -f historyserver.yaml` | History Server Pod + Service (`historyserver:30080`)                                    |
| 9    | `port-forward`                        | 本機可訪問 History Server API                                                           |

### MinIO 中的資料結構範例

```
ray-historyserver/
└── default/
    └── raycluster-historyserver/
        └── session_2026-01-16_02-00-00_123456_1/
            ├── events.log
            └── metadata.json
```

---

## 四、要觀察什麼？

### 1. **MinIO 是否收到日誌？**

- 登入 MinIO Console（`http://localhost:9001`，帳號/密碼 `minioadmin`）
- 檢查 bucket `ray-historyserver` 是否有 `session_xxx` 目錄
- 如果沒有 → Collector 沒上傳成功（常見原因：MinIO 連不上、權限錯誤）

### 2. **History Server 是否能讀取 MinIO？**

- 查看 History Server Pod 日誌：
  ```bash
  kubectl logs -l app=historyserver
  ```
- 應看到類似：
  ```
  Connected to MinIO at minio-service.minio-dev:9000
  Loaded sessions: [session_2026-...]
  ```

### 3. **API 是否回傳資料？**

```bash
# 先取得 session 名稱（從 MinIO 或 logs）
SESSION="session_2026-01-16_02-00-00_123456_1"

# 進入 session
curl -c ~/cookies.txt "http://localhost:8080/enter_cluster/default/raycluster-historyserver/$SESSION"

# 查詢 tasks
curl -b ~/cookies.txt "http://localhost:8080/api/v0/tasks"
```

- 如果回傳 JSON 陣列 → 成功！
- 如果空陣列 → 任務沒產生事件（檢查 RayJob 是否真的執行）
- 如果 404/500 → History Server 啟動失敗或 MinIO 讀取錯誤

### 4. **Live Cluster 模式是否工作？**

- 不要刪除 RayCluster（跳過 Step 7）
- 直接部署 History Server
- 用 `SESSION="live"` 進入
- 應能即時看到 Tasks/Actors（Collector 會持續推送事件到 History Server）

---

## 架構圖（簡化版）

```
+----------------+       +---------------------+       +------------------+
|   Ray Job      |       |   Ray Head Pod      |       |     MinIO        |
| (Step 6)       | ----> | - ray-head          | ----> | (S3 bucket)      |
|                |       | - collector (sidecar)|       |                  |
+----------------+       +----------+----------+       +------------------+
                                    |
                                    | (Step 7: delete cluster)
                                    v
                      +-----------------------------+
                      |    History Server Pod       |
                      | - Connects to MinIO         |
                      | - Serves API on :30080      |
                      +-----------------------------+
                                    |
                                    | (kubectl port-forward)
                                    v
                          Your curl / Browser
```

## 遇到問題

### 文件中是使用 kind 拉下 image，因為我這邊是完整的 k8s 所以用 ssh 的方式送到 worker node

因為 worker node 裡面沒有 historyserver image

```
docker save historyserver:v0.1.0 | ssh -i ~/private.key ubuntu@192.168.200.108 "sudo ctr -n=k8s.io images import -"
docker save collector:v0.1.0 | ssh -i ~/private.key ubuntu@192.168.200.108 "sudo ctr -n=k8s.io images import -"
docker save collector:v0.1.0 | ssh -i ~/private.key ubuntu@192.168.200.237 "sudo ctr -n=k8s.io images import -"
```

### 檢查 history server 是否有正確綁到 minio

```bash
kubectl logs -f deployment/historyserver-demo
```

這邊 log 顯示 Bucket ray-historyserver already exists，代表有連到 MinIo，在 Ray Pod 消失前的最後一刻，Collector 會把日誌「備份」到 MinIO 這個持久化的儲存空間，即使 Pod 沒了，資料還在

```
time="2026-01-16T01:05:38Z" level=info msg="add config from in cluster config"
time="2026-01-16T01:05:38Z" level=info msg="create client manager successfully, clients: 1"
time="2026-01-16T01:05:38Z" level=info msg="Begin to create s3 client ..."
time="2026-01-16T01:05:38Z" level=info msg="Checking if bucket ray-historyserver exists..."
time="2026-01-16T01:05:39Z" level=info msg="Bucket ray-historyserver already exists"
time="2026-01-16T01:05:39Z" level=info msg="Clean logdir is logs"
time="2026-01-16T01:05:39Z" level=info msg="Starting EventHandler in background..."
time="2026-01-16T01:05:39Z" level=info msg="Starting event file reader loop"
time="2026-01-16T01:05:39Z" level=info msg="Starting server on :8080"
time="2026-01-16T01:05:39Z" level=info msg="Starting a event processor channel"
time="2026-01-16T01:05:39Z" level=info msg="Starting a event processor channel"
time="2026-01-16T01:05:39Z" level=info msg="[List]Returned objects in log/metadir/. length of page.Contents: 0, length of page.CommonPrefixes: 0"
```

### 在 raycluster 有資源上的限制，所以為了要避開要進去改 mem 的用量

但是在部署 raycluster 遇到了問題

```bash
kubectl apply -f historyserver/config/raycluster.yaml
```

```
Events:
  Type     Reason            Age   From               Message
  ----     ------            ----  ----               -------
  Warning  FailedScheduling  62s   default-scheduler  0/3 nodes are available: 1 node(s) had taint {node-role.kubernetes.io/master: }, that the pod didn't tolerate, 2 Insufficient memory.
  Warning  FailedScheduling  0s    default-scheduler  0/3 nodes are available: 1 node(s) had taint {node-role.kubernetes.io/master: }, that the pod didn't tolerate, 2 Insufficient memory.
```

去改了 historyserver/config/raycluster.yaml mem 的 limit 為 256Mi

```yaml
containers:
  - name: ray-worker
    # ... 其他配置 ...
    resources:
      limits:
        cpu: "2"
        memory: 2G
      requests:
        cpu: "50m"
        memory: "256Mi" # <--- 將 1G 改為 256Mi
```

### rayjob 沒辦法順利地運行

```
Events:
  Type     Reason              Age                 From               Message
  ----     ------              ----                ----               -------
  Warning  RayClusterNotFound  15s (x13 over 35s)  rayjob-controller  RayCluster default/raycluster-historyserver set in the clusterSelector is not found. It must be created manually
```

## raycluster worker 一直卡在 GCS 沒有啟動

```
Events:                                                                                                  │
│   Type    Reason     Age    From               Message                                                   │
│   ----    ------     ----   ----               -------                                                   │
│   Normal  Scheduled  7m43s  default-scheduler  Successfully assigned default/raycluster-historyserver-cp │
│ u-worker-n8hqj to k8s-n0                                                                                 │
│   Normal  Pulled     7m43s  kubelet            spec.initContainers{wait-gcs-ready}: Container image "ray │
│ project/ray:2.52.0" already present on machine                                                           │
│   Normal  Created    7m43s  kubelet            spec.initContainers{wait-gcs-ready}: Created container wa │
│ it-gcs-ready                                                                                             │
│   Normal  Started    7m43s  kubelet            spec.initContainers{wait-gcs-ready}: Started container wa │
│ it-gcs-ready
```

等待 Head 節點的 GCS（Global Control Store）準備好後，才啟動 Worker 容器，所以要先確認 Head Pod 是否正常運行，發現了 head 出現了 `Readiness probe failed` 這個錯誤，有可能是 Head Service 可能沒有正確暴露 GCS（Redis）port 6379
