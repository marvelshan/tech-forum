## K8S Lab Day_36

# Linux 常用系統維運指令實戰：從 dmesg 到 ethtool 一次搞懂

## 前言

當一位軟體工程師和 SRE 相當重要的是要跟 Linux 相處在一起，而這陣子在準備一些面試的時候，要回來準備這些用法和熟悉這些用法也是相當的重要的，針對於 debug 才能融會貫通

## 1. dmesg — 系統開機與核心訊息觀察器 （display message）

用途：顯示 kernel ring buffer 中的訊息，包含開機、硬體偵測、網路卡、磁碟、USB 事件等。這是排查硬體或驅動問題的第一個好幫手，實際應用場景為插入一個 USB 想確認他是否有被偵測到、系統無法掛載磁碟，需要查看他的原因、網路介面異常時檢查驅動的訊息

<img width="629" height="467" alt="image" src="https://github.com/user-attachments/assets/b36ab110-3ed6-42e8-ab39-10c2f5998da4" />

```bash
# 顯示最近的 kernel log
dmesg | tail

# 只看網路相關訊息
dmesg | grep eth

# 以人類可讀時間格式顯示
dmesg -T
```

## 2. dd — 低階資料複製與磁碟映像神器（Data Duplicator）

用途：用來進行位 block-level 的資料複製、磁碟備份或測試磁碟速度，實際的應用場竟是備份整顆磁碟或分割區、做硬碟效能壓力測試

```bash
# 建立磁碟映像檔
dd if=/dev/sda of=/backup/disk.img bs=4M status=progress

# 寫入 ISO 到 USB
dd if=ubuntu.iso of=/dev/sdb bs=4M status=progress

# 測試磁碟寫入速度
dd if=/dev/zero of=testfile bs=1G count=1 oflag=dsync
```

## 3. du — 檔案或資料夾大小統計（disk usage）

用途：快速統計目錄大小、找出占空間的資料夾，實際應用在磁碟快滿時查是哪個資料夾占空間、日誌（log）暴增時找出罪魁禍首

```bash
# 顯示目前目錄各資料夾大小
du -h --max-depth=1

# 顯示某個資料夾總大小
du -sh /var/log
```

## 4. dc — 命令列計算機（Desk Calculator）

用途：做簡單數學運算的小工具，在不想開 Python 或 bc 的環境下快速算數、腳本中需要簡單運算時（雖然 awk / expr 更常用）

```bash
# 進入互動模式
dc
# 輸入：2 3 + p   （p代表print）
# 結果：5
```

## 5. passwd — 修改使用者密碼

用途：設定或修改使用者密碼，建立新使用者後第一次設定密碼，管理員重設密碼。

```bash
# 修改自己密碼
passwd

# 修改其他使用者密碼（需 root）
sudo passwd username
```

## 6. usermod — 使用者屬性管理

用途：修改使用者資訊，如群組、登入 shell、家目錄位置，想讓新使用者可以執行 docker，改變某個使用者的登入設定

```bash
# 把使用者加入 docker 群組
sudo usermod -aG docker username

# 改變登入 shell
sudo usermod -s /bin/bash username
```

## 7. awk — 文本處理神器

用途：逐行讀取文本並以欄位操作資料，非常適合 log、CSV 或系統輸出分析，可以快速分析系統輸出結果、統計或過濾 log、結合 shell script 做報表。

```bash
# 顯示 /etc/passwd 檔的第一欄（使用者名稱）
awk -F: '{print $1}' /etc/passwd

# 計算某欄數值總和
awk '{sum += $2} END {print sum}' data.txt
```

## 8. ip — 網路介面與路由管理

用途：取代傳統的 ifconfig、route，用於管理 IP、route、link、bridge 等，可以用於設定多網卡主機的路由、臨時調整 IP 設定或測試網段

```bash
# 顯示網卡資訊
ip addr show

# 顯示路由表
ip route

# 新增靜態路由
sudo ip route add 10.0.0.0/24 via 192.168.1.1
```

## 9. ethtool — 網卡層級除錯工具

用途：查看或設定 Ethernet device 相關參數，例如速率、雙工模式、driver，用於檢查實體網卡連線狀態、優化網路延遲（例如關閉 offloading）、在資料中心或虛擬機中排查低速問題

