# Backlog

- Nix 的版本會影響到 isitoctl 下載的版本，那今天我不用 unstable 又想使用新的版本怎麼辦呢？

# 常用指令

```bash
# 把外部流量導到 Ingress Gateway
kubectl port-forward -n istio-ingress svc/istio-ingressgateway 8080:80

# 同時也把 Prometheus 開到本地 9090
kubectl -n istio-system port-forward svc/prometheus 9090:9090
```

```bash
curl -I http://localhost:8080/productpage
```

```bash
kubectl get pods -n istio-system | grep istiod
```

```bash
ssh -i private.key ubuntu@192.168.200.249
```

檢查遠端主機 103.122.116.69 的 22 埠是否正在監聽並接受連線

```bash
nc -zv 103.122.116.69 22
```

```yaml
# sudo vim /etc/netplan/50-cloud-init.yaml
enp8s0:
  dhcp4: true
  dhcp6: true
  match:
    macaddress: fa:16:3e:80:71:41
  set-name: enp8s0
```

```bash
sudo netplan apply
```

檢查是否有殘留進程

```bash
ps aux | grep nix
```

## 插入網卡

```bash
ip addr
```

先查詢網卡的名稱和 Mac address

```bash
3: enp*s0: <BROADCAST,MULTICAST> mtu 1442 qdisc noop state DOWN group default qlen 1000
    link/ether fa:16:3e:80:71:** brd ff:ff:ff:ff:ff:ff
```

然後將 enp\*s0 加入設定

```bash
sudo vim /etc/netplan/50-cloud-init.yaml
```

```yaml
enp*s0:
  dhcp4: true
  match:
    macaddress: fa:16:3e:80:71:** # 填寫 enp*s0 的 Mac address
  set-name: enp*s0
```

接著套用上去就可以了

```bash
sudo netplan apply
```
