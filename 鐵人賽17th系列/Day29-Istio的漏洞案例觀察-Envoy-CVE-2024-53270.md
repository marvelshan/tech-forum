## K8S Lab Day_31

# Day_29: Istio 的漏洞案例觀察：Envoy CVE-2024-53270

## 前言

昨天提到 SMI 的 interface，了解了 service mesh 過去的歷史，再來要來個實戰，在使用這些工具多多少少一定會經歷到一些還有被發現的問題，接著我們要來透過這個 Envoy 的這個案例來了解一下～

## 漏洞簡述

這個漏洞的編號是 CVE-2024-53270，這個 CVE 的意思是 Common Vulnerabilities and Exposures，對於漏洞有一個評分 - Common Vulnerability Scoring System, CVSS，它是由美國國家基礎建設諮詢委員會經過一系列的變更和演化最後發展出這套漏洞的標準，評分是 0.0-10.0 分數越高危險程度越高，假如有興趣可以去看一下他的文件，這個漏洞的評分是 7.5，其實算是一個蠻高的評分，這個是發生在配置了某個 load-shed 點（`http1_server_abort_dispatch`）時，如果 downstream 在某個時刻被 reset，而 Envoy 在處理 overload 時假設 active request 一定存在，可能會 dereference 空指標導致進程 crash。這個的修補在 1.29.12 / 1.30.9 / 1.31.5 / 1.32.3（以及之後版本），使用這些或更新的 Envoy 版本可完全解決。若不能立即升級，官方建議暫時**禁用 `http1_server_abort_dispatch` load-shed 點**或把對應 threshold 設很高作為應急方案

