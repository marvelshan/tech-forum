這個 YAML 是用 **KubeRay** 部署一個特殊用途的 **RayCluster**，主要目的在於實現 **Ray 任務/執行歷史的持久化收集與儲存**（特別是 Ray 2.5x 系列的 **Event Export 功能**），通常被稱為 **Ray History Server** 或 **Ray 事件/任務歷史收集叢集**。

這個叢集本身**不跑主要的運算工作**，而是作為一個「接收器 + 儲存器」，收集所有其他 Ray 叢集的事件，並把這些結構化事件持久化到 MinIO（S3 相容物件儲存），供後續的歷史查詢、任務回溯、效能分析使用。

以下是這個 YAML 的**結構化解析與重點說明**，適合用來寫系列文章的第一篇或第二篇基礎介紹：

### 1. 整體架構與目的

```yaml
kind: RayCluster
metadata:
  name: raycluster-historyserver
  namespace: default
  labels:
    ray.io/cluster: raycluster-historyserver
```

- 這是一個**專門的 RayCluster**，名字帶有 `historyserver`
- 它通常**不會**被用來跑真正的機器學習/資料處理任務
- 真正目的是當作**事件收集與持久化中樞**

### 2. Head 節點的特殊設計（最核心部分）

```yaml
headGroupSpec:
  rayStartParams:
    dashboard-host: 0.0.0.0
    num-cpus: "0" # ← 故意不分配 CPU，因為不跑 task
  serviceType: ClusterIP

  template:
    spec:
      containers:
        - name: ray-head
          image: rayproject/ray:2.52.0
          resources:
            limits:
              cpu: "5"
              memory: 10G
            requests:
              cpu: "50m"
              memory: 1G
          securityContext:
            privileged: true # ← 為了能順利 ps、讀取 raylet 資訊

          # 非常重要的環境變數（Ray 2.52.0 的事件匯出機制）
          env:
            - name: RAY_enable_ray_event
              value: "true"
            - name: RAY_enable_core_worker_ray_event_to_aggregator
              value: "true"
            - name: RAY_DASHBOARD_AGGREGATOR_AGENT_EVENTS_EXPORT_ADDR
              value: "http://localhost:8084/v1/events"
            - name: RAY_DASHBOARD_AGGREGATOR_AGENT_EXPOSABLE_EVENT_TYPES
              value: "TASK_DEFINITION_EVENT,TASK_LIFECYCLE_EVENT,..." # 列出要收集的事件類型

          # 重要的 postStart hook：取得 raylet node_id
          lifecycle:
            postStart:
              exec:
                command: ["/bin/sh", "-lc", "... GetNodeId() ..."]

        # 關鍵的側車容器（sidecar）——事件收集器
        - name: collector
          image: collector:v0.1.0 # ← 自製或第三方收集器
          command:
            - collector
            - --role=Head
            - --runtime-class-name=s3
            - --ray-cluster-name=raycluster-historyserver
            - --ray-root-dir=log
            - --events-port=8084
```

**Head 節點真正要做的事**：

1. 啟用 Ray 的事件產生機制（`RAY_enable_ray_event`）
2. 把事件推向本地（localhost:8084）→ 這就是給旁邊的 collector 容器
3. 透過 **postStart** 腳本拿到 raylet 的 node_id（很多自訂收集邏輯會需要）
4. 跑一個 sidecar 容器 `collector` 負責真正接收 8084 port 的事件，並上傳到 S3/MinIO

### 3. Worker Group 的設計（也很特別）

```yaml
workerGroupSpecs:
- groupName: cpu
  replicas: 1
  minReplicas: 1
  maxReplicas: 1000     # 可擴展，但通常只會用 1~幾個
  template:
    spec:
      containers:
      - name: ray-worker
        image: rayproject/ray:2.52.0
        resources:
          limits:
            cpu: "2"
            memory: 2G
        env:   # 同樣開啟事件匯出
          ... 跟 head 一樣的四個重要 env ...

      - name: collector
        image: collector:v0.1.0
        command:
        - collector
        - --role=Worker          # ← 差別在這裡
        - --runtime-class-name=s3
        ...
```

**Worker 的 collector 也會把事件推到同一個 MinIO bucket**，但會帶上自己的 node_id 做區分。
