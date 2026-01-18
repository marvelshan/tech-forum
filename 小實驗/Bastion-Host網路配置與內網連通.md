## K8S Lab Day_2

# Bastion-Host網路配置與內網連通

今天要繼續來開機器了，首先昨天把 bastion-host 的機器開起來，也順利地連上了，接著要做的是利用我們的堡壘機，跳到其他的機器，快速的先將其他機器開啟。
我所開的機器有
- k8s-m0, k8s-n0, k8s-n1, gitlab-runner

按照文件說明，堡壘機會 ping 不到內網的其他機器，我們必須先使用`ip addr`來查詢第二張網卡的資訊，`sudo vim /etc/netplan/50-cloud-init.yaml` 並且用這個指令去編輯 Ubuntu 系統中網路配置的檔案，在以下的第三點可以看到 enp8s0 並沒有出現在以下的 50-cloud-init 檔案當中，所以我們必須手動添加，這樣我們就可以成功的 ping 到內網的機器。

```ip addr
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
2: enp3s0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP group default qlen 1000
    link/ether fa:16:3e:e7:e6:e9 brd ff:ff:ff:ff:ff:ff
    inet 103.122.116.69/23 metric 100 brd 103.122.117.255 scope global dynamic enp3s0
       valid_lft 39481sec preferred_lft 39481sec
    inet6 2403:8ec0::f816:3eff:fee7:e6e9/64 scope global dynamic mngtmpaddr noprefixroute 
       valid_lft 2591870sec preferred_lft 604670sec
    inet6 fe80::f816:3eff:fee7:e6e9/64 scope link 
       valid_lft forever preferred_lft forever
3: enp8s0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1442 qdisc fq_codel state UP group default qlen 1000
    link/ether fa:16:3e:80:71:41 brd ff:ff:ff:ff:ff:ff
    inet 192.168.200.100/24 metric 100 brd 192.168.200.255 scope global dynamic enp8s0
       valid_lft 39481sec preferred_lft 39481sec
    inet6 fe80::f816:3eff:fe80:7141/64 scope link 
       valid_lft forever preferred_lft forever
```
```sudo vim /etc/netplan/50-cloud-init.yaml
network:
    ethernets:
        enp3s0:
            dhcp4: true
            match:
                macaddress: fa:16:3e:e7:e6:e9
            set-name: enp3s0
        #後續才加入的
        enp8s0: #後續才加入的
            dhcp4: true
            match:
                macaddress: fa:16:3e:80:71:41
            set-name: enp8s0
    version: 2
```

<img width="615" height="213" alt="截圖 2025-09-14 上午11 11 44" src="https://github.com/user-attachments/assets/b6865332-33a4-47c8-97f9-aa8ef16f689c" />

本日推薦日文歌曲：[Fujii Kaze - damn](https://youtu.be/yP7K2lXr6GA?si=wLPf19bXaOeA-vyg)

藤井風這種 chill 的歌曲真的很讚，這種日本戀愛的歌曲都可以比擬成人生，有時候自己愚蠢無濟於事，但往往那些青澀，能夠讓自己重新愛上很多東西，繼續往明天邁進～～


