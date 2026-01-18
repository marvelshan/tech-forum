## K8S Lab Day_11

# Day9: Istio 安全防線實戰，mTLS Identity 與 Authorization Policy 的跨命名空間存取控制

## 前言

昨天實作了 Istio 基本的指令，那今天要來操作 Mesh 基本流量控制 mTLS、Policy、Timeout、Retry，要來啟用 mTLS，實作流量限速、timeout 與 retry 規則，驗證失敗重試與熔斷行為，並且要來試試看 Canary / A/B 測試實作，又要來比較看看跟原本 k8s deployment 的 rolling update 又有什麼差別呢？但應該會分成很多天說～

## Istio identity

在任何分散式系統中，Identity 都是安全的基礎，Istio 也不例外，Istio 採用 X.509 certificates 來識別與驗證各個 workload，istiod 提供 gRPC API 負責接收憑證簽發請求，Istio agent 啟動時會在本地產生一組 private key 與 CSR，並帶著憑證請求送給 istiod，istiod 的 CA 驗證請求會檢查 CSR 的身份是否合法，若通過則簽發憑證，Envoy 透過 SDS API 取得憑證，在 workload 啟動後，sidecar Envoy 向 Istio agent 請求憑證與金鑰，Istio agent 回傳憑證與金鑰並透過 SDS API 傳給 Envoy，最後他會自動 Rotation，Istio agent 會監控憑證有效期限，並在快到期時重新執行簽發流程

在這個過程是不是跟 HTTPS 的憑證機制很像，HTTPS 也是用 X.509 certificates 來驗證伺服器身份，並建立加密通道其中過程的差異是 HTTPS 所以賴的第三方 CA 就像是 Istio 內建的 istiod，HTTPS 主要針對 client side 和 server side 的單向驗證，而 istio 是針對服務與服務之間的雙向驗證

<img width="380" height="423" alt="截圖 2025-09-21 上午11 03 39" src="https://github.com/user-attachments/assets/8454a69e-f1c3-45d2-938a-c18dc8044bbe" />


> Istio 的 identity 機制其實就是把「HTTPS 憑證的觀念」套用到服務之間，只是它做到更**自動化、內建化、以 workload 為中心**，這就是為什麼能在大規模微服務環境裡實現 **零信任 (Zero Trust)** 的原因

## Authentication

在 Istio 中，Authentication 是透過 Envoy sidecar 來執行的，所有 service-to-service 的流量都會先進入本地的 Envoy，再透過 mTLS 建立安全通道，首先 client 發出的 Outbound 流量會被攔截並轉發到 local sidecar Envoy，client side Envoy 與 server side Envoy 進行 mTLS 握手，client side Envoy 還會檢查 server side Envoy 的憑證，確保它對應正確的 service account，雙方 Envoy 建立好加密的 mTLS 連線後，才會傳送實際的應用流量，server side Envoy 收到流量後，會進行授權檢查，若符合規則才轉發給後端服務

<img width="640" height="466" alt="截圖 2025-09-21 上午11 11 12" src="https://github.com/user-attachments/assets/9ad341ab-6b4c-4b7f-983a-fdc203f2c5c6" />

## Authorization Policy

在 Istio 中，它提供了 mesh- 、namespace- 與 workload- 三種範圍，讓你能夠依照需求逐步導入 mTLS 與存取控制，它的好處是支援 workload-to-workload 以及 end-user-to-workload 的授權模式，提供統一的 AuthorizationPolicy CRD，支援自訂條件，可以根據 request header、IP、service account 等屬性決定是否允許或拒絕

<img width="725" height="336" alt="截圖 2025-09-21 上午11 14 42" src="https://github.com/user-attachments/assets/efa0d830-3ca0-41b6-9b98-7d94f1d8e2b9" />

## 實作

因為今天會使用到 istioctl，我們需要下載 istioctl 的套件，但是回想起前幾天，我們需要做到環境版控，所以我們就會需要在 `flake.nix` 加上 `istioctl`

```nix
...
      devShells.${system}.default = pkgs.mkShell {
        buildInputs = with pkgs; [
          ansible_2_16  # 指定 Ansible 版本，與 Kubespray 相容
          python3
          python3Packages.pip
          python3Packages.netaddr
          python3Packages.jmespath
          kubectl
          kubernetes-helm # Helm CLI
          istioctl        # 加入 Istio CLI
        ];
...
```

然後再跑這個指令，並等他把包下載下來

```bash
nix develop
```

我們就可以用這個指令來檢查是否安裝完畢

```bash
istioctl version
```

接著我們要做一個小實驗來更了解 istio identity 中的運作，在 Istio 裡，每個 workload 會透過 SPIFFE ID (`spiffe://cluster.local/ns/<namespace>/sa/<serviceaccount>`) 來識別身分，這個 ID 是 mTLS 憑證裡自動帶的，我們就用 AuthorizationPolicy 來驗證「只有特定 Identity 才能存取服務」