```bash
# 顯示網卡資訊
sudo ethtool eth0

# 測試是否支援特定速率
sudo ethtool eth0 | grep Speed

# 關閉網卡自動協商
sudo ethtool -s eth0 autoneg off speed 100 duplex full
```

## 奇牙

其實熟練 linux 就要像是肌肉記憶，雖然我也還沒熟練他，但是這看起來很零散的資料和方法，但只要抓住這幾個步驟**觀察、修改、驗證**，久而久之就能很熟練的運用他們，遇到問題就能以非常快的速度迎刃而解，最後我也整理了一些比較常用的指令當作工具本

## 系統監控與效能分析

### 進程管理

使用場景：系統變慢時查哪個程式佔 CPU 或記憶體最多、偵測背景中殘留的服務或無法關閉的程序。

```bash
# 查看系統進程
ps aux
ps -ef

# 動態監控進程（類似任務管理員）
top
htop    # 顯示更清晰，需先安裝

# 結束進程
kill <PID>
killall <程序名稱>
pkill <模式>

# 顯示進程樹
pstree
```

### 系統狀態監控

```bash
# 查看系統運行時間與平均負載
uptime

# 查看記憶體使用情況
free -h

# 監控系統資源
vmstat 1          # 每秒更新一次
iostat            # 磁碟 I/O 統計
sar               # 系統活動報告（需 sysstat 套件）

# 查看即時系統事件（例如核心錯誤或驅動訊息）
dmesg | tail
journalctl -xe    # 以 systemd 為核心的 log 查看方式
```

## 檔案與目錄操作

### 基礎操作

在刪除前可以用 ls 或 du -sh 檢查，若是 production 系統，強烈建議搭配 trash-cli

```bash
mkdir dirname
mkdir -p parent/child       # 建立多層目錄

cp file1 file2              # 複製檔案
cp -r dir1 dir2             # 複製整個資料夾

mv oldname newname          # 移動或重新命名
rm filename
rm -r dirname               # 刪除資料夾
rm -rf dirname              # 強制刪除（危險操作）
```

### 檔案內容與分析

```bash
cat filename                # 顯示全部內容
less filename               # 分頁查看
head -n 10 filename         # 前10行
tail -n 10 filename         # 後10行
tail -f logfile             # 實時監控日誌

diff file1 file2            # 比對差異
cmp file1 file2             # 二進位比較
```

### 檔案搜尋

```bash
# 以名稱查找
find /path -name "*.log"
find . -type f -name "config*"

# 以內容查找
grep "pattern" filename
grep -r "error" /var/log/
grep -i "warning" file

# 快速定位（需先建立資料庫）
locate filename
updatedb
```

## 使用者與權限管理

### 權限操作

```bash
chmod 755 filename           # rwxr-xr-x
chmod u+x script.sh          # 給擁有者執行權限

chown user:group filename
chown -R user:group dir/     # 遞迴更改擁有者

chgrp groupname filename     # 改變檔案群組
```

### 使用者管理

```bash
passwd                      # 修改密碼
sudo passwd username         # 管理員修改他人密碼
usermod -aG docker user      # 加入群組
usermod -s /bin/bash user    # 修改登入 shell
```

## 網路指令與除錯工具

### 網路連線診斷

```bash
ping google.com
ping -c 4 8.8.8.8

traceroute google.com
tracepath google.com

# 查看連線與端口
netstat -tuln
ss -tuln                   # netstat 的替代工具

# DNS 查詢
nslookup example.com
dig example.com
host example.com
```

### 網路介面管理（搭配 ip、ethtool）

應用狀況是在雙網卡環境下調整主路由或測試虛擬機或容器的網卡速率

```bash
ip addr show                # 顯示所有網卡
ip route                    # 顯示路由表
ip link set eth0 down       # 關閉網卡
ip link set eth0 up         # 開啟網卡

sudo ethtool eth0           # 查看網卡設定
sudo ethtool -s eth0 autoneg off speed 100 duplex full
```

## 磁碟與儲存管理

```bash
df -h                       # 查看磁碟使用率
df -i                       # 查看 inode 使用情況

find /path -type f -size +500M      # 找出大檔案
du -h --max-depth=1 /path | sort -hr
```

## 套件管理

### Debian / Ubuntu

```bash
sudo apt update
sudo apt upgrade
sudo apt install package
sudo apt remove package
```

### CentOS / Fedora

