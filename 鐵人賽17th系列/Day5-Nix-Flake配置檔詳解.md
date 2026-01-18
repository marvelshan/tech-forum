## K8S Lab Day_7

## 前言
昨天要提到 Nix 但卻沒有詳細的介紹詳細的配置內容，今天就來介紹一下配置內容吧！

### Nix 的配置檔：`flake.nix`

昨天有說到 Nix 的特色就是可以把「環境」也寫進檔案裡，然後每個人只要拿到同樣的檔案，就能重現一模一樣的環境，這個檔案就叫做 **`flake.nix`**。

我們來看一個昨天的範例：

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

---

### 一段一段來解釋

1. **description**

   ```nix
   description = "Kubespray environment with Nix";
   ```

   就是一個描述，讓你知道這個 flake 是幹嘛用的

2. **inputs**

   ```nix
   inputs = {
     nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
   };
   ```

   Nix 的套件來源（相當於 repo），這裡我們指定用 **24.05 穩定版的 nixpkgs**，確保環境可重現

3. **outputs**

   ```nix
   outputs = { self, nixpkgs }: ...
   ```

   定義「我這個 flake 要產生什麼東西」，在這裡，我們要輸出的是 **一個開發環境（devShell）**。

4. **system & pkgs**

   ```nix
   system = "x86_64-linux";
   pkgs = import nixpkgs { inherit system; };
   ```

   指定系統架構（這裡是 Linux x86\_64），然後把 nixpkgs 匯入成套件集合 `pkgs`

5. **devShells**

   ```nix
   devShells.${system}.default = pkgs.mkShell { ... };
   ```

   這裡定義「我想要進入的開發環境」

   * `buildInputs` → 我需要哪些工具？
     這裡裝了 Ansible、Python、kubectl、Helm 等，專門為 Kubespray 準備
   * `shellHook` → 進入這個環境的時候要跑的指令
     例如：輸出一個提示訊息，還有設定 `ANSIBLE_CONFIG`、`KUBECONFIG` 環境變數

---

### 實際用法

有了這個 `flake.nix` 之後，每個人只要跑：

```bash
nix develop
```

就會進到一個一模一樣的環境裡，裡面已經有：

* Ansible 2.16（跟 Kubespray 相容）
* Python + 必要套件（pip、netaddr、jmespath）
* kubectl、Helm

這樣就不用擔心「有人 Ansible 版本不同、有人少裝 Python 套件」之類的問題。

---

接著我們來看比較用 nix 和 沒用的差別，我們利用 `ansible --version` 來查看執行檔案路徑與版本

```bash
ansible --version
```

#### 在未執行 `nix develop`

```bash
ansible [core 2.16.14]
  config file = None
  configured module search path = ['/home/ubuntu/.ansible/plugins/modules', '/usr/share/ansible/plugins/modules']
  ansible python module location = /home/ubuntu/.local/lib/python3.10/site-packages/ansible
  ansible collection location = /home/ubuntu/.ansible/collections:/usr/share/ansible/collections
  executable location = /home/ubuntu/.local/bin/ansible
  python version = 3.10.12 (main, Aug 15 2025, 14:32:43) [GCC 11.4.0] (/usr/bin/python3)
  jinja version = 3.0.3
  libyaml = True
```

#### 執行 `nix develop`

```bash
ansible [core 2.16.5]
  config file = None
  configured module search path = ['/home/ubuntu/.ansible/plugins/modules', '/usr/share/ansible/plugins/modules']
  ansible python module location = /nix/store/7ic5w44dss0x88lx4r4c7k18z064jxc1-python3.11-ansible-core-2.16.5/lib/python3.11/site-packages/ansible
  ansible collection location = /home/ubuntu/.ansible/collections:/usr/share/ansible/collections
  executable location = /nix/store/7ic5w44dss0x88lx4r4c7k18z064jxc1-python3.11-ansible-core-2.16.5/bin/ansible
  python version = 3.11.10 (main, Sep  7 2024, 01:03:31) [GCC 13.2.0] (/nix/store/s0p1kr5mvs0j42dq5r08kgqbi0k028f2-python3-3.11.10/bin/python3.11)
  jinja version = 3.1.5
  libyaml = True
```

從上面結果可以看到，**在沒有用 Nix 的情況下**，Ansible 是裝在使用者目錄下（`~/.local/bin/ansible`，模組在 `~/.local/lib/python3.10/...`），這代表每個人可能因為安裝方式不同、Python 版本不同，導致環境不一致

但**用了 `nix develop` 之後**，執行檔和模組路徑都變成 `/nix/store/*****`。這個路徑的意思是：

* `/nix/store` 是 Nix 把所有套件存放的地方，每個套件都會有一個獨一無二的 hash 路徑（像 `7ic5w44dss0x88lx4r4c7k18z064jxc1`）
* 這確保了套件版本、依賴、編譯方式完全固定，不會因為系統差異而出錯
* 換句話說，不管誰執行 `nix develop`，進去之後看到的 `ansible` 都是**同一份、同一版本、同一環境**

---

當我們用 Nix 建立專案時，會出現兩個檔案，拿較常看到的 Node.js 來比較，flake.nix 就像 package.json，只負責描述「我要用哪些套件、哪些版本範圍」，就像 package-lock.json 或 yarn.lock，把實際解析到的版本精準鎖下來，保證大家裝起來的環境完全一致，Node.js 裡面會生一個超肥的 node_modules，塞滿各種套件；但在 Nix 裡，這些套件其實都放在 `/nix/store`，然後透過 flake.lock 指向正確的版本，環境乾淨很多

那我們把 `flake.nix` 寫好後，假如我們後續要更新呢？這時候，就會用到以下指令：

* **更新 flake.lock**（更新所有 inputs）：

  ```bash
  nix flake update
  ```
* **只更新特定 input**

  ```bash
  nix flake update home-manager
  ```
* **部署新配置**（如果配置在 `/etc/nixos` 可以省略 `--flake .`）：

  ```bash
  sudo nixos-rebuild switch --flake .
  ```
* **同時更新 flake.lock 並部署**（相當於先 `nix flake update`）：

  ```bash
  sudo nixos-rebuild switch --recreate-lock-file --flake .
  ```

---

在日常開發中，還會常見一些跟 flake 有關的指令：

* **編譯某個 target**：
當我們的專案越來越大時，我們通常都會針對於某些 target 去做編譯，在 flake.nix 裡，我們可以定義多個 output（例如 packages.index、packages.target、packages.app），這時候就能用以下指令只 build 需要的部分：
  ```bash
  nix build .#index -L
  nix build .#target -L
  nix build .#app -L
  ```

* 列出 flake 能提供的所有 target：

  ```bash
  nix flake show
  ```

* 在 commit 前跑 formatter / pre-commit hooks：

  ```bash
  nix fmt
  nix run .#pre-commit -L
  ```


我們就可以看到這就是 Nix 的威力啊，原本大家各自亂裝套件就像駕駛員直接赤手空拳硬幹使徒，一下版本不合、一下缺套件，結果都暴走！但有了 Nix，就好像坐進 EVA，所有武裝、同步率、裝備都固定下來，無論誰上去，都是同一台 01 號機，能穩定出擊！

## Reference

https://nixos-and-flakes.thiscute.world/zh/nixos-with-flakes/update-the-system

https://github.com/unionlabs/union
