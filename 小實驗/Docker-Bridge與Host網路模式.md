## K8S Lab Day_37

# Docker 網路實戰：從容器互聯到 Bridge 與 Host 模式解析

## 前言

昨天在 linux 的練工坊讓自己好像馬步紮了比較穩了，但是還是要常常練習把他刻在腦袋裡面，可以時不時就 nslookup、dig、netstat 一下，多敲一下 google.com 或 yahoo.tw 來練習一下 XD，話說從 istio 到現在的 linux 排解，很大一部分都是網路的問題，其實在容器和我們所使用的軟體的世界很大一部分都是跟網路有關，像是最近 XD 就是最近，AWS 在前天大當機，好像是 Dynamo DB 的 API 的 DNS 解析失敗，造成一系列的 Cascade 效應，導致一堆服務死光光，而且還停機停超久，超級 GG，所以這也告訴我們網路是相當重要的一個角色，今天我們要來了解一下 docker 裡面網路的，也就是容器之間的連接

## 為什麼需要容器網路？

在 Docker 的世界裡，容器就像獨立的虛擬環境，但它們往往需要互相溝通。例如，一個 Web 應用容器可能需要連線到資料庫容器來存取資料。Docker 提供了多種方式來處理這件事，包括 Port Mapping 和 Linking。Port Mapping 是將容器內的端口暴露到宿主機上，讓外部存取；但 Linking 則是另一種更安全的內部互動方式，它在容器間建立隧道，讓接收端容器能看到來源端容器的特定資訊，而不用暴露到外部網路

## 容器之間的連線 (Linking)

在早期的 Docker 版本中，若要讓兩個容器互相溝通，官方提供的方式是使用 --link 參數建立「容器連線」

當運行 `docker run` 時，系統會自動分配一個隨機名稱（如 ospwn3y803hc），但這不好記。自訂名稱有兩個好處，一是更容易記住，例如將 Web 應用容器命名為 web、再來是作為連線其他容器的參考點，例如連線 web 到 db

```bash
sudo docker run -d -P --name web practice/webapp python app.py
```

```bash
# 檢查命名
sudo docker ps -l
CONTAINER ID  IMAGE                  COMMAND        CREATED       STATUS       PORTS                    NAMES
ospwn3y803hc  practice/webapp:latest python app.py  12 hours ago  Up 2 seconds 0.0.0.0:49154->5000/tcp  web
```

```bash
# 或用 docker inspect
sudo docker inspect -f "{{ .Name }}" ospwn3y803hc
/web
```

再來就是要連接了，使用 `--link`

```bash
sudo docker run -d --name db practice/postgres
```

```bash
# 再建立一個 web 容器，並讓它連線到 db
sudo docker run -d -P --name web --link db:db practice/webapp python app.py
```

```bash
# 檢查連線
docker ps
CONTAINER ID  IMAGE                     COMMAND               CREATED             STATUS             PORTS                    NAMES
349169744e49  training/postgres:latest  su postgres -c '/usr  About a minute ago  Up About a minute  5432/tcp                 db, web/db
aed84ee21bde  training/webapp:latest    python app.py         16 hours ago        Up 2 minutes       0.0.0.0:49154->5000/tcp  web
```

這裏可以看到 db 的 NAMES 欄有 web/db，表示連線成功。Linking 建立安全隧道，不需映射端口到宿主機，避免暴露資料庫到外部

## Docker Network drivers

Docker 的網路系統是 pluggable 的，也就是說它透過不同的 Network Drivers 來提供多樣化的網路模式與功能

### Bridge 預設網路驅動

如果在建立容器時沒有特別指定網路，Docker 就會使用 Bridge 模式。這是最常見的網路模式，適合容器之間在同一台主機上互相通訊的場景，容器有自己的私有 IP，對外連線透過宿主 NAT，通常是 Web 應用 + 資料庫在同一台主機上

<img width="1154" height="1076" alt="image" src="https://github.com/user-attachments/assets/e6ac3aa0-8a0e-41c7-82ca-76370150063e" />

### Host 共享宿主的網路堆疊

使用 Host 模式時，容器與宿主主機共用相同的網路環境，容器不再擁有獨立的 IP，而是直接使用宿主的網卡與 port，優點是效能佳、延遲低，但缺點是無網路隔離，可能 port 衝突，通常應用在需要監控代理、metrics exporter