```bash
sudo yum install package
sudo dnf install package
```

## 文本處理三劍客（grep / sed / awk）

```bash
# 搜尋關鍵字
grep -r "error" /var/log/
grep -v "debug" file.log
grep -E "warn|fail" file.log
```

```bash
# 快速編輯
sed 's/old/new/g' file
sed -n '10,20p' file
sed '/pattern/d' file
```

```bash
# 欄位處理
awk -F: '{print $1}' /etc/passwd
awk '{sum += $2} END {print sum}' data.txt
```

### 輔助工具

```bash
cut -d: -f1 /etc/passwd
sort file | uniq
sort -nr file
```

## 日常技巧與命令操作

### 背景工作與多任務

```bash
./script.sh &       # 背景執行
jobs                # 查看背景任務
fg %1               # 回到前台
bg %1               # 繼續背景執行
```

### terminal control

```bash
screen              # 建立分離式會話
tmux                # 功能更強的多工終端
```

### 歷史命令與自動補全

```bash
history
!vim                # 執行上一次的 vim 指令
Ctrl + R            # 搜尋歷史命令
Tab                 # 自動補全
```

## 快速檢查服務狀況

```bash
# 查看系統啟動時間與負載（Load Average）
uptime

# 查看目前登入的使用者與運行時間
w

# 顯示記憶體總量與使用量（人類可讀格式）
free -h

# 查看系統可用記憶體（不含 cache）
cat /proc/meminfo | grep -i memavailable

# 檢查磁碟使用量
df -h

# 檢查 inode 使用率（是否太多小檔案）
df -i

# 以批次模式列出前 20 行 CPU 使用狀況
top -n 1 -b | head -20

# 互動式查看（需安裝）
htop

# 檢查指定服務狀態
systemctl status your-service-name

# 僅檢查是否運行中
systemctl is-active your-service-name

# 列出所有進程並搜尋指定服務
ps aux | grep your-service

# 直接用 pgrep 搜尋進程名稱
pgrep -f your-service

# 查看指定 port 是否被監聽
netstat -tulpn | grep :port
ss -tulpn | grep :port
lsof -i :port

# 追蹤指定服務的 systemd log
journalctl -u your-service-name -f
journalctl -u your-service-name --since "1 hour ago"
journalctl -u your-service-name -n 100

# 系統主日誌（Ubuntu）
tail -f /var/log/syslog

# CentOS 或 RHEL 系統主日誌
tail -f /var/log/messages

# 查看應用日誌
tail -f /var/log/your-app/app.log

# 查找應用相關日誌檔
find /var/log -name "*your-app*" -type f

# 同時查看多個 nginx 日誌
tail -f /var/log/nginx/access.log /var/log/nginx/error.log

# 使用 multitail（需安裝）
multitail /var/log/nginx/access.log /var/log/nginx/error.log

# 搜尋錯誤字樣
grep -i "error" /var/log/your-app/app.log

# 顯示上下文 5 行的例外訊息
grep -C 5 "exception" /var/log/syslog

# 測試服務是否可通
curl -v http://localhost:port

# telnet 舊式測試工具
telnet localhost port

# nc（netcat）快速測試
nc -zv localhost port

# iptables 規則
iptables -L -n

# Ubuntu 使用 UFW
ufw status

# DNS 解析是否正常
nslookup your-domain.com
dig your-domain.com

# 統計目前已建立的連線數
netstat -an | grep ESTABLISHED | wc -l

# socket 統計摘要
ss -s

# 查看即時流量使用
iftop
nethogs

# 檢查延遲與路由
traceroute your-domain.com
mtr your-domain.com

# 磁碟 I/O 狀況（每秒刷新一次）
iostat -x 1

# 查看哪個進程 I/O 較高
iotop

# 僅監控特定服務進程
top -p $(pgrep -d',' your-service)

# 查看進程樹結構
pstree -p $(pgrep your-service)
ps -ef --forest | grep your-service

# 查看進程打開的檔案
lsof -p $(pgrep your-service)

# 查看進程狀態
cat /proc/$(pgrep your-service)/status

# 跟蹤系統呼叫
strace -p $(pgrep your-service)

# 查看目前 shell 限制
ulimit -a

# 查看特定進程限制
cat /proc/$(pgrep your-service)/limits

# 驗證 Nginx 配置
nginx -t

# 驗證 Apache 配置
apache2ctl configtest

# 查看應用環境變數
env | grep -i your_app
printenv

# 測試資料庫連線
mysql -h host -u user -p -e "SELECT 1;"
redis-cli ping

# PostgreSQL 連線數
psql -c "SELECT count(*) FROM pg_stat_activity;"

# MySQL 慢查詢或卡住的執行緒
mysql -e "SHOW FULL PROCESSLIST;"
```

