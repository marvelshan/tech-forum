## K8S Lab Day_27

# Day25: Istio 與 Sidecar 資源調優：效能最佳化、Envoy Proxy Tuning

## 前言

在昨天又重複複習了授權的設定，今天要來講效能，在 mesh 的架構下，每個 pod 都會被注入一個 sidecar，但是每個 pod 有多少流量就代表 sidecar 也會被影響，假如沒有更進階的 tunning 的話 Envoy 會默默的在服務之中產生負擔

## 1. 檢查 control plane 指標

這段輸出來自 Envoy 代理的 control plane 狀態和指標，主要顯示了 Envoy 與 Istiod 之間的同步狀況

```bash
kubectl exec -it <pod> -c istio-proxy -- pilot-agent request GET /stats | head -20
```

![pilot-agent request GET /stats](https://github.com/user-attachments/assets/b6bf5d68-bff1-49f2-a8cb-2e08b91a5e5a)

其中有的資訊有 `XDS (Discovery Service)` 配置版本資訊、`XDS gRPC` 連接狀態、`Circuit Breakers` 確認是否過載，假如目前都是 `0` 的話就是都還是沒有超過限制的狀態

## 2. 調整 sidecar 的 Requests & Limits

有兩種方法可以做到，第一個假如對 `istio-proxy` 的資源利用比較保守的話就暴力的去更改 Sidecar Injector ConfigMap 或 Pod Annotation

```bahs
kubectl edit configmap istio-sidecar-injector -n istio-system
```

```yaml
resources:
  requests:
    cpu: 50m
    memory: 64Mi
  limits:
    cpu: 500m
    memory: 256Mi
# 自己去設定要限制的資源
```

第二個就是直接去設定單一服務的 `annotations`

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: productpage
  annotations:
    sidecar.istio.io/proxyCPU: "200m"
    sidecar.istio.io/proxyMemory: "256Mi"
```

然後我們也可以看到一些 source code 怎麼去利用 `sidecar.istio.io/proxyCPU` 去做判斷然後來限制使用的資源，像是這邊使用 `Go Template` 利用了 if else 的判斷來 `Override` 預設的資源使用量，並且動態的去做調整

```yaml
        resources:
      {{- if or (isset .ObjectMeta.Annotations `sidecar.istio.io/proxyCPU`) (isset .ObjectMeta.Annotations `sidecar.istio.io/proxyMemory`) (isset .ObjectMeta.Annotations `sidecar.istio.io/proxyCPULimit`) (isset .ObjectMeta.Annotations `sidecar.istio.io/proxyMemoryLimit`) }}
        {{- if or (isset .ObjectMeta.Annotations `sidecar.istio.io/proxyCPU`) (isset .ObjectMeta.Annotations `sidecar.istio.io/proxyMemory`) }}
          requests:
            {{ if (isset .ObjectMeta.Annotations `sidecar.istio.io/proxyCPU`) -}}
            cpu: "{{ index .ObjectMeta.Annotations `sidecar.istio.io/proxyCPU` }}"
            {{ end }}
            {{ if (isset .ObjectMeta.Annotations `sidecar.istio.io/proxyMemory`) -}}
            memory: "{{ index .ObjectMeta.Annotations `sidecar.istio.io/proxyMemory` }}"
            {{ end }}
```

## 3. Connection Pool & Circuit Breaker

Istio 是允許透過 DestinationRule 來設定 Envoy 的連線池與超時行為，我們可以利用 `connectionPool` 控制 Envoy 可維持多少連線，避免爆量導致高 latency，`outlierDetection` 則能主動隔離異常的目標端點，提升整體穩定度，而我們這邊可以看到截取的 istio 開發者所寫的 unit test，這類 Control Plane 的驗證測試確保配置進入 Pilot 時能被正確接收與轉換成 Envoy cluster 配置，當然在這個測試還有提到像是 `bad max requests per connection`、`valid connection pool, tcp timeout disabled`、`invalid connection pool, bad max concurrent streams` 等等的行為可以看到 `http1MaxPendingRequests`、`Http2MaxRequests` 的配置來去測試說他是不是有在這個安全範圍內，雖然這段只是去測試配置這些參數在測試狀況下是否有異常，但我們還是可以透過這些測試文件來了解說可以怎麼去配置這些參數來限制和去保護我們的服務

```go
func TestValidateConnectionPool(t *testing.T) {
	cases := []struct {
		name  string
		in    *networking.ConnectionPoolSettings
		valid bool
	}{
		{
			name: "valid connection pool, tcp and http",
			in: &networking.ConnectionPoolSettings{
				Tcp: &networking.ConnectionPoolSettings_TCPSettings{
					MaxConnections: 7,
					ConnectTimeout: &durationpb.Duration{Seconds: 2},
				},
				Http: &networking.ConnectionPoolSettings_HTTPSettings{
					Http1MaxPendingRequests:  2,
					Http2MaxRequests:         11,
					MaxRequestsPerConnection: 5,
					MaxRetries:               4,
					IdleTimeout:              &durationpb.Duration{Seconds: 30},
					MaxConcurrentStreams:     5,
				},
			},
			valid: true,
		},
	}
}
```

## 總結

抓抓頭發現，好像有點複雜了，這些設定在大部分的狀況下還是不太會使用到啦，但主要是我們要在需要的時候發會用場，透過網路上的 source code 可以讓我們更了解設計者的思考邏輯和運作，以前很多前輩都說：『要找文件？去看測試』，就來發揮這個用處啦，當然要去限制到我們的 sidecar 還是有很多種方法，但今天就大概介紹到這吧，再抓頭頭髮都抓沒了ＱＱ

## Reference

https://istio.io/latest/docs/ops/deployment/performance-and-scalability/

https://github.com/projectcalico/calico/blob/master/manifests/alp/istio-inject-configmap-1.6.yaml

https://github.com/istio/istio/blob/master/pkg/config/validation/validation_test.go
