## K8S Lab Day_24

# Day22: Istio 與 CLI 整合：服務拓樸與流量觀察實戰

## 前言

昨天講到 Istio 的 health check 和 holdApplicationUntilProxyStarts 設定，那今天我們就來看看，假如系統 deploy 完成後，我們要怎麼透過 CLI 工具來觀察整個 Mesh 的服務拓樸與流量狀況

## 觀察 Sidecar 狀態

這個是最基本觀察 Envoy sidecar synced 狀況的指令，這裏的 CDS / LDS / RDS / EDS 分別代表 CDS：Cluster Discovery Service，LDS：Listener Discovery Service，RDS：Route Discovery Service，EDS：Endpoint Discovery Service

```bash
istioctl proxy-status
```

輸出的結果會是

![proxy-status output](https://github.com/user-attachments/assets/740633f8-5dc7-4e0e-97cd-8453fa170fd3)

假如任何一項有問題可以用這個指令來去查看他的狀況，但這邊要注意到假如使用的版本是在 `Istio 1.26` 之前就會是輸出跟文件上寫的方式一樣，而全部顯示為 4 (CDS,LDS,EDS,RDS) 就代表該 Pod 的 sidecar 已完全同步控制面下發的流量控制設定

![proxy-status output document](https://github.com/user-attachments/assets/bcc483cc-8785-4a81-91b0-9e5287a32873)

接著假如要查看流量的 configuration 可以使用以下的指令去查詢

```bash
istioctl proxy-config cluster <pod-name>.<namespace>
```

```bash
Pod: productpage-v1-54bb874995-77p7m
   Pod Revision: default
   Pod Ports: 9080 (productpage), 15090 (istio-proxy)
   # 9080 是應用程式 productpage 自身服務的 HTTP port
   # 15090 是 Envoy sidecar 的監控端口，用於 metrics / status
--------------------
Service: productpage
   Port: http 9080/HTTP targets pod port 9080
   # Service export 的 port 名稱是 http（對外 9080），對應 Pod 內部的容器 port 9080
--------------------
Effective PeerAuthentication:
   Workload mTLS mode: PERMISSIVE
   # 這裏 PeerAuthentication PERMISSIVE 表示這個 workload 可同時接受 plaintext 和 mTLS 加密 traffic
--------------------
Exposed on Ingress Gateway http://192.168.200.200
    # 這裏說明服務有被 Ingress Gateway 掛在這個 ip 上
VirtualService: bookinfo
   Match: /productpage*
```

這裏最下面也可以把這裡對應想成(簡化版)，當然還有 PeerAuthentication 等等的配置可以去修改

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: bookinfo
spec:
  hosts:
    - "*"
  gateways:
    - bookinfo-gateway
  http:
    - match:
        - uri:
            prefix: /productpage
      route:
        - destination:
            host: productpage
            port:
              number: 9080
```

還可以使用 istioctl proxy-config 查詢更多的 envoy 配置的狀況

```bash
# 查詢 cluster
istioctl proxy-config cluster <pod>.<namespace>

# 查詢 listener
istioctl proxy-config listener <pod>.<namespace>

# 查詢 route
istioctl proxy-config route <pod>.<namespace>

# 查詢 endpoint
istioctl proxy-config endpoint <pod>.<namespace>
```

## 總結

當然還是有很多很使用的指令可以去查詢的，還是參照於[官網的文件](https://istio.io/latest/docs/reference/commands/istioctl/)吧，可以自己去試試看這些指令，會跳出一些有趣的資訊，一個一個資訊去查就會發現自己更了解為何 istio 的流量會這樣流動和這樣配置！

## Reference

https://istio.io/latest/docs/ops/diagnostic-tools/proxy-cmd/

https://istio.io/latest/docs/reference/commands/istioctl/
