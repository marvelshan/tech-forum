## K8S Lab Day_33

# Pod 生命週期

<img width="1229" height="1051" alt="image" src="https://github.com/user-attachments/assets/f72dbd23-0cf5-4572-ab9b-8d2146420507" />

pod 的生命週期是從他被排程到節點上開始直到被終止為止的過程，這個過程是被嚴格按照順序進行，必且確保都是處於正確的狀態

## 1. Init Container

當你 kubectl apply 或 Kubelet 建立 Pod，Pod 會先進入 Pending / ContainerCreating 階段，這個階段主要是在啟動之前所有的前置工作，主要是要設定配置的檔案，檢查有沒有必要的服務，並執行資料庫的遷移，或是 istio sidecar 等等需要的 injection

## 2. Pod Hook

當 init 完成後，main container 會開始啟動，這是會觸發 pod hook，在主容器的 Entry Point 執行後立即觸發，但不需要等待主容器內的應用程式啟動完畢

### 3. Probes

在 main container 進入運行狀態後，會透過兩種 probes 來監控他們的健康狀態，第一是 readiness probe 用來判斷應用程式是否已經準備好接受流量，再來是 liveness peobe 用來判斷 container 是否仍存活著，是否需要 restart

以下它的主要職責是根據節點上容器運行時的內部狀態，計算並產生一個準確的、準備發送給 k8s API Server 的 v1.PodStatus，這個 v1.PodStatus 透過 Kubelet 的 Status Manager 發送到 API Server，作為 Pod 狀態更新（Status Subresource Update）的數據回報給 contol plane，可以讓 schedule 知道 Pod 的實際生命週期階段（如 Pending 或 Failed），Controllers 決定是否需要創建新的 Pod 或刪除舊的 Pod，通常是根據 Pod 的 READY 條件，Service & Endpoints Controller 根據 Ready 條件（由 Kubelet 根據 Readiness Probe 計算）決定是否將 Pod 的 IP 加入到 Service 的 Endpoints 列表中

