# Kubernetes 上運行 Ray 叢集的強大工具，KubeRay

在現代的分散式計算和 AI/ML 領域，Ray 作為一個高效的開源框架，已被廣泛用於大規模資料處理、機器學習訓練和服務部署。然而，將 Ray 部署到 K8s 上時，常常面臨管理複雜度高的挑戰<
這時，KubeRay 就成為一個關鍵的解決方案

## 什麼是 KubeRay？

Ray 本身就是一個很強的分散式框架，能處理任務調度、actor 模型以及資料並行計算，不過如果直接把 Ray 跑在 Kubernetes 上，實務上會遇到不少麻煩，像是要自己管理 Pod 的建立與重啟、資源怎麼分配、節點怎麼擴縮，還有怎麼跟 Kubernetes 既有的機制（例如 autoscaling）整合，這些都需要額外處理，KubeRay 就是在這樣的背景下出現，目標是把這些運維層面的複雜度交給 Operator 處理，讓使用者可以更專注在 Ray 的應用本身

透過 KubeRay，使用者只需要定義像 RayCluster 這類的 CRD，Operator 就會自動建立與管理底層的 Pod、Service 以及相關資源，不需要再自己寫一大堆 Deployment 或 StatefulSet 的 YAML，cluster 的生命週期也能自動化處理，包括依照負載動態調整 worker 數量、在節點或程序異常時進行故障恢復，甚至支援零停機升級，這些對正式環境來說都是非常重要的能力，可以大幅提升整體可用性與穩定度

在 workload 上，KubeRay 也做了區分，像是一次性的批次任務，例如模型訓練或大規模資料處理，可以用 RayJob 來跑，任務結束後資源會自動回收，對雲端成本控制很有幫助；如果是長時間對外提供服務的情境，例如用 Ray Serve 部署模型 API，則可以用 RayService 來確保服務的高可用性，並支援像藍綠部署這類不中斷服務的更新方式，這讓 Ray 不只是拿來跑實驗或訓練，也能更自然地進入正式線上服務的架構

另一方面，KubeRay 也很好地融入 Kubernetes 的整個生態系，可以使用原本熟悉的 Pod template 來客製資源設定，搭配 PVC 管理儲存空間，也能和像 Istio 這類服務網格一起使用，在網路流量控管與觀測性上都有彈性，對企業環境來說，還可以結合 Prometheus 等監控工具，來管理多租戶的 Ray 叢集與資源使用情況

整體來看，KubeRay 特別適合用在 AI／ML 與大數據相關場景，不論是大規模離線運算，還是需要隨流量擴縮的模型推論服務，都能在 Kubernetes 上用比較標準化、雲原生的方式來運作。同時它也降低了 Ray 開發者接觸 Kubernetes 的門檻，透過 kubectl 外掛或 Dashboard 就能管理叢集狀態，開發時可以更專注在任務邏輯與模型本身，而不是花大量時間處理基礎設施。

## 透過 issue 來學習 KubeRay 底層機制

在 PR #1386（標題為 "Check Ray container status for deletion in shouldDeletePod"）是一個針對邊緣案例的改進。它關閉了 Issue #1355，並衍生出後續的 #1392（參數化測試）和 #1393（文件化重啟邏輯）。
這個 PR 的核心是修改 KubeRay Operator 的 reconcile 邏輯，特別是在決定是否刪除 Ray Pod 時，從只檢查 Pod 整體狀態轉為額外檢查主 Ray 容器的狀態。這解決了在多容器 Pod（尤其是帶 sidecar）下的故障恢復問題。

### 為何會有這個 Issue？

