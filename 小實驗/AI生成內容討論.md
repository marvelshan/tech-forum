## K8S Lab Day_38

# Study1-內容討論

## 前言

今天意外在看書的時候再胡思亂想，在想說我未來的職涯發展，因為我自己的最終極目標是可以遠端工作，並且可以很熟練，因為我覺得遠端工作是一件很神聖的事情，不只是要很自律，還要對於交辦的事物還有各個同事之間溝通的狀況要很熟練才能達到，所以在這之前我要怎麼讓我達到這件事，就是去做開源！其實之前就有貢獻開源專案的經驗，但是感覺有點像是體驗開源，而不是享受開源，要享受勢必要對這個工具有一定的了解和實作，剛好我最近又在寫雲服務和微服務這塊的資訊，所以我決定要來學 golang 順便用一些小實驗的方式把 k8s 和 istio 把它搭建起來，也只是構想不知道自己可不可以達到，所以就開啟了這個新主題！

# 用 AI 從頭開始學習開源專案

透過 AI 輔助，以 SRE/DevOps 工程師視角，從 Golang 基礎開始，逐步學習 Kubernetes 和 Istio 的核心概念與實作。每一篇文章包含一個小實驗，模擬這些開源專案的核心功能，從簡單到複雜，逐步搭建完整架構，並融入 SRE/DevOps 的實務觀點（如可觀測性、可靠性設計）

## 為什麼選擇 Golang？

Go 在雲原生時代中可說是一代新星，主要是他的語法簡潔，編譯速度快，執行效率接近於 C/C++，他的 goroutines 和 channels 讓 code 變得更直觀和安全，對於像是 k8s controller 是相當的重要，他也有相對應的 library 像是 `net/http` 等等的好處

## 小實驗：用 Go 寫一個簡單的 HTTP 伺服器

但是首先我們還是要先把環境管理好，我們要用到的是 nix 的環境版本控制

```nix
# vi flake.nix
{
  description = "Golang Dev Environment for Kubernetes & Istio Learning";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };

        goTools = with pkgs; [
          go_1_24  # 最新 Go 版本
          golangci-lint  # Go linting 工具
          git  # 版本控制
          curl  # HTTP 測試
          jq  # JSON 處理（未來解析 K8s API）
        ];
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = goTools;

          shellHook = ''
            echo "🚀 Golang Dev Environment Activated! 🚀"
            echo "Go version: $(go version)"
          '';
        };
      }
    );
}
```

## 實驗一

這邊用一個簡單建立一個基本的 http server，利用 `http.HandleFunc` 來註冊 router，在 `/` 這隻 api 會回覆 `"Hello, Kubernetes and Istio!"`，`http.ListenAndServe(":8080", nil)` 會在 local 啟動一個 port 8080 的 server，這裏主要是使用了 go 的 anonymous function `func(w, r)` 和內建的 library `net/http`

```go
// vi simple_server.go
package main

import (
    "fmt"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, Kubernetes and Istio!")
    })

    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, "Server is healthy")
    })

    fmt.Println("Server starting on :8080...")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        fmt.Println("Server failed:", err)
    }
}
```

將程式跑起來，然後 curl 看看可不可以得到回覆

```bash
go run simple_server.go
```

## 實驗二

以下這段 code 是使用 goroutine 同時啟動兩個 http server，`func startServer` 用來建立新的 `http.ServeMux`，在 main 中，使用 `go startServer` 讓每個 server 在獨立的 goroutine 執行，也就是非同步的運行，`select {}` 是一個 block

```go
// vi multi_server.go
package main

import (
	"fmt"
	"net/http"
	"time"
)

func startServer(name, addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s: Hello from %s!\n", time.Now().Format(time.RFC3339), name)
	})
	fmt.Printf("[%s] starting on %s\n", name, addr)
	http.ListenAndServe(addr, mux)
}

func main() {
	go startServer("api", ":8080")
	go startServer("metrics", ":9090")

	select {} // block forever
}
```

### select

這邊要來介紹 select，這是 go 專門用於 channel 的語法結構，但這邊是要讓 main goroutine 永久 blocking 不退出程式，那假如不使用 select 呢？`main()` 就會不阻塞，繼續往下跑完，程式就會直接結束，而通常在 `select {}` 裡面會放一些 `case` 的操作，但是這邊的是 `沒有任何 case 的 select`，所以會 block forever，不會消耗 CPU 也不會退出，所以才可以確保 goroutine 持續的運作

那再回頭想想，為何第一段 code 不需要 `select`，因為在 `http.ListenAndServe(":8080", nil)` 本身會 block goroutine，每次有請求的時候會啟動新的 goroutine 去處理，對比這段程式，這邊使用的 `http.ListenAndServe` 是被包在 goroutine 裡面，因為 goroutine 是非 block 的，所以只要 main() 不被 block 他就會直接結束

### mux

