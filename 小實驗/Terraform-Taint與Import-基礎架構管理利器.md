## K8S Lab Day_35

# Terraform Taint 與 Import：基礎架構管理利器

## 前言

今天很突然的講到了 terraform，也是因為過去有稍微玩過，也被搞過...，所以今天就來了解一下他吧。

在現代雲端運維中，將基礎架構管理成程式碼（Infrastructure as Code, IaC）已經成為最佳實務。Terraform 是這個領域中廣受歡迎的工具，它提供了強大的自動化、版本控制與一致性保障。但在實務操作中，經常會遇到資源失效、漂移或需要整合現有資源的情況。本文將深入探討 Terraform 的兩個重要指令：taint 與 import，以及它們在基礎架構管理中的應用與最佳實務。

### Terraform Taint：強制重新建立資源

#### 什麼是 Terraform Taint？

terraform taint 是 Terraform 的工作流程指令，用於標記需要重建的資源。當某個資源被染色後，下一次執行 terraform apply 時，Terraform 會先銷毀該資源，然後依照配置重新建立，確保資源從頭開始生成，回到乾淨狀態。

#### 為什麼要使用 Terraform Taint？

Terraform Taint 在以下情況特別有價值，外部修改或手動變更可能使資源與 Terraform 配置不一致。Taint 可強制重建資源，使其回到宣告式配置狀態、若資源因軟體錯誤或其他事件失效，taint 提供快速替換的方法、某些配置變更無法就地更新，taint 可安排資源重新建立

#### Terraform Taint 操作步驟

首先，使用 terraform state list 檢視 Terraform 狀態中的資源，確認要染色的目標

```bash
terraform taint <resource_type>.<resource_name>
```

再來就是執行 terraform apply，Terraform 會銷毀染色資源並重新建立

- 優勢：

  - 快速隔離與修復故障資源

  - 強制資源與 Terraform 配置一致

  - 提供基礎架構變更彈性與自動化能力

- 限制：

  - 可能導致停機或服務中斷

  - 忽略資源依賴性可能引發級聯故障

  - 資源刪除過程不可逆，可能造成資料遺失

### Terraform Import：將現有資源納入管理

#### 什麼是 Terraform Import？

terraform import 允許將在 Terraform 之外建立的資源納入管理。透過這個指令，Terraform 可以追蹤現有資源的狀態，並與配置檔案中的資源建立對應，實現一致性管理。

#### 為何要使用 Import？

1. 無縫整合現有環境：不需重建資源，逐步接管既有基礎架構。

2. 統一管理：將所有資源納入 Terraform，簡化管理流程。

3. 確保一致性：避免手動或外部配置造成的偏差，維持宣告式管理。

#### 使用 Terraform Import 範例

```bash
mkdir terraform-import-tutorial
cd terraform-import-tutorial
touch main.tf
```

```hcl
provider "aws" {
  region = "us-west-2"
}

resource "aws_instance" "example" {
  # 匯入後填寫配置
}
```

```bash
terraform init
```

```bash
terraform import aws_instance.example i-1234567890abcdef0
```

```bash
terraform show
```

```hcl
resource "aws_instance" "example" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.micro"
  key_name      = "my-key-pair"
}
```

```bash
terraform plan
terraform apply
```

## 結論

Terraform 的 taint 與 import 指令提供了靈活且強大的基礎架構管理能力。taint 適用於修復故障或強制一致性，而 import 則是整合現有環境的利器。透過這兩個工具，運維工程師可以有效控制資源生命周期，提升自動化管理與運維效率。

## 那 istio 呢 XD

在 k8s 中部署 Istio 通常需要配置命名空間、證書以及相關資源。透過 Terraform，我們可以將這些操作自動化，將 Istio 的部署流程納入 IaC 管理。以下將介紹一個使用 Terraform 部署 Istio 的範例流程

### 1. 建立 Istio 命名空間

部署 Istio 前，需要先在 Kubernetes 集群中建立專屬的命名空間，例如 istio-system。Terraform 提供了 kubernetes_namespace 資源來實現這個操作

`local.istio_namespace` 指定 Istio 的命名空間名稱，例如 istio-system。`depends_on` 是確保在 EKS 集群建立完成後再創建命名空間，避免資源依賴問題

