## K8S Lab Day_47

## 前言

平常都會使用 ssh 連到終端機，都會跳出超多的資訊，今天就來看一下這些訊息的內容

```
Welcome to Ubuntu 22.04 LTS (GNU/Linux 5.15.0-160-generic x86_64)

 * Documentation:  https://help.ubuntu.com
 * Management:     https://landscape.canonical.com
 * Support:        https://ubuntu.com/advantage

  System information as of Thu Nov  6 17:49:54 UTC 2025

  System load:             0.0
  Usage of /:              39.6% of 27.55GB
  Memory usage:            11%
  Swap usage:              0%
  Processes:               133
  Users logged in:         0
  IPv4 address for enp3s0: 103.122.***.69
  IPv6 address for enp3s0: 2403:8ec0::****:3eff:fee7:e6e9
  IPv4 address for enp4s0: 192.168.200.100


122 updates can be applied immediately.
5 of these updates are standard security updates.
To see these additional updates run: apt list --upgradable


*** System restart required ***
Last login: Thu Nov  6 14:48:57 2025 from 42.77.**.90
```

首先可以看到 `Welcome to Ubuntu 22.04 LTS (GNU/Linux 5.15.0-160-generic x86_64)`，這裏可以看到我們最首席的 Ubuntu 的版本，這邊是使用到長期支援的版本也就是 22.04 LTS，官方可以支援到 2032，然後後面就是 Linux kernel 的版本 `5.15.0-160`，然後系統架構是使用 64 位元的 x86 架構

```
 * Documentation:  https://help.ubuntu.com
 * Management:     https://landscape.canonical.com
 * Support:        https://ubuntu.com/advantage
```

接著可以看到這裡說明到的是 ubuntu 官方文件，然後 landscape 是 canonical 提供企業級的 ubuntu 管理平台喔，這家公司在台北也有辦公司，可說是很有競爭力的公司！然後就是 Support 這邊講到的是一些技術支援還有合規認證和安全更新等等

然後可以看到這邊 `System information as of Thu Nov  6 17:49:54 UTC 2025` 是現在系統的即時狀態，也就是我現在連進來的時間，因為 UTC 是全球通用時間標準，像是台灣就是 +8 所以可見我現在凌晨 1 點連進來寫文章 XD

```
  System load:             0.0
  Usage of /:              39.6% of 27.55GB
  Memory usage:            11%
  Swap usage:              0%
  Processes:               133
  Users logged in:         0
  IPv4 address for enp3s0: 103.122.***.69
  IPv6 address for enp3s0: 2403:8ec0::****:****:fee7:e6e9
  IPv4 address for enp4s0: 192.168.200.100
```

接著就是顯示當前主機的運行狀況了，

- `System load: 0.0` 這邊表示系統負載極低，0.0 表示幾乎沒有負載

- `Usage of /:              39.6% of 27.55GB` 表示在根目錄，也就是 `/` 已經使用了 39.6%，然後總共的容量有 `27.55GB`

- `Memory usage:            11%` 目前的實體記憶體的使用率有 11%

- `Swap usage:              0%` 表示這邊沒有使用 swap，表示記憶體充足，不需要使用硬碟模擬記憶體，假如要了解可以[參考](https://enohuang.com/blog/2024/use-swap-space-to-avoid-oom/)

- `Users logged in:         0` 目前沒有其他使用者登入

- `IPv4 address for enp3s0: 103.122.***.69` 這裡顯示了兩張網路卡的 IP，一張是 `enp3s0` 另一張是 `enp4s0`，然後第一張有 IPv4 和 IPv6 然後是公網 IP，另一張是內網的 IP

小補充：enp3s0 和 enp4s0 是現代 Linux 使用的 predictable network interface names，取代舊的 eth0

```
122 updates can be applied immediately.
5 of these updates are standard security updates.
To see these additional updates run: apt list --upgradable
```

接著這邊就是套件的小提示，告訴多少套件是安全的，哪些是需要更新，然後應該要執行 `apt list --upgradable`

```
*** System restart required ***
```

這邊表示系統內有更新套件已安裝完成，但需要重新開機才能完全生效，可以使用這個指令來確認一下重開機的原因 `cat /var/run/reboot-required`，如果確定當前沒有服務正在執行就可以重開呦，但好像不太會這樣做 `sudo reboot`，當你重開之後就不會出現這則訊息囉

```
Last login: Thu Nov  6 14:48:57 2025 from 42.77.**.90
```

接著就是說到上次登入的時間和來源的 ip address，這裏有可能是使用本地端的電腦或是跳板機，假如忘記自己是從哪裡來的可以從這邊回朔

## 小結

一直都不太知道為何連到機器的時候都會跳出這些訊息，今天有稍微認識了一下，雖然不太可能什麼都能了解，但在這種時間之餘可以了解一下平常不太會注意到的地方也蠻有趣的～