### Overlay 多主機通訊的基礎

Overlay 網路用於連接多個 Docker Daemon 讓容器或 Swarm 服務能夠跨節點通訊，而不需設定 OS 層級的路由，主要使用在 Swarm、k8s，優點是簡化多節點部署，場景是建立微服務集群時

<img width="573" height="299" alt="image" src="https://github.com/user-attachments/assets/9e9cc771-fe41-408a-b674-47958b334778" />

### IPvlan 控制 IP 與 VLAN 的進階方案

IPvlan 提供使用者對 IPv4/IPv6 網址的完整控制，VLAN 模式則可進一步實現 Layer 2 VLAN 標記與 L3 路由，適合需要與底層實體網路深度整合的場景，場景是在大型企業網路、資料中心

### Macvlan 讓容器看起來像實體機器

Macvlan 允許你給每個容器分配獨立的 MAC 位址，讓容器在網路上看起來就像獨立的實體主機，Docker 會根據 MAC 位址來路由封包，主要應用在遷移 legacy app 或需要實體化網路識別的應用

<img width="1030" height="1048" alt="image" src="https://github.com/user-attachments/assets/f2ea19c5-0192-4cb9-a2d9-1d5ef70b51ff" />

### None 完全隔離的網路模式

none 模式會將容器完全隔離，不與宿主或其他容器連線，此模式不適用於 Swarm 服務，主要用途是在建立無網路的安全或測試環境，場景是封閉式運算、離線任務、網路安全測試

### Third-party network plugins

Docker 也支援安裝第三方網路插件，讓使用者可整合特殊的網路堆疊或 SDN（Software-Defined Networking）方案，例如 Calico、Cilium 等

## `docker network ls`

我們可以利用以上的指令來查看目前的網路類型

```bash
docker network ls
NETWORK ID     NAME      DRIVER    SCOPE
8ddb7e9846c6   bridge    bridge    local
48e785b7efb3   host      host      local
7e07c5b5ae34   none      null      local
```

## Bridge mode

我們比較常見的 docker 網路類型是 bridge mode

<img width="1295" height="643" alt="image" src="https://github.com/user-attachments/assets/4b85ff86-c5d2-4dfc-871f-f1670b197290" />

```bash
docker network create my-net

# 創建容器時指定 --network
docker create --name my-nginx \
  --network my-net \
  --publish 8080:80 \
  nginx:latest

docker network connect my-net my-nginx
```

```bash
# 創建網路時用 --ipv6 啟用 IPv6，若無指定 --subnet，會自動選擇 Unique Local Address (ULA) 前綴

docker network create --ipv6 --subnet 2001:db8:1234::/64 my-net

docker network create --ipv6 --ipv4=false v6net
```

### Docker Bridge 的原理

我們一直在探究網路，勢必要了解底層的原理吧！其實也都是來自 Linux 標準工具的組合，如 network namespaces、virtual Ethernet devices (veth)、virtual network switches (bridge)、IP routing 和 network address translation (NAT)，我們可以基於 Linux kernel 的網路虛擬化來了解這些容器底層的網路運作

- Network namespaces (netns)：虛擬化網路堆疊，讓每個容器有獨立網路環境，包括裝置、路由和防火牆規則
- Virtual Ethernet devices (veth)：虛擬乙太網路裝置，用來連接容器和主機
- Virtual network switches (bridge)：虛擬交換器，連接多個 veth，讓容器互相通訊
- IP routing 和 NAT：路由讓流量轉發，NAT 讓容器連外網並隱藏內部 IP

### 開始吧～

#### 1. 使用 Network Namespaces 建立容器網路隔離

```bash
# Docker Bridge 的基礎是 network namespace (netns)，它複製網路堆疊，讓容器有獨立路由、防火牆和裝置

sudo ip netns add netns0
sudo ip netns add netns1

ip netns list

ip link list
# Output:
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
2: enp3s0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP mode DEFAULT group default qlen 1000
    link/ether fa:16:3e:e7:e6:e9 brd ff:ff:ff:ff:ff:ff
3: enp4s0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1442 qdisc fq_codel state UP mode DEFAULT group default qlen 1000
    link/ether fa:16:3e:80:71:41 brd ff:ff:ff:ff:ff:ff

# 進入 namespace 跑 bash
sudo nsenter --net=/run/netns/netns0 bash

ip link list

# 輸出應只顯示，就跟原本的有差異
1: lo: <LOOPBACK> mtu 65536 qdisc noop state DOWN mode DEFAULT group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00

# 然後可以跳出
exit
```

