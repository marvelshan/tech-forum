# KubeRay issue 學習

## 一、[Issue #3793](https://github.com/ray-project/kuberay/issues/3793)

<img width="1271" height="659" alt="issue #3793" src="https://github.com/user-attachments/assets/c1fcde9c-3cb8-47fd-85ae-a1d95176b611" />

### 使用者遇到的實際場景

> 在 Kubernetes 上用 **containerd + NVIDIA GPU runtime**（例如 k3s 或 RKE）部署 KubeRay 集群
> 要讓 Pod 能正確使用 GPU，**必須在 PodSpec 中設定 `runtimeClassName: nvidia`**

但目前 KubeRay 的 Helm chart **不支援直接在 `values.yaml` 裡設定這個欄位**

### 當前限制

KubeRay 的 Helm template 只會 render 一個「白名單」內的 PodSpec 欄位（例如 `containers`, `volumes`, `affinity` 等），而 **`runtimeClassName` 不在這個白名單中**

結果：

- 即使在 `values.yaml` 寫了：
  ```yaml
  worker:
    template:
      spec:
        runtimeClassName: nvidia
  ```
- Helm 會 **靜默忽略這行**（不會報錯，但也不會套用）
- 最終生成的 `RayCluster` CR 中 **沒有 `runtimeClassName`**，導致 GPU 無法使用

### 臨時 workaround

使用者只能在 Helm 安裝後，手動 patch：

```bash
kubectl patch raycluster <name> --type=json -p='[{"op":"add","path":"/spec/workerGroupSpecs/0/template/spec/runtimeClassName","value":"nvidia"}]'
```

這很不方便，尤其在 CI/CD 或自動化部署中

---

## 二、如何思考這個問題？

### Step 1：確認這是「功能缺失」而非 bug

- 這不是程式崩潰或邏輯錯誤，而是 **Helm chart 缺少對某個合法 Kubernetes 欄位的支援**
- 屬於 **UX / 可配置性問題**（configurability gap）

### Step 2：定位問題根源

- 查看 Helm chart 的 template 檔案（通常是 `templates/ray-cluster.yaml`）
- 發現它只顯式 render 特定欄位，例如：
  ```go
  {{- with .Values.worker.template.spec.containers }}
  containers: {{- toYaml . | nindent 8 }}
  {{- end }}
  ```
- 但 **沒有處理 `runtimeClassName`**

### Step 3：設計通用解法

在此案例中，**明確新增 `runtimeClassName` 支援**，因為：

- 它是 GPU 場景的關鍵欄位，需求明確且常見。

---

## 三、PR #4184 是怎麼修正的？

<img width="1262" height="767" alt="PR #4184" src="https://github.com/user-attachments/assets/d58afda2-4794-4576-a833-bd5fef436566" />

### 改動內容

#### 1. **更新 Helm values.yaml**

新增 `runtimeClassName` 欄位到 head 和 worker 的 template spec：

```yaml
head:
  template:
    spec:
      runtimeClassName: "" # <-- 新增
worker:
  template:
    spec:
      runtimeClassName: "" # <-- 新增
```

#### 2. **修改 Helm template（ray-cluster.yaml）**

在 render PodTemplateSpec 時，加入對 `runtimeClassName` 的條件判斷：

```yaml
{{- if .Values.head.template.spec.runtimeClassName }}
runtimeClassName: {{ .Values.head.template.spec.runtimeClassName }}
{{- end }}
```

同樣為 worker group 加上類似邏輯

#### 3. **更新文件與測試**

- 在 `README.md` 中說明此新參數
- 補充 e2e 測試

## 細節說明

### 1. 為何是改動 `helm-chart/ray-cluster/templates/raycluster-cluster.yaml`？

它是用 Helm 產生 RayCluster 自定義資源（Custom Resource, CR）實例的模板，它不是 CRD 本身的定義（CRD 通常在 crds/ 目錄下，例如 ray.io_rayclusters.yaml）

### 2. 為什麼需要 values.yaml？template 已經可以接收參數了啊？

`values.yaml` 的功能為提供所有可調參數的結構與預設值，使用者可覆蓋，`templates/*.yaml` 是根據 .Values 動態渲染最終的 Kubernetes YAML

Helm 需要 values.yaml 來提供 helm show values 命令的輸出，在 CI/CD 中做 schema validation（如果搭配 values.schema.json），讓使用者只需寫「差異部分」（例如只改 runtimeClassName，其他用預設）

### 3. 在 issue 中有提到一個問題，`「我打算修改 kuberay repo 裡的 Helm chart（即 helm-chart/ray-cluster/...），但我注意到還有一個獨立的 repo 叫 kuberay-helm，所以我不確定該改哪邊？」`

<img width="863" height="214" alt="截圖 2026-01-14 上午11 53 56" src="https://github.com/user-attachments/assets/a8b5dc76-17c6-45ef-9336-ea80c675c358" />

KubeRay 專案原本把 Helm chart 放在主 repo 的 helm-chart/ 目錄下，後來為了方便 Helm Hub 發佈、版本管理，也同步維護了一個獨立的 Helm repo：ray-project/kuberay-helm，這兩個 repo 的內容應該保持同步，通常透過自動化腳本（如 scripts/sync-helm-chart.sh）從主 repo 同步到 kuberay-helm

### 4. 為什麼測試中的 path 寫法不一樣？

這是因為 RayCluster CR 的結構本身就不對稱：

```yaml
spec:
  headGroupSpec: # <- 這是單一物件（object）
    template:
      spec:
        runtimeClassName: ...

  workerGroupSpecs: # <- 這是陣列（array）
    - groupName: "workergroup"
      template:
        spec:
          runtimeClassName: ...
    - groupName: "smallGroup"
      template:
        spec:
          runtimeClassName: ...
```

- Head 只有一個，直接用路徑 `spec.headGroupSpec.template.spec.runtimeClassName`
- Worker 是一個 list，必須用 JSONPath 過濾語法 找出特定 group：

```jsonpath
spec.workerGroupSpecs[?(@.groupName=="workergroup")].template.spec.runtimeClassName
```

這表示：「在 workerGroupSpecs 陣列中，找 groupName 等於 "workergroup" 的那一項」

#### 那為什麼不統一路徑？

- 因為 Kubernetes CRD 設計是 Head 是單例，Worker 是多例（支援多種 worker group）所以在測試工具（如 helm-unittest）使用 JSONPath 查詢，必須符合實際結構
