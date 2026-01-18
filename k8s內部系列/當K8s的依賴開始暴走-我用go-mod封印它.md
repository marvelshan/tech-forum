## K8S Lab Day_41

# Study3-當 K8s 的依賴開始暴走，我用 go mod 封印它

## 前言

昨天有說明到如何實作 `kubectl get pod` 的 CLI 工具，去拿到實際應用場景的 k8s cluster 資訊，今天會 focus 在 module 也就是昨天也有稍微用到的 `go mod`，昨天也有提到指令像是 `go mod init <module-name>` 可以來初始化 module，還有 `go mod tidy` 來同步 `go.mod` 和 `go.sum`，還有 `go get <module>@<version>` 添加依賴還有 `go mod download` 等等

## Kubernetes 的 go.mod

在以下可以看到在 [k8s github 底下的 `go.mod`](https://github.com/kubernetes/kubernetes/blob/master/go.mod)，首先是 `k8s.io/kubernetes` 是用來定義這個 module 的名稱，在 require 裏可以看到依賴內部的 k8s.io 模組像是 client-go、api 等等，這些模組在 `staging/src/` 目錄獨立維護，確保模組化

```go
module k8s.io/kubernetes

go 1.25.0

godebug default=go1.25

require (
	bitbucket.org/bertimus9/systemstat v0.5.0
	github.com/JeffAshton/win_pdh v0.0.0-20161109143554-76bb4ee9f0ab
	github.com/Microsoft/go-winio v0.6.2
	github.com/Microsoft/hnslib v0.1.1
    	k8s.io/api v0.0.0
	k8s.io/apiextensions-apiserver v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/apiserver v0.0.0
	k8s.io/cli-runtime v0.0.0
	k8s.io/client-go v0.0.0
...
```

接著我們就要回來看昨天的程式碼，為何會使用到 `k8s.io/client-go/kubernetes`？ 這裏主要是讓我們可以更輕鬆的調用 k8s api，而這裡做到的是包含了 client-go 封裝了與 k8s API Server 的 HTTP 請求，開發者無需手動處理 RESTful API 還有一些 CRD 的操作街口，還有一些內建的 cache 和監聽機制，我們可以看到以下透過 kubeconfig 檔案初始化了一個 Clientset，用於後續的 API 操作

```go
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}
	client := &K8sClient{clientset: clientset}
```

## CoreV1()：指向核心 API 的 Client

以下這邊可以看到會用到 `client-go` 裡面的 `CoreV1()` 來去返回一個指向核心 API（core/v1）的客戶端接口，包含 Pod、Service、Node 等核心資源的操作，這裏也可以在 [github repo](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/client-go/kubernetes/clientset.go#L324) 看到這個 `func (c *Clientset) CoreV1() corev1.CoreV1Interface` function

然後 `Pods(namespace)` 這邊就可以看到他要返回送進來帶 `namespace` 的 pod，`List(ctx context.Context, opts metav1.ListOptions)` 這邊 List 方法返回一個 PodList 結構，包含所有 Pod 的詳細資訊，它使用了 RESTful API 的語義，並內建錯誤處理與重試機制

```go
	pods, err := c.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Error listing pods: %v", err)
		return nil, err
	}
```

## Pods(namespace) 與 newPods() 的連結

那好像還不是很清楚他們的連結性，這就可以看到[這個檔案](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/client-go/kubernetes/typed/core/v1/core_client.go#L90C1-L92C2) `kubernetes/staging/src/k8s.io/client-go/kubernetes/typed/core/v1/core_client.go`

```go
func (c *CoreV1Client) Pods(namespace string) PodInterface {
	return newPods(c, namespace)
}
```

這裡每當使用者透過 CoreV1Client 呼叫 Pods(namespace) 方法時，就會觸發 `newPods(c, namespace)`，進而建立一個專屬於該 namespace 的 Pod 操作客戶端

## newPods() 與 Struct Embedding 的應用

接著就可以進到 [github](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/client-go/kubernetes/typed/core/v1/pod.go) `staging/src/k8s.io/client-go/kubernetes/typed/core/v1/pod.go`，這個位置來看怎麼使用的

在這裡透過 newPods 這個 function 獲取到的是過 PodInterface，他的底層是一個 `*pod` 的一個 struct，透過 Struct Embedding 的方法，內嵌了泛型客戶端 `*gentype.ClientWithListAndApply[...]`，而這邊也繼承了 PodInterface CRUD 的 method，也就是可以在上面看到的 `Pods(namespace).List()`

```go
// PodInterface has methods to work with Pod resources.
type PodInterface interface {
	Create(ctx context.Context, pod *corev1.Pod, opts metav1.CreateOptions) (*corev1.Pod, error)
	Update(ctx context.Context, pod *corev1.Pod, opts metav1.UpdateOptions) (*corev1.Pod, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, pod *corev1.Pod, opts metav1.UpdateOptions) (*corev1.Pod, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Pod, error)
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error)
    ...
}

// pods implements PodInterface
type pods struct {
	*gentype.ClientWithListAndApply[*corev1.Pod, *corev1.PodList, *applyconfigurationscorev1.PodApplyConfiguration]
}

// newPods returns a Pods
func newPods(c *CoreV1Client, namespace string) *pods {
	return &pods{
		gentype.NewClientWithListAndApply[*corev1.Pod, *corev1.PodList, *applyconfigurationscorev1.PodApplyConfiguration](
			"pods",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *corev1.Pod { return &corev1.Pod{} },
			func() *corev1.PodList { return &corev1.PodList{} },
			gentype.PrefersProtobuf[*corev1.Pod](),
		),
	}
}
```

## List() 方法的實際呼叫鏈

接著接著，還沒完喔，這邊我們要來看到 `List()`，這邊是要來到 [staging/src/k8s.io/client-go/gentype/type.go](https://github.com/kubernetes/kubernetes/blob/fb10a2995459c52238024adbb10ffdfbdafd2c4d/staging/src/k8s.io/client-go/gentype/type.go#L133)，這裏就是剛剛看到 newPods 這個 function 裡面的 `gentype.NewClientWithListAndApply[]()`，當我們使用到了 List，實際是呼叫了 `gentype.ClientWithListAndApply.List`，接著在這邊有 `ClientWithListAndApply`，然後在前面的 struct 有定義了 `alsoLister`，那這個定義的 `alsoLister` 就是實際實作 `func (l *alsoLister[T, L]) List()` 的地方，這也是 go Method Promotion 和 Anonymous Embedding 的特性，將內嵌類型的所有方法都會被「提升」到外層 struct，就好像外層 struct 自己定義了這些方法一樣

```go
// NewClientWithListAndApply constructs a client with support for lists and applying declarative configurations.
func NewClientWithListAndApply[T objectWithMeta, L runtime.Object, C namedObject](
	resource string, client rest.Interface, parameterCodec runtime.ParameterCodec, namespace string, emptyObjectCreator func() T,
	emptyListCreator func() L, options ...Option[T],
) *ClientWithListAndApply[T, L, C] {
	typeClient := NewClient[T](resource, client, parameterCodec, namespace, emptyObjectCreator, options...)
	return &ClientWithListAndApply[T, L, C]{
		typeClient,
		alsoLister[T, L]{typeClient, emptyListCreator},
		alsoApplier[T, C]{typeClient},
	}
}

...

// List takes label and field selectors, and returns the list of resources that match those selectors.
func (l *alsoLister[T, L]) List(ctx context.Context, opts metav1.ListOptions) (L, error) {
	list := l.newList()
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	err := l.client.client.Get().
		UseProtobufAsDefaultIfPreferred(l.client.prefersProtobuf).
		NamespaceIfScoped(l.client.namespace, l.client.namespace != "").
		Resource(l.client.resource).
		VersionedParams(&opts, l.client.parameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(list)
	return list, err
}
```

## 呼叫鏈總結與結構圖

用這樣的方式應該就可以更清楚的了解了他實際的運作，這就是很典型的 composition over inheritance 設計，利用 go 的 Generic Type 來避免重複的使用 CRUD

```text
pods (struct)
 └─ *ClientWithListAndApply[P, PL, A]
      └─ *client[P, PL, A]
           └─ *alsoLister[P, PL]  ← List 方法在此實作
```

## Anonymous Embedding 和 Method Promotion

我們在前面提到 `gentype.NewClientWithListAndApply[]()` 這個呼叫，實際上就是一個 Anonymous Embedding 的例子，這種設計並非 inheritance，而是 Go 所強調的 composition over inheritance，藉由結構的內嵌組合，讓功能模組彼此協作，而不是依賴類似 OOP 中的繼承階層，在 Go 中，可以將一個 struct 直接嵌入另一個 struct，而不必給它命名，這就叫 `Anonymous Embedding`，當一個 struct anonymous embedding 另一個 struct，內層 struct 所定義的方法也會被自動「提升」成為外層 struct 的方法，我們來使用一個簡單的例子來介紹

```go
package main

import "fmt"

// 子結構
type details struct {
	name    string
	age     int
	psalary int
}

// details 有自己的方法
func (d details) totalSalary(days int) int {
	return d.psalary * days
}

// 父結構匿名嵌入 details
type employee struct {
	post string
	id   int
	details
}

func main() {
	e := employee{
		post: "Engineer",
		id:   42,
		details: details{
			name:    "Yusuke",
			age:     22,
			psalary: 1000,
		},
	}

	// Promoted Fields
	fmt.Println("Name:", e.name)
	fmt.Println("Post:", e.post)

	// Promoted Method
	// 雖然 totalSalary 定義在 details，但外層 employee 可以直接呼叫
	fmt.Println("Total Salary:", e.totalSalary(30))
}

// Output
// Name: Yusuke
// Post: Engineer
// Total Salary: 30000
```

## Composition over Inheritance

在傳統物件導向語言像是 Java、C++、TypeScript 裡，如果我們希望 Client 能同時具備 List 與 Apply 功能，通常會使用 Inheritance 來達成，例如建立一個 BaseClient，再讓 ListClient 與 ApplyClient 繼承它，最後再建立一個 FullClient 同時繼承兩者，但這種層層繼承的做法，會導致程式架構僵化、耦合度高、難以維護，Go 採取了完全不同的方向，透過 Composition 而非 Inheritance 來達到可擴充性與重用性，也就是說，Go 不會建立繼承階層，而是用小型、可獨立運作的 struct 彼此組合，讓每個部分都專注於一件事，最後再透過 Anonymous Embedding 整合成完整的功能體

在 client-go 的例子中，我們看到 `ClientWithListAndApply` 並不是繼承自 Client，而是 embedding 了三個模組化的 struct

```go
return &ClientWithListAndApply[T, L, C]{
    typeClient,
    // typeClient：封裝最基本的 REST 操作與通用邏輯
    alsoLister[T, L]{typeClient, emptyListCreator},
    // alsoLister：負責實作 List() 方法
    alsoApplier[T, C]{typeClient},
    // alsoApplier：負責實作 Apply() 方法
}
```

這三者組合後的 ClientWithListAndApply 就像一個「靈壓融合體」，繼承了三者的行為，卻沒有繼承階層的束縛

## 小結

原本今天就想說用如何 import package 帶過這天，但一研究下來，好像就停不下來，一直往下去 trace，也發揮了工程師 trace bug 練就的技巧，不過這樣看下來，對於 k8s 的結構和 go 的語法又更加的了解，今天也是學了很多東西，發現真的是很讚呀～

## Reference

https://www.geeksforgeeks.org/go-language/promoted-methods-in-golang-structure/

https://kevin-yang.medium.com/golang-embedded-structs-b9d20aadea84

https://aran.dev/posts/go-and-composition-over-inheritance/