#### 2. 建構 Bridge 網路，讓容器互相通訊

```bash
# Docker 的 bridge (如 docker0) 是虛擬交換器，連接容器的 veth 裝置
sudo ip link add name bridge0 type bridge
sudo ip link set bridge0 up

# 為容器建立 veth pair (一端在容器，一端連 bridge)
# 配置 netns0
sudo ip link add veth0 type veth peer name veth0-br
sudo ip link set veth0 netns netns0
sudo ip link set veth0-br master bridge0
sudo ip link set veth0-br up
sudo nsenter --net=/run/netns/netns0 ip addr add 192.168.1.2/24 dev veth0
sudo nsenter --net=/run/netns/netns0 ip link set veth0 up

# 配置 netns1
sudo ip link add veth1 type veth peer name veth1-br
sudo ip link set veth1 netns netns1
sudo ip link set veth1-br master bridge0
sudo ip link set veth1-br up
sudo nsenter --net=/run/netns/netns1 ip addr add 192.168.1.3/24 dev veth1
sudo nsenter --net=/run/netns/netns1 ip link set veth1 up
```

對另一容器 netns1 重複做一樣的事，IP 是 192.168.1.3，兩個容器透過 bridge0 通訊，使用 192.168.1.x 的 IP，這就是 Docker Bridge 的核心，docker0 是 bridge，每容器一個 veth 連到它，讓容器間用 IP 通訊

#### 3. 連外網與 NAT

```bash
# 在 bridge0 加 IP
sudo ip addr add 192.168.1.1/24 dev bridge0

# 啟用 IP forwarding
sudo sysctl -w net.ipv4.ip_forward=1

# NAT 規則（讓容器流量偽裝成主機 IP）
sudo iptables -t nat -A POSTROUTING -s 192.168.1.0/24 -o eth0 -j MASQUERADE
```

容器能 ping 外網，但外部不知容器 IP，這就是 Docker Bridge 的 NAT 機制

#### 4. 從外部連容器與端口發佈

```bash
# 外部連容器需 port mapping，假設容器跑服務在 80 port，需要加規則
sudo iptables -t nat -A PREROUTING -p tcp --dport 8080 -j DNAT --to-destination 192.168.1.2:80
sudo iptables -A FORWARD -p tcp -d 192.168.1.2 --dport 80 -j ACCEPT
```

最後這實現 Docker 的 -p 8080:80：外部連主機 8080，轉到容器 80，但假如要操作這個實驗的話還是要在 `netns0` 這個 ns 啟動 port 為 80 的服務 `curl http://localhost:8080` 才有效喔

```bash
# 也可以再進入到 ns 中查看
sudo nsenter --net=/run/netns/netns0 bash

ip link list

# Output
1: lo: <LOOPBACK> mtu 65536 qdisc noop state DOWN mode DEFAULT group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
6: veth0@if5: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP mode DEFAULT group default qlen 1000
    link/ether fe:de:63:5f:37:7d brd ff:ff:ff:ff:ff:ff link-netnsid 0
```

或是可以從 netns0 ping netns1 來查看是否有正確的連接

```bash
sudo nsenter --net=/run/netns/netns0 ping -c 3 192.168.1.3
```

```bash
# 在主機 namespace 檢查 bridge0
sudo ip link list master bridge0

# Output:
5: veth0-br@if6: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master bridge0 state UP mode DEFAULT group default qlen 1000
    link/ether 92:59:0b:1c:8a:8f brd ff:ff:ff:ff:ff:ff link-netns netns0
7: veth1-br@if8: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master bridge0 state UP mode DEFAULT group default qlen 1000
    link/ether 4e:dd:22:a9:88:0f brd ff:ff:ff:ff:ff:ff link-netns netns1
```

