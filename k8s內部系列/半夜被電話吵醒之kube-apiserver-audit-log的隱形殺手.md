## K8S Lab Day_54

# 半夜被電話吵醒之 kube-apiserver audit log 的隱形殺手

## 前言

昨天寫到沒有設定 pod 的 memory limit，今天要來看到 etcd，凌晨的手機又響了，發現是 cluster API 掛了，所有的 pods pending，kublet 連不上 apiserver，揉揉眼睛登入 dashboard，看到 Prometheus 警報又炸開來了，etcd write latency P99 > 500ms，kube-apiserver response 都變成 500 internal server error

```bash
“etcdserver: request timed out, possibly due to previous leader failure"
...
```

## 事情是怎麼發生的？

首先要看到 etcd，是 k8s 的心臟，存了所有 cluster 的狀態，他對寫入延遲非常的敏感，官方建議是 <100ms，超過就有可能 leader election 失敗，導致整個 cluster 癱瘓

這邊要看到這次的問題是出現在 kube-apiserver 的 audit log，在 default 的狀態下，audit log 會寫到 apiserver Pod 的 local file 裡，但假如 master node 同時也跑 etcd，這些 log 會瘋狂的上升

思考一個情境，在 cluster 接受數萬個 API call、每小時 audit log GB 級的進入、磁碟從 80% 到 100%、etcd 因為 I/O 競爭寫入變慢、延遲飆高到 300ms、heartbeat 丟失、cluster API 因此崩潰，最後發現是維運團隊一開始沒注意到 audit policy 設成 log all requests，結果就在辦也爆開

## 怎麼找到問題？

首先要檢查 apiserver 的狀態和 event

```bash
kubectl get componentstatuses   # etcd unhealthy
kubectl describe pod kube-apiserver-xxx -n kube-system
# 看到 etcd connection refused 或 timeout
kubectl get events --all-namespaces | grep etcd
```

利用 Prometheus 和 etcd exporter 監控 etcd metrics

```bash
curl http://etcd-ip:2379/metrics | grep etcd_disk_wal_fsync_duration_seconds
# 或 Grafana dashboard: etcd_disk_backend_commit_duration_seconds > 0.1
```

檢查 node 的磁碟使用量

```bash
kubectl get nodes
ssh master-node
df -h /var/lib/etcd   # 發現 99% full
du -sh /var/log/kubernetes/*
```

確認 audit log 是兇手

```bash
journalctl -u kube-apiserver | grep "audit"
```

## 怎麼緊急排除狀況，並回覆到原先的狀態？

裡急清空 audit log 止血

```bash
ssh master-node
truncate -s 0 /var/log/kubernetes/audit.log
systemctl restart kube-apiserver
```

如果 etcd 已經掛了就要先恢復 leader

```bash
etcdctl --endpoints=https://127.0.0.1:2379 member list
etcdctl endpoint health
```

如果是使用 PV，擴大磁碟

```bash
kubectl edit pvc etcd-pvc -n kube-system
```

## 怎樣才是安全的配置

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: etcd
spec:
  containers:
    - name: etcd
      image: etcd:3.5.x
      command:
        - etcd
        - --data-dir=/var/lib/etcd
        - --quota-backend-bytes=8589934592 # 8GB quota，避免無限增長
        - --auto-compaction-retention=8h # 自動壓縮
      volumeMounts:
        - name: etcd-data
          mountPath: /var/lib/etcd
  volumes:
    - name: etcd-data
      persistentVolumeClaim:
        claimName: etcd-pvc
```

audit policy 安全的寫法來最小化 lod

```yaml
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
  - level: Metadata # 只 log metadata，不是 full request/response
    resources:
      - group: ""
        resources: ["pods", "secrets"]
    verbs: ["get", "create"]
omitStages:
  - "RequestReceived"
```

把 audit log 導到外部的 ELK 或是 S3，不要寫在 local，並用 dedicated etcd nodes 並設 QoS Guaranteed

## 要怎麼預防呢

必須要設告警的，Prometheus rule 設定 `etcd_disk_wal_fsync_duration_seconds_bucket` > 0.1s 就要 alert，磁碟 >80%，也可以使用 AWS EKS 的 etcd DB size 監控

或是自動化 compaction 與 defrag

```bash
etcdctl defrag
```

又或是使用 Kyverno

```yaml
apiVersion: wgpolicyk8s.io/v1alpha2
kind: ClusterPolicy
spec:
rules:

- name: require-audit-backend
  validate:
  pattern:
  spec:
  auditLogPath: "!/var/log/\*"
```

## etcd 的 Leader Election

接著在講完事件之後要來看一下這個選舉機制，etcd 是使用 Raft 共識演算法來確保 cluster 的高可用性和資料的一致性，有三種角色 Leader、Follower、Candidate，用來在 leader 出問題時選出新的 leader 來確保 cluster 能夠持續進行，有點像是 radis 的哨兵機制

當領導者崩潰或網路延遲導致心跳遺失時，某個追隨者的選舉超時先到期，它會增加自己的 Term Number 轉換成候選人，並向其他節點發送 RequestVote RPC，請求中包含自己的 Term 和 Last Log Index，證明自己是最新的，候選人需要獲得 Quorum 才能當選領導者，獲得 Quorum 後，轉換成領導者，立即發送心跳給所有人，通知新領導者上線

## etcd 的 Split-Brain

Split-Brain 是分布式系統的經典故障問題，指的是 Network Partition 導致 cluster 分成多個孤立子集，每個子集都以為自己是唯一的領導者，造成資料不一致或衝突

那要如何緩解 Split-Brain？etcd 的 Raft 演算法設計上就避免 Split-Brain，透過嚴格的 Quorum 和 Term 機制就可以有效的預防

## 小結

etcd 滿了就像是 cluster 隱形心臟病，平常沒事已有是直接掛點，audit log 就是常見的問題，但需要利用外部的監控去監測與告警就能靜量的避開

## Reference

https://docs.cloud.google.com/kubernetes-engine/distributed-cloud/vmware/docs/troubleshooting/resource-contention?hl=zh-cn

```

```