接著我們這邊要看到 mux，這邊主要是要建立一個 multiplexer 的 router，用來管理同一個 http server 裡面有多個 endpoint，這邊也可以看到 `http.ServeMux` 是 go 的一個 library 提供的 http request router，可以將不同的 URL path mapping 到不同的 handler function

那我們就可以繼續看到為何要用到 mux，如果直接使用到 `http.HandleFunc` 多個 server 會共用同一個 router 會產生衝突，所以要使用到 `NewServeMux()` 讓每個 server 都可以有自己的 router

## 實驗三

channel 是 go 一個很重要的一個角色，在 k8s controller，controller-runtime 也大量的使用到 channel，在這邊每個 worker 會從 jobs channel 接收工作的編號，處理後將結果寫入 results channel，這裏的 jobs 和 results 都是 channel (`make(chan Type, buffer)`)，分別用於傳入任務和接收結果，其中的 `<-chan` 表示只讀，`chan<-` 表示只寫，在 main() 中建立三個 worker goroutine 並將 5 個工作放入 jobs channel，最後關閉 channel，然後再將其印出

```go
// vi channel_demo.go
package main

import (
	"fmt"
	"time"
)

func worker(id int, jobs <-chan int, results chan<- string) {
	for j := range jobs {
		time.Sleep(500 * time.Millisecond)
		results <- fmt.Sprintf("Worker %d finished job %d", id, j)
	}
}

func main() {
	jobs := make(chan int, 5)
	results := make(chan string, 5)

	for w := 1; w <= 3; w++ {
		go worker(w, jobs, results)
	}

	for j := 1; j <= 5; j++ {
		jobs <- j
	}
	close(jobs)

	for i := 1; i <= 5; i++ {
		fmt.Println(<-results)
	}
}
```

### Channel

在這邊的 `func worker(id int, jobs <-chan int, results chan<- string)` 可以看到 channel 的方向限制，`<-chan int` 只讀 channel，`chan<- string` 只寫 channel，然後後面有 `for j := range jobs { ... }` 利用 for range loop 來讀取 channel，直到 channel 被關閉，而這邊的 `close(jobs)` 就是關閉 channel 表示沒有更多的值需要傳，然後 `range` loop 就會自動結束

那我就想了又想，為何需要 channel 呢？在我們聽到 go 的時候，就是可以處理高併發的狀況，那所謂的高併發會有什麼問題呢？也就是 Race condition 還有需要有同步的機制，而這個 channel 就可以更安全的去交換資料

那再繼續往下想，剛剛有建立 buffer，發送者可以在 buffer 未滿的情況下 non-blocking 的傳送資料，那假如 buffer 已經滿了呢？發送者就會被 block 直到 receiver 拿走資料，這就是 backpressure，如果沒有適當的處理會有 deadlock 的狀況導致資料遺失

```go
// vi buffer_block_demo.go
package main

import (
	"fmt"
	"time"
)

func main() {
	ch := make(chan int, 2) // 緩衝區大小 2

	// 發送者 goroutine
	go func() {
		for i := 1; i <= 5; i++ {
			fmt.Println("Sending", i)
			ch <- i // 當 buffer 滿時會阻塞
			fmt.Println("Sent", i)
		}
		fmt.Println("Sender done")
	}()

	// 接收者延遲啟動
	time.Sleep(3 * time.Second)

	go func() {
		for v := range ch {
			fmt.Println("Received", v)
			time.Sleep(1 * time.Second)
		}
	}()

	// 等待
	time.Sleep(10 * time.Second)
}
```

這邊可以看到 channel buffer 只有 2 個容量，第一次會送出 1 和 2 不會 block，但送到第 3 個的時候 sender 會被 block，直到 receiver 讀取資料，因為這邊設定 receiver 會在 3 秒後開始讀，所以發送者在送第三個資料前會停住，這邊可以了解到 buffered channel 可以緩衝資料，但不是無限大，所以在高併發的系統中需要搭配適當的 buffer size，non-block 的 send(select + default)，會是用 goroutine + queue 的方式來避免 deadlock

接著再想想為何 go channels 要限制 buffer size，在 stack overflow 就有提到 back-pressure 的狀況，因為機器的資源有限，所以還是要考慮到真正需要的 buffer 和系統能 afford 的 workload，如果真的需要非常大量的資料，理論上可以建立超大 channel，但大多數的狀況，0 或 1 的 buffer 就夠了。並且這頁可以讓系統自然的 block 生產者，避免不必要的記憶體浪費，所以可以把 channel buffer 想成不只是資料的暫存，也是一個控制系統穩定性與併發速度的機制

## 小結

開始了一個新的系列，也開始學新的東西，透過這樣簡單的小實驗了解其中的機制也相當的有趣呢！比起一個小 topic 一直在撿來撿去好像有點零碎，接著我會將我想要學的東西一點一點的加入到這個主題當中，所以假如看到這個主題中有一些很突兀的工具和內容出現不要覺得太奇怪，因為我突然想到 XD

## Reference

https://stackoverflow.com/questions/41906146/why-go-channels-limit-the-buffer-size

https://go.dev/ref/spec#Channel_types
