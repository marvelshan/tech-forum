## K8S Lab Day_42

# Study4-當無數個 Goroutine 同時甦醒，我看見了雲原生的靈界之門

## 前言

昨天我們學習了 go module 底層是怎麼去呼叫 function 並使用其方法，那今天我們要來學習 goroutines 與 channels，並對比 k8s 控制器模式的併發需求，也要透過一些小實驗來實作 k8s 的 pod 調度，來了解底層原理的運作，讓我們更清楚 go 的特性對 k8s 和 istio 帶來的好處

## 什麼是 Goroutines？

Goroutine 是 Go 的輕量化執行緒，因為他創建和切換的成本極低，可以輕鬆啟動數千甚至十萬個 goroutine，那要如何使用，在第一天的時候就有使用到，在 k8s 中，deployment controller 需要同時監控多個資源的狀態變化 goroutines 就非常適合來處理這類的任務

```go
go func() {
    fmt.Println("Runnung in a goroutine")
}
```

## 什麼是 Channel？

Channel 是 go 提供於同步和通信的機制，用在 goroutines 之間傳遞資料，channel 卻報資料傳輸的安全，避免 mutex 的複雜性，在第一天也有使用到，在 istio 的 control plane 中，goroutine 可以用來處理多個 sidecar 配置的更新，而 channels 就是用於協調這些更新

```go
ch := make(chan string)
go func() {
    ch <- "Hello from goroutine"
}()
msg := <-ch
fmt.Println(msg)
```

## k8s 中的併發機制

在 k8s 中， controller pattern 依賴併發來確保 cluster 的 actual state 和 desired state 一致，而其中的工作原理是 Deployment Controller、ReplicaSet Controller 透過 reconcile loop 來監控資源的狀態，當檢查到差異時執行動作，像是創建或刪除 Pod，每個 controller 需要併行處理多個資源，且快速地響應事件，K8s controller-runtime 內部廣泛運用這些機制，以保證系統在高頻率事件下仍能穩定運作

## Concurrent Reconciling

