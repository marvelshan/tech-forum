## K8S Lab Day_52

# 一份讓 Loki 在 24 小時內磁碟爆滿的案例

## 前言

昨天提到的 Kiro 真的蠻厲害的，之後應該會想要拿來嘗試寫一些呼叫 k8s function 的一些小實驗，回到今天，因為在維運的部分不外乎會注意到可觀測性，所以今天就要來講一下我之前遇過的問題，順便回來熟悉一下 Loki 的內容

## Promtail 的危險配置呀

在之前我們去導入 Loki 是在使用 K8s 的狀況下，基本上有個簡單的 POC 就是直接抄網路上的 Promtail DaemonSet YAML，這個配置在網路上幾乎都長得一模壹樣，但他的環境在開啟 istio service mesh 之後基本會在 1~3 天內把磁碟塞爆，導致查詢速度變得相當的緩慢、寫入失敗、甚至整個 Loki Pod OOM 或是 PVC 滿了要重啟

```yaml
# 1. 使用 DaemonSet + hostPath 直接掛 /var/log 與 /var/lib/docker/containers
volumeMounts:
  - name: varlibdockercontainers
    mountPath: /var/lib/docker/containers
    readOnly: true

# 2. scrape_configs 完全沒有任何過濾
scrape_configs:
  - job_name: pod-logs
    kubernetes_sd_configs:
      - role: pod # 沒有 namespace 限制 → 抓全 cluster 所有 Pod
    relabel_configs:
      - replacement: /var/log/pods/*$1/*.log # 直接用 __path__ 抓所有容器 log
        source_labels:
          - __meta_kubernetes_pod_uid
          - __meta_kubernetes_pod_container_name
        target_label: __path__
      # 完全沒有 drop istio-proxy、沒有 annotation 控制、沒有排除 kube-system
```

看到以上的範例，假如是只有在 k8s 的環境下，這份文件就是一份相當危險的配置了

- 第一可以看到他收集了全 cluster 所有 ns 的 pod log，包含了 `kube-system`、`istio-system`、`monitoring`、`logging` 這些原有系統配置的 log 就相當的大，`kube-proxy`、`calico`、`metrics-server` 這種也是每天都幾十 GB 的

- 第二每個應用的 pod 都會被收集，鮑含了 `initContainer`、`debug 容器`、`fluentbit/promtail` 他自己的 log，這種 promtail 會收集自己 log 是會造成無限遞迴的風險的，雖然有 positions 檔來避免他無限重複，但還是很吃硬碟

- 第三 path 和 hostpath 會造成雙重收集， promtail 同時透過 kubenetes_sd（Promtail/Prometheus 的 Service Discovery 機制） 發現 pod，又直接讀 Dcoker 的 json log，在 CRI 環境下會重複讀到同一份日誌

再來看到 Istio，這個會是一個指數型的超大災難，因為會收到 istio-proxy 的 log，也就是我們的 Envoy，因為每個 pod 都會被注入一個 istio-proxy 的 sidecar，這個 pod 會 default 輸出詳細的 access log，像是每一個 HTTP/gRPC 都會輸出一行，單一個 Pod QPS 會有 1000 多行，這樣會造成每天有 80 多 GB 的 log，假如有 100 的 pod 就可想而知

```yaml
# 沒有這段關鍵過濾
- source_labels: [__meta_kubernetes_pod_container_name]
  regex: istio-proxy
  action: drop
```

### 要如何做才會是安全的配置呢？

如果一定要使用 DaemonSet 而不使用 sidecar 的方法

```yaml
scrape_configs:
  - job_name: pod-logs
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: [prod, staging] # 明確限制 namespace
    relabel_configs:
      # 最重要的地方！排除 istio-proxy
      - source_labels: [__meta_kubernetes_pod_container_name]
        regex: istio-proxy|promtail|linkerd-proxy
        action: drop

      # 只收集標註要收集的容器
      - action: keep
        source_labels: [__meta_kubernetes_pod_annotation_promtail_io_scrape]
        regex: "true"

      # ...
```

並在 pod 加上 annotation

```yaml
annotations:
  promtail.io/scrape: "true"
```

## 假如真的發生了！！！

### 立即關閉

首先要先把水龍頭關掉，要不就 `systemctl stop promtail` 要不就 `kubectl scale deploy --all -n <promtail-ns> --replicas=0`，直接切斷 log 的寫入來源，再來把 Loki 的節點 scale 到 0，`kubectl scale deploy loki -n loki --replicas=0`，讓 Loki 停止 compaction、ingester flush，避免在磁碟滿的時候寫壞了 index，確認磁碟真的不會再增加，要先進到 worker node 裡面 `watch -n 2 "du -sh /mnt/data/loki"` 或是 `df -h`

### 找到誰把磁碟塞爆

大部分的狀況都是 istio-proxy