## healthcheck.sh

```bash
#!/usr/bin/env bash
# ---------------------------------------
# Linux 系統健康檢查腳本
# 功能: 由上到下偵測系統、服務、網路是否正常
# ---------------------------------------

SERVICE_NAME=${1:-"nginx"}        # 預設檢查 nginx，可用參數傳入
CHECK_URL=${2:-"http://localhost"} # 預設檢查 localhost
PORT=${3:-80}                      # 預設 port

set -e  # 有任何命令失敗就結束 (防止連續錯誤)
trap 'echo "[!] 檢查中斷，可能有錯誤發生"; exit 1' ERR

echo "=== [Step 1] 系統資源檢查 ==="

# 檢查 CPU Load
LOAD=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | sed 's/,//')
LOAD_INT=${LOAD%.*}
if [ "$LOAD_INT" -ge 4 ]; then
  echo "[ERROR] CPU 負載過高: $LOAD"
  exit 1
else
  echo "[OK] CPU 負載正常: $LOAD"
fi

# 檢查記憶體
AVAILABLE=$(grep -i memavailable /proc/meminfo | awk '{print $2}')
if [ "$AVAILABLE" -lt 500000 ]; then
  echo "[ERROR] 可用記憶體過低: ${AVAILABLE}KB"
  exit 1
else
  echo "[OK] 記憶體充足: ${AVAILABLE}KB 可用"
fi

# 檢查磁碟空間
DISK_USAGE=$(df / | tail -1 | awk '{print $5}' | tr -d '%')
if [ "$DISK_USAGE" -gt 90 ]; then
  echo "[ERROR] 根目錄磁碟使用率過高: ${DISK_USAGE}%"
  exit 1
else
  echo "[OK] 磁碟使用率正常: ${DISK_USAGE}%"
fi

echo
echo "=== [Step 2] 服務檢查 ($SERVICE_NAME) ==="

# 檢查服務是否啟動
if ! systemctl is-active --quiet "$SERVICE_NAME"; then
  echo "[ERROR] 服務未啟動: $SERVICE_NAME"
  exit 1
else
  echo "[OK] 服務運行中: $SERVICE_NAME"
fi

# 檢查進程是否存在
PID=$(pgrep -f "$SERVICE_NAME" || true)
if [ -z "$PID" ]; then
  echo "[ERROR] 找不到 $SERVICE_NAME 的進程"
  exit 1
else
  echo "[OK] 進程存在, PID: $PID"
fi

echo
echo "=== [Step 3] 網路層檢查 ==="

# 檢查 port 是否監聽
if ! ss -tulpn | grep -q ":$PORT"; then
  echo "[ERROR] Port $PORT 未被監聽"
  exit 1
else
  echo "[OK] Port $PORT 監聽正常"
fi

# 檢查 DNS 解析
if ! nslookup google.com >/dev/null 2>&1; then
  echo "[ERROR] DNS 解析失敗"
  exit 1
else
  echo "[OK] DNS 解析正常"
fi

# 檢查外網連線
if ! ping -c 2 8.8.8.8 >/dev/null 2>&1; then
  echo "[ERROR] 外部網路不通"
  exit 1
else
  echo "[OK] 外部網路可連線"
fi

echo
echo "=== [Step 4] 應用層檢查 ($CHECK_URL) ==="

# HTTP 健康檢查
STATUS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$CHECK_URL")
if [ "$STATUS_CODE" -ne 200 ]; then
  echo "[ERROR] HTTP 狀態碼異常: $STATUS_CODE"
  exit 1
else
  echo "[OK] HTTP 正常回應 200"
fi

echo
echo "=== 所有檢查通過，系統狀況良好 ==="
```

```bash
chmod +x healthcheck.sh

# 範例1：檢查 nginx + localhost
./healthcheck.sh

# 範例2：檢查 postgresql 服務、指定 URL 與 Port
./healthcheck.sh postgresql http://127.0.0.1:5432 5432
```
