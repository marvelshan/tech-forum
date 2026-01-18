## K8S Lab Day_10

# Day8: Istio 實戰開戰，從 Helm 部署到 Control Plane 的維運戰術

## 前言

昨天介紹完 Service mesh 的概念，沒有實作過還是不太清楚他在幹嘛，今天就要來實際操作透過實作來一探究竟，實作用的工具選用 Istio，市面上也比較常見，社群也比較完善，假如有機會未來希望能講到 Ambient mode，自己蠻喜歡的部分～

## 安裝 Istio

```bash
helm repo add istio https://istio-release.storage.googleapis.com/charts
helm repo update
```

眼尖的夥伴一定會發現在我們寫 nix 安裝檔的時候，在 `buildInputs` 這邊就有寫到 `helm` 的包需要安裝，也就是在這個時候啦，因為要實際演示，就沒有在 nix 下直接寫 command 直接把 istio 安裝起來

```
buildInputs = with pkgs; [
    ansible_2_16 # 指定 Ansible 版本，與 Kubespray 相容
    python3
    python3Packages.pip
    python3Packages.netaddr
    python3Packages.jmespath
    kubectl
    kubernetes-helm # option：如果需要 Helm
];
```

接著就繼續完成 istio 的基礎設定

```bash
kubectl create namespace istio-system
helm install istio-base istio/base -n istio-system
helm install istiod istio/istiod -n istio-system --wait
```

為何要下這三個指令呢？

1. `kubectl create namespace istio-system`：這個比較簡單，就是在 Kubernetes 裡先創建一個專門給 Istio 用的 Namespace，需要把 Istio 的資源（Pod、Service、Config）都放在這個區域，避免跟其他應用互相干擾

2. `helm install istio-base istio/base -n istio-system`：把 Istio 的基礎元件安裝進來，Base 裡面包含一些共用的 CRD（Custom Resource Definition）和基本設定，是 Istio 運作的核心依賴

3. `helm install istiod istio/istiod -n istio-system --wait`：Istiod 是 Istio 的 Control Plane，負責統一管理 Sidecar 的設定，處理流量規則、策略、證書、自動注入等功能，`--wait` 參數的意思是等這個 Helm Release 全部啟動完成、Pod 都 ready 了再回到命令行，確保 Control Plane 已經可以使用

然後安裝完畢你就會看到

```
"istiod" successfully installed!

To learn more about the release, try:
  $ helm status istiod -n istio-system
  $ helm get all istiod -n istio-system
```

然後好奇心驅使了我打了 `helm get all istiod -n istio-system`，你就會發現你的 terminal 就會炸出一堆你會直接 cirl C 謝謝再聯絡的資訊，裡面就包含了一些 helm release 的 metadata，還有生成的 k8s 資源，包含 Deployment、Service、ConfigMap、Secret、CRD 等等，那你會想說為何需要那麼多，其實是因為 Istio 的 control plane 本身就很複雜，而 Helm 為了完整呈現這個 Release 的內容，把所有生成的資源都列出來了

那如果你只是想快速看看 Istio 安裝後到底建立了哪些東西，不需要被 helm get all 轟炸，建議可以先用這幾個指令：

```bash
# 查看所有 namespace 下的 Pod，確認 Istio Control Plane 啟動狀態
kubectl get pods -n istio-system

# 查看所有 Service，確認 Istio 的各個入口和內部服務
kubectl get svc -n istio-system

# 查看所有 Deployment，了解 Control Plane 組成
kubectl get deploy -n istio-system

# 查看所有 CRD（Custom Resource Definition），Istio 用它來擴展 Kubernetes API
kubectl get crds | grep istio.io

# 也可以利用官方文件上的指令來確認 istiod 服務是否有成功建立
kubectl get deployments -n istio-system --output wide
```

會建立也要會刪除誒，這邊也附上刪除的指令吧

```bash
helm delete istiod -n istio-system
helm delete istio-base -n istio-system
kubectl delete namespace istio-system
```

## Reference

https://istio.io/latest/docs/setup/install/helm/

https://ithelp.ithome.com.tw/articles/10220314

https://ithelp.ithome.com.tw/articles/10301328

https://ithelp.ithome.com.tw/articles/10306305?sc=iThelpR