![漏洞](https://github.com/user-attachments/assets/9cd690a1-cb94-45df-a711-d95be05a4708)

## 技術背景

Envoy 在 high memory/conn 情況下會啟用 _overload manager_，可以在不同生命週期（例如接收 header、建立連線、dispatch 請求）放置「load-shed points」來決定哪些請求或連線要被放棄或回覆本地錯誤，以保護系統不中斷服務。但如果程式在某個特殊 race 條件下（例如 downstream 剛好 reset 了 stream）呼叫了假設「active request 一定存在」的 code path，就會出現 null pointer 並導致 envoy 進程當掉。CVE-2024-53270 就是在 `http1_server_abort_dispatch` 這個 load-shed 點的處理路徑出現了這種假設，當 downstream reset 與 upstream H/2 reset 發生交互時就會觸發崩潰

## 為什麼 Istio 使用時要在意？

istio 的 data plane 就是 Envoy，因此 Envoy 的任何 crash 都會直接影響 Pod 或整個 ingress/egress gateway 的可用性，可能導致 Pod 被 kubelet 重啟或流量短暫中斷，若你在集群中有自訂 bootstrap / EnvoyFilter 去啟用 Overload Manager 或特定 load-shed 點（例如針對 ingress 做 edge overload 設定），你更可能受影響。Istio 官方在安全公告中也特別提醒：若使用自訂 EnvoyFilter 啟用了 Overload manager，避免使用 `http1_server_abort_dispatch`

前面聽的霧煞煞，那我們要再來講細一點，`http1_server_abort_dispatch` 是一個 Envoy Load Shed Point， 它允許 Envoy 在偵測到系統資源（例如記憶體、CPU 或您自定義的監控指標）達到高 loading 時，主動對新進的 HTTP/1.1 請求採取動作，用來 deny 或 abort 它們，是為了服務還未過載到完全 crash 之前就開始拒絕部分流量，使用他的核心原因是要 Graceful Degradation，來防止 Cascading Failure，主要應用場景是對於流量尖峰保護、記憶體限制環境防止 OOM、依賴故障隔離，可以看到下面的 integration test，它的主要目的是驗證當 Envoy 處於高負載或資源耗盡狀態時，能否正確地使用特定的負載卸載點來 shed 新的 HTTP/1 請求，從而保護系統不至於崩潰

```c++
TEST_P(LoadShedPointIntegrationTest, Http1ServerDispatchAbortShedsLoadWhenNewRequest) {
  // Test only applies to HTTP1.
  if (downstreamProtocol() != Http::CodecClient::Type::HTTP1) {
   // 確保只在 HTTP/1 協議模式下運行此測試，因為這個 Load Shed Point 專門針對 HTTP/1 處理
    return;
  }
  autonomous_upstream_ = true; // 設置上游為 autonomous 模式 (例如不等待 Envoy 主動發送數據)

   // 1. 設定過載管理器與 http1_server_abort_dispatch 的 Load Shed Point
  initializeOverloadManager(
      TestUtility::parseYaml<envoy::config::overload::v3::LoadShedPoint>(R"EOF(
      name: "envoy.load_shed_points.http1_server_abort_dispatch"
      triggers: // 設置觸發負載卸載的條件
        - name: "envoy.resource_monitors.testonly.fake_resource_monitor" // 使用一個模擬的資源監控器
          threshold:
            value: 0.90 // 設定閾值為 90%。當資源使用率超過此值時，即判定為過載
    )EOF"));
  test_server_->waitForCounterEq("http.config_test.downstream_rq_overload_close", 0);
  // 檢查計數器，確認目前因為過載而被關閉的 Downstream Request 數量為 0

  // 2. 模擬過載狀態並發送請求（預期被拒絕）

  // Put envoy in overloaded state and check that the dispatch fails.
  updateResource(0.95); // 手動將模擬資源使用率設置為 95%，超過 90% 的閾值
  test_server_->waitForGaugeEq(
      "overload.envoy.load_shed_points.http1_server_abort_dispatch.scale_percent", 100);
      // 等待 Metric 確認過載管理器的 Scale Percent 達到 100%，表示完全啟用負載卸載

  codec_client_ = makeHttpConnection(makeClientConnection((lookupPort("http"))));
  auto [encoder, decoder] = codec_client_->startRequest(default_request_headers_);

  // We should get rejected local reply and connection close.
  test_server_->waitForCounterEq("http.config_test.downstream_rq_overload_close", 1);
  // 檢查計數器，確認因過載而被關閉的請求數量變為 1 (即剛才發送的請求)
  ASSERT_TRUE(decoder->waitForEndStream());
  EXPECT_EQ(decoder->headers().getStatusValue(), "500");
  // 檢查 http code，預期收到 500 Internal Server Error
  ASSERT_TRUE(codec_client_->waitForDisconnect());
  // 驗證連線因為過載而被 Envoy 關閉

  // 3. 禁用過載狀態並發送請求（預期成功）

  // Disable overload, we should allow connections.
  updateResource(0.80); // 手動將模擬資源使用率設置為 80%，低於 90% 的閾值，解除過載狀態
  test_server_->waitForGaugeEq(
      "overload.envoy.load_shed_points.http1_server_abort_dispatch.scale_percent", 0);
   // 等待 Metric 確認負載卸載的 Scale Percent 降為 0，表示機制禁用
  codec_client_ = makeHttpConnection(makeClientConnection((lookupPort("http"))));
  // 建立第二個新的 HTTP 連線
  auto response = codec_client_->makeHeaderOnlyRequest(default_request_headers_);
  // 發送第二個新的 HTTP 請求
  ASSERT_TRUE(response->waitForEndStream());
  EXPECT_EQ(response->headers().getStatusValue(), "200");
}
```

這段我們就可以看到 Load Shedding 在 HTTP/1 協議下，如何使用 http1_server_abort_dispatch 負載卸載點來拒絕新請求

---

## 回頭來檢查版本，看看現在是不是有遇到

1. 檢查 Istio / Envoy 版本（先確認是否受影響）

```bash
# 檢查 Istio control plane / ingress pods
kubectl get pods -n istio-system

# 在一個 sidecar pod 或 ingressgateway pod 裡查 envoy 版本
kubectl exec -n <namespace> -it <pod-name> -c istio-proxy -- envoy --version

# 或針對 ingressgateway（有些安裝會把 ingress 放 istio-system）
kubectl exec -n istio-system -it $(kubectl get pod -n istio-system -l app=istio-ingressgateway -o jsonpath='{.items[0].metadata.name}') -c istio-proxy -- envoy --version

# export:
envoy  version: 1a53bf14a57976dd0f509752b748caf3b1125d54/1.35.2-dev/Clean/RELEASE/BoringSSL

# 這邊就可以看到版本是 1.35.2 已經是漏洞之後的版本了
```

2. 檢查是否有自訂的 Overload / EnvoyFilter 啟用該 load-shed

```bash
# 列出所有 EnvoyFilter（檢查是否有啟用 overload manager/自訂 bootstrap）
kubectl get envoyfilter -A

# 針對疑似有自訂 bootstrap 的 filter 做 describe / grep 查詢
kubectl describe envoyfilter -n <ns> <filter-name> | sed -n '1,200p'
kubectl get configmap -n istio-system | grep bootstrap
```

若有看到自訂 bootstrap 或 `overload_manager`、`load_shed_points` 的設定，必須要看是否包含 `http1_server_abort_dispatch`，Istio 的公告也指出，如果你建立了自訂 EnvoyFilter 去啟用 Overload manager，避免使用 `http1_server_abort_dispatch`

3. 監控與日誌

當 envoy 當掉時， kubelet 會重啟 sidecar 或整個 Pod，觀察 Pod 重啟次數，`kubectl get pods -n <ns> -o wide` 或 `kubectl describe pod <pod>` 看 RestartCount，在 envoy 日誌中搜索 crash/backtrace 或 segfault，如果在 ingress/edge 出現間歇性 DoS（請求突然 5xx 或 pod 重啟），也要提高警覺

## 緩解與修補

官方建議的優先順序，不管就是升級就對了！但還是要確認升級的版本有沒有更新，但一定有目前正在使用但又好像沒什麼問題的時候，就要避免使用 / 移除 `http1_server_abort_dispatch` load-shed（不要在自訂 bootstrap 或 EnvoyFilter 中啟用它），或將對應的閾值設得很高，使 load-shed 幾乎不會觸發（但這可能削弱 overload manager 的保護效果），也可以在 ingress 邊界做好 rate-limiting、WAF 規則或 layer-7 防護，減少能觸發特殊 race 的外部流量模式，建立監控告警（envoy 重啟、sidecar crash、5xx ratio 顯著上升）以便快速回應等等，但這種有問題的還是儘速的更新最妥

## 總結

今天來講到這個漏洞的地方，我自己是覺得很酷，其實我們使用的工具一直都會有這種小細節是我們需要觀察的，我們可以透過這種過去的案例來觀察出假如系統出問題，我們可以先從哪邊開始排查，但是問題千千萬萬種，一定有我們不知道的，所以多看多了解一定會變強的XD，或是把這種漏洞當故事書看也蠻有趣的，看看為何這種漏洞會出現，然後來案例分析也是不錯的體驗～～然後下面是這個漏洞修正的 github，有興趣也可以看一下他是怎麼修正的～

![漏洞修正](https://github.com/user-attachments/assets/ea207e8d-9c9c-4884-b525-2d3c396dbbbc "漏洞修正 github")

## Reference

[Envoy Security Advisory (GHSA / GitHub Advisory) — CVE-2024-53270 details & mitigation.](https://github.com/envoyproxy/envoy/security/advisories/GHSA-q9qv-8j52-77p3?utm_source=chatgpt.com)

[NVD (CVE-2024-53270) — 技術描述與修補版本](https://nvd.nist.gov/vuln/detail/CVE-2024-53270?utm_source=chatgpt.com)

[Istio Security Bulletin（ISTIO-SECURITY-2024-007）— 列出 Envoy CVE 與對 Istio 使用者的建議（避免在自訂 EnvoyFilter 使用 `http1_server_abort_dispatch` 等）](https://istio.io/latest/news/security/istio-security-2024-007/?utm_source=chatgpt.com)

[Envoy docs: Overload Manager / Runtime 設定（解釋 load_shed_points 與 runtime 覆寫概念）](https://www.envoyproxy.io/docs/envoy/latest/configuration/operations/overload_manager/overload_manager?utm_source=chatgpt.com)

[Common Vulnerability Scoring System version 4.0](https://www.first.org/cvss/specification-document)

[EnvoyFilter](https://jimmysong.io/book/envoy-made-simple/service-mesh/envoy-filter/)
