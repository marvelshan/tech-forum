## K8S Lab Day_34

# Jenkins 的 CICD 部署之路

## 前言

想說最近要把履歷打開來了，要開始複習前以前所使用過的技術，順便來繼續鑽研有沒有更有趣的做法可以應用

<img width="796" height="398" alt="image" src="https://github.com/user-attachments/assets/729a078e-1636-4bac-a3d1-e6738ed76e79" />

## CICD 流程

```
Bitbucket Repo
   ↓ (Webhook or Manual Trigger)
Jenkins Pipeline
   ↓
[1] Checkout 選定版本 (branch/tag)
   ↓
[2] Build Docker Image
   ↓
[3] Push to GCR
   ↓
[4] Deploy to GCP Service (Cloud Run / GKE / VM)
```

這邊所使用的技術是 Jenkins，因為過去這個技術較為大宗吧，因為過去主要都是使用網頁的方式來讓開發者，也就是那時候的我來去部署相對應的版本，但是我這邊沒有那時候的截圖，所以我就使用 jenkinsfile 的方式介紹那時候的使用，其實 jenkins 的流程可以把它理解為一行一行的指令讓他去執行，首先我們要選用我們要使用的版號，也就是我們每開發完成後我們都會打版，不管是打 tag (lightweight tag) `git tag <標籤名>`，或是 `git tag -a v<新版號> -m "<標籤訊息>"` 去打 release version 的版本號，然後把這個版本推到我們所需要拉下來的存放位置，像是 bitbucket 或是 gitlab 或是 github 等等，然後下一步就是要將 Image 丟在 GCR 上面，最後就是部署了

```Jenkinsfile
pipeline {
  agent any

  environment {
    PROJECT_ID = 'your-gcp-project-id'
    REGION = 'asia-east1'
    REPO_NAME = 'my-node-service'
    IMAGE = "gcr.io/${PROJECT_ID}/${REPO_NAME}"
    GCLOUD_CREDENTIALS = credentials('gcp-service-account') // Jenkins Credential ID
  }

  parameters {
    string(name: 'GIT_TAG', defaultValue: 'main', description: '要部署的 Git branch 或 tag')
  }

  stages {
    stage('Checkout Code') {
      steps {
        git branch: "${params.GIT_TAG}",
            credentialsId: 'bitbucket-credentials',
            url: 'git@bitbucket.org:your-team/your-repo.git'
      }
    }

    stage('Build Docker Image') {
      steps {
        sh """
          echo "${GCLOUD_CREDENTIALS}" > /tmp/key.json
          gcloud auth activate-service-account --key-file=/tmp/key.json
          gcloud auth configure-docker -q
          docker build -t ${IMAGE}:${GIT_TAG} .
        """
      }
    }

    stage('Push Image to GCR') {
      steps {
        sh """
          docker push ${IMAGE}:${GIT_TAG}
        """
      }
    }

    stage('Deploy to Cloud Run') {
      steps {
        sh """
          gcloud run deploy ${REPO_NAME} \
            --image ${IMAGE}:${GIT_TAG} \
            --region ${REGION} \
            --platform managed \
            --allow-unauthenticated
        """
      }
    }
  }

  post {
    always {
      cleanWs()
    }
  }
}
```

接著就是測試啦，看是要跑測試還是手動測試，完成測試就可以從 staging 版本準備上到 beta 版本囉～

## 結合 Istio

接著我要來試試看結合我所學習到的 Istio 來學習，原本是已經把 image 放在 GCR 上了，接著我想要做的事是將這個 image deploy to Beta Namespace，然後再用 VirtualService 來去控制 traffic，首先我們先預設原本的設置是像是以下，有兩個 deployment 的設定，搭配一個 ns 為 beta 的 service

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: website-v1
  namespace: beta
  labels:
    app: website
    version: v1
spec:
  replicas: 3
  selector:
    matchLabels:
      app: website
      version: v1
  template:
    metadata:
      labels:
        app: website
        version: v1
    spec:
      containers:
        - name: website
          image: gcr.io/YOUR_PROJECT/website:v1
          ports:
            - containerPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: website-v2
  namespace: beta
  labels:
    app: website
    version: v2
spec:
  replicas: 3
  selector:
    matchLabels:
      app: website
      version: v2
  template:
    metadata:
      labels:
        app: website
        version: v2
    spec:
      containers:
        - name: website
          image: gcr.io/YOUR_PROJECT/website:v2
          ports:
            - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: website
  namespace: beta
spec:
  selector:
    app: website
  ports:
    - port: 80
      targetPort: 8080
```

接著就是 VirtualService / DestinationRule，負責流量切換

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: website
  namespace: beta
spec:
  host: website
  subsets:
    - name: v1
      labels:
        version: v1
    - name: v2
      labels:
        version: v2
---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: website
  namespace: beta
spec:
  hosts:
    - website
  http:
    - route:
        - destination:
            host: website
            subset: v1
          weight: 90
        - destination:
            host: website
            subset: v2
          weight: 10
```

但其實也可以把這個流程透過 `kubectl patch` 來加入到 jenkins pipeline 中

