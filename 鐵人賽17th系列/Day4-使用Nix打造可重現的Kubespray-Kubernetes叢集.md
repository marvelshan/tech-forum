## K8S Lab Day_6

## 前言
ai 在生成 30天的時候我也沒詳細的說明我會用到哪些工具，所以後來在找資料的時候我是使用 kubespray 來去建立我的 k8s cluster，所以就沒有使用到 Minikube，那這邊還是簡單介紹一下這兩個的差別：
### Minikube
Minikube 適合在自己的電腦（macOS、Linux、Windows）快速啟動一個單節點的 k8s cluster，它會在本地端啟一個 VM，會在這個 VM 建立一個 signle-node k8s Cluster，本身不支援 HA，所以只建議實驗性操作，不太適合正式環境

### Kubespray
Kubespray 是一個基於 Ansible 的開源工具，用於在多個節點上部署生產級 k8s cluster它支援高可用性（HA）配置、複雜網路選項（如 Calico、Flannel）和多節點架構

## 使用 Nix 設置 Kubernetes 環境

因為過去有玩一些開源專案是使用到 Nix 來安裝環境配置，覺得是一個蠻有趣的工具，就拿來寫鐵人賽玩玩看～

### Nix：超嚴謹的包管理系統 + 環境管理工具
<img width="1280" height="640" alt="image" src="https://github.com/user-attachments/assets/9699cdcc-fcc8-4dae-a920-0ea35f453799" />


我們常常在開發的時候會遇到新進的同事可能用的電腦版本不一樣，或者是他本機的開發環境跟我們不同。結果就是，他在安裝一些工具的時候，會冒出一些奇怪的錯誤，而這些錯誤你自己根本沒遇過。我相信這是很多人一定都有過的經驗：「我這邊能跑，為什麼你那邊不行？」

**這時候 Nix 就登場了。**

Nix 的特點就是：它會把軟體跟它需要的依賴，全部乾乾淨淨地打包在一起，而且裝在獨立的路徑底下。所以同一台電腦上，你可以同時有：

- Python 3.8 + Django 專案
- Python 3.11 + FastAPI 專案
- 甚至再加一個 Ruby on Rails 專案

完全不會互相衝突！

---
Nix 還有一個超級大的優勢就是**可重現性**，假如今天把專案的環境寫在一個檔案（例如 flake.nix）裡，別人只要拿到這個檔案，不管是在 Linux、macOS，甚至另一台乾淨的電腦上，跑一個指令就能得到一模一樣的環境，有點像是『環境也能版控』，但 Nix 的思維跟我們平常用的 apt、brew 有點不一樣，傳統的套件管理工具是「全域裝一份」，裝了新版本就可能蓋掉舊版本，Nix 是「每一個版本都存在 /nix/store/ 裡，而且路徑都不一樣」，所以舊的版本不會消失，也不會被污染

> Nix 就是讓你從此不用再怕環境不一樣，大家都能站在同一個基準上開發

---

講完了 Nix 的一堆好處，大家可能會覺得：「哇，這東西是不是完美解決環境問題啊？」，其實也不是，它還是有一些缺點的，安裝每個環境都會花蠻久的時間，因為 Nix 會把東西都打包、重建，有些時候下載、編譯的過程會比較慢，尤其第一次裝的時候，常常要等一段時間。再來就是 Nix 還算是比較小眾，所以你在找資料或解 bug 的時候，可能會需要花更多時間去爬論壇或看官方文件，但是網路上其實已經有很多[中文的資源](https://nixos-and-flakes.thiscute.world/zh/)，還有人家已經寫好的包可以用。

---
## 實作

目標：使用 Nix 管理 Kubespray 相關工具（如 Ansible、kubectl、Python）以及其他依賴，確保環境配置可重現

### 確保有 curl 或 wget

```
sudo apt update
sudo apt install -y curl
```
### 安裝 Nix

```
curl -L https://nixos.org/nix/install | sh
```
安裝完畢就會看到以下，按照他所說的執行 `. /home/ubuntu/.nix-profile/etc/profile.d/nix.sh` 或是 `. ~/.nix-profile/etc/profile.d/nix.sh` 都可以
```
Installation finished!  To ensure that the necessary environment
variables are set, either log in again, or type

  . /home/ubuntu/.nix-profile/etc/profile.d/nix.sh

in your shell.
```

### 使用 Nix Flakes

```
mkdir -p ~/.config/nix
echo "experimental-features = nix-command flakes" >> ~/.config/nix/nix.conf
```
可試著檢查版本，確認是否安裝完畢
```
nix --version
```

### 建立 Nix 配置文件（flake.nix）
```
mkdir ~/kubespray-nix
cd ~/kubespray-nix
vi flake.nix
```

```nix
{
  description = "Kubespray environment with Nix";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05"; # 使用穩定的 Nixpkgs 版本
  };

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        buildInputs = with pkgs; [
          ansible_2_16 # 指定 Ansible 版本，與 Kubespray 相容
          python3
          python3Packages.pip
          python3Packages.netaddr
          python3Packages.jmespath
          kubectl
          kubernetes-helm # option：如果需要 Helm
        ];
        shellHook = ''
          echo "Kubespray environment with Nix is ready!"
          export ANSIBLE_CONFIG=$PWD/ansible.cfg
          export KUBECONFIG=$HOME/.kube/config
        '';
      };
    };
}
```

### 設置 Kubespray 所需的工具

進入 Nix 環境 (這個時間會花有點久)
```
nix develop
```

確認工具的版本是否正確
```
ansible --version
kubectl version --client
python3 --version
```

### 把 Kubespray 專案抓下來

```
git clone --depth 1 --branch v2.28.0 https://github.com/kubernetes-sigs/kubespray.git
```

```
cp -rfp inventory/sample inventory/mycluster
```
接著要編輯 `inventory/mycluster/inventory.ini` 對應的 server 資訊，

```
[all]
k8s-m0 ansible_host=192.168.200.*** ansible_user=ubuntu
k8s-n0 ansible_host=192.168.200.*** ansible_user=ubuntu
k8s-n1 ansible_host=192.168.200.*** ansible_user=ubuntu

[kube_control_plane]
k8s-m0

[etcd]
k8s-m0

[kube_node]
k8s-n0
k8s-n1

[calico_rr]

[k8s_cluster:children]
kube_control_plane
kube_node
calico_rr
```

### 啟動 Kubesprau
```
ansible-playbook -i inventory/mycluster/inventory.ini --private-key=~/private.key --become --become-user=root cluster.yml
```

### 確認是否安裝完成
```
kubectl get nodes
```
```
No resources found in default namespace.
```
這樣就成功啦～

## Reference
[使用 Kubespray 建立自己的 K8S（一）](https://ithelp.ithome.com.tw/articles/10294526)

[NixOS 与 Flakes一份非官方的新手指南](https://nixos-and-flakes.thiscute.world/zh/)
(網路上很多人都不建議看 Nix 官方的資訊)
