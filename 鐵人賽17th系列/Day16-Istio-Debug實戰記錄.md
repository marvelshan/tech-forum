## K8S Lab Day_18

# Day16: Istio Debug 實戰記錄

## 前言

昨天介紹了簡單的 log 的用法，因為有在嘗試做一些 istio 文件上的小實驗，然後就遇到了一些 bug，所以今天是一日 debug day

### Debug 實戰！

```bash
# 安裝 Loki 與 OpenTelemetry Collector
istioctl install -f samples/open-telemetry/loki/iop.yaml --skip-confirmation
kubectl apply -f samples/addons/loki.yaml -n istio-system
kubectl apply -f samples/open-telemetry/loki/otel.yaml -n istio-system
```

這時候我們去檢查服務是否有正常啟動

```bash
kubectl get pods -n istio-system -l app=loki
kubectl get pods -n istio-system -l app=opentelemetry-collector
```

發現了還沒有任何帶有 app=loki label 的 Pod 出現

```bash
No resources found in istio-system namespace.
NAME                                       READY   STATUS    RESTARTS   AGE
opentelemetry-collector-684c6f9f4c-sdk5d   1/1     Running   0          2m12s
```

現在我們就要一步一步的來去找到問題，首先我們要先看 loki 是否有嘗試被創建

```bash
kubectl get pods -n istio-system | grep loki
```

```bash
loki-0                                     0/2     Pending   0          4m11s
```

看起來 pod 是存在，但是他是處於一個 pending 的狀態

```bash
kubectl describe pod -n istio-system loki-0
```

接著我們就看到他的 event 是說我們沒有正確的綁定 PersistentVolumeClaim，看來我們是要自己創建並且綁定了

```bash
...
Events:
  Type     Reason            Age    From               Message
  ----     ------            ----   ----               -------
  Warning  FailedScheduling  4m15s  default-scheduler  0/4 nodes are available: pod has unbound immediate PersistentVolumeClaims. preemption: 0/4 nodes are available: 4 Preemption is not helpful for scheduling.
```

首先還是必須要先看一下有沒有

```bash

kubectl get pv loki-pv
```

然後創建 loki-pv.yaml

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: loki-pv
spec:
  capacity:
    storage: 10Gi
  volumeMode: Filesystem
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: manual
  hostPath:
    path: "/mnt/data/loki"
```

接著我們要讓我們的 loki 綁定上手動創建的 PV

```bash

kubectl edit pvc -n istio-system storage-loki-0
```

```yaml
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: manual # 新增這一行
  volumeMode: Filesystem
```

就可以來檢查是否有正確的被 bound

```bash
kubectl get pvc -n istio-system storage-loki-0
```

再來檢查一下是否有正確啟動啦

```bash
kubectl describe pod -n istio-system loki-0
```

看到下面的 event 出現 `Successfully assigned istio-system/loki-0 to k8s-n1` 就恭喜綁定成功啦！

```bash
Events:
  Type     Reason            Age                 From               Message
  ----     ------            ----                ----               -------
  Warning  FailedScheduling  8m12s               default-scheduler  0/4 nodes are available: pod has unbound immediate PersistentVolumeClaims. preemption: 0/4 nodes are available: 4 Preemption is not helpful for scheduling.
  Warning  FailedScheduling  3m (x2 over 8m10s)  default-scheduler  0/4 nodes are available: pod has unbound immediate PersistentVolumeClaims. preemption: 0/4 nodes are available: 4 Preemption is not helpful for scheduling.
  Normal   Scheduled         28s                 default-scheduler  Successfully assigned istio-system/loki-0 to k8s-n1
  Normal   Pulled            29s                 kubelet            Container image "kiwigrid/k8s-sidecar:1.30.7" already present on machine
  Normal   Created           28s                 kubelet            Created container: loki-sc-rules
  Normal   Started           28s                 kubelet            Started container loki-sc-rules
  Normal   Pulled            6s (x3 over 29s)    kubelet            Container image "docker.io/grafana/loki:3.5.3" already present on machine
  Normal   Created           6s (x3 over 29s)    kubelet            Created container: loki
  Normal   Started           6s (x3 over 29s)    kubelet            Started container loki
  Warning  BackOff           4s (x5 over 27s)    kubelet            Back-off restarting failed container loki in pod loki-0_istio-system(2b89cfc0-c408-4358-9d78-a5453c584d1e)
```

但又往下看發現另一個 event `Back-off restarting failed container loki in pod loki-0_istio-system`，loki 的 pod 目前呈現 CrashLoopBackOff 的狀態，他在 `mkdir /var/loki/rules: permission denied` 的時候出現了權限的問題，所以我們現在要 ssh 進入我們的 worker node 去開啟權限

```bash
ssh <k8s-n1_IP_or_hostname>
sudo chmod -R 755 /mnt/data/loki
```

然後返回並刪除 loki 的 pod 讓他去重新啟動

```bash
kubectl delete pod -n istio-system loki-0
```

我們在查看一下是否有正確啟動

```bash
kubectl get pods -n istio-system | grep loki
```

```bash
loki-0                                     2/2     Running   0          118s
```

看起來就有正確的被啟動啦！

## 總結

今天大概介紹了怎麼查看 pod 狀態的問題，先說這不一定是 best practice，但可以用這種方法來查看問題，但 root cause 還是必須要去細查，因為我的功力還不算是太強，假如各界大師有好的方法可以再麻煩告訴小弟我了！

## Reference

https://istio.io/latest/docs/tasks/observability/logs/access-log/
