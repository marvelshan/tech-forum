## K8S Lab Day_44

# Study6-Kubernetes API 請求的 Rate Limiting 與錯誤重試機制

## 前言

昨天講到了如何解析在建立資源的時候 YAML 是如何被解析的，今天會繼續介紹 api 的穩定性和可靠性，探討到 rate limiting 和 retry 的機制，最後會繼續補充到昨天還沒有講完的 k8s 建立資源的流程

## 為何需要 Rate Limiting 與錯誤重試？

k8s api server 是整個 cluster 的 control plane，承載了所有資源操作的請求，假如在高負載貨異常的情況下，假如沒有很好的防護，api server 崩潰會導致整個 cluster 不可用，這樣是相當的危險的，所以 rate limiting 和 retry 的機制就相當的重要

### Flow Control 機制：Client-side vs Server-side

首先我們要先來看到有兩種 rate limiting 的機制，分為 Client-side Flow Control 和 Server-side Flow Control，Client-side Flow Control 是發生在 kubectl 和 controller，是由 client-go 來實現的、而 Server-side Flow Control 是發生在 api server 內部的，主要是由 [Priority and Fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/) 機制去實現的

### Server-side Flow Control：API Priority and Fairness (APF)

接著我們可以看到 [staging/src/k8s.io/apiserver/pkg/server/filters/priority-and-fairness.go](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/server/filters/priority-and-fairness.go) 這個檔案，先來介紹 APF 的機制，他是針對不同優先等級的請求進行陪對和控制，核心目標是防止低優先的請求阻塞系統，確保高優先的請求能夠被優先處理，再來是看到程式碼 `priorityAndFairnessHandler.Handle`，每個請求抵達時，[會先取出 `RequestInfo` 和 `User`，並判斷是否為 watch 請求或是長時間的請求](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/server/filters/priority-and-fairness.go#L103C1-L108C3)

```go
	// Skip tracking long running non-watch requests.
	if h.longRunningRequestCheck != nil && h.longRunningRequestCheck(r, requestInfo) && !isWatchRequest {
		klog.V(6).Infof("Serving RequestInfo=%#+v, user.Info=%#+v as longrunning\n", requestInfo, user)
		h.handler.ServeHTTP(w, r)
		return
	}
```

### Classification

再來是針對一般的請求做 classification，決定請求對應的 FlowSchema 與 PriorityLevel，並估算他的 workEstimator

```go
	var classification *PriorityAndFairnessClassification
	noteFn := func(fs *flowcontrol.FlowSchema, pl *flowcontrol.PriorityLevelConfiguration, flowDistinguisher string) {
		classification = &PriorityAndFairnessClassification{
			FlowSchemaName:    fs.Name,
			FlowSchemaUID:     fs.UID,
			PriorityLevelName: pl.Name,
			PriorityLevelUID:  pl.UID,
		}
        ...
	}
    ...
    estimateWork := func() flowcontrolrequest.WorkEstimate {
        return h.workEstimator(r, classification.FlowSchemaName, classification.PriorityLevelName)
    }
```

### Work Estimation

在以上的 `estimateWork` 估算完工作量後，APF 會根據優先等級和資源佔用決定是否立即執行、排隊等待或是拒絕請求，[而等待的上限是由 `getRequestWaitContext` 控制的](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/server/filters/priority-and-fairness.go#L396)，避免請求無限的排隊

```go
func getRequestWaitContext(ctx context.Context, defaultRequestWaitLimit time.Duration, clock utilsclock.PassiveClock) (context.Context, context.CancelFunc) {
    thisReqWaitLimit := defaultRequestWaitLimit
    if deadline, ok := ctx.Deadline(); ok {
        thisReqWaitLimit = deadline.Sub(clock.Now()) / 4
    }
    if thisReqWaitLimit > time.Minute {
        thisReqWaitLimit = time.Minute
    }
    return context.WithDeadline(ctx, clock.Now().Add(thisReqWaitLimit))
}
```

而 watch 請求，APF 只會在初始化階段紀錄資源的消耗，[watch 本身的長時間存活不會阻塞 APF 的併發數](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/server/filters/priority-and-fairness.go#L211)

```go
forgetWatch = h.fcIfc.RegisterWatch(r)
close(shouldStartWatchCh)
watchInitializationSignal.Wait()
```

假如 request 被拒絕像是 queue 超時或是資源不足，[APF 會回傳 HTTP 429 並設定 retry after](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/server/filters/priority-and-fairness.go#L317)

```go
tooManyRequests(r, w, strconv.Itoa(int(h.droppedRequests.GetRetryAfter(classification.PriorityLevelName))))
```

以上可以看到 APF 利用 FlowSchema 和 Priority 來對 request 分類，並針對不同請求類型像是 mutating、readonly、watch 給予差異化的處理，來保護 API server 不會過載

### Client-side Flow Control：Token Bucket Rate Limiter

那 client-side 的 rate limiting 呢？可以看到 [`func NewTokenBucketRateLimiter(qps float32, burst int) RateLimiter`](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/client-go/util/flowcontrol/throttle.go#L63C1-L63C67)，這裏是使用 Token Bucket 來去防止 client-side 短時間對 api server 發送過多的請求，造成 control plane 過載，Token Bucket 基本的概念是每個請求需要消耗一個 token，假如 bucket 有 token 請求會立即執行，假如沒有請求就會等待或是被拒絕，而 bucket 內的 token 上限是由 burst 來決定

code 可以看到 `tokenBucketRateLimiter` 與 `tokenBucketPassiveRateLimiter` 兩種 struct，前者提供阻塞的 `Accept()` 與 `Wait(ctx)` 方法，後者則是被動型，不會阻塞，只能立即嘗試消耗 token

```go
type tokenBucketPassiveRateLimiter struct {
    limiter *rate.Limiter
    qps     float32
    clock   clock.PassiveClock
}

type tokenBucketRateLimiter struct {
    tokenBucketPassiveRateLimiter
    clock Clock
}
```

使用 `NewTokenBucketRateLimiter` 可以建立一個可用於 client 端的 rate limiter

```go
func NewTokenBucketRateLimiter(qps float32, burst int) RateLimiter {
    limiter := rate.NewLimiter(rate.Limit(qps), burst)
    return newTokenBucketRateLimiterWithClock(limiter, clock.RealClock{}, qps)
}
```

當 client 發送請求時，會呼叫 Accept() 或 Wait(ctx) 來確認是否可以立即發送

```go
// Accept will block until a token becomes available
func (tbrl *tokenBucketRateLimiter) Accept() {
    now := tbrl.clock.Now()
    tbrl.clock.Sleep(tbrl.limiter.ReserveN(now, 1).DelayFrom(now))
}

func (tbrl *tokenBucketRateLimiter) Wait(ctx context.Context) error {
    return tbrl.limiter.Wait(ctx)
}
```

如果使用被動型 rate limiter，則呼叫 TryAccept() 嘗試立即消耗 token，如果沒有 token 則直接返回 false

```go
func (tbprl *tokenBucketPassiveRateLimiter) TryAccept() bool {
    return tbprl.limiter.AllowN(tbprl.clock.Now(), 1)
}
```

最後來小結一下這邊，client-side rate limiting 利用 Token Bucket 對 client 端請求進行平滑限制，配合 burst 控制短期突發流量，避免 API Server 被瞬間大量請求淹沒；而 server-side 的 APF 則在 API Server 內部進行排隊、分類和優先處理，兩者結合，形成 Kubernetes 對控制平面的完整保護機制

## etcd3/store.Create()：核心儲存邏輯

接著要來介紹寫入 etcd 的機制，因為昨天有提到創建 pod 時經過 decode 後會陸續經過幾個步驟，像是 defaulting、validation 還有 storage，也就是將資源的狀態儲存或更新到 etcd 中，而現在的 etcd 已經更新到 3 了，所以在檔案 [kubernetes/staging/src/k8s.io/apiserver/pkg/storage/etcd3/store.g](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/storage/etcd3/store.go)o 中，他會是 etcd3，這個檔案中也包含了讀取現有的資源、加密、Optimistic Concurrency Control 還有寫入 etcd

```go
func (s *store) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {
    // ...
	if version, err := s.versioner.ObjectResourceVersion(obj); err == nil && version != 0 {
		return storage.ErrResourceVersionSetOnCreate
	}
    // ...
    txnResp, err := s.client.Kubernetes.OptimisticPut(ctx, preparedKey, newData, 0, kubernetes.PutOptions{LeaseID: lease})
    if !txnResp.Succeeded {
        return storage.NewKeyExistsError(preparedKey, 0)
    }
    // ...
}
```

在這邊我們看到 `ObjectResourceVersion` 來檢查 ResourceVersion 是否是 0，那我們要先來了解這是什麼，在 etcd 中每個物件都會帶有這個欄位 metadata.resourceVersion，這個是由 etcd 自動產生，並且自己遞增版本號，用來追蹤該物件的版本

### OptimisticPut

接著我們也看到很重要的部分，也就是 `OptimisticPut` 實現樂觀鎖的部分，`OptimisticPut(ctx, key, value, 0, ...)` 但是這邊也把期望的 version 先輸入了 version 是 0，也就是這個版本是第一版，那我們來看一下在更新的時候會怎麼做呢？

```go
func (s *store) GuaranteedUpdate(...) error {
    // 1. 先讀取當前狀態
    origState, err := getCurrentState() // ← 這裡會呼叫 Get()

    for {
        // 2. 執行業務邏輯（tryUpdate）
        ret, ttl, err := s.updateState(origState, tryUpdate)

        // 3. 嘗試寫入，期望 revision = origState.rev
        txnResp, err := s.client.Kubernetes.OptimisticPut(ctx, key, newData, origState.rev, ...)

        if !txnResp.Succeeded {
            // 4. 若失敗（也許別人先改了），重新讀取
            origState, err = getCurrentState() // ← 再次 Get()
            continue
        }
        break
    }
}
```

這邊是不是就可以看到這裏的 `OptimisticPut` 是使用到 `origState.rev`，要先拿到 ResourceVersion 才可以更新現有的版本，那假如這個要新創建的資源已經被建立了呢？我們可以看到在 Create 這個 function 中有以下，也就是假如 key 已經存在，這裏會是 `txnResp.Succeeded = false`，就會直接 return 掉

```go
	if !txnResp.Succeeded {
		return storage.NewKeyExistsError(preparedKey, 0)
	}
```

那我們接著就要繼續往下看 `OptimisticPut()`，[這邊是怎麼實作的](https://github.com/etcd-io/etcd/blob/main/client/v3/kubernetes/client.go)，這裏正是實作 `Optimistic Concurrency` 的機制，以下可以看到 `Txn` 這邊會判斷說是否 `expectedRevision` 跟版本一致，假如不一致的話才會 `clientv3.OpPut(...)` 去執行 Put，所以這邊就對應到 Create 其中的一個 `txnResp, err := s.client.Kubernetes.OptimisticPut`，這邊會拿到 `txnResp`，假如 key 這裡存在，這裏的 txn 就會失敗，進而 `txnResp.Succeeded` 就會是 false，而假如是 update 的話，也是比較前面的 key 版本有沒有跟預期的相符，假如有跟預期的相符，代表說這個期間沒有被他人更改，就可以更新，也實現了樂觀鎖的效果，跟 SQL 的樂觀鎖的機制是不是也一樣呀！

```go
func (k Client) OptimisticPut(ctx context.Context, key string, value []byte, expectedRevision int64, opts PutOptions) (resp PutResponse, err error) {
    txn := k.KV.Txn(ctx).If(
        clientv3.Compare(clientv3.ModRevision(key), "=", expectedRevision),
    ).Then(
        clientv3.OpPut(key, string(value), clientv3.WithLease(opts.LeaseID)),
    )

    if opts.GetOnFailure {
        txn = txn.Else(clientv3.OpGet(key))
    }

    txnResp, err := txn.Commit()
    if err != nil {
        return resp, err
    }
    resp.Succeeded = txnResp.Succeeded
    resp.Revision = txnResp.Header.Revision
    if opts.GetOnFailure && !txnResp.Succeeded {
        if len(txnResp.Responses) == 0 {
            return resp, fmt.Errorf("invalid OptimisticPut response: %v", txnResp.Responses)
        }
        resp.KV = kvFromTxnResponse(txnResp.Responses[0])
    }
    return resp, nil
}
```

## 小結

今天不只介紹了 rate limiting 保護 api server 的機制，還講到了 etcd 背後儲存資料和更新資料的方式，後續還會再把建立資源的流程全部順一次～
