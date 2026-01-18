## K8S Lab Day_26

# Day24: 打造 istio 微服務的 Zero-Trust 安全，OPA 的進階授權實戰

## 前言

昨天講到如何使用 WASM plugin 去客製化 service mesh 的流量，今天提到的是在傳統的網路安全架構下，內部網路都是預設是可信的，在介紹 mTLS 的時候有提到因為現今較多的微服務和雲服務的架構下，就產生出了 Zero-Trust，Never trust, always verify 就變為更加的重要

從前幾天的訓練，我們應該可以次性的把 yaml 完成

```yaml
apiVersion: security.istio.io/v1beta1
kind: RequestAuthentication
metadata:
  name: google-jwt
  namespace: default
spec:
  selector:
    matchLabels:
      app: productpage
  jwtRules:
    - issuer: "https://accounts.google.com"
      jwksUri: "https://www.googleapis.com/oauth2/v3/certs"
      audiences:
        - "<your-service-audience>" # 替換為實際的服務或應用程式 ID
---
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: allow-jwt-users
  namespace: default
spec:
  selector:
    matchLabels:
      app: productpage
  action: ALLOW
  rules:
    - from:
        - source:
            requestPrincipals: ["https://accounts.google.com/*"]
      to:
        - operation:
            methods: ["GET"]
            paths: ["/api/public"]
    - from:
        - source:
            requestPrincipals: ["https://accounts.google.com/special-user"]
      to:
        - operation:
            methods: ["POST"]
            paths: ["/api/admin"]
---
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default-mtls
  namespace: default
spec:
  mtls:
    mode: STRICT
```

接著就可以直接驗證了！

```bash
export INGRESS_IP=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
curl -v http://$INGRESS_IP/api/public
```

假如都是安裝正確的話就會出現 `401 Unauthorized` 的 return 訊息

## OPA 與 Istio 整合強化 Zero-Trust

接著我們要來介紹 OPA，為什麼要用 OPA？雖然 AuthorizationPolicy 可以依據來源與 JWT claim 做到 fine-grained control，但還是受限於 yaml 的規格，沒辦法進行太多的邏輯判斷，這時候我們就可以使用到 `Open Policy Agent` 把邏輯抽象到 `Rego` 這個語法當中，達到可以動態調整授權的策略

![opa](https://github.com/user-attachments/assets/bb73a644-d7a1-46e0-ab9d-e3e73c20afae)

```bash
kubectl apply -f https://raw.githubusercontent.com/open-policy-agent/opa-envoy-plugin/main/examples/istio/quick_start.yaml
```

接著我們就可以設定剛剛提到的 rego 檔來去控制授權

```rego
## authz.rego
package istio.authz

# 預設 deny，除非符合 allow 的規則
default allow := false

# 允許健康檢查的 GET 請求
allow if {
    input.parsed_path[0] == "health"
    input.attributes.request.method == "GET"
}

# 根據使用者角色與權限授權
allow if {
    # 取得使用者角色
    some user_role in _user_roles[_user_name]
    # 取得角色的權限清單
    some permission in _role_permissions[user_role]
    # 比對 request method 與 path 是否符合角色權限
    permission.method == input.attributes.request.http.method
    permission.path == input.attributes.request.http.path
}

# 從 Authorization header 中解析出使用者名稱
_user_name := parsed if {
    [_, encoded] := split(input.attributes.request.http.headers.authorization, " ")
    [parsed, _] := split(base64url.decode(encoded), ":")
}

# 定義每個使用者的角色
_user_roles := {
    "alice": ["guest"],
    "bob": ["admin"],
}


# 定義每個角色對應的權限
_role_permissions := {
    "guest": [{"method": "GET", "path": "/productpage"}],
    "admin": [
        {"method": "GET", "path": "/productpage"},
        {"method": "GET", "path": "/api/v1/products"},
    ],
}
```

接著我們就可以在 Istio Mesh 中註冊 OPA External Authorizer

```bash
kubectl edit configmap -n istio-system istio
```

需要在 `data.mesh` 區塊中加入以下內容

```yaml
extensionProviders:
  - name: opa-ext-authz-grpc
    envoyExtAuthzGrpc:
      service: opa-ext-authz-grpc.local
      port: 9191
```

接著讓 target namespace 自動注入 OPA 與 Istio 的 sidecar

```bash
kubectl label namespace default opa-istio-injection="enabled"
kubectl label namespace default istio-injection="enabled"
```

然後接著就是測試的流程了

```bash
export INGRESS_IP=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# alice 帶 Basic Auth
curl -v -H "Authorization: Basic YWxpY2U6cGFzc3dvcmQ=" http://$INGRESS_IP/productpage

# bob 帶 Basic Auth
curl -v -H "Authorization: Basic Ym9iOnBhc3N3b3Jk" http://$INGRESS_IP/api/v1/products
```

以上做完就是測試成功啦！

## 總結

這種 opa 的做法算是把授權的規則外部化，因為今天控制這些權限的假如是後端的夥伴，也不太可能讓他來來回回的去改 yaml 檔，所以這樣抽象化出來就可以很好的去管理授權的規則，詳細的 opa rego 的語法可以再去到[官網查詢文件](https://www.openpolicyagent.org/docs/policy-language)呦～

## Reference

https://www.solo.io/blog/compliance-zero-trust-istio-ambient-mesh

https://istio.io/latest/docs/reference/config/security/request_authentication/

https://www.solo.io/blog/fine-grained-service-authorizations-istio-opa

https://www.openpolicyagent.org/docs/envoy/tutorial-istio
