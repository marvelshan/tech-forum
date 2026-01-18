## K8S Lab Day_29

# Day27: Istio 與 Kubernetes Admission Webhook：動態注入與驗證機制解析

## 前言

昨天有講到 webhook 的內容，但前面好像都沒有特別提到，但是這個又特別重要，在 k8s 中 Admission Webhook 是一項強大的擴展機制，可以動態的 mutation 和 validation，這種設計讓系統能夠在不修改 API Server source code 的情況下，實現如自動 Sidecar 注入、自訂驗證規則等進階功能，Istio 也利用這個 Webhook 機制實現了 Envoy Sidecar 自動注入，讓使用者不需要在每個 Pod YAML 裡手動新增 proxy 容器

## 什麼是 Admission Webhook

可以看到下面這張圖，當使用者透過 kubectl apply 或 API 請求建立資源時，會經歷下面的階段，k8s 提供了兩個 webhook，MutatingAdmissionWebhook 是用於修改請求（例如自動注入容器），ValidatingAdmissionWebhook 是用於驗證請求是否合法（例如檢查配置）

![Admission Webhook](https://github.com/user-attachments/assets/bf0939a0-6236-4c3b-9588-d1aaa3b5fe7b)

## 那 istio 又跟 Admission Webhook 的關係？

Istio 在 Mesh 內使用 Mutating Webhook 自動將 Envoy Sidecar 注入至 Pod 中，Validating Webhook 來驗證 Istio 資源（如 VirtualService、DestinationRule）的正確性，而這兩個 webhook 都是由 control plane Istiod 來託管

在部署或 debug 之前我們要確認一下 k8s 的一些狀況，首先我們要先確認 webhook 要在 k8s 1.29 以上

```
bash
kubectl version --short
```

```
Client Version: v1.32.5
Server Version: v1.32.5
```

再來要確認 admissionregistration API 已啟用

```bash
kubectl api-versions | grep admissionregistration.k8s.io/v1
```

確認完之後我們就要開始了解 Mutating Admission Webhook，Webhook 會自動攔截 Pod 建立請求，並在 Pod 被建立前自動加入 Envoy 容器，這也是 Automatic Injection 的方法，另外一種就是手動 injection，透過 istioctl kube-inject 將 Sidecar YAML 手動加入到 Pod 的配置中

最簡單的做法是對 ns 加上標籤，Istio 的 Sidecar Injector Webhook 會根據 namespaceSelector 設定，只對帶有 istio-injection=enabled 的 Namespace 進行攔截並注入

```bash
kubectl label ns demo istio-injection=enabled
```

假如要確認現在 ns 的狀態可以執行以下

```bash
kubectl get namespace -L istio-injection
```

Istio 的 MutatingAdmissionWebhook 定義了兩個重要條件，Namespace 必須有 istio-injection=enabled，且 Pod 不能標註 sidecar.istio.io/inject=false，符合以上條件時，Webhook 才會執行 Sidecar 注入

```yaml
namespaceSelector:
  matchLabels:
    istio-injection: enabled
objectSelector:
  matchExpressions:
    - key: sidecar.istio.io/inject
      operator: NotIn
      values:
        - "false"
```

在某些情況下，像是第三方應用、Pipeline 暫時性 Pod ，不方便在 Pod YAML 加 Annotation，Istio 提供了黑白名單機制來補充，這兩個設定存在於 istio-sidecar-injector ConfigMap 中，而這邊也優先的順序：Pod Annotation→neverInjectSelector→alwaysInjectSelector→default policy，也就是說若 Pod 加了 annotation，一律以 annotation 為準，若沒有 annotation，看是否命中黑名單 (neverInjectSelector)，命中則不注入，最後才是依據 default policy 決定

```bash
kubectl get configmap istio-sidecar-injector -n istio-system -o yaml
```

```yaml
neverInjectSelector:
  - matchLabels:
      sidecar/autoinject: disabled
  - matchExpressions:
      - key: sidecar-inject
        operator: In
        values:
          - "false"
          - "disabled"

alwaysInjectSelector:
  - matchLabels:
      sidecar/autoinject: enabled
  - matchExpressions:
      - key: sidecar-inject
        operator: In
        values:
          - "true"
          - "enabled"
```

## 總結

今天又抓抓頭，Automatic Sidecar Injection 是 Istio 控制流量與策略落地的核心環節，假如能夠掌握他的運作邏輯，就能再多環境的情況下更彈性的去管理 injection 的策略

## Reference

https://www.zhaohuabing.com/2018/05/23/istio-auto-injection-with-webhook/

https://morpheushuang.medium.com/istio-automatic-sidecar-injection-543-4981ae7375f7

https://istio.io/latest/docs/ops/configuration/mesh/webhook/
