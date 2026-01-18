## K8S Lab Day_45

# Study7-深入理解 Kubernetes Defaulting 與 Validation：從 Pod 建立流程看自動補值與欄位驗證

## 前言

昨天有提到 etcd 資料的儲存，今天會繼續來看到當建立 pod 的時候底層還會做到哪些事，今天會來講到 defaulting 和 validation，那為何要 defaulting 和 validation 呢？ defaulting 要確定會自動填入 default 的值，因為有時候不是所有的參數我們都會寫，而 validation 是要確保欄位符合規範，防止無效的配置，像是 `containerPort` 就必須在 1~65535

## 小實驗

### 用 client-go 觀察 defaulting 行為

```bash
go mod init demo
```

```go
// vi defaulting_demo.go
package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	corev1 "k8s.io/api/core/v1"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "incomplete-pod-demo",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:alpine",
					// 故意不填：restartPolicy, ports, resources, etc.
				},
			},
			// 故意不填：restartPolicy, dnsPolicy, etc.
		},
	}

	createdPod, err := clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Println("Pod 建立成功！觀察 defaulting 結果：")
	fmt.Printf("  restartPolicy: %s\n", createdPod.Spec.RestartPolicy)
	fmt.Printf("  dnsPolicy: %s\n", createdPod.Spec.DNSPolicy)
	fmt.Printf("  container[0].ports: %v\n", createdPod.Spec.Containers[0].Ports)
	fmt.Printf("  serviceAccountName: %s\n", createdPod.Spec.ServiceAccountName)
}
```

```bash
go mod tidy

go run defaulting_demo.go
Pod 建立成功！觀察 defaulting 結果：
  restartPolicy: Always
  dnsPolicy: ClusterFirst
  container[0].ports: []
  serviceAccountName: default
# 這裏自動輸入了我們所沒有填到的部分
```

接著我們就可以看一下我們輸出的內容

```bash
kubectl get pod

NAME                              READY   STATUS              RESTARTS   AGE
incomplete-pod-demo               1/1     Running             0          2m6s
# 這邊確實也有創建出我們想要的內容
kubectl get pod incomplete-pod-demo -o yaml
# 假如有興趣也可以輸入這個指令得到更多內容
```

## defaulting

接著我們要來看一下 source code 是怎麼做到 defaulting 的，k8s 在 api server 接收到物件後，會將未填寫欄位補齊 default，這些 defaulting function 會在 scheme 中註冊，並在物件 decode 完成後套用，確保所有進入系統的 api 物件都保持著一致的欄位狀態

```go
func SetDefaults_Pod(obj *v1.Pod) {
    // 若 container 有設定 limits 但未設定 requests，則自動將 requests 設為 limits
    for i := range obj.Spec.Containers {
        if obj.Spec.Containers[i].Resources.Limits != nil {
            if obj.Spec.Containers[i].Resources.Requests == nil {
                obj.Spec.Containers[i].Resources.Requests = make(v1.ResourceList)
            }
            for key, value := range obj.Spec.Containers[i].Resources.Limits {
                if _, exists := obj.Spec.Containers[i].Resources.Requests[key]; !exists {
                    obj.Spec.Containers[i].Resources.Requests[key] = value.DeepCopy()
                }
            }
        }
    }
    // ...

    // 若未指定 EnableServiceLinks，預設啟用
    if obj.Spec.EnableServiceLinks == nil {
        enableServiceLinks := v1.DefaultEnableServiceLinks
        obj.Spec.EnableServiceLinks = &enableServiceLinks
    }

    // 若 HostNetwork = true，則所有 container port 預設等於 host port
    if obj.Spec.HostNetwork {
        defaultHostNetworkPorts(&obj.Spec.Containers)
        defaultHostNetworkPorts(&obj.Spec.InitContainers)
    }
}
```

但是這邊我們好像沒有看到他沒有把全部都補齊，這邊其實他只補齊了 api 層的 default 像是 `EnableServiceLinks` 未填寫時就自動設為預設值，還有當 container 有 limits 但沒 requests 時，為避免資源配置不一致，requests 會自動等於 limits，而後續的資料就會由 `Admission Controllers` 和 `kubelet` 等等做填入

## validation

接著我們就要來看到 Validation，確保物件內容符合格式和規範，而 validation 是在寫入 etcd 前對內容做合法性的檢查，以 pod 的 `ResourceRequirements` 驗證邏輯為例，可以看到 `ValidateResourceRequirements` 會逐一檢查 limits 和 requests 的合理性

```go
func ValidateResourceRequirements(requirements *v1.ResourceRequirements, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	limPath := fldPath.Child("limits")
	reqPath := fldPath.Child("requests")

	for resourceName, quantity := range requirements.Limits {
		fldPath := limPath.Key(string(resourceName))
		allErrs = append(allErrs, ValidateContainerResourceName(core.ResourceName(resourceName), fldPath)...)
		allErrs = append(allErrs, ValidateResourceQuantityValue(core.ResourceName(resourceName), quantity, fldPath)...)
	}

	for resourceName, quantity := range requirements.Requests {
		fldPath := reqPath.Key(string(resourceName))
		allErrs = append(allErrs, ValidateContainerResourceName(core.ResourceName(resourceName), fldPath)...)
		allErrs = append(allErrs, ValidateResourceQuantityValue(core.ResourceName(resourceName), quantity, fldPath)...)

		// request 必須 <= limit（除非是允許 overcommit 的資源）
		limitQuantity, exists := requirements.Limits[resourceName]
		if exists {
			if quantity.Cmp(limitQuantity) != 0 && !v1helper.IsOvercommitAllowed(resourceName) {
				allErrs = append(allErrs, field.Invalid(reqPath, quantity.String(), fmt.Sprintf("must be equal to %s limit of %s", resourceName, limitQuantity.String())))
			} else if quantity.Cmp(limitQuantity) > 0 {
				allErrs = append(allErrs, field.Invalid(reqPath, quantity.String(), fmt.Sprintf("must be less than or equal to %s limit of %s", resourceName, limitQuantity.String())))
			}
		}
	}
	return allErrs
}
```

## 小結

這邊看似不是填入所有的 default，光是剛剛有提到的 `serviceAccountName` 就沒有看到他的 default 是怎麼被填進去的，這就會是在 Admission Controller 補上，在後續我們會繼續看到 Admission Controller 完成了哪些事情，為何會這樣創建資源的