```Jenkinsfile
stage('Progressive Rollout') {
  steps {
    sh """
      kubectl patch virtualservice website -n beta --type='json' \
        -p='[{"op": "replace", "path": "/spec/http/0/route/0/weight", "value": 70},
             {"op": "replace", "path": "/spec/http/0/route/1/weight", "value": 30}]'
      sleep 300
      kubectl patch virtualservice website -n beta --type='json' \
        -p='[{"op": "replace", "path": "/spec/http/0/route/0/weight", "value": 50},
             {"op": "replace", "path": "/spec/http/0/route/1/weight", "value": 50}]'
    """
  }
}
```

## 為什麼「不直接用 Jenkins」，而要引入 Istio

這時候一定會跳出這個問題在腦袋中，Jenkins 確實能完成部署自動化，但是它的功能跟 istio 不太相同，jenkins 只需要知道 apply 哪些檔案，不太管之後的 traffic management，但假如要做到也不是不行，會覺得有點奇怪而已

假如要針對於 traffic management 的話，istio 還是專業大哥啦，假如多版本運行，在 istio 還是有很好的控管方式來更簡單的去運作他，也可以更好的使用 istio 背後的可觀測性來去有效的觀察服務的運作，相對於安全性也是 istio 可以使用 mTLS 來去管理各個服務相互的流量加密

## 那假如不使用 jenkins 呢？

這時候就要來看一下 Argo CD 啦！Argo CD 是近年非常流行的 GitOps 工具，特色是一切以 Git 為中心，把應用程式的 Kubernetes YAML、Helm Chart、或 Kustomize 設定放在 Git 裡，Argo CD 會自動偵測變更，並讓叢集狀態與 Git repo 保持一致，這樣的好處是不再需要手動執行 kubectl apply，每次的版本變更、回滾都有 Git commit 記錄，在 UI 上可以看到目前叢集的實際狀態 和 Git 狀態是否一致。當然還有 FluxCD、GitLab CI/CD、GitHub Actions 等等的部署工具，各個都有他們優點～

## Argo Rollouts

<img width="1280" height="710" alt="image" src="https://github.com/user-attachments/assets/f2d51d4d-c831-4425-8b15-dfd57d0760f1" />

Argo CD 本身主要負責 GitOps 同步，也就是讓 cluster 的狀態與 Git 版本保持一致，但當我們想要做到藍綠部署、自動回滾這類更進階的 Progressive Delivery 時，單靠 Deployment 資源就不太夠用了，這時候就可以使用 Argo Rollouts，Argo Rollouts 是一個 k8s controller，擴充原本的 Deployment，做到更精細的 traffic management，想當然他也是一個 CRD，每個階段都能設定 pause，讓團隊觀察指標，如果某階段監控失敗（例如延遲上升、錯誤率飆高），可以自動回滾

<img width="390" height="263" alt="image" src="https://github.com/user-attachments/assets/6776aee5-a078-4612-b6c7-c8fa7f481676" />

<img width="1024" height="363" alt="image" src="https://github.com/user-attachments/assets/984878c2-6d84-4f41-ba14-5e1cba28d75f" />


```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: abort-retry-promote
spec:
  strategy:
    canary:
      steps:
        - setWeight: 50
        - pause: { duration: 3s }
  selector:
    matchLabels:
      app: abort-retry-promote
  template:
    metadata:
      labels:
        app: abort-retry-promote
    spec:
      containers:
        - name: abort-retry-promote
          image: nginx:1.19-alpine
          resources:
            requests:
              memory: 16Mi
              cpu: 1m
```