```go
// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/kubelet_pods.go

// generateAPIPodStatus creates the final API pod status for a pod, given the
// internal pod status. This method should only be called from within sync*Pod methods.
func (kl *Kubelet) generateAPIPodStatus(pod *v1.Pod, podStatus *kubecontainer.PodStatus, podIsTerminal bool) v1.PodStatus {
	// 記錄日誌，表示開始生成 Pod 狀態。
	klog.V(3).InfoS("Generating pod status", "podIsTerminal", podIsTerminal, "pod", klog.KObj(pod))

	// --- 1. 獲取舊狀態作為基準 (Get Old Status as Baseline) ---
	// use the previous pod status, or the api status, as the basis for this pod
	oldPodStatus, found := kl.statusManager.GetPodStatus(pod.UID)
	if !found {
		// 如果本地狀態管理器沒有記錄，則使用 API Server 傳來的狀態作為舊狀態。
		oldPodStatus = pod.Status
	}
	// 將容器運行時回報的內部狀態 (podStatus) 轉換成 API 格式 (v1.PodStatus)，並與舊狀態合併。
	s := kl.convertStatusToAPIStatus(pod, podStatus, oldPodStatus)

	// --- 2. 計算新的 Pod 階段 (Calculate New Pod Phase) ---
	// calculate the next phase and preserve reason
	// 合併所有容器（主容器和初始化容器）的狀態列表。
	allStatus := append(append([]v1.ContainerStatus{}, s.ContainerStatuses...), s.InitContainerStatuses...)
	// 根據所有容器狀態、Pod 是否應終止、以及是否有任何主容器啟動，來計算 Pod 的新階段 (Phase)。
	s.Phase = getPhase(pod, allStatus, podIsTerminal, kubecontainer.HasAnyActiveRegularContainerStarted(&pod.Spec, podStatus))
	klog.V(4).InfoS("Got phase for pod", "pod", klog.KObj(pod), "oldPhase", oldPodStatus.Phase, "phase", s.Phase)

	// --- 3. 終端階段的三向合併與保護 (Terminal Phase Merge & Protection) ---
	// Perform a three-way merge between the statuses from the status manager,
	// runtime, and generated status to ensure terminal status is correctly set.
	// 如果新計算的階段不是終端狀態 (Failed/Succeeded)。
	if s.Phase != v1.PodFailed && s.Phase != v1.PodSucceeded {
		switch {
		// 如果 Kubelet 本地記錄的舊狀態是終端狀態，則強制使用該終端狀態 (保護終端狀態不被覆蓋)。
		case oldPodStatus.Phase == v1.PodFailed || oldPodStatus.Phase == v1.PodSucceeded:
			klog.V(4).InfoS("Status manager phase was terminal, updating phase to match", "pod", klog.KObj(pod), "phase", oldPodStatus.Phase)
			s.Phase = oldPodStatus.Phase
		// 如果 API Server 記錄的狀態是終端狀態，則強制使用該終端狀態 (保護終端狀態不被覆蓋)。
		case pod.Status.Phase == v1.PodFailed || pod.Status.Phase == v1.PodSucceeded:
			klog.V(4).InfoS("API phase was terminal, updating phase to match", "pod", klog.KObj(pod), "phase", pod.Status.Phase)
			s.Phase = pod.Status.Phase
		}
	}

	// --- 4. 保留階段原因和訊息 (Preserve Reason and Message) ---
	if s.Phase == oldPodStatus.Phase {
		// preserve the reason and message which is associated with the phase
		// 如果新舊階段相同，則保留舊階段的 Reason 和 Message，避免覆蓋重要的錯誤資訊。
		s.Reason = oldPodStatus.Reason
		s.Message = oldPodStatus.Message
		// 如果保留的欄位為空，則嘗試從 API Server 的原始狀態中獲取。
		if len(s.Reason) == 0 {
			s.Reason = pod.Status.Reason
		}
		if len(s.Message) == 0 {
			s.Message = pod.Status.Message
		}
	}

	// --- 5. 檢查內部驅逐請求 (Internal Eviction Check) ---
	// check if an internal module has requested the pod is evicted and override the reason and message
	// 迭代所有 Pod 同步處理器（例如，處理 Node 資源壓力驅逐的模組）。
	for _, podSyncHandler := range kl.PodSyncHandlers {
		if result := podSyncHandler.ShouldEvict(pod); result.Evict {
			// 如果模組請求驅逐，則覆蓋狀態，強制設為 Failed 並寫入驅逐原因。
			s.Phase = v1.PodFailed
			s.Reason = result.Reason
			s.Message = result.Message
			break
		}
	}

	// --- 6. 再次執行終端狀態的不可逆保護 (Final Terminal State Lock) ---
	// pods are not allowed to transition out of terminal phases
	if pod.Status.Phase == v1.PodFailed || pod.Status.Phase == v1.PodSucceeded {
		// 如果 API Server 記錄的狀態是終端狀態...
		// API server shows terminal phase; transitions are not allowed
		if s.Phase != pod.Status.Phase {
			// 並且新計算的狀態試圖變為非終端狀態（非法轉換）
			klog.ErrorS(nil, "Pod attempted illegal phase transition", "pod", klog.KObj(pod), "originalStatusPhase", pod.Status.Phase, "apiStatusPhase", s.Phase, "apiStatus", s)
			// 強制將狀態修正回 API Server 記錄的終端狀態。
			// Force back to phase from the API server
			s.Phase = pod.Status.Phase
		}
	}

	// --- 7. 更新 Kubelet 內部狀態管理員 (Update Internal State Managers) ---
	// ensure the probe managers have up to date status for containers
	// 更新 Liveness 和 Readiness 探針管理器，確保它們使用最新的容器狀態來進行健康檢查。
	kl.probeManager.UpdatePodStatus(context.TODO(), pod, s)

	// update the allocated resources status
	// 如果啟用 Feature Gate，更新資源管理器中 Pod 的資源分配狀態。
	if utilfeature.DefaultFeatureGate.Enabled(features.ResourceHealthStatus) {
		kl.containerManager.UpdateAllocatedResourcesStatus(pod, s)
	}

	// --- 8. 處理 Pod 條件 (Pod Conditions) ---
	// preserve all conditions not owned by the kubelet
	// 複製 API Server 狀態中所有非 Kubelet 所擁有的條件 (Condition)，避免覆蓋。
	s.Conditions = make([]v1.PodCondition, 0, len(pod.Status.Conditions)+1)
	for _, c := range pod.Status.Conditions {
		if !kubetypes.PodConditionByKubelet(c.Type) {
			s.Conditions = append(s.Conditions, c)
		}
	}
	// 處理 Pod 垂直擴容（如果啟用 Feature Gate）。
	if utilfeature.DefaultFeatureGate.Enabled(features.InPlacePodVerticalScaling) {
		resizeStatus := kl.determinePodResizeStatus(pod, podIsTerminal)
		for _, c := range resizeStatus {
			// 處理調整大小 (Resize) 相關的條件。
			gen := podutil.CalculatePodConditionObservedGeneration(&oldPodStatus, pod.Generation, c.Type)
			if gen == 0 {
				c.ObservedGeneration = 0
			}
			s.Conditions = append(s.Conditions, *c)
		}
	}

	// copy over the pod disruption conditions from state which is already
	// updated during the eviciton (due to either node resource pressure or
	// node graceful shutdown).
	// 複製 Pod 中斷條件 (如由於節點資源壓力或優雅關機而導致的驅逐)。
	cType := v1.DisruptionTarget
	if _, condition := podutil.GetPodConditionFromList(oldPodStatus.Conditions, cType); condition != nil {
		s.Conditions = utilpod.ReplaceOrAppendPodCondition(s.Conditions, condition)
	}

	// set all Kubelet-owned conditions
	// 設置所有 Kubelet 負責維護的標準 Pod 條件 (Pod Conditions)。
	if utilfeature.DefaultFeatureGate.Enabled(features.PodReadyToStartContainersCondition) {
		s.Conditions = append(s.Conditions, status.GeneratePodReadyToStartContainersCondition(pod, &oldPodStatus, podStatus))
	}
	allContainerStatuses := append(s.InitContainerStatuses, s.ContainerStatuses...)
	// 設置 PodInitialized (初始化完成) 條件。
	s.Conditions = append(s.Conditions, status.GeneratePodInitializedCondition(pod, &oldPodStatus, allContainerStatuses, s.Phase))
	// 設置 PodReady (準備就緒) 條件。
	s.Conditions = append(s.Conditions, status.GeneratePodReadyCondition(pod, &oldPodStatus, s.Conditions, allContainerStatuses, s.Phase))
	// 設置 ContainersReady (容器準備就緒) 條件。
	s.Conditions = append(s.Conditions, status.GenerateContainersReadyCondition(pod, &oldPodStatus, allContainerStatuses, s.Phase))
	// 設置 PodScheduled (已排程) 條件。
	s.Conditions = append(s.Conditions, v1.PodCondition{
		Type: v1.PodScheduled,
		ObservedGeneration: podutil.CalculatePodConditionObservedGeneration(&oldPodStatus, pod.Generation, v1.PodScheduled),
		Status: v1.ConditionTrue,
	})

	// --- 9. 設定 IP 地址 (Set IP Addresses) ---
	// set HostIP/HostIPs and initialize PodIP/PodIPs for host network pods
	if kl.kubeClient != nil {
		// 獲取主機（節點）的 IP 地址。
		hostIPs, err := kl.getHostIPsAnyWay()
		// 處理 IP 獲取錯誤並記錄日誌。
		// 設定 HostIP 和 HostIPs。
		// 如果 Pod 使用 HostNetwork 模式，則將節點 IP 設定為 Pod 的 PodIP，處理單棧和雙棧情況。
		// ... 處理 HostIP 和 HostIPs 設置邏輯 ...
	}

	// --- 10. 返回最終狀態 (Return Final Status) ---
	return *s // 返回最終計算並建構完成的 v1.PodStatus 物件。
}
```