看到這邊也實作完了 docker bridge 的原理啦！看圖可以更清楚呦！

<img width="2000" height="1136" alt="image" src="https://github.com/user-attachments/assets/31287771-08e0-4d79-90b8-cc03214270cf" />

#### Cleanup

```bash
sudo ip netns delete netns0 2>/dev/null
sudo ip netns delete netns1 2>/dev/null
sudo ip link delete bridge0 2>/dev/null
sudo iptables -t nat -F
sudo iptables -F
```

## Reference

https://docs.docker.com/engine/network/drivers/

https://ithelp.ithome.com.tw/articles/10305163

https://ithelp.ithome.com.tw/articles/10305163

https://www.docker.com/blog/understanding-docker-networking-drivers-use-cases/

https://labs.iximiuz.com/tutorials/container-networking-from-scratch

# PID 1 的重要性、Signals 與關閉機制

## 前言

今天時間比較多，好像可以把前幾天有看到的東西也把它寫一下，話說現在有 ai 可以整理資料之後，閱讀的速度真的差很多，可以整理的實驗也快很多，那剛剛是在網路的地方去著手，那接著就繼續往 docker 的生命週期去了解，使用者常忽略但至關重要的部分，尤其在生產環境中，理解這些能避免意外的中斷或資源洩漏，會從 CMD 執行模式開始，逐步解釋 signals、關閉命令的行為，並介紹 cgroup freezer 子系統作為進階資源管理工具

## PID 是什麼？

在 Linux 系統中，每個執行中的「程序（Process）」都有一個唯一的「Process ID, PID」。他是系統中第一個被啟動的進程，在一般 Linux 系統中是 systemd 或 init，所有其他進程（PID 2、3、4...）都是由 PID 1 啟動或衍生的。當啟動一個 Docker 容器時，`docker run --name myapp nginx` 這個容器的「第一個進程」就是 nginx，在容器內部的世界裡 nginx 的 PID = 1，它就是這個容器裡的「init process」

這時候我們就要了解到 Zombie Process，在 Linux 中，父進程需要負責「回收」子進程的資源，若 PID 1 不做這件事，容器內會出現大量的 zombie process，一般的應用程式（如 node、python）不會自動做這件事

而容器中止時，也就是 PID 1 的進程結束時，只要 PID 1 結束，整個容器就會被 Docker 判定為「已停止」

## CMD 執行模式的差異：Exec vs Shell

在 Dockerfile 中，CMD 指令定義容器啟動時執行的命令，但它的執行方式有兩種：exec mode 和 shell mode。這不僅影響 PID 1 的身分，還會影響容器如何處理信號和關閉

### Exec Mode

它使用 JSON 陣列格式，這裏直接執行 sleep 1000，容器中的 PID 1 就是 sleep 進程。透過 docker top 或 docker ps 查看，你會看到 PID 1 是 sleep

```text
CMD ["sleep", "1000"]
```

### Shell Mode

使用字串格式，這透過 `/bin/sh -c` 執行，所以 PID 1 是 `/bin/sh -c 'sleep 1000'`，docker ps 會顯示類似 "/bin/sh -c 'sleep 1…"。這多了一層 shell 包裝

```text
CMD sleep 1000
```

### 差異影響與使用場景

這兩種模式的主要差異在於容器內進程的啟動方式與信號傳遞機制，由於 Shell Mode 會多經過一層 `/bin/sh -c`，因此信號傳遞會有差異，在 Exec Mode 中，Docker 送出的信號（例如 SIGTERM、SIGINT）會直接傳給 PID 1 的應用程式（如 sleep、python、nginx），這樣應用可以正確接收並處理信號，進行 graceful shutdown; 在 Shell Mode 中，信號會先傳給 `/bin/sh`，而 `/bin/sh` 不一定會把信號轉發給子進程，因此常見情況是應用程式收不到關閉信號，導致容器停止時需等超時再被 SIGKILL 強制殺死

### PID 1 的兩個責任

1. 回收孤兒/ zombie 子進程

   Linux 規定 PID 1 需要呼叫 wait() 來回收 zombie

   如果 PID 1 是 sleep（Exec Mode），sleep 本身不會呼叫 wait() → 無法回收 zombie。

   如果 PID 1 是 /bin/sh -c（Shell Mode），大部分 shell 會有簡單的 wait 機制 → 可以回收 zombie。

