## K8S Lab Day_8

## 前言

前幾天講了 Nix 的環境版本控制，今天要回來說明 K8S 的調度和資源管理，而且第一版 ai 給的 30 天看來已經跑偏了，所以接下來的版本會跟第一天說的有差～完成 30 天後會再重新更新一次。

## 調度與資源管理

當我們在 k8s 裡面跑 pod 的時候我們一定會去管控 CPU 和 Ram 會消耗多少，假如不多加設定的話各個 cluster 使用的資源就會相互打架，就像是 EVA 一樣，每一台 EVA（就像 Node）都有它能負荷的極限，如果我們在 Pod 上完全不去設定 CPU 和 RAM 的用量，就好像駕駛員不帶同步率數據、也不裝限制器，結果就是各個 EVA 都在暴走，開始互相干擾、甚至拖垮整個戰場

1. Requests：Pod 至少要多少的資源才能運作，假如我說明我需要 200m CPU，512Mi Ram，scheduler 就會找到一個有空間的 Node 去放這個 Pod，只有當前 Node 可分配資源大於 Request 才能將 Pod 調度到 Node 上

2. Limits：Pod 能使用的最大上限，假如設置為 0 時代表對資源不做限制

## QoS（Quality of Service）

k8s 會根據你設定的 Requests/Limits 來分為各個等級：

1. Guaranteed（保證型）：只有當 Requests 和 Limits 都有設定時，並且設定的數值都一樣才會是 Guaranteed

   - 適合的場景：關鍵核心服務（例如資料庫、金流系統），不能被驅逐、必須長時間穩定運行的工作負載

2. Burstable（可突發型）：有設定 Requests 但 Limits 設定比較高，偶爾會有高流量進來，好處就是彈性大，壞處就是當系統壓力大時可能會被淘汰掉

   - 適合的場景：大部分的業務邏輯服務、API Server，平常用量可控，但偶爾需要爆衝處理流量高峰

3. BestEffort（盡力而為型）：Requests 和 Limits 都沒有設定，缺定式資源有剩才跑得動，壓力大的時候會最先被犧牲

   - 開發測試用 Pod，非關鍵性任務，像是 batch job、log 分析等，容忍被隨時驅逐的應用

可以來看一下 document 給的範例：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: qos-demo
  namespace: qos-example
spec:
  containers:
    - name: qos-demo-ctr
      image: nginx
      resources:
        # 超過限制時，容器會被 OOMKilled (Ram) 或被 Throttle (CPU)
        limits:
          # 記憶體限制：容器最多可以使用 200MiB，如果超過此限制，容器會被 Kubernetes 終止 (OOMKilled)
          memory: "200Mi"
          # CPU 限制：容器最多可以使用 700 毫核 (0.7 核心)，超過此限制，容器會被 CPU Throttling
          cpu: "700m"

        # Kubernetes 會根據 Requests 來調度 Pod 到合適的 Node
        requests:
          # 記憶體請求：Kubernetes 保證容器至少有 200MiB 可用，用於 Node 選擇和資源預留
          memory: "200Mi"
          # CPU 請求：Kubernetes 保證容器至少有 700m 核可用
          cpu: "700m"

      # Kubernetes QoS 類別的三種類型：
      # 1. Guaranteed：requests = limits (最高優先級)
      # 2. Burstable：requests < limits (中等優先級)
      # 3. BestEffort：無 requests 和 limits (最低優先級)
      #
      # 在節點記憶體壓力時，Kubernetes 會按優先級終止 Pod：
      # BestEffort > Burstable > Guaranteed
```

當你在創建完畢 `kubectl apply -f https://k8s.io/examples/pods/qos/qos-pod.yaml --namespace=qos-example`，並查看 pod 狀態時 `kubectl get pod qos-demo --namespace=qos-example --output=yaml` 會得到以下資訊

```yaml
spec:
  containers:
    ...
    resources:
      limits:
        cpu: 700m
        memory: 200Mi
      requests:
        cpu: 700m
        memory: 200Mi
    ...
status:
  qosClass: Guaranteed
```

## LimitRange

剛剛有說到 Requests 跟 Limits 可以自己設定，但問題來了，如果大家都不設呢？或者有人亂設，一個 Pod 就要吃 64 核心、128Gi RAM，那整個 cluster 不就被他霸佔了？這時候就需要 LimitRange 來當「社區公約」的角色，LimitRange 是 Namespace 等級的規則，用來幫這個 Namespace 裡的 Pod 設定「預設值」跟「上下限」，我們可以設定一個 yaml 檔讓之後的 pod 來去遵守它

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: mem-limit-range
  namespace: default
spec:
  limits:
    - default:
        memory: 512Mi
        cpu: 500m
      defaultRequest:
        memory: 256Mi
        cpu: 200m
      max:
        memory: 1Gi
        cpu: 1
      min:
        memory: 128Mi
        cpu: 100m
      type: Container
```

當 Pod 沒設值，系統會自動給 200m CPU / 256Mi Memory 的 request，500m CPU / 512Mi Memory 的 limit，不管怎麼設，Memory 最少要 128Mi，最多不能超過 1Gi；CPU 最少 100m，最多 1 core。

那什麼情況下會使用呢？團隊開發，大家愛亂設資源、有些人偷懶，乾脆不設資源、避免服務起不來、公司要做資源規範，方便管理等等

## 結論

前幾天大概都把 K8S 基本講完，雖然還有很多東西沒講，但鐵人賽還有很多資源都比我強太多了！接著開始前往 CICD 和 service mesh 還有 Ceph 的東西，怎麼感覺越來越變成我自己想要搞的實驗室的感覺，看到什麼就想用用看 XD

## Reference

https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/

https://ithelp.ithome.com.tw/articles/10295419
