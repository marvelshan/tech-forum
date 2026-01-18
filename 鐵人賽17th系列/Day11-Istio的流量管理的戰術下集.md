## K8S Lab Day_13

# Day_11: istio 的流量管理的戰術下集

## 前言

昨天說明明了 ssh 和部分的 traffic management 今天繼續講剩下的～

### 4. TCP Traffic Shifting

跟 HTTP 的 Traffic Shifting 一樣，但應用在 TCP 協議（例如資料庫 Proxy），應用場景在你要升級一個 Redis/MongoDB 後端，先導 10% 流量到新 DB，觀察行為

```yaml
apiVersion: networking.istio.io/v1
kind: VirtualService
metadata:
  name: redis
spec:
  hosts:
    - redis
  tcp:
    - match:
        - port: 6379
      route:
        - destination:
            host: redis
            subset: v1
            port:
              number: 6379
          weight: 80
        - destination:
            host: redis
            subset: v2
            port:
              number: 6379
          weight: 20
```

---

### 5. Request Timeouts

設定一個請求的等待上限，使用場景通常是避免上游被下游卡死，超過 0.5s 就返回錯誤，主要防止「整個過程因為一個卡住的服務拖垮」

```yaml
apiVersion: networking.istio.io/v1
kind: VirtualService
metadata:
  name: reviews
spec:
  hosts:
    - reviews
  http:
    - route:
        - destination:
            host: reviews
            subset: v2
      timeout: 0.5s
```

---

### 6. Circuit Breaking

如果某個服務回應一直失敗，就暫時不送流量給它，主要是保護整個系統，避免「壞掉的服務拖累所有人」，像是 ratings 連線錯誤率 >50%，就先斷掉，過幾秒再恢復

```yaml
kind: DestinationRule
---
spec:
  host: httpbin
  trafficPolicy:
    connectionPool:
      http:
        http1MaxPendingRequests: 1 # HTTP 1.1 最大待處理請求數，超過會拒絕
        maxRequestsPerConnection: 1 # 每個 TCP 連線允許的最大 HTTP 請求數
      tcp:
        maxConnections: 1 # TCP 最大連線數
    outlierDetection: # 異常檢測 (Outlier Detection) 設定
      baseEjectionTime: 3m # 異常 pod 被踢掉後等待時間 (3 分鐘)，過後自動恢復
      consecutive5xxErrors: 1 # 連續 5xx 錯誤超過 1 次就將該 pod 暫時踢掉
      interval: 1s # 每 1 秒檢查一次 pod 健康狀態
      maxEjectionPercent: 100 # 最多踢掉 100% 的 pod，避免全數流量被切斷可視情況調整
```

---

### 7. Mirroring

複製一份真實流量給另一個版本，但不影響使用者，應用場景主要是想要線上測試新版本，但不想影響用戶

```yaml
apiVersion: networking.istio.io/v1
kind: VirtualService
metadata:
  name: httpbin
spec:
  hosts:
- httpbin
  http:
  - route:
- destination:
    host: httpbin
    subset: v1
  weight: 100
mirror:
  host: httpbin
  subset: v2
mirrorPercentage:
  value: 100.0
```

---

### 8. Locality Load Balancing

盡量把流量導到最近的實例，場景多使用在跨多地部署（台灣、東京、美國），要讓台灣用戶優先打到台灣的服務

```yaml
apiVersion: networking.istio.io/v1
kind: DestinationRule
metadata:
  name: helloworld
spec:
  host: helloworld.sample.svc.cluster.local
  trafficPolicy:
    loadBalancer:
      localityLbSetting:
        enabled: true
        distribute: # 地區加權
          - from: region1/zone1/*
            to:
              "region1/zone1/*": 70
              "region1/zone2/*": 20
              "region3/zone4/*": 10
    outlierDetection:
      consecutive5xxErrors: 100
      interval: 1s
      baseEjectionTime: 1m
```

---

### 9. 還有 [Ingress](https://istio.io/latest/docs/tasks/traffic-management/ingress/) 和 [Egress](https://istio.io/latest/docs/tasks/traffic-management/egress/)

## 結論

更細緻的管控外部進來的流量怎麼進出 mesh，完全取決於場景的設計，這裏就不多說了～

## Reference

https://ithelp.ithome.com.tw/articles/10301327

https://ithelp.ithome.com.tw/articles/10301329