這個 Issue 的原因是來自 Kubernetes 的 Pod 生命週期設計，和 Ray 在 KubeRay 中的部署與重啟策略之間產生了落差，特別是在 Pod 內包含 sidecar 容器的情況下，問題會被放大，在 KubeRay 的設計中，不論是 head 還是 worker Pod，通常都會把 restartPolicy 設為 Never 或 OnFailure，這代表當容器失敗時，K8s 不會在原地重啟容器，而是交由 KubeRay Operator 來刪除整個 Pod，再由上層控制器建立一個新的 Pod，這樣的設計是為了避免 Ray cluster 內部狀態不一致，尤其在啟用 GCS 容錯或需要重新加入叢集拓撲時，由 Operator 統一處理會比較安全，也能避免殘留資源造成浪費

在先前的 PR #1341 中，KubeRay 加入了一段邏輯，只要 Pod 進入 terminated 狀態，也就是 Failed 或 Succeeded，而且 restartPolicy 不是 Always，就會由 Operator 主動刪除該 Pod，觸發重新建立。不過這個判斷邏輯只依賴 Pod 的整體 Phase，例如 Pending、Running 或 Failed，而沒有進一步檢查各個容器的實際狀態，問題就出在 Kubernetes 對 Pod Phase 的定義是聚合式的。只要 Pod 裡面還有任何一個容器正在運行或啟動中，整個 Pod 的 Phase 就會被標記為 Running。這在單一容器的 Pod 裡通常不會有問題，但在實務上，Ray 的 Pod 幾乎都是多容器架構，除了主要負責執行 Ray 程序的主容器之外，常常還會搭配 sidecar 容器來處理日誌收集、監控指標匯出、自動擴縮邏輯，或是服務網格相關的網路代理，例如 Envoy。

> Pod 的 Phase 並不是針對 Pod 內每個容器各自計算後再精確彙總的狀態，而是一個把所有容器狀況整合起來的高階摘要，用來表示整個 Pod 在生命週期中的大致位置。它的設計目的不是提供細節層級的健康判斷，而是讓系統與使用者能快速理解這個 Pod 目前是在啟動中、正在運行，還是已經結束。
>
> Kubernetes 官方文件:
> "The phase of a Pod is a simple, high-level summary of where the Pod is in its lifecycle. The phase is not intended to be a comprehensive rollup of observations of container or Pod state..."
>
> 因此在實際行為上，只要 Pod 裡還有任何一個容器處於運行或啟動中的狀態，整個 Pod 的 Phase 就可能仍然是 Running，即使其他關鍵容器早已停止或失效。這也是為什麼 Phase 比較適合拿來判斷「這個 Pod 大概還活著還是已經結束」，但並不適合用來判斷「這個 Pod 的主要功能是否仍然正常」

在這種情境下，如果 Ray 的主容器因為程序崩潰或被手動終止而停止運作，因為 restartPolicy 設為 Never，它不會被 Kubernetes 自動重啟；但只要 sidecar 容器仍在正常運行，整個 Pod 的 Phase 就會繼續維持在 Running。對 KubeRay 來說，這個 Pod 看起來仍然是健康狀態，因此不會觸發刪除與重建流程，導致實際上已經失效的 Ray 節點被卡在 cluster 中，cluster 也就無法自動恢復，這種情況在生產環境其實非常容易發生，因為 sidecar 幾乎是標配，用來補足可觀測性、資安與平台整合需求。Ray 官方文件本身也提供 sidecar 的範例，用於日誌持久化或其他輔助功能。因此這個 Issue 本質上不是少見的邊角案例，而是只要進入真實部署場景就很可能遇到的問題

## 解決方案細節

### （1）主要流程更新

```go
r.Log.Info("reconcilePods", "Found 1 head Pod", headPod.Name, "Pod status", headPod.Status.Phase,
	"Pod restart policy", headPod.Spec.RestartPolicy,
	"Ray container terminated status", getRayContainerStateTerminated(headPod))

shouldDelete, reason := shouldDeletePod(headPod, rayv1alpha1.HeadNode)
r.Log.Info("reconcilePods", "head Pod", headPod.Name, "shouldDelete", shouldDelete, "reason", reason)
if shouldDelete {
	// 刪除 Pod 的邏輯（未顯示）
}
```

