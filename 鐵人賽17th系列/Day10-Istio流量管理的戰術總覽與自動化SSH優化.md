## K8S Lab Day_12

# Day10: Istio 流量管理的戰術總覽與自動化 SSH 優化

## 前言

昨天做了 istio 比較基本可以做到的功能認證和授權，接著就是比較應用層面的，今天會講到比較應用層面的場景，在微服務架構下，traffic management 是一件很重要的事，今天就稍微來介紹一下吧

另外，我自己在操作的時候因為都要一直連線 ssh，然後每次都要先進到放 `flake.nix` 的資料夾底下，再執行 `nix develop`，所以我就想說優化他，在 `~/.bashrc` 檔案內加上我需要執行的指令

```bash
vi ~/.bashrc
```

```bash
if [ -z "$IN_NIX_DEVELOPMENT" ]; then
  # Set a flag to prevent re-running in the subshell
  echo $IN_NIX_DEVELOPMENT
  export IN_NIX_DEVELOPMENT=true
  if [ -z "$SSH_ORIGINAL_COMMAND" ]; then
    cd ~/kubespray-nix
    /home/ubuntu/.nix-profile/bin/nix develop
  fi
fi
```

將這些放入整個檔案的最後面，這邊的邏輯給大家自己細細品味，其中的 `/home/ubuntu/.nix-profile/bin/nix develop` 這邊不是使用 `nix develop` 是因為 Shell 腳本的執行環境中，nix 這個指令可能不在 $PATH 環境變數裡，當腳本在 ~/.bashrc 裡自動執行時，$PATH 可能還沒有被完整設定，導致 Shell 找不到 nix 這個指令，從而報錯 `Command 'nix' not found`

## Traffic Management

### 1. Request Routing

是根據請求的條件（HTTP Header、URL Path、使用者 ID）把流量導到不同版本的服務，應用的場景通常在你要做 A/B 測試 或 Canary 部署

```yaml
spec:
  hosts:
    - reviews
  http:
    - match:
        - headers:
            end-user:
              exact: jason
      route:
        - destination:
            host: reviews
            subset: v2
    - route:
        - destination:
            host: reviews
            subset: v1
```

---

### 2. Fault Injection

在某些強況下會有一些 delay 或 error code

```yaml
spec:
  hosts:
    - ratings
  http:
    - fault:
        abort:
          httpStatus: 500
          percentage:
            value: 100
      match:
        - headers:
            end-user:
              exact: jason
      route:
        - destination:
            host: ratings
            subset: v1
    - route:
        - destination:
            host: ratings
            subset: v1
```

---

### 3. Traffic Shifting

把流量比例分配到不同版本，在部署的時候有時會需要做 Canary release 這種漸進式的部署，像是先給 10% v2 確認穩定後再拉到 50% 最後 100%

```yaml
apiVersion: networking.istio.io/v1
kind: DestinationRule
metadata:
  name: reviews
spec:
  host: reviews
  subsets:
    - name: v1
      labels:
        version: v1
    - name: v2
      labels:
        version: v2
```

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
            subset: v1
          weight: 90
        - destination:
            host: reviews
            subset: v2
          weight: 10
```

然後在調整成

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
            subset: v1
          weight: 50
        - destination:
            host: reviews
            subset: v2
          weight: 50
```

最後

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
          weight: 100
```

## 結論

因為真的蠻多種應用場景的，大家可以花些時間去玩玩看文件上面的一些流程，會更加認識 DestinationRule 和 VirtualService 的功能的～

## Reference

https://ithelp.ithome.com.tw/articles/10301327

https://ithelp.ithome.com.tw/articles/10301329

https://medium.com/brobridge/%E6%98%8F%E5%80%92-service-mesh-%E5%8E%9F%E4%BE%86%E4%B8%8D%E6%98%AF%E7%B6%B2%E6%A0%BC%E6%9C%8D%E5%8B%99-9a4b0636371f