2. 信號轉發 / graceful shutdown

   Docker 對容器發送信號（如 SIGTERM）時，會傳給 PID 1

   Exec Mode：PID 1 就是你的應用程式本身，應用程式可以直接接收信號並處理 → graceful shutdown 正常。

   Shell Mode：PID 1 是 /bin/sh -c，shell 可能不會把 SIGTERM 傳給子進程（你的應用程式），所以應用可能收不到信號 → 容器會等超時再被 SIGKILL。

兩者在不同需求上各有利弊，如果關心 zombie 回收 → Shell Mode 會比較穩定，如果關心應用程式能正確收到 SIGTERM → Exec Mode 是最佳選擇

所以在生產環境，常用 Exec Mode + init wrapper，像是 tini，tini 會作為 PID 1 負責回收 zombie、轉發信號，應用程式直接作為子進程運行，信號可以正常接收，容器也能 graceful shutdown，這真的就兩全其美了呀～

```dockerfile
ENTRYPOINT ["tini", "--"]
CMD ["nginx", "-g", "daemon off;"]
```

## Linux Signals

Signals 是 Linux 進程間的非同步通知機制，用來中斷、終止或暫停進程。每個 signal 有預設行為，但可以被忽略、捕捉或自訂處理，像是有幾種例子

- Ctrl+C 送出 SIGINT (Interrupt from keyboard)
- kill [pid] 預設送 SIGTERM (Termination signal)
- kill -9 [pid] 送 SIGKILL (Kill signal)

### 那容器關閉機制與 PID 1 的關係為何呢？

Docker 使用 signals 關閉容器，PID 1 是接收者，而這反映 PID 1 的特殊性，它是 namespace 的 init process，終止它會導致 kernel SIGKILL 其他進程

docker stop 會送 SIGTERM 給 PID 1，若 PID 1 處理並退出，容器關閉，否則 10 秒後送 SIGKILL（可改 `--time=5`）

docker kill 直接送 SIGKILL（可改 `--signal`）快速關閉，無 graceful stop

docker rm -f 對運行容器送 SIGKILL 後移除

所以 PID 1 真的很重要！它需回收 zombie、處理 SIGTERM 做 graceful stop，並轉發給 child processes，Shell mode 常讓 `/bin/sh` 當 PID 1，導致問題，所以推薦 exec mode 或使用 tini 等 init 工具

## Cgroup Freezer

Cgroup freezer 是 linux 核心的一個子系統，用來凍結/解凍一組 task，讓管理員可以根據需求調度機器，他特別適合使用在 batch job management，像是在 HPC（高效能運算）cluster 調度整個 cluster 的存取權，freezer 使用 cgroups 定義要啟動或停止的任務提供一個機制來批量管理這些 task

另一個重點是 checkpointing 運行中的 task group，freezer 讓 checkpointing code 取得 consistent image，透過強制 cgroup 中的任務進入 quiescent state，一但任務停止，他的任務可以造訪 `/proc` 或呼叫和新介面來收集資訊，checkpointing task 可以在發生可恢復錯誤時重新啟動，也允許任務遷移到 cluster 的其他節點，複製收集的資訊到心節點並在那裡重新啟動

### 為什麼不直接用 SIGSTOP 和 SIGCONT？

序列化的 SIGSTOP 和 SIGCONT 信號不總是足夠用來在用戶空間停止/恢復任務。這些信號對要凍結的任務是可觀察的，SIGSTOP 無法被捕捉、阻擋或忽略，但可以被等待或 ptrace 的父任務看到、SIGCONT 更不適合，因為它可以被任務捕捉

### Cgroup Freezer 的階層結構

凍結一個 cgroup 會凍結屬於該 cgroup 和所有後代 cgroups 的任務，每個 cgroup 有 self-state 和 parent-state，只有兩個狀態都是 THAWED（解凍）時，cgroup 才是 THAWED，為何這樣設計呢？如果 parent 凍結，而 child 自己設為 THAWED，實際上仍然不能運行，這樣的設計保證了上層 cgroup 的控制權優先，還有管理員或容器 runtime（如 Docker、systemd）凍結一個 cgroup，就可以保證整個子樹的所有任務都被暫停，不必逐一控制每個子 cgroup，像是 Docker 暫停容器時會使用 freezer cgroup，將容器整棵進程樹凍結、systemd 進行服務管理時，想暫停整個服務（包括所有子進程和子 cgroup）以釋放資源或進行 checkpoint/restore