# Istio 與 Gateway API Inference Extension：為 Kubernetes 上的 AI 推理打造智慧流量路由

## 前言

在 k8s 運行 ai workload 一直都是一個很大的挑戰，在我昨天所提到的 ai 會需要判斷他的資源使用率才能達到最好的成本考量，也有提到因為他的推理請求每個都相當的不同，像是他的記憶體消耗量大，有時候會需要使用到動態載入 Lora Low-Rank Adapter，在平常的服務，我們常常使用 ingres、gateway api、service mesh 等等的 L7 的流量控制，但這些往往對 ai 不理想，所以 Gateway API Inference Extension 是要來解決這個問題，讓 gateway 能理解 ai 的推理，然後比較聰明的方式去使用 GPU 的資源

## Gateway API Inference Extension 的設計

<img width="1298" height="602" alt="截圖 2025-10-15 下午3 10 50" src="https://github.com/user-attachments/assets/2c5681c7-1e34-43e0-9f5d-27af8d261ec0" />

### 1. InferenceModel

這裏主要是由 ai 工程師來決定，這是邏輯模型的入口，支持多個模型來去做切分

```yaml
apiVersion: inference.networking.x-k8s.io/v1alpha2
kind: InferenceModel
metadata:
  name: inferencemodel-llama2
spec:
  modelName: llama2
  criticality: Critical
  poolRef:
    name: vllm-llama2-7b-pool
  targetModels:
    - name: vllm-llama2-7b-2024-11-20
      weight: 75
    - name: vllm-llama2-7b-2025-03-24
      weight: 25
```

### 2. InferencePool

這裏主要是由維運人員來去做配置，HTTPRoute 可以將流量導向一個智能的推理池，也就是這裏所説到的 `InferencePool`

```yaml
apiVersion: inference.networking.x-k8s.io/v1alpha2
kind: InferencePool
metadata:
  name: vllm-llama2-7b-pool
spec:
  targetPortNumber: 8000
  selector:
    app: vllm-llama2-7b
  extensionRef:
    name: vllm-llama2-7b-endpoint-picker
```

### 3. HTTPRoute

這裏就從剛剛的推理入口送到了 InferencePool 的 HTTPRoute

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: llm-route
spec:
  parentRefs:
    - name: inference-gateway
  rules:
    - backendRefs:
        - group: inference.networking.x-k8s.io
          kind: InferencePool
          name: vllm-llama2-7b
      matches:
        - path:
            type: PathPrefix
            value: /
```

## Reference

https://cloud.tencent.com/developer/article/2522826

https://istio.io/latest/zh/blog/2025/inference-extension-support/

https://www.solo.io/blog/llm-d-distributed-inference-serving-on-kubernetes

https://cloudnativecn.com/blog/gateway-api-inference-extension-deep-dive/
