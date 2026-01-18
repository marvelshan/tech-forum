## K8S Lab Day_25

# Istio WASM Plugin 戰術：撰寫與掛載自訂過濾器

## 前言

昨天我們了解一些查詢 service mesh 流量運作的狀況，這時候我就很好奇啦，因為 sidecar 能設定的東西就那些，假如我要做到更多底層的變化，有些不是我想要的設定，我應該怎麼克制化我的 sidecar 去滿足一些特別的需求（無理？），沒錯！我們這時候就可以使用到現在當下蠻有名的 WASM，來去做到這些操作～

## WASM Plugin

Istio 自 1.12 起正式支援 WebAssembly 作為 extensibility 的機制，讓我們可以不在 rebuild Envoy 的情況下動態的 inject 我們所需要的邏輯，在 istio 中，WASM Plugin 是以 Envoy Filter 的形式動態載入的，envoy proxy 在執行的期間可以去 inject `.wasm` 的 model，並根據配置來決定掛在 Inbound、Outbound、Listener、HTTP Filter

但我們在回頭想想，為何要使用 WASM 呢？因為 envoy 是用 C++ 寫的，假如我們要去修改他的話，我們是必要使用 C++ 來去改他，原本是使用 Rust、Go 等等的開發者就被限制了。另外，假如我們直接去更動原生的 envoy，假如在我們不經意的地方有 bug 導致服務 crash，反而得不償失，所以我們需要 WASM 去做到 Isolation，並且 WASM 他有很好的優勢是他的 model 是很輕量的二進位檔，可以動態的去 update 它，不需要影響到 envoy

![wasm plugin](https://github.com/user-attachments/assets/e351d121-7b81-44a0-81e1-62c0b5c662d2)

首先我們要先去 flake.nix 把我們所需的環境安裝起來

```nix
{
  description = "Kubespray environment with Nix";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
    in
    {
      devShells.${system} = {
        kubespray = pkgs.mkShell {
          buildInputs = with pkgs; [
            ansible_2_16
            python3
            python3Packages.pip
            python3Packages.netaddr
            python3Packages.jmespath
            kubectl
            kubernetes-helm
            istioctl
            jq
          ];
          shellHook = ''
            echo "Kubespray environment with Nix is ready!"
            export ANSIBLE_CONFIG=$PWD/ansible.cfg
            export KUBECONFIG=$HOME/.kube/config
          '';
        };

        wasm = pkgs.mkShell {
          buildInputs = with pkgs; [
            rustc
            cargo
            lld
            llvmPackages.clang
          ];
          shellHook = ''
            echo "WASM environment is ready!"
          '';
        };
      };
    };
}
```

我們需要先把我們環境先 setting 好，再來就是建立我們所需要的 plugin 了

```bash
nix develop .#wasm
```

```bash
cargo new istio-wasm-demo --lib
cd istio-wasm-demo
```

```toml
[lib]
crate-type = ["cdylib"]

[dependencies]
proxy-wasm = "0.2"
```

```rs
// src/lib.rs
use proxy_wasm::traits::*;
use proxy_wasm::types::*;
use proxy_wasm::hostcalls;

struct HeaderLogger;

impl Context for HeaderLogger {}

impl HttpContext for HeaderLogger {
    fn on_http_request_headers(&mut self, _: usize, _: bool) -> Action {
        if let Some(value) = self.get_http_request_header("user-agent") {
            hostcalls::log(LogLevel::Info, &format!("User-Agent: {}", value)).unwrap_or(());
        }
        Action::Continue
    }
}

impl RootContext for HeaderLogger {
    fn on_configure(&mut self, _: usize) -> bool {
        hostcalls::log(LogLevel::Info, "HeaderLogger configured").unwrap_or(());
        true
    }
}

#[unsafe(no_mangle)]
pub fn _start() {
    proxy_wasm::set_root_context(|_| Box::new(HeaderLogger));
}
```

我們這邊就寫了一個簡單的 pluggin，每當有 HTTP Request 進入時，會在 Envoy log 中記錄 User-Agent，編譯完我們就會得到 `target/wasm32-unknown-unknown/release/istio_wasm_demo.wasm`

```bash
rustup target add wasm32-unknown-unknown
cargo build --release --target wasm32-unknown-unknown
```

接下來我們就需要部署到 istio 上面了～我們要先確保已經啟動了 `wasmExtensions`，並建立 ConfigMap 儲存 wasm，然後完成 `WasmPlugin`，然後利用 Volume 掛載 ConfigMap 到 Envoy 容器內

```bash
kubectl get crd wasmplugins.extensions.istio.io
```

```bash
NAME                              CREATED AT
wasmplugins.extensions.istio.io   2025-09-30T07:38:51Z
```

```bash
kubectl create configmap header-logger \
  --from-file=istio_wasm_demo.wasm=target/wasm32-unknown-unknown/release/istio_wasm_demo.wasm \
  -n istio-system
```

```yaml
apiVersion: extensions.istio.io/v1alpha1
kind: WasmPlugin
metadata:
  name: header-logger
  namespace: default
spec:
  url: file:///etc/wasm/header-logger/istio_wasm_demo.wasm
  phase: AUTHZ
  pluginConfig:
    log_level: info
  selector:
    matchLabels:
      app: productpage
```

```bash
vi istio/samples/bookinfo/platform/kube/bookinfo.yaml
```

```yaml
volumeMounts:
  - name: wasm-filters
    mountPath: /etc/wasm/header-logger
volumes:
  - name: wasm-filters
    configMap:
      name: header-logger
```

然後就可以試試看有沒有成功了～

```bash
kubectl exec -it <pod> -c istio-proxy -- curl -s -H "User-Agent: MyTestClient" http://productpage:9080/productpage
```

```bash
kubectl logs <pod> -c istio-proxy | grep User-Agent
```

```bash
istioctl proxy-config listener <pod> -n default
```

## 總結

透過以上的操作是不是又更接近 istio master 一點呀，但我不是 QQ，但是會了 plugin 好像又可以多解決一些問題了，好像不是好事 XD

## Reference

https://www.alibabacloud.com/help/tc/asm/sidecar/developing-a-wasm-plugin-for-grid-agents-using-rust

https://cloud.tencent.com/developer/article/2369728

https://konghq.com/blog/engineering/proxy-wasm

https://istio.io/latest/docs/concepts/wasm/

https://istio.io/latest/docs/reference/config/proxy_extensions/wasm-plugin/

```
istioctl proxy-config listener productpage-v1-54bb874995-77p7m -n default
```
