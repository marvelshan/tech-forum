## K8S Lab Day_43

# Study5-Kubernetes API 的反序列化之旅，apimachinery 核心揭秘

## 前言

在昨天我們學到了 controller 他是怎麼處理資源更新的方法，今天要繼續來學習 k8s 的 api 是怎麼實作的，首先要先來了解一下 k8s 的 api，k8s 的所有資源（Pod、Deployment、Service 等）都是透過 API 來定義與管理的，這些物件最終會被序列化為 JSON 或 YAML，透過 API Server 與 cluster 互動，在 Go 中這些 API 物件被定義為 struct，並使用 tags 來控制序列化行為

而 k8s 的 api 包括了以下幾個部分，Metadata 包含名稱、命名空間、標籤、註解等、Spec 描述資源的期望狀態，例如 Pod 的容器清單、Status 描述資源的實際狀態，由控制器更新、Kind 和 APIVersion 等等，以下可以看看 [source code 這邊 pod 的結構定義](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/api/core/v1/types.go#L5369)

```go
// staging/src/k8s.io/api/core/v1/types.go
type Pod struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the pod.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec PodSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Most recently observed status of the pod.
	// This data may not be up to date.
	// Populated by the system.
	// Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Status PodStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}
...
type PodAffinity struct {
    ...
}
type PodAntiAffinity struct {
    ...
}
```

以上這邊就可以看到 `metav1.TypeMeta` (Kind/APIVersion) 提供了 API 的元資訊，定義了資源的類型和 API 版本，`metav1.ObjectMeta` 包含標準物件的識別資訊，像是 name、namespace、labels 和 annotations 等，用於物件管理與查找、`Spec PodSpec` 描述了資源的期望狀態，這是使用者定義的部分，用來定義 Pod 應該如何運行，例如其中包含的容器清單、網路配置、Volume 掛載等資訊、`Status PodStatus` 描述了資源的實際狀態，這部分由 k8s 的 Controller 負責更新，指示 Pod 當前的運行狀態，像是生命週期階段 Running, Pending, Failed 等等

## 小實驗

### 用 Golang 定義一個簡單的 Pod 結構，並序列化為 YAML

```bash
go mod init pod-yaml
```

```go
package main

import (
    "fmt"
    "log"

    "gopkg.in/yaml.v2"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// 簡化的 Container 結構
type Container struct {
    Name  string `yaml:"name"`
    Image string `yaml:"image"`
}

// 簡化的 PodSpec 結構
type PodSpec struct {
    Containers []Container `yaml:"containers"`
}

// 簡化的 Pod 結構
type Pod struct {
    APIVersion string `yaml:"apiVersion"`
    Kind       string `yaml:"kind"`
    Metadata   metav1.ObjectMeta `yaml:"metadata"`
    Spec       PodSpec           `yaml:"spec"`
}

func main() {
    // 創建一個 Pod 物件
    pod := Pod{
        APIVersion: "v1",
        Kind:       "Pod",
        Metadata: metav1.ObjectMeta{
            Name:      "my-pod",
            Namespace: "default",
            Labels: map[string]string{
                "app": "my-app",
            },
        },
        Spec: PodSpec{
            Containers: []Container{
                {
                    Name:  "nginx",
                    Image: "nginx:1.19",
                },
            },
        },
    }

    // 序列化為 YAML
    yamlData, err := yaml.Marshal(pod)
    if err != nil {
        log.Fatalf("Error marshaling to YAML: %v", err)
    }

    // 輸出 YAML
    fmt.Println("---")
    fmt.Println(string(yamlData))
}
```

```bash
go mod tidy
```

去執行 `go run pod_yaml.go` 就可以直接使用到 `metav1.ObjectMeta` k8s 的 metadata 的資料結構，像是 Name、Namespace、Labels，而最後 `yaml.Marshal` 會將 go 的結構序列化為 yaml 的格式

```yaml
# go run pod_yaml.go
---
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  generatename: ""
  namespace: default
  selflink: ""
  uid: ""
  resourceversion: ""
  generation: 0
  creationtimestamp: "0001-01-01T00:00:00Z"
  deletiontimestamp: null
  deletiongraceperiodseconds: null
  labels:
    app: my-app
  annotations: {}
  ownerreferences: []
  finalizers: []
  managedfields: []
spec:
  containers:
    - name: nginx
      image: nginx:1.19
```

這樣的輸出是不是就很像是我們平常在定義資源的 yaml 了呀，接下來我們要來看到 k8s 是怎麼實作這段，這時候我們就要看到這個 repo `kubernetes/apimachinery`，我們來看看他是怎麼說明的

## apimachinery

> Scheme, typing, encoding, decoding, and conversion packages for Kubernetes and Kubernetes-like API objects.

apimachinery 的角色是提供一組通用的基礎設施，用來處理 k8s api 的結構化資料與轉換機制，它包含了我們所想了解的 Encoding / Decoding 也就是物件與 JSON、YAML 等格式的序列化與反序列化，在 apimachinery 有幾個核心的元件

[**Scheme**](https://github.com/kubernetes/apimachinery/tree/master/pkg/runtime/schema) 是 k8s 物件的註冊中心，它負責將 Go struct（例如 v1.Pod）與其對應的 Group/Version/Kind（G/V/K）關聯起來，這樣在反序列化 YAML/JSON 時，系統才知道該用哪個 struct 來 decode

所有的 k8s 資源都內嵌 **TypeMeta 與 ObjectMeta**，TypeMeta 包含 apiVersion 和 kind，ObjectMeta 包含 name、namespace、labels 等等

**Encoder / Decoder** 支援多種序列化格式，包括 JSON、YAML、Protobuf，能根據 media type（如 application/json）自動選擇對應的編解碼器

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
```

當我們寫下此 YAML 時，apimachinery 的 Decoder 解析這段，根據 `apiVersion: apps/v1` 和 `kind: Deployment` 查詢 Scheme，找到對應的 Go struct，將 YAML 內容反序列化成該 struct 的實例，後續進行 validation、defaulting、conversion、storage 等流程

再來我們要來看一下 source code 是怎麼做到這些步驟的，首先我們要看到 `YAMLToJSONDecoder`

```go
// Decode reads a YAML document as JSON from the stream or returns
// an error. The decoding rules match json.Unmarshal, not
// yaml.Unmarshal.
func (d *YAMLToJSONDecoder) Decode(into interface{}) error {
	bytes, err := d.reader.Read()
	if err != nil && err != io.EOF { //nolint:errorlint
		return err
	}

	if len(bytes) != 0 {
		err := yaml.Unmarshal(bytes, into)
		if err != nil {
			return YAMLSyntaxError{err}
		}
	}
	d.inputOffset += len(bytes)
	return err
}
```

這邊的 `YAMLToJSONDecoder.Decode` 實際上並非轉為 JSON，而是直接使用 `yaml.Unmarshal` 將 YAML 內容依 JSON 語意規則反序列化到目標 struct，其行為與 `json.Unmarshal` 一致，確保後續處理符合 k8s runtime 的解碼預期

接著看到 Scheme，Scheme 透過內部的 `gvkToType` 映射，根據 GroupVersionKind（GVK）快速查找對應的 Go struct 類型（reflect.Type），使 k8s 能在反序列化時，依據 YAML 中的 apiVersion 與 kind 欄位，精確實例化出正確的資源物件結構

```go
// KnownTypes returns the types known for the given version.
func (s *Scheme) KnownTypes(gv schema.GroupVersion) map[string]reflect.Type {
	types := make(map[string]reflect.Type)
	for gvk, t := range s.gvkToType {
		if gv != gvk.GroupVersion() {
			continue
		}

		types[gvk.Kind] = t
	}
	return types
}
```

接著往下看，`Serializer.Decode` 首先將 YAML 轉為 JSON，再透過 `meta.Interpret` 從資料中提取 GVK，接著結合預設 GVK 與目標物件類型，利用 `ObjectCreater` 建立對應的 Go struct 實例，並透過嚴格或寬鬆的 JSON 反序列化將資料載入物件，最終返回具體資源物件與其 GVK

```go
// Decode attempts to convert the provided data into YAML or JSON, extract the stored schema kind, apply the provided default gvk, and then
// load that data into an object matching the desired schema kind or the provided into.
// If into is *runtime.Unknown, the raw data will be extracted and no decoding will be performed.
// If into is not registered with the typer, then the object will be straight decoded using normal JSON/YAML unmarshalling.
// If into is provided and the original data is not fully qualified with kind/version/group, the type of the into will be used to alter the returned gvk.
// If into is nil or data's gvk different from into's gvk, it will generate a new Object with ObjectCreater.New(gvk)
// On success or most errors, the method will return the calculated schema kind.
// The gvk calculate priority will be originalData > default gvk > into
func (s *Serializer) Decode(originalData []byte, gvk *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	data := originalData
	if s.options.Yaml {
		altered, err := yaml.YAMLToJSON(data)
		if err != nil {
			return nil, nil, err
		}
		data = altered
	}

	actual, err := s.meta.Interpret(data)
	if err != nil {
		return nil, nil, err
	}

	if gvk != nil {
		*actual = gvkWithDefaults(*actual, *gvk)
	}

	if unk, ok := into.(*runtime.Unknown); ok && unk != nil {
		unk.Raw = originalData
		unk.ContentType = runtime.ContentTypeJSON
		unk.GetObjectKind().SetGroupVersionKind(*actual)
		return unk, actual, nil
	}

	if into != nil {
		_, isUnstructured := into.(runtime.Unstructured)
		types, _, err := s.typer.ObjectKinds(into)
		switch {
		case runtime.IsNotRegisteredError(err), isUnstructured:
			strictErrs, err := s.unmarshal(into, data, originalData)
			if err != nil {
				return nil, actual, err
			}

			// when decoding directly into a provided unstructured object,
			// extract the actual gvk decoded from the provided data,
			// and ensure it is non-empty.
			if isUnstructured {
				*actual = into.GetObjectKind().GroupVersionKind()
				if len(actual.Kind) == 0 {
					return nil, actual, runtime.NewMissingKindErr(string(originalData))
				}
				// TODO(109023): require apiVersion here as well once unstructuredJSONScheme#Decode does
			}

			if len(strictErrs) > 0 {
				return into, actual, runtime.NewStrictDecodingError(strictErrs)
			}
			return into, actual, nil
		case err != nil:
			return nil, actual, err
		default:
			*actual = gvkWithDefaults(*actual, types[0])
		}
	}

	if len(actual.Kind) == 0 {
		return nil, actual, runtime.NewMissingKindErr(string(originalData))
	}
	if len(actual.Version) == 0 {
		return nil, actual, runtime.NewMissingVersionErr(string(originalData))
	}

	// use the target if necessary
	obj, err := runtime.UseOrCreateObject(s.typer, s.creater, *actual, into)
	if err != nil {
		return nil, actual, err
	}

	strictErrs, err := s.unmarshal(obj, data, originalData)
	if err != nil {
		return nil, actual, err
	} else if len(strictErrs) > 0 {
		return obj, actual, runtime.NewStrictDecodingError(strictErrs)
	}
	return obj, actual, nil
}
```

這也是為什麼在開發 CRD 時，我們要在程式中 [register](https://github.com/kubernetes/apimachinery/blob/master/pkg/apis/meta/v1/register.go) CR 到 Scheme，否則系統無法根據 GVK 找到對應的 struct 來 decode

## 小結

這邊介紹了 `apimachinery` 是如何實現辨認 yaml 資源的步驟，後續流程當然還有 Validation、Defaulting、Conversion、Storage 等步驟。這邊我也是看了頭很痛，看了很久，假如有說明不正確的地方，或需要補充的地方再麻煩跟我說了！