由於 Kubernetes 的 Pod.Status.Phase 是對所有容器狀態的粗粒度聚合，這裏新增了一個輔助函數 getRayContainerStateTerminated()，專門用來，根據預期的容器索引（common.RayContainerIndex）取得 Ray 主容器的名稱；遍歷 pod.Status.ContainerStatuses，透過 容器名稱精確匹配（而非依賴列表順序）；並且回傳該容器的 Terminated 狀態（\*corev1.ContainerStateTerminated）。

#### 那這裏為何必須用名稱匹配？

Kubernetes 官方明確指出，spec.containers 與 status.containerStatuses 不保證順序一致，因此，直接用索引（如 [0]）取狀態是不安全的，若在測試環境（如 envtest）中因缺乏 Kubelet 而無法取得 ContainerStatuses，則安全地回傳 nil，代表「無法確認終止狀態」，避免誤刪

### （2）`shouldDeletePod()`

#### 情境 1：Pod 處於終止狀態（`Failed` / `Succeeded`）

```go
if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
	if isRestartPolicyAlways {
		// 即使狀態是 Failed，但 restartPolicy=Always → 不刪（依賴 K8s 自動重啟）
		return false, "..."
	}
	return true, "..." // 否則可以安全刪除
}
```

#### 情境 2：Pod Phase 是 `Running`，但 **Ray 容器已 Terminated**

```go
rayContainerTerminated := getRayContainerStateTerminated(pod)
if pod.Status.Phase == corev1.PodRunning && rayContainerTerminated != nil {
	if isRestartPolicyAlways {
		return false, "..." // 會自動重啟，不刪
	}
	return true, "..." // 容器掛了又不能重啟 → 刪掉
}
```

- 以前只看 `Phase == Running` 就認為是一切正常 -> **會漏掉容器崩潰的情況**
- 現在會檢查 **Ray 主容器是否已經 Terminated**（比如 `ExitCode != 0`）。
- 即使 Pod Phase 還是 `Running`，只要主容器掛了，就需要處理！

> 舉例：假如運行 `ray start --head`，但程式 crash 了（exit code 1）。Kubernetes 會嘗試重啟（如果 restartPolicy=Always），但如果重啟失敗多次，容器會處於 Terminated 狀態，而 Pod Phase 可能仍是 `Running`（因為還有 init container 或 sidecar 活著）。

#### 情境 3：其他情況 → 不刪除

```go
reason := fmt.Sprintf("KubeRay does not need to delete the %s Pod %s. ...")
return false, reason
```

- 這包括：`Pending`、`Running` 且容器沒掛、或狀態不明。
- 回傳明確原因，方便除錯。

### （3）`getRayContainerStateTerminated()` 用來安全取得容器狀態

```go
func getRayContainerStateTerminated(pod corev1.Pod) *corev1.ContainerStateTerminated {
	rayContainerName := pod.Spec.Containers[common.RayContainerIndex].Name
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == rayContainerName {
			return containerStatus.State.Terminated
		}
	}
	return nil // 找不到
}
```

這裡是在多容器 Pod 的情境下，取得 Ray 主容器是否已經進入 Terminated 狀態，它先從 Pod 的 spec 依照既定的索引抓出 Ray 主容器的名稱，接著遍歷 pod.Status.ContainerStatuses，也就是 Kubernetes 回報的每個容器實際執行狀態，找到名稱相符的那一個容器後，回傳其 State.Terminated 欄位。如果該容器尚未終止，或根本找不到對應的狀態（例如在某些測試或尚未回報狀態的情況），就回傳 nil，讓呼叫端可以明確區分「尚未終止」與「無法取得狀態」，避免直接依賴 Pod Phase 而誤判整個 Pod 的健康狀況

## Reference

https://github.com/ray-project/kuberay/pull/1386

https://github.com/ray-project/kuberay/pull/1386/files
