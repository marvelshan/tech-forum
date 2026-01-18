## K8S Lab Day_48

# Study8-Admission-Control深入探討

## 前言

前幾天講到了 Validating 和 defaulting 的機制，今天要繼續深入來了解一下 admission control

## Admission Control

### 為何需要 admission control 呢？

在創建資源的過程中，defaulting 和 validation 已經有補植和驗證的機制，但是有些規則還是需要動態的和權限控制的機制，像是禁止使用 hostPath 掛載、強制加上 securityContext、注入 Sidecar 等等，這血都是沒辦法寫死在 defaults.go 裡面的，所以需要由 admission controller 來動態處理

```go
// InstallAPIGroup exposes the given api group in the API.
// The <apiGroupInfo> passed into this function shouldn't be used elsewhere as the
// underlying storage will be destroyed on this servers shutdown.
func (s *GenericAPIServer) InstallAPIGroup(apiGroupInfo *APIGroupInfo) error {
	return s.InstallAPIGroups(apiGroupInfo)
}

func (s *GenericAPIServer) getAPIGroupVersion(apiGroupInfo *APIGroupInfo, groupVersion schema.GroupVersion, apiPrefix string) (*genericapi.APIGroupVersion, error) {

	// ...

	version := s.newAPIGroupVersion(apiGroupInfo, groupVersion)
	version.Root = apiPrefix
	version.Storage = storage
	return version, nil
}
func (s *GenericAPIServer) newAPIGroupVersion(apiGroupInfo *APIGroupInfo, groupVersion schema.GroupVersion) *genericapi.APIGroupVersion {

	allServedVersionsByResource := map[string][]string{}
	for version, resourcesInVersion := range apiGroupInfo.VersionedResourcesStorageMap {
		for resource := range resourcesInVersion {
			if len(groupVersion.Group) == 0 {
				allServedVersionsByResource[resource] = append(allServedVersionsByResource[resource], version)
			} else {
				allServedVersionsByResource[resource] = append(allServedVersionsByResource[resource], fmt.Sprintf("%s/%s", groupVersion.Group, version))
			}
		}
	}

	return &genericapi.APIGroupVersion{

		// ...

		Admit:             s.admissionControl,
		MinRequestTimeout: s.minRequestTimeout,
		Authorizer:        s.Authorizer,
	}
}
```

[首先我們要先從 `newAPIGroupVersion` 來開始了解](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/server/genericapiserver.go#L876)，這邊為何要把 `s.admissionControl` 直接三塞進每個 `APIGroupVersion` 裡面呢？這邊可以了解每個 API GroupVersion 在處理 Create / Update / Delete 的請求時，都能經過相同的 admission 流程

[接著可以看到 `func createHandler`](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/create.go#L182)，看到這裏是怎麼處理 `POST /apis/{group}/{version}/{resource}`

```go
func createHandler(r rest.NamedCreater, scope *RequestScope, admit admission.Interface, includeName bool) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// ...

		// 1. 建構 admission attributes
		admissionAttributes := admission.NewAttributesRecord(
			obj, nil, scope.Kind, namespace, name,
			scope.Resource, scope.Subresource, admission.Create, options,
			dryrun.IsDryRun(options.DryRun), userInfo,
		)

		// 2. 內部會呼叫底層 storage
		requestFunc := func() (runtime.Object, error) {
			return r.Create(
				ctx,
				name,
				obj,
				rest.AdmissionToValidateObjectFunc(admit, admissionAttributes, scope),
				options,
			)
		}

		// 3. 透過 finisher.FinishRequest 包裝，確保在 storage 寫入前後都能正確執行 admission
		result, err := finisher.FinishRequest(ctx, func() (runtime.Object, error) {
			// ... 準備 liveObj、field manager 更新等

			// === Mutating Admission ===
			if mutatingAdmission, ok := admit.(admission.MutationInterface); ok && mutatingAdmission.Handles(admission.Create) {
				if err := mutatingAdmission.Admit(ctx, admissionAttributes, scope); err != nil {
					return nil, err
				}
			}

			// Mutating 後再次 dedup owner references
			dedupOwnerReferencesAndAddWarning(obj, req.Context(), true)

			// 真正呼叫 storage.Create
			result, err := requestFunc()

			// ...
		})

		// ...
	}
}
```

在這裡可以看到 `admission.NewAttributesRecord(...)` 建立一個描述這次操作的物件，提供 admission controller 判斷使用，像是 admission 可以知道這是 Create，或是要操作哪些 namespace、user、resource 等等，接著看到 Create 是使用 `rest.ValidateObjectFunc`，這裏會把 admission 轉成 storage 呼叫的 validation function

這裏的在呼叫 requestFunc 前的準備與 mutating admission，這裏可以看到 `if mutatingAdmission, ok := admit.(admission.MutationInterface); ok && mutatingAdmission.Handles(admission.Create)` 判斷 admission 是否實作了 MutationInterface，這裏也判斷是否處理 Create，如果是就呼叫 `Admit(...)` 讓它做 mutation

最後會呼叫 `requestFunc()` 這裏 `storage.Create` 會在 commit 前呼叫 `createValidatingAdmission` 也就是 `rest.AdmissionToValidateObjectFunc`，所以這裏的 validation admission 才會被執行做驗證

## 結語

今天的時間沒有很多，所以簡單了看了一下 Create 的時候是怎麼去呼叫 mutating admission 和 validation admission 的
