## K8S Lab Day_53

# 半夜被電話吵醒之沒設 memory limit 的不定時炸彈

## 前言

昨天寫了關於 promtail 收到爆量的 logs 導致 loki 連帶 grafana 無法顯示圖表，這個給我了一個啟發，現在就來開啟一系列的半夜被電話吵醒系列文～

今天要講到的是假如有服務的 latency 爆了，有半數的 Pod 變成 OOMKilled 並且無限循環重啟，然後看到 Grafana 跳出的 alert

```text
oom-killer invoked. OOM killer terminated this process.
```

## 事情是怎麼發生的？

預設在出事的服務是一個 java spring boot 的服務，一直以來在 local 和 staging 環境都沒有問題，但 memory 的使用量大概都在 600-800Mi，然後，部署到因為要快速所以在 k8s 的 yaml config 這樣寫

```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    cpu: "1000m"
```

結果發現就是這個 Pod memory 無上限的提升，某天晚上剛好遇到一大批的檔案要理，Java Heap 持續的膨脹，Off-Heap 也爆炸了，讓整個 Container 的 memory 直接上升到 8GB，Node 的 memory 也開始變得吃緊了，kublet 開始啟動了 oom-killer 機制，優先砍掉沒有 memory limit 的 pod，然後最後導致整個服務爆炸，最後因為沒有 limit，kublet 直接使用 kernel oom-killer，而不是 container level 的 OOM，所以整個連 restartPolicy 都沒辦法救回來，Pod 一直處於 CrashLoopBackOff 的狀態

## 要怎麼找到問題？

第一個當然先去看到 log 的狀況

```bash
kubectl get pod -n prod | grep oom
kubectl describe pod xxxxxxxxxx

The node was low on resource: memory.
         Container xxx was using 8.2Gi, which exceeds its request ...
```

然後再去看到 node level 的 memory

```bash
kubectl top node
# 發現某台 node memory usage 99%
```

進到 node 去找 oom-killer log

```bash
dmesg | grep -i "killed process"
```

然後最後再去確認 container memory 的使用量

```bash
kubectl exec -it <pod> -- cat /sys/fs/cgroup/memory/memory.usage_in_bytes
```

## 怎麼緊急故障排除且回到可使用狀態呢？

最快的方法就是直接手動給他一個 memory limit

```yaml
resources:
  limits:
    memory: "4Gi"
```

```bash
kubectl apply -f deploy.yaml
```

如果來不及改 yaml 就直接使用 kubectl patch

```bash
kubectl patch deployment my-svc -n prod --patch '
spec:
  template:
    spec:
      containers:
      - name: my-container
        resources:
          limits:
            memory: "4Gi"
'
```

可以先觀察有沒有再 OOM，latency 有回歸正常

## 那怎樣才是安全的配置呢？

```yaml
resources:
  requests:
    memory: "1Gi"
    cpu: "500m"
  limits:
    memory: "2Gi" # 通常是 requests 的 1.5~3 倍，視應用而定
    cpu: "2"
```

另外也要開啟 QoS Guaranteed(requests=limits) 是最好的，相對於成本較高，但多數人還是使用 Burstable + limit 的配置

## 那要怎麼預防呢？

- 強制在 CI/CD 的流程上檢查

```yaml
# Kyverno policy
apiVersion: wgpolicyk8s.io/v1alpha2
kind: ClusterPolicy
metadata:
  name: require-memory-limits
spec:
  validationFailureAction: Enforce
  rules:
    - name: check-memory-limits
      match:
        resources:
          kinds:
            - Pod
            - Deployment
      validate:
        message: "Memory limits are required"
        pattern:
          spec:
            containers:
              - resources:
                  limits:
                    memory: "?*"
```

- 使用 Vertical Pod Autoscaler(VPA) Recommender 模式，使用建議的 requests/limits

- 開啟 Node 的 memory QoS，讓 BestEffort Pod 先被砍掉

- 監控加上 Container menory usage > 90% of limit 就要發出 alert 而不是只有看 node memory

- 開發時要加上壓力測試，並且要測 memory limit

## 小結

沒有 memory limit 是常見的 prod issue，他不會再測資安或是效能的時候出現，常常就會在你在睡覺的時候爆開，在 k8s 中，memory requests 是排程用的，那 memory limits 就是用來救命的，也希望大家能睡得安穩啦，不要再被 OOMKilled 叫醒了～

## Reference

https://blog.csdn.net/yanghangwww/article/details/111992079

https://medium.com/@reefland/tracking-down-invisible-oom-kills-in-kubernetes-192a3de33a60

https://qzy.im/blog/2020/07/oom-killer-killed-java-process-in-linux/

https://www.hwchiu.com/docs/2023/autoscaler