```bash
kubectl -n loki exec -it loki-0 -- sh

curl -G --data-urlencode 'query=sum(rate({} [1h])) by (namespace,container)' \
  http://localhost:3100/loki/api/v1/query_range --silent | jq

# 就會看到類似這種的 log
# namespace="prod", container="istio-proxy"      → 2.8MB/s
```

只要看到 istio-proxy 排在第一個就是這個的問題，如果 loki 已經完全掛掉且 query 都不行使用

```bash
kubectl -n loki exec -it loki-0 -- sh
du -sh /loki/chunks/* | sort -hr | head -20
# 或者直接看 index 大小
ls -lh /loki/chunks/ | grep index

sudo du -sh /mnt/data/loki/chunks/* | sort -hr | head
```

### 清理並回覆

直接砍到 chunks

```bash
# 在 node 上直接砍掉最舊的 chunks
sudo ls -ld /mnt/data/loki/chunks/*/ | sort | head -n -6 | awk '{print $9}' | xargs rm -rf
# 或者只砍 index 目錄
sudo find /mnt/data/loki/chunks -name "index_*" -mtime +1 -delete
```

或是直接砍到 PVC

```bash
helm uninstall loki -n loki
kubectl delete pvc storage-loki-0 -n loki
```

或是直接 scale 讓 log 時間到自己把 log 清掉

```bash
kubectl patch pvc storage-loki-0 -n loki -p '{"spec":{"resources":{"requests":{"storage":"200Gi"}}}}'
```

然後就是像剛剛一樣，去把 yaml 加上 filter 避免收掉不必要的 log

```yaml
# 1. 立刻加上這段到 Promtail
relabel_configs:
  - source_labels: [__meta_kubernetes_pod_container_name]
    regex: istio-proxy|linkerd-proxy|promtail|grafana-agent
    action: drop

  # 2. 加上 annotation 控制，避免未來新服務又爆炸
  - action: keep
    source_labels: [__meta_kubernetes_pod_annotation_promtail_io_scrape]
    regex: "true"

# 3. Loki 加上嚴格限制
limits_config:
  ingestion_rate_mb: 8
  ingestion_burst_size_mb: 16
  max_global_streams_per_user: 5000
```

最後呢就是要寫事後報告啦！`事件原因：Promtail 配置未排除 istio-proxy 容器，導致 Envoy access log 暴量寫入，影響範圍：Loki PVC 100% 滿，服務中斷 ＊ 小時 ＊＊ 分...`

然後就要去 promise 之後不會發生了，不然又要被電到臭頭

## 簡單實用的 loki 配置

看到 loki 的 document 然後按照上面去做配置，結果爆了，就開始不禁懷疑起 loki 這個工具，但其實這個東西是真的實用且好用

```yaml
# loki-values.yaml
loki:
  auth_enabled: false # 小規模可先關，之後接 gateway 再開
  commonConfig:
    replication_factor: 1
  schemaConfig:
    configs:
      - from: "2025-01-01"
        store: tsdb
        object_store: filesystem
        schema: v13
        index:
          prefix: loki_index_
          period: 24h

  storage:
    type: filesystem
    filesystem:
      dir: /loki/chunks

  pattern_ingester:
    enabled: true

  limits_config:
    ingestion_rate_mb: 16 # 防止 Promtail 爆炸時直接把 Loki 搞掛
    ingestion_burst_size_mb: 32
    max_global_streams_per_user: 10000
    retention_period: 336h # 14 天（依磁碟調整）
    retention_stream_count: 5000
    volume_enabled: true

  compactor: # 建議開啟，避免 index 暴增
    working_directory: /loki/compactor
    retention_enabled: true

persistence:
  enabled: true
  size: 50Gi # 至少 50Gi 起跳，10Gi 幾小時就滿
  storageClassName: "longhorn" # 改用 longhorn、rook-ceph、純雲碟等

singleBinary:
  replicas: 1
  resources:
    requests:
      cpu: "500m"
      memory: "2Gi"
    limits:
      cpu: "2000m"
      memory: "4Gi"

# 關閉不需要的組件
promtail:
  enabled: false
gateway:
  enabled: true # 建議開 nginx gateway，之後好加 basic auth / oauth
```

然後按照以下指令去做部署

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

kubectl create ns loki
helm install loki grafana/loki -n loki \
  -f loki-values.yaml \
  --version 6.29.0
```

## 結論

希望我過去採得坑可以幫助更多團隊能夠避免，也透過這個文章也可以更了解 promtail 收 log 的機制啦！

## Reference

https://grafana.com/docs/loki/latest/send-data/promtail/installation/

https://sagar-srivastava.medium.com/setting-up-grafana-loki-on-kubernetes-a-simplified-guide-97fbf850ba55

https://grafana.com/grafana/dashboards/14876-grafana-loki-dashboard-for-istio-service-mesh/

https://developer.hashicorp.com/consul/tutorials/observe-your-network/proxy-access-logs#proxy-access-logs