主要的目標我們會建立兩個 ns（team-a 和 team-b），分別會部署 httpbin (server) 與 curl (client)，設定一個 AuthorizationPolicy，讓 httpbin 只允許來自 team-a 的 Pod 存取，去驗證 curl in team-a 是可以存取，curl in team-b 是被拒絕

- 在實作前可以先確認環境 istio 的 control plan 是否正確運行

```bash
kubectl get pods -n istio-system
```

1. 先 clone 我們要測試的 sample file

```bash
git clone https://github.com/istio/istio.git
cd istio
```

2. create ns

```bash
kubectl create ns team-a
kubectl create ns team-b
```

3. deploy httpbin + curl（帶 sidecar）

```bash
kubectl apply -n team-a -f <(istioctl kube-inject -f samples/httpbin/httpbin.yaml)
kubectl apply -n team-a -f <(istioctl kube-inject -f samples/curl/curl.yaml)
# 會需要 curl 和 httpbin 這兩個來驗證和測試，httpbin 提供測試 API 的服務端，curl 模擬客戶端發送請求

kubectl apply -n team-b -f <(istioctl kube-inject -f samples/httpbin/httpbin.yaml)
kubectl apply -n team-b -f <(istioctl kube-inject -f samples/curl/curl.yaml)
```

```bash
kubectl get pod -n team-a
# 確認是否建立完畢
NAME                      READY   STATUS    RESTARTS   AGE
curl-559c7c864d-llrqg     2/2     Running   0          37m
httpbin-5974566f6-85xk5   2/2     Running   0          37m
```

4. 稍微等他一下去 apply 我們所用到 pod，接著去驗證是否都有正確連線到

```bash
kubectl exec -n team-a \
  -it $(kubectl get pod -n team-a -l app=curl -o jsonpath={.items[0].metadata.name}) \
  -c curl -- curl -sS http://httpbin.team-a:8000/ip

# 進入 team-a namespace 裡的 curl Pod，其中會執行 get pod 獲取名稱，並執行 curl 去呼叫 httpbin.team-a:8000/ip
```

output:

```
{
  "origin": "127.0.0.6:38429"
}
```

```bash
kubectl exec -n team-b \
  -it $(kubectl get pod -n team-b -l app=curl -o jsonpath={.items[0].metadata.name}) \
  -c curl -- curl -sS http://httpbin.team-a:8000/ip
```

output:

```
{
  "origin": "127.0.0.6:43307"
}
```

但你在下次執行這個指令又會看到不同的 source port 這是為什麼呢？在 TCP/IP 通訊裡，client 連到 server 的時候，每次建立新的 TCP 連線，kernel 會自動分配一個臨時的 source port（ephemeral port），所以你會看到不同的 source IP:port 組合，每次你執行 kubectl exec … curl ...，其實都是在 Pod 裡重新跑一個新的 curl process，它就會重新發起一次新的 TCP 連線，kernel 會再分配一個新的 source port

5. 接著要建立 AuthorizationPolicy 去限制 Identity（`allow-only-team-a.yaml`）

```yaml
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: httpbin-policy
  namespace: team-a
spec:
  selector:
    matchLabels:
      app: httpbin
  rules:
    - from:
        - source:
            principals: ["cluster.local/ns/team-a/sa/curl"]
```

```bash
kubectl apply -f allow-only-team-a.yaml
```

6. 接著也要過一段時間等他把 policy apply 上去驗證才會成功，但也可以在指令一下完馬上去測試，去感受他部署的時間

```bash
kubectl exec -n team-a \
  -it $(kubectl get pod -n team-a -l app=curl -o jsonpath={.items[0].metadata.name}) \
  -c curl -- curl -s -o /dev/null -w "%{http_code}\n" http://httpbin.team-a:8000/ip

# 200
```

```bash
kubectl exec -n team-b \
  -it $(kubectl get pod -n team-b -l app=curl -o jsonpath={.items[0].metadata.name}) \
  -c curl -- curl -s -o /dev/null -w "%{http_code}\n" http://httpbin.team-a:8000/ip

# 403
```

你也可以在執行一次上面那個指令，他會回覆`RBAC: access denied`

7. 最後記得清理完實驗的環境

```bash
kubectl delete ns team-a team-b
kubectl delete -f allow-only-team-a.yaml
```

## 結論

Istio 會根據 sidecar Envoy 帶的 SPIFFE Identity 判斷來源 ServiceAccount 與 Namespace，並且我們可以想想看這個的應用場景，是不是在公司有各個團隊或是各個服務，每個服務與服務之間都有敏感的資料，我們針對 team a ns 的 ServiceAccount 開放權限，但 team-b 就會被 deny，這樣我們在 AuthorizationPolicy 就可以更有效率的管控身份的驗證

## Reference

https://istio.io/latest/docs/reference/config/security/authorization-policy/

https://github.com/istio/istio/tree/master/samples/curl

https://github.com/istio/istio/tree/master/samples/httpbin

https://istio.io/latest/docs/reference/config/security/request_authentication/
（有空可以搭配 RequestAuthentication 來玩玩看）