想當然他也可以搭配 istio 來針對於流量去做到管理，這個範本來自 [istio-ecosystem/admiral](https://github.com/istio-ecosystem/admiral/blob/master/install/sample/overlays/rollout-bluegreen/greeting.yaml)，而我們這邊就可以看出來 Argo Rollouts 是一個 Deployment Controller，在 spec.strategy 中，我們可以看到使用的是 blueGreen 策略，activeService 指定哪個 Service 是目前 prod 版本，當 Rollout 推升新版成功時，controller 會自動更新這個 Service 的 selector，讓它指向新版本的 ReplicaSet，previewService 讓新版（尚未上線）的 Pod 可以被預覽，這在真實場景非常實用，因為團隊可以先測試、驗證，再決定是否被推上線，這也就是 Blue-Green 的 `Green Stack`，`autoPromotionEnabled: false` 代表系統不會自動切換流量，需要人工確認完 `kubectl argo rollouts promote greeting` 才會上線，避免意外事故

接著是 `template.metadata.annotations` 這邊的 admiral 是 stage 環境，用於多環境（multi-cluster / multi-tenant）場景的分層流量控制，並啟用 Istio 的 Sidecar 注入，讓 Envoy proxy 能接管網路流量，讓 VirtualService、DestinationRule 能介入運作，這裏可以理解成 **Rollouts 負責控制 Deployment 邏輯，Istio 負責實際流量導向**

雖然後面兩個 Service 的 selector 看起來相同，但 Argo Rollouts Controller
會在實際執行過程中自動修改 selector，讓 active 與 preview 分別指向不同版本的 ReplicaSet

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: greeting
  labels:
    identity: greeting
spec:
  replicas: 1
  selector:
    matchLabels:
      app: greeting
  template:
    metadata:
      annotations:
        admiral.io/env: stage
        sidecar.istio.io/inject: "true"
      labels:
        app: greeting
        identity: greeting.bluegreen
    spec:
      containers:
        - image: nginx
          name: greeting
          ports:
            - containerPort: 80
          volumeMounts:
            - mountPath: /etc/nginx
              name: nginx-conf
              readOnly: true
            - mountPath: /var/log/nginx
              name: log
          resources:
            requests:
              cpu: "10m"
              memory: "50Mi"
            limits:
              cpu: "20m"
              memory: "75Mi"
      volumes:
        - configMap:
            items:
              - key: nginx.conf
                path: nginx.conf
            name: nginx-conf
          name: nginx-conf
        - emptyDir: {}
          name: log
  strategy:
    blueGreen:
      # activeService specifies the service to update with the new template hash at time of promotion.
      # This field is mandatory for the blueGreen update strategy.
      activeService: rollout-bluegreen-active
      # previewService specifies the service to update with the new template hash before promotion.
      # This allows the preview stack to be reachable without serving production traffic.
      # This field is optional.
      previewService: rollout-bluegreen-preview
      # autoPromotionEnabled disables automated promotion of the new stack by pausing the rollout
      # immediately before the promotion. If omitted, the default behavior is to promote the new
      # stack as soon as the ReplicaSet are completely ready/available.
      # Rollouts can be resumed using: `kubectl argo rollouts resume ROLLOUT`
      autoPromotionEnabled: false
---
kind: Service
apiVersion: v1
metadata:
  name: rollout-bluegreen-active
  labels:
    app: greeting
    identity: greeting.bluegreen
  namespace: sample
spec:
  ports:
    - name: http
      port: 80
      targetPort: 80
  selector:
    app: greeting

---
kind: Service
apiVersion: v1
metadata:
  name: rollout-bluegreen-preview
  labels:
    app: greeting
    identity: greeting.bluegreen
  namespace: sample
spec:
  ports:
    - name: http
      port: 80
      targetPort: 80
  selector:
    app: greeting
```

## 那為什麼要搭配 Istio？

雖然 Argo Rollouts 本身可以操作 k8s Service 的 selector，但若要在 `active 流量`與`preview 流量`之間做更 fine-grained 的流量控制，像是只讓 5% 的真實使用者流量先導入新版、根據 header 或 cookie 導向不同版本、自動監測延遲、錯誤率、流量峰值後再決定是否 promote 等等的，都需要 Istio 的 VirtualService + DestinationRule

<img width="867" height="605" alt="image" src="https://github.com/user-attachments/assets/23b9541a-fcb3-4718-bd96-16973da1c6df" />

## 部署方式與比較

這時候講到部署，勢必要比較一下各種的部署方式吧，才不愧發明這些部署方式還有那時候遇到挑戰的先人

| 部署方式                  | 主要特徵                                                 | 優點                     | 缺點                             | 適用場景                   |
| ------------------------- | -------------------------------------------------------- | ------------------------ | -------------------------------- | -------------------------- |
| **Recreate**              | 先刪舊版本，再部署新版本                                 | 簡單直接                 | 有停機時間，使用者體驗不佳       | 開發或非關鍵系統           |
| **Rolling Update**        | 舊版本逐步被新版本替換                                   | 無明顯停機，K8s 原生支援 | 難以精準控制流量比例             | 中小型系統的自動部署       |
| **Blue-Green Deployment** | 兩組完整環境（Blue 舊版 / Green 新版）並行，切換入口流量 | 切換快速、可回滾         | 成本高（雙倍資源）               | 關鍵應用、穩定需求高       |
| **Canary Deployment **    | 逐步導入流量到新版本觀察表現                             | 可觀察性強、低風險       | 需要額外流量控制機制（如 Istio） | 高可用、高併發應用         |
| **A/B Testing**           | 基於使用者屬性（如 Header、Cookie）動態分流              | 支援實驗與評估           | 較複雜，需額外邏輯               | ML 模型、推薦系統、UI 測試 |

## 總結

看完以上的部署方式，覺得自己有點像是拿著各種工具來面對各種實際的狀況，每個工具都有他實用的地方，組裝起來可以打片天下無敵手的感覺，感覺啦ＸＤ，原本沒想到要寫那麼多，默默的也把這些做完，看著前方的海，在這邊敲鍵盤真的是很爽ＸＤ

![IMG_1250](https://github.com/user-attachments/assets/86901745-567e-4b2e-9291-8e3c68299f7a)

## Reference

https://github.com/PacktPublishing/Hands-On-Microservices-with-Spring-Boot-and-Spring-Cloud

https://ithelp.ithome.com.tw/articles/10305105

https://github.com/istio-ecosystem/admiral/blob/master/install/sample/overlays/rollout-bluegreen/greeting.yaml

https://tachingchen.com/tw/blog/kubernetes-rolling-update-with-deployment/
