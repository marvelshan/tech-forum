## K8S Lab Day_28

# Day26: Istio 驗證錯誤與 Webhook 問題排查

## 前言

昨天說到了 sidecar 的效能 tuning，常常在部署 istio 的時候，明明已經寫好了 `DestinationRule` 或 `VirtualService`，語法檢查也都通過，但一執行 kubectl apply -f，系統卻沒辦法正確的安裝，我們就來跟著官方文件來排查看看～

## 1. 好像看起來正確？

我們在前幾天有介紹到 istioctl 的指令，這時候假如遇到 YAML 沒辦法正確的去安裝之後，我們就可以利用以下的指令來去查詢 istio 的狀況

```bash
istioctl validate -f your-config.yaml
istioctl analyze
```

這兩個命令會執行靜態驗證與整體配置關聯分析，可以看到像是錯誤的 API schema、缺少必要欄位、無效的 namespace 範圍等等，這些很多很小的工作都會導致 istio webhook 跳出錯誤的

## 2. 真的沒錯呀！

明明 configuration 真的沒錯呀！那有可能是 validating webhook 沒在運作，我們可以利用這個指令來去了解 istio-validator-istio-system 的配置

```bash
kubectl get validatingwebhookconfiguration istio-validator-istio-system -o yaml
```

我們就會看到跳出了一個 yaml 檔，我們可以觀察到 `caBundle` 這個不能是空的，這個是 istiod 給的憑證，還有 service 指向的 ns 和 path 要是正確的，最後還有 `failurePolicy` 建議是 Fail，這是確保驗證失敗時不放行

## 3. x509 啥勒錯了？！

假如有看到 `x509: certificate signed by unknown authority` 通常是 webhook 的 caBundle 有錯，而我們要怎麼排查呢

首先我們要確認 `istiod` 是否有正常運行

```bash
kubectl -n istio-system get pod -lapp=istiod
```

```bash
NAME                      READY   STATUS    RESTARTS   AGE
istiod-7dfd5689c8-swqkb   1/1     Running   0          6h10m
```

然後要查看 istiod 日誌中是否出現 patching 錯誤，假如有 `failed to patch caBundle` 或 `permission denied`，需要檢查 `RBAC` 的權限

```bash
for pod in $(kubectl -n istio-system get pod -lapp=istiod -o jsonpath='{.items[*].metadata.name}'); do
  kubectl -n istio-system logs $pod
done
```

接著要驗證 ClusterRole 權限，官方文件說是 `istiod-istio-system`，但我們這邊查一下應該是版本問題要用 `istiod-clusterrole-istio-system` 才查得到

```bash
kubectl get clusterrole istiod-clusterrole-istio-system -o yaml
```

這邊就會看到 rules 有 `validatingwebhookconfigurations` 這個的權限

```bash
rules:
- apiGroups:
  - admissionregistration.k8s.io
  resources:
  - mutatingwebhookconfigurations
  verbs:
  - get
  - list
  - watch
  - update
  - patch
- apiGroups:
  - admissionregistration.k8s.io
  resources:
  - validatingwebhookconfigurations
  verbs:
  - get
  - list
  - watch
  - update
```

## 4. `no such host` 和 `no endpoints available`

在建立 Istio 時出現這兩個錯誤時，是在 webhhok 嘗試建立 istiod 時失敗的，原因通常是 istiod Pod 尚未啟動完成或沒有可用 Endpoint

```bash
kubectl -n istio-system get pod -lapp=istiod
kubectl -n istio-system get endpoints istiod
```

```bash
NAME                      READY   STATUS    RESTARTS   AGE
istiod-7dfd5689c8-swqkb   1/1     Running   0          6h15m
NAME     ENDPOINTS                                                                    AGE
istiod   10.233.118.179:15012,10.233.118.179:15010,10.233.118.179:15017 + 1 more...   10d
```

假如沒有 endpoints，代表 istiod 並未成功啟動或 service selector 錯誤，可以透過以下繼續判斷是哪裡有問題

```bash
for pod in $(kubectl -n istio-system get pod -lapp=istiod -o jsonpath='{.items[*].metadata.name}'); do
  kubectl -n istio-system describe pod $pod
  kubectl -n istio-system logs $pod
done
```

## 總結

以前在沒有那麼詳細的在看文件的時候，都馬是隨便亂試，試到成功為止，不然就在那邊埋頭苦幹，都不知道原來文件上面有那麼多好方法可以使用，好像又那麼認識一點 istio 了（開腸剖腹 XD）～

## Reference

https://istio.io/latest/docs/ops/common-problems/validation/

https://istio.io/latest/docs/reference/commands/istioctl/#istioctl-analyze

https://cloud.google.com/service-mesh/v1.26/docs/troubleshooting/troubleshoot-webhook?hl=zh-tw