在 [k8s controller-runtime](https://github.com/kubernetes-sigs/controller-runtime/blob/main/pkg/controller/controller.go) 中，`MaxConcurrentReconciles` 這個參數決定了同時可以有多少個 reconcile loop 被啟動，當監控的物件頻繁變化時像是大量 Pod 更新，會產生許多 reconcile 請求，如果僅使用單一執行緒處理，reconcile queue 很快就會塞滿，導致延遲累積。此時開啟多執行緒（Concurrent Reconciling）能顯著提升吞吐量

```go
	// MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run. Defaults to 1.
	MaxConcurrentReconciles int
...
	// Reconciler reconciles an object
	Reconciler reconcile.TypedReconciler[request]
```

這時候就會想開啟多執行緒後，會不會有兩個 reconcile loop 同時處理同一個物件，造成狀態不一致？

答案是不會！是因為 controller-runtime 內部採用了 k8s client-go 提供的 workqueue 實作，該資料結構在 goroutines 間協調物件的處理狀態

當一個 reconcile request 進入 queue 時，

- 若該物件在 processing set 中正在被處理，僅將它加入 dirty set
- 若該物件尚未被處理，則加入 queue 並標記進入 processing set
- 當該物件處理完畢後，若 dirty set 中仍有它的紀錄，代表在期間內又發生了變化，物件會被重新加入 queue

<img width="670" height="808" alt="image" src="https://github.com/user-attachments/assets/4411f7d6-49cd-4fc8-96c9-c27c05f6602b" />

> Dirty Set：記錄所有已經被標記為「需再次處理」的物件
>
> Processing Set：記錄當前正在被某個 reconcile loop 處理的物件

這樣的設計保證不會同時有兩個 goroutine 處理同一個物件，並能夠在高頻事件中自動合併多次變更，避免重複計算

但這種機制也帶來一個副作用，因為物件可能會在 queue 尾端被重新排入，因此某些請求可能遭遇延遲，特別是長時間的 reconcile 任務中

controller 的行為是 level-triggered，而非 edge-triggered，並不保證會對每一個事件都逐一響應，而是確保系統最終能收斂至正確的狀態，它追求的是狀態一致性，而非事件順序的完整追蹤

## 小實驗

### 用 goroutines 實現一個簡單的工作佇列，模擬 Pod 調度

這裏啟動了三個 worker，每個運行在獨立的 goroutine，模擬 K8s controller 在多個節點上並行分配 Pod，在往後使用了 `sync.WaitGroup` 確保所有 worker 完成任務後，主程式才繼續執行，模擬 k8s 中 controller 等待所有操作完成的場景

```go
// vi workqueue.go
package main

import (
    "fmt"
    "sync"
    "time"
)

// Pod 模擬 k8s 的 Pod
type Pod struct {
    Name string
    Node string
}

// Worker 模擬 controller 的 worker
func worker(id int, jobs <-chan Pod, results chan<- string, wg *sync.WaitGroup) {
    defer wg.Done()
    for pod := range jobs {
        fmt.Printf("Worker %d: Scheduling pod %s\n", id, pod.Name)
        // 模擬調度耗時操作
        time.Sleep(100 * time.Millisecond)
        // 假如想要更明顯的延遲可以把這邊加成 1000
        pod.Node = fmt.Sprintf("node-%d", id)
        results <- fmt.Sprintf("Pod %s scheduled to %s", pod.Name, pod.Node)
    }
}

func main() {
    // 創建任務 channel
    jobs := make(chan Pod, 10)
    results := make(chan string, 10)
    var wg sync.WaitGroup

    // 啟動三個 goroutines，模擬多節點調度
    numWorkers := 3
    for i := 1; i <= numWorkers; i++ {
        wg.Add(1)
        go worker(i, jobs, results, &wg)
    }

    // 模擬提交 Pod 任務
    pods := []Pod{
        {Name: "pod-1"},
        {Name: "pod-2"},
        {Name: "pod-3"},
        {Name: "pod-4"},
        {Name: "pod-5"},
    }
    for _, pod := range pods {
        jobs <- pod
    }
    close(jobs) // 關閉任務佇列，表示沒有更多任務

    // 等待所有工作者完成
    wg.Wait()
    close(results) // 關閉結果佇列

    // 輸出調度結果
    for result := range results {
        fmt.Println(result)
    }
}
```

## Workqueue

剛剛透過 goroutine 與 channel 模擬了一個簡易的 workqueue，將資源變更事件轉為任務，並交由多個 worker 併發處理，在 `jobs := make(chan Pod, 10)` 非常直觀，但在實際 controller 中不會直接使用這種內建的、未經包裝的 chan 類型，也不會將整個 Pod 物件丟進去，可以再想想前面的那張圖，假如同一個 pod 多次變更的話 `chan pod` 的方式就會讓他多次進 queue，造成重複處理、浪費資源，甚至引發 race condition

那我們這邊就要看到 [source code](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/client-go/util/workqueue/queue.go#L218) 的 `func (q *Typed[T]) Add(item T)`，只要 item 已在 dirty 或 processing，就不會重複入 queue

```go
// Add marks item as needing processing. When the queue is shutdown new
// items will silently be ignored and not queued or marked as dirty for
// reprocessing.
func (q *Typed[T]) Add(item T) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	if q.shuttingDown {
		return
	}
	if q.dirty.Has(item) {
		// the same item is added again before it is processed, call the Touch
		// function if the queue cares about it (for e.g, reset its priority)
		if !q.processing.Has(item) {
			q.queue.Touch(item)
		}
		return
	}

	q.metrics.add(item)

	q.dirty.Insert(item)
	if q.processing.Has(item) {
		return
	}

	q.queue.Push(item)
	q.cond.Signal()
}
```

那接著我們要來看一下 controller 是怎麼來使用這個 workqueue，我們可以看到 sample-controller 這個 k8s 官方的 repo，他主要是實作了 Reconcile Loop 的原理，這邊我們也可以看到他這邊是怎麼去使用 `workqueue.Add()`，這裡的 `cache.ObjectToName(obj)` 會將一個 k8s 物件（例如 \*v1alpha1.Foo）轉換成一個字串形式的 key，像是 `<namespace>/<name>`

```go
// enqueueFoo takes a Foo resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Foo.
func (c *Controller) enqueueFoo(obj interface{}) {
	if objectRef, err := cache.ObjectToName(obj); err != nil {
		utilruntime.HandleError(err)
		return
	} else {
		c.workqueue.Add(objectRef)
	}
}
```

而這個 `enqueueFoo` 並不會直接被 main 呼叫，[而是會被 Informer 註冊成 EventHandler ](https://github.com/kubernetes/sample-controller/blob/master/controller.go#L128)

```go
	// Set up an event handler for when Foo resources change
	fooInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueFoo,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueFoo(new)
		},
	})
```

那接著我們再往下想，Worker 是怎麼 consume queue 的，看到以下 `go wait.UntilWithContext(ctx, c.runWorker, time.Second)` 這邊會啟動多個 worker goroutine，這邊的 `runWorker` 會去執行 `processNextWorkItem`

```go
// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	logger := klog.FromContext(ctx)

	// Start the informer factories to begin populating the informer caches
	logger.Info("Starting Foo controller")

	// Wait for the caches to be synced before starting workers
	logger.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(ctx.Done(), c.deploymentsSynced, c.foosSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logger.Info("Starting workers", "count", workers)
	// Launch two workers to process Foo resources
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	logger.Info("Started workers")
	<-ctx.Done()
	logger.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}
```

這邊我們可以看到 `func processNextWorkItem` 裡面的邏輯，`workqueue.Get()` 會將 key 從 dirty 移到 processing，接著 `workqueue.Done(objRef)` 會將 key 從 processing 移除，此時才允許再次進 queue

```go
// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	objRef, shutdown := c.workqueue.Get()
	logger := klog.FromContext(ctx)

	if shutdown {
		return false
	}

	// We call Done at the end of this func so the workqueue knows we have
	// finished processing this item. We also must remember to call Forget
	// if we do not want this work item being re-queued. For example, we do
	// not call Forget if a transient error occurs, instead the item is
	// put back on the workqueue and attempted again after a back-off
	// period.
	defer c.workqueue.Done(objRef)

	// Run the syncHandler, passing it the structured reference to the object to be synced.
	err := c.syncHandler(ctx, objRef)
	if err == nil {
		// If no error occurs then we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(objRef)
		logger.Info("Successfully synced", "objectName", objRef)
		return true
	}
	// there was a failure so be sure to report it.  This method allows for
	// pluggable error handling which can be used for things like
	// cluster-monitoring.
	utilruntime.HandleErrorWithContext(ctx, err, "Error syncing; requeuing for later retry", "objectReference", objRef)
	// since we failed, we should requeue the item to work on later.  This
	// method will add a backoff to avoid hotlooping on particular items
	// (they're probably still not going to work right away) and overall
	// controller protection (everything I've done is broken, this controller
	// needs to calm down or it can starve other useful work) cases.
	c.workqueue.AddRateLimited(objRef)
	return true
}
```

接著，`syncHandler` 比較了資源的實際狀態和期望狀態，最後更新

```go
// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *Controller) syncHandler(ctx context.Context, objectRef cache.ObjectName) error {
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "objectRef", objectRef)

	// Get the Foo resource with this namespace/name
	foo, err := c.foosLister.Foos(objectRef.Namespace).Get(objectRef.Name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleErrorWithContext(ctx, err, "Foo referenced by item in work queue no longer exists", "objectReference", objectRef)
			return nil
		}

		return err
	}
...
	// Finally, we update the status block of the Foo resource to reflect the
	// current state of the world
	err = c.updateFooStatus(ctx, foo, deployment)
	if err != nil {
		return err
	}

	c.recorder.Event(foo, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}
```

## 小結

今天也是講了非常多的東西，但是也是學習到很多呀！看到了 controller 背景是怎麼運作了，發現其實 k8s 有很多很好玩的小細節，可以更清楚前人在設計的時候的智慧，明天也要繼續努力呢！

## Reference

https://openkruise.io/blog/learning-concurrent-reconciling