```hcl
resource "kubernetes_namespace" "istio_system" {
  metadata {
    name = local.istio_namespace
  }
  depends_on = [module.eks.cluster_name]
}
```

### 2. 配置自簽 Root CA 證書

Istio 的控制平面需要證書來進行 mTLS 以及安全通信。可以使用 cert-manager 生成自簽證書，並透過 Terraform 的 kubectl_manifest 將 YAML 文件套用到集群中，`kubectl_path_documents` 會讀取本地 YAML 文件，例如自簽 Root CA 的配置。`kubectl_manifest` 將 YAML 文件套用到 Kubernetes 集群，Terraform 會自動追蹤資源狀態。`depends_on` 確保 EKS 的相關附加元件（addons）已安裝，避免套用失敗

```hcl
data "kubectl_path_documents" "self_signed_ca" {
  pattern = "${path.module}/cert-manager-manifests/self-signed-ca.yaml"
}

resource "kubectl_manifest" "self_signed_ca" {
  for_each  = toset(data.kubectl_path_documents.self_signed_ca.documents)
  yaml_body = each.value

  depends_on = [module.eks_blueprints_addons]
}
```

### 3. 配置 Istio 證書

接下來，為 Istio 本身生成證書，用於控制平面與工作負載之間的安全通信，與自簽 CA 類似，透過 kubectl_path_documents 與 kubectl_manifest 讀取並套用 Istio 證書的 YAML 文件。確保依賴 EKS 附加元件，避免資源套用順序錯誤

```hcl
data "kubectl_path_documents" "istio_cert" {
  pattern = "${path.module}/cert-manager-manifests/istio-cert.yaml"
}

resource "kubectl_manifest" "istio_cert" {
  for_each  = toset(data.kubectl_path_documents.istio_cert.documents)
  yaml_body = each.value

  depends_on = [module.eks_blueprints_addons]
}
```

#### 4. Terraform 部署 Istio 的優勢

使用 Terraform 部署 Istio 相比手動套用 YAML，有以下優勢：

1. 自動化管理：將命名空間、證書與其他 Kubernetes 資源納入 Terraform 管理，資源變更可追蹤、版本化。

2. 依賴管理：透過 depends_on，Terraform 可確保資源按照正確順序建立，減少部署錯誤。

3. 可重複性：Terraform 配置可在不同環境（開發、測試、正式）重複使用，確保一致性。

4. 狀態追蹤：Terraform 會追蹤資源狀態，未來可結合 taint 或 import 來處理資源漂移或整合既有集群。

以上的文件是利用 SPIFFE（Secure Production Identity Framework For Everyone）/SPIRE（SPIFFE Runtime Environment） 與 Istio，在多個 EKS Kubernetes 集群間建立信任橋樑。透過這種方式，不同集群中的微服務能夠相互信任並安全地通信，確保不同組件間的安全通信非常重要，但各系統獨立的身分認證模型往往不兼容，增加了安全管理與應用修改的複雜度，此範例透過 Istio mesh federation 與 SPIFFE/SPIRE 身分驗證系統，解決跨集群信任與安全身分統一的問題。示範中有兩個獨立 EKS 集群（foo-eks-cluster 與 bar-eks-cluster），各自有由 cert-manager 生成的根 CA。SPIRE 在每個集群中充當中間 CA，並透過交換信任 bundle 的方式啟用聯邦(federation)，使不同根 CA 的 workload 可以安全通信

## 小結

透過 Terraform 結合 `kubectl_manifest`，可以將 Istio 部署流程完全程式化，從命名空間建立到證書配置，都能納入 IaC 管理。這不僅提升自動化效率，也方便日後的資源更新與版本控制。所以看到以上，其實 istio 不一定就是一定會使用到他的 sidecar，這裡的目的是利用 Istio + Spiffe/Spire 建立跨 cluster 的信任域，所以即便沒有手動管理 sidecar，Istio 的 CA 和 Spiffe 會自動為 workloads 發放 mTLS 證書，Istio CA 不只是為了 sidecar，而是確保服務間的身份可信任和通訊加密

## Reference

https://github.com/aws-samples/istio-on-eks/blob/main/patterns/eks-istio-mesh-spire-federation/terraform/1.foo-eks/main.tf

https://www.purestorage.com/tw/knowledge/what-is-terraform-import.html

https://www.purestorage.com/tw/knowledge/what-is-terraform-taint.html
