## K8S Lab Day_20

# Day18: Istio Ingress Gateway 的進階流量管理：mTLS、TLS Termination

## 前言

昨天說明到 VirtualService / DestinationRule / ServiceEntry 的基礎，今天要往 Ingress Gateway 的細節邁進～

## Ingress Gateway 與 mTLS

為何會需要使用到 mTLS 呢？因為傳統的服務，通常都只需要單向的認證就好，因為他需要實現 Zero-Trust Networking，確保服務與服務之間都是透過身份驗證加密的，在傳統的 TLS 網路中，一旦流量進入了防火牆內部，他就是被信任的，但是在 cloud native 的環境中，已經變得模糊了，主要是原本是 monolithic 的架構，可以較好的被判斷，但是現在為服務跟微服務之間的保護又變為更加的重要，每個 pod 都有獨有的 IP 但又一直創建和銷毀，變成我們要去增加這個邊界的安全性就是使用 mTLS 來去達到

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: ingress-mtls
  namespace: istio-ingress
spec:
  selector:
    matchLabels:
      istio: ingressgateway
  mtls:
    mode: STRICT
```

這邊就是一個強制 Ingress Gateway 只會接受 mTLS 連線的範例，在這邊的 mtls.mode 設定是 STRICT 代表他的 workload 只能接受 mTLS 的 Traffic，還有其他的像是 default 的 `PERMISSIVE` 寬鬆版本; `DISABLE` 禁用 mTLS，除非有自己的安全 solution 不然通常不建議; 還有 `UNSET` 直接繼承父級的設定

## TLS Termination at Gateway

剛剛有提到 mTLS，是屬於 istio 這個世界觀下的加密，但在流量進來之前他還是 TLS 的連線，這時候就需要 Termination，當流量送到 Gateway 的時候，Gateway 會負責解密，並透過 sidecar 的 mTLS 加密傳給 Mesh 內部的 Service，或是也可以直接將 mode 設定為 passthrough，直接讓後端的服務去處理，這樣就沒有 termination 到了 Zzzz

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: ingress-gateway
  namespace: istio-system
spec:
  selector:
    istio: ingressgateway
  servers:
    - port:
        number: 443
        name: https
        protocol: HTTPS
      tls:
        mode: SIMPLE
        credentialName: istio-ingress-cert # 由 Secret 存放憑證
      hosts:
        - "example.com"
```

像是以下的範例為更多元的處理方式，今天假如進來的流量是 http 沒有被加密，它就會幫我們自動 redirect 到 HTTPS 的 port，這樣就可以強制加密讓整個流程增加安全性

```yaml
servers:
  - hosts:
      - "*.danielstechblog.de"
    port:
      number: 80
      name: http
      protocol: HTTP
    tls:
      httpsRedirect: true
```

接下來我們就要看到 `PASSTHROUGH`，為何這邊要使用這個？在這邊 Gateway 它只是 L4 轉發，無法查看或修改數據，這對於需要滿足如 HIPAA、PCIDSS 等嚴格合規性要求的場景相當的重要，因為它確保了數據在整個傳輸路徑中都處於加密狀態

```yaml
- hosts:
    - "*.tls.danielstechblog.de"
  port:
    number: 10443
    name: tls-passthrough
    protocol: TLS
  tls:
    mode: PASSTHROUGH
```

另外，這邊再設定 Gateway 的 host 參數的時候，要特別注意使用 wildcard 的設定，因為這會有 Same Domain Conflict 的問題去影響到 TLS 的行為，如果想要同時在 443 port 上做 TLS termination 和 TLS passthrough，那就必須用完整的 FQDN，而不能只靠萬用字元，不只是 Gateway 連 VirtualService 也一樣要用 FQDN 來設定

## FQDN

FQDN 的全名是 Fully Qualified Domain Name，從最左邊的子域名一直寫到最右邊的 root domain，像我們常常會看到 host 的設定是 `productpage.bookinfo.svc.cluster.local` 這就是一個 FQDN，而他又是什麼 `<service>.<namespace>.svc.cluster.local` 看成這樣應該就很清楚了吧，這個是 service 的，那 pod 也有屬於他的 FQDN `<pod-ip>.<namespace>.pod.cluster.local`，這樣我們就可以更容易和清楚的知道 cluster 的服務是怎麼分配的，也可以讓 DNS 快速的 lookup 到相對應的資源

## 總結

今天介紹到了流量送進來 Ingress 可以去做到的設定，另外還講到了 FQDN，一直都沒有機會提到，感覺慢慢地把我們腦中 istio 不齊的地方慢慢梳理順了～

## Reference

https://istio.io/latest/docs/tasks/traffic-management/ingress/ingress-sidecar-tls-termination/

https://jimmysong.io/blog/understanding-the-tls-encryption-in-istio/

https://www.cnblogs.com/whuanle/p/17538694.html

https://www.danielstechblog.io/run-the-istio-ingress-gateway-with-tls-termination-and-tls-passthrough/