```text
* Examples of usage :

   # mkdir /sys/fs/cgroup/freezer
   # mount -t cgroup -ofreezer freezer /sys/fs/cgroup/freezer
   # mkdir /sys/fs/cgroup/freezer/0
   # echo $some_pid > /sys/fs/cgroup/freezer/0/tasks

to get status of the freezer subsystem :

   # cat /sys/fs/cgroup/freezer/0/freezer.state
   THAWED

to freeze all tasks in the container :

   # echo FROZEN > /sys/fs/cgroup/freezer/0/freezer.state
   # cat /sys/fs/cgroup/freezer/0/freezer.state
   FREEZING
   # cat /sys/fs/cgroup/freezer/0/freezer.state
   FROZEN

to unfreeze all tasks in the container :

   # echo THAWED > /sys/fs/cgroup/freezer/0/freezer.state
   # cat /sys/fs/cgroup/freezer/0/freezer.state
   THAWED
```

以上範例展示如何使用 cgroup freezer 子系統控制進程：首先建立 freezer cgroup 並將指定進程加入，透過讀寫 freezer.state 可以暫停（FROZEN）、解凍（THAWED）或查看（THAWED/FREEZING/FROZEN）整個 cgroup 的所有任務，達到容器或進程凍結與解凍的效果

## 小傑

今天看到了 PID 和 cgroup freezer，其實平常是不會直接操作到 cgroup freezer，他是比較在背景發生的場景，當你執行 `docker pause <container>` 時，Docker 會透過 cgroup freezer 將該容器內的所有進程 FROZEN，暫停 CPU 調度，容器內進程就像被冰凍一樣，不會執行任何 code，執行 `docker unpause <container>` 時，Docker 會將 cgroup 狀態改為 THAWED，讓所有進程恢復運作，在一些 high-level Docker 或容器管理平台，可能會用 freezer 來暫停不活躍的容器或進程，節省 CPU 或暫時隔離進程，其實我們也可以看到圖片的 docker 生命週期，就可以知道今天所說的的細節環節，雖然在途中沒有講到，但已經可以從其中的流程了解到了細節的運作了～～

<img width="1079" height="648" alt="image" src="https://github.com/user-attachments/assets/9fde7f0d-2ef2-4f31-87ea-a98b6707e1a5" />

## Reference

https://kernel.meizu.com/2024/07/12/sub-system-cgroup-freezer-in-Linux-kernel/

https://ithelp.ithome.com.tw/articles/10304865

https://blog.miniasp.com/post/2021/07/09/Use-dumb-init-in-Docker-Container

https://www.kernel.org/doc/Documentation/cgroup-v1/freezer-subsystem.txt

https://www.threads.com/@willh_tw/post/DPXjEPaDvoV/%E5%AD%B8%E7%BF%92-docker-%E5%AE%B9%E5%99%A8%E7%94%9F%E5%91%BD%E9%80%B1%E6%9C%9F%E7%AE%A1%E7%90%86%E7%9A%84%E5%AE%8C%E6%95%B4%E6%8C%87%E5%8D%97%E4%BE%86%E4%BA%86%E9%9B%96%E7%84%B6-docker-run-%E6%8C%87%E4%BB%A4%E8%83%BD%E5%BF%AB%E9%80%9F%E5%95%9F%E5%8B%95%E5%AE%B9%E5%99%A8%E4%BD%86%E5%9C%A8%E5%AF%A6%E9%9A%9B%E6%87%89%E7%94%A8%E5%A0%B4%E6%99%AF%E4%B8%AD%E9%96%8B%E7%99%BC%E8%80%85%E5%BE%80%E5%BE%80%E9%9C%80%E8%A6%81%E6%9B%B4%E7%B2%BE%E7%B4%B0%E7%9A%84%E6%8E%A7%E5%88%B6%E8%83%BD%E5%8A%9Bivan-vel
