## K8S Lab Day_55

# 半夜被電話吵醒之 Control Plane / etcd 手動升級損毀

## 事件背景與發生情況

在 On-Prem 環境裡，Control Plane 與 etcd 幾乎都是要自己升級，沒有像是雲端託管服務背後有官方自動化的升級流程和 rollback 機制，在 On-Prem 面對到的的是沒有官方 one-click upgrade、沒有內建 blue/green control plane、沒有自動 etcd snapshot + S3 儲存、常常只有 3 台 master，磁碟還是 HDD，最後升級就會變成這樣

```yaml
ssh master01
yum update kubeadm kubelet kubectl -y
kubeadm upgrade apply v1.28.5
systemctl restart kubelet
# 然後手動 cp /etc/kubernetes/manifests/*.yaml 到 master02、master03
# 再一台一台重開 kubelet
```

常常出事的場景像是三台 master 同時重啟，etcd quorum 瞬間消失 30 秒，最後 API server 全掛、升到一半發現 `v1.28.5` 的 kube-apiserver 跟舊版 etcd `3.5.6` 不相容，然後整個 cluster 直接不認得新 binary、master01 先升完變成 `v1.28.5`，master02、master03 還是 `v1.27.8`，版本 skew 然後 scheduler 永久卡在「Unable to create pod: etcd version too old」、etcd 成員列表沒更新，舊節點還留在 member list，網路ㄘㄨㄚ ˋ 一下就產生 split-brain，兩邊都認為自己是合法 leader，然後最後導致公司全部的 Pods Pending，大家的手機就 alert 瘋狂的響

## 要怎麼 Troubleshooting？

先查看 apiserver 是不是活著

```bash
journalctl -u kube-apiserver -n 100 | grep -i etcd
```

查看 etcd member 狀態

```bash
ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  member list -w table
```

看 health 與 leader

```bash
etcdctl endpoint status --cluster -w table
etcdctl endpoint health --cluster
```

再來就是看有沒有 split-brain

```bash
etcdctl endpoint status --cluster -w table | grep -v "true"
```

查看 raft log 是否 diverge

```bash
for endpoint in master01:2379 master02:2379 master03:2379; do
  echo "=== $endpoint ==="
  ETCDCTL_API=3 etcdctl --endpoints=https://$endpoint ... get / --prefix --keys-only | wc -l
done
```

## 怎麼緊急故障排除且回到可使用狀態呢？

### 顯判斷哪一份資料是 authoritative

最後達成 quorum 的那一份規則就會是最新的

```bash
# 看哪台有最多 peers 認為是 leader
etcdctl member list -w table
# 看 raft index 誰最高，但要先有 quorum 才能看
etcdctl endpoint status --cluster -w table | sort -k5 -nr | head -1
```

### 要來救命的 etcd snapshot 恢復流程

等第一台起來有 leader 後，再把其他兩台 member remove

```bash
scp backup/etcd-snapshot-2024-12-01.db master01:/tmp/

# 先停掉所有 static pod
mkdir /etc/kubernetes/manifests.bak
mv /etc/kubernetes/manifests/*.yaml /etc/kubernetes/manifests.bak/

# 只留一台當 donor（通常是最健康的）
etcdctl snapshot restore /tmp/etcd-snapshot-2024-12-01.db \
  --data-dir=/var/lib/etcd-restored \
  --initial-cluster="master01=https://10.0.0.11:2380,master02=https://10.0.0.12:2380,master03=https://10.0.0.13:2380" \
  --initial-advertise-peer-urls=https://10.0.0.11:2380

# 移動資料目錄並啟動
mv /var/lib/etcd /var/lib/etcd-old
mv /var/lib/etcd-restored /var/lib/etcd
chown -R etcd:etcd /var/lib/etcd
systemctl start etcd
```

### 收斂版本不一致

```bash
kubeadm upgrade apply v1.27.8 --force   # 會強制所有 node 降回去

# 或升級其餘節點
for node in master02 master03; do
  ssh $node
  yum install kubeadm-1.28.5 kubelet-1.28.5 kubectl-1.28.5
  kubeadm upgrade node
  systemctl restart kubelet
done
```

## 那要怎麼預防呢？

使用 kubeadm 或自動化工具，像是 Kuberspray 搭配 Ansible，或是 Terraform + Ansible + kubeadm，最後寫一個 script 去跑升級檢查清單，像是`./pre-upgrade-check.sh` 去檢查 etcd health、snapshot 年齡、磁碟使用率、版本 skew，並且建立 Runbook，最後要設定監控指標

```yaml
etcd_server_leader_changes_seen_total > 1
etcd_disk_wal_fsync_duration_seconds > 0.5
etcd_mvcc_db_total_size_in_bytes > 6GB
kube_apiserver_request_duration_seconds{code="500"} > 10
```

## 結語

在 On-Prem 環境裡，Control Plane 跟 etcd 永遠是最容易被忽略、卻也是最容易讓整個集群直接停機的環節，只要升級流程沒規劃好、版本相依性沒確認、一個不小心三台 master 同時重啟，整個 Kubernetes 就會原地爆開，這其實也不是工程師的能力不足，而是 On-Prem 的現實本來就缺乏雲端託管的保護機制，沒有 snapshot、沒有 rollback、沒有安全升級 pipeline，一切都得靠自己，但只要控好流程、守住紀律，它還是能穩穩地跑起來、不讓你在半夜被電話吵醒～
