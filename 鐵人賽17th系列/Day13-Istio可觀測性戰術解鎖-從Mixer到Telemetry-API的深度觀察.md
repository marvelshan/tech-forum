## K8S Lab Day_15

# Day13: Istio 可觀測性戰術解鎖，從 Mixer 到 Telemetry API 的深度觀察

## 前言

之前大概也把流量的操作講完了，今天就來進入到我覺得 service mesh 蠻厲害的部分了，就是可觀測性，它能夠深入了解微服務之間的互動、流量模式、錯誤率以及延遲情況，這不只是「看得見」流量，更可以用來分析、排錯、優化服務～

## Observability on Istio

講到 observability 就要提到曾經出現過的組件 Mixer，他主要會從 Envoy sidecar 收集各種流量資，會根據配置好的策略對流量進行控制，但是在 Istio 1.5 以前，Mixer 是核心，負責所有 telemetry 與 policy，Envoy sidecar 主要做資料收集，真正的策略判斷和資料處理在 Mixer，但後來 Istio 逐步移除 Mixer（deprecate），引入 Telemetry v2 / WASM-based extensions，遙測和策略處理直接在 Envoy sidecar 內部完成，不再依賴 Mixer，因此也提升了效能、降低延遲，並且 Telemetry API 可以取代 Mixer 配置自訂 metrics、tag、過濾規則等，這邊做了一個比較

![mixer](https://github.com/user-attachments/assets/ffe9a54d-49a0-4b4b-8997-d4147fa308c6)

![Telemetry v2](https://github.com/user-attachments/assets/7c087702-79ca-4879-bc17-d245b156e810)

| 功能                | Mixer                      | Telemetry v2 / Envoy        |
| ------------------- | -------------------------- | --------------------------- |
| 收集指標            | Envoy → Mixer → Prometheus | Envoy → Prometheus          |
| Tag 自訂 / 指標修改 | Mixer 配置                 | Telemetry API 配置          |
| Rate Limit / Policy | Mixer 做決策               | Envoy WASM / Envoy 原生實作 |
| 性能                | 存在額外網路開銷           | 直接在 Envoy sidecar 處理   |

### Telemetry API

這裏分成了三種 configuration 層級 Scope, Inheritance, and Overrides，Telemetry API 的資源可以繼承 Istio 配置層級中的設定，Scope 這邊氛圍三種層級：root configuration namespace 通常是 istio-system，提供整個 mesh 配置，amespace-scoped resource 很明顯是針對 ns 的，workload-scoped resource 就是針對 selector 的 workload，我這邊理解為某個 pod 或 deployment。Inheritance 我覺得就是 OOP 的概念，子層級會「繼承」父層級的設定，除非被覆寫，像是 Workload 繼承 Namespace 設定，Namespace 繼承 Mesh-wide 設定等等。Overrides 這邊也是，子層級可以「覆寫」父層級的設定，完全替換對應欄位，Workload 的設定覆寫 Namespace 的同欄位設定等等的

另外，telemetry 的 provide 也可以客製化更改，像是官方文件有提到的，像是這裡就是指定 zipkin 把追蹤資料送到你自己在 cluster 裡部署的 Zipkin collector，下一個 provide 是用 Stackdriver 來把追蹤資料送到 Google Cloud Stackdriver

```yaml
data:
  mesh: |-
    extensionProviders: # The following content defines two example tracing providers.
    - name: "localtrace"
      zipkin:
        service: "zipkin.istio-system.svc.cluster.local"
        port: 9411
        maxTagLength: 56
    - name: "cloudtrace"
      stackdriver:
        maxTagLength: 256
```

接著來介紹各個 scope 配置的行為

1. mesh-wide behavior

設定 mesh-wide 的 tracing provider 為 localtrace，取樣率 100%，並加上自訂 tag foo: bar

```yaml
apiVersion: telemetry.istio.io/v1
kind: Telemetry
metadata:
  name: mesh-default
  namespace: istio-system
spec:
  tracing:
    - providers:
        - name: localtrace
      customTags:
        foo:
          literal:
            value: bar
      randomSamplingPercentage: 100
```

2. namespace-scoped tracing behavior

覆寫 mesh-wide 設定，只針對 myapp namespace，取樣率維持 100%，provider 使用 localtrace，但自訂 tag 改為取 userId header 的值

```yaml
apiVersion: telemetry.istio.io/v1
kind: Telemetry
metadata:
  name: namespace-override
  namespace: myapp
spec:
  tracing:
    - customTags:
        userId:
          header:
            name: userId
            defaultValue: unknown
```

3. workload-specific behavior

對 frontend workload 停用 tracing，但仍會傳遞 tracing headers，只是不報 span 到 provider

```yaml
apiVersion: telemetry.istio.io/v1
kind: Telemetry
metadata:
  name: workload-override
  namespace: myapp
spec:
  selector:
    matchLabels:
      service.istio.io/canonical-name: frontend
  tracing:
    - disableSpanReporting: true
```

## 總結

今天主要介紹的是 telemtry API 針對於可觀測性資料收集的簡介，必須要了解了這些才會對整個收集的概念更加清楚，後面幾天會針對於 metrics，logs，trace 做更細微的操作

## Reference

https://istio.io/latest/docs/tasks/observability/telemetry/

https://outshift.cisco.com/blog/istio-mixerless-telemetry
