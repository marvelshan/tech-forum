## K8S Lab Day_51

# Kiro Spec-Driven Development Workshop

<img width="1200" height="675" alt="image" src="https://github.com/user-attachments/assets/e7ccdf9b-eb5a-442e-8858-64fcd8fb444b" />

## 前言

昨天是高雄的 AWS 雲端大會，雖然跟台北差不多，然後小了些，但能在高雄參加到這種活動也是一個不一樣的體驗，其中最喜歡的還是參加到了 Kiro 的實作，差不多 1.5 小時，但是感覺認真要做的話應該會要花更久的時間，開始用的時候覺得 vscode 要掰掰了，真的是太好用了，使用 Spec 模式可以快速的把 prototype 做出來，而且需求越符合就越不會出錯，平常我就很常使用 copilot 和 cursor，但是這個的 AI 輔助開發真的是超出我的想像，讓 AI 輔助開法不只是在自己的電腦，而且不只是在各自開發者不同的表達方式不同，而是利用 spec 的方式來好好管理需要開發需求的功能，並且未來也可以讓非 dev 的人員能夠更快速的理解現在開發的運作，變成一個真正懂專案、會規劃、會自動化、並且也會畫架構圖的超級隊友！

## 不得不介紹這些

### Spec Mode 真的香到不行

以前我用 AI 玩 project 都是 vibe 式使用，給他一個簡單的 prompt 然後就一直使用 prompt 一直修一直改，但是最後總是發現這個架構很可怕，並且不是自己想要的，然後就把 project cancel 掉，不是有一個系統性的流程，但是現在的 spec mode 的方式就大幅的改進

1. 寫高階需求
2. Kiro 自動生成 requirements.md
3. 進到 design.md
4. 最後生成 tasks.md 然後看哪些有符合我們的需求並且一個一個執行

整個過程就像有個 tech lead 幫我把功能和流程切得整整齊齊，並且每一步都有文件可以回朔，讓過去覺得寫文件是一個麻煩的事都一次把它解決

### Steering + Hooks 讓 AI 真的懂我

Steering 就像 AI 的一份開發 paradigm，寫完之後以後都可以不用重複把一樣的事情放在 prompt 中，而且 hooks 也超厲害，在這次的 workshop 中，是只要更改 CDK 檔案，就自動叫 AWS Diagram MCP Server 幫我畫一張雲端架構圖，並存成 diagram.png，這樣未來有這個模式，跟主管或是同事溝通就如虎添翼，不用還要在那邊約個會議然後畫白板，花一堆時間溝通討論，用這個就可以加快開發的速度啊！！！

### MCP Server 不只強還可以更強

這次使用了三個 AWS 官方的 MCP，aws-documentation 問 CDK 問題直接查官方文件，aws-diagram 自動產生美到不行的架構圖，cdk-mcp-server 直接告訴我哪裡違反 Well-Architected，這些概念就像 vscode 的 extension，重點是還可以用 json 來管理，直接超級方便

## 來做個練習吧！

### 1. 先去官網下載 [Kiro](https://kiro.dev)

### 2. 建立空資料夾並用 Kiro 開啟

### 3. 左下角登入帳號（我是用 builder id 登入，現在登入的話還有 30 天的免費 50 credit）

### 4. 實際操作

在左邊有個 Kiro 的小精靈，會出現四個格子，點選 MCP 然後把官方的 MCP 加入，存檔後 Kiro 會自動啟動，之後問 AWS 問題、畫架構圖、檢查 CDK 最佳實務就直接可用

```json
{
  "mcpServers": {
    "aws-docs": {
      "command": "uvx",
      "args": ["awslabs.aws-documentation-mcp-server@latest"]
    },
    "aws-diagram": {
      "command": "uvx",
      "args": ["awslabs.aws-diagram-mcp-server"]
    },
    "cdk-mcp": { "command": "uvx", "args": ["awslabs.cdk-mcp-server@latest"] }
  }
}
```

### 5. 產生專案 Steering 檔案

點選 Generate foundation steering files，並且會產生出三個檔案 `product.md`、`tech.md`、`structure.md`

### 6. 用 Spec 模式做完整功能

在開啟新的 Spec 並且貼上需求，Kiro 這時候就會按照你的 prompt 然後生成四個檔案，並且他不會一次產生完畢，在這個過程中每個環節都可以更改完再繼續進行，這樣可以確認產生出來的去貼和我們所需要的功能，最後會產生出三個檔案 `requirements.md`、`design.md`、`tasks.md`，然後最後可以按照 task 上面的需求一個一個的去 run，並且一步一步的完成我們需要開發的程式

### 7. 一間部署到 AWS

我們前面有講到很厲害 Hook 的功能，我們在設定的時候，當我們有更改 CDK，就會自動產生架構圖，就跟 webhook 概念很像，要接到我們要做的事情，就會繼續往下一步去做，這時候就會產生我們需要的架構圖，也不用自己去畫

**這樣就完成一次簡單的實作啦！**

## [Multimodal development with Kiro](https://kiro.dev/blog/multimodal-development-with-kiro-from-design-to-done/)

<img width="3840" height="2160" alt="image" src="https://github.com/user-attachments/assets/7ca22ec9-3d82-4596-8648-fbf92d148054" />

接著要講到講者有提到的這個 Multimodal 的文章，這裏是找到資深架構師 Kandyce Bohannon 所分享的內容，如何從手繪白板的 ERD，然後直接打造出一個支援多雲的交易系統

主要他的所講述的，從把白板圖 → ERD → UML → Schema → Kubernetes YAML 一關一關進行，就直接把整套系統從資料模型到 IaC 全部生成出來，真的是超強

> The future isn't about choosing between visual design and code generation; it's about AI that can seamlessly work with both.

最後它有提到，未來不是在圖和 code 之間二選一，而是要讓 AI 都了解，可能從過去的 vibe coding 我們以為這個只是剛開始，但現在他已經可以幫專業級的架構師把整套金融系統從白板直接到完成，雖然這位架構師一定在過程中利用他的知識去寫出更好的 md 檔讓 Kiro 能夠更好的了解需求，但可以說只要你越強，Kiro 就越讓你更強大！

## 這個東西還在快速的發展！

後續講者也提到說 Kiro 現在也是非常快速的發展並改變我們現在開發的模式，可以看到[這邊文章](https://kiro.dev/changelog/spec-correctness-and-cli/)，Kiro 也更新了 v0.6 這個版本，推出了很多非常棒的功能，像是 **Spec 自動驗證（Property-based Testing）**，在寫完 Spec 之後，Kiro 會自動生成數百筆的資料去測試邊界條件，抓出理論上會出現的邏輯漏洞、**企業版上線**代表了 Kiro 已經開始要商業化，也代表要規模化的去使用了、**Kiro CLI**的誕生，我們也可以使用 terminal 去下指令去完成我們想要做到的事情

```bash
curl -fsSL https://cli.kiro.dev/install | bash
kiro "幫我用 Next.js + Tailwind 做一個 dark mode 切換的 Dashboard，部署到 Vercel"
```

**Checkpointing** 一鍵回到上一個對話節點，所有檔案瞬間還原，還會告訴你剛剛改了哪幾份檔案、建議怎麼接下去，這樣就不用害怕 AI 改了什麼不知道的東西，並也可以快速回復到原本開發可用的階段，這個真的是超級方便！**Multi-root Workspace**可以一次開 monorepo + shared library + infra CDK 資料夾，不用再開三個視窗。

看來 AWS 真的是下了很多重本要跟 VScode 對抗呢！不只是每個 Userflow 和 UI 體驗都超級好，而且小精靈好可愛，看來這個 IDE 的競爭會越來越激烈呢嗎 XD

## 總結

用過之後真的覺得他跟之前的 cursor 和 copilot 這種 AI 輔助不一樣，他是一個完整的 Agentic IDE，把原本分散的能力全部整合在一起，而且是使用人類的自然語言來寫文件寫規則然後驅動，對於那種想要快速建立測試 project 的人，Kiro 真的是救贖，並且在團隊規模擴大的時候，可以利用前人產生的文件快速的去 pickup，就不會流於過去的口耳相傳式開發，真的建議大家去把它下載來玩玩看！

另外也特別感謝 Yu-Hsiang Li 這次的 workshop，這個演講深入淺出，不只是從基礎的 prompt 開發，到後續介紹到未來的方向，我都學到相當多的東西，從一開始帶著大家理解 spec-driven 為什麼重要，到中間示範怎麼把 Kiro、MCP、CDK 這些工具串在一起，最後再拉回到團隊到底該怎麼用 AI，整個流程非常完整！講者的分享真的讓我重新思考 AI 工具的定位，也讓我覺得這條路未來一定會越來越熱門，希望之後還有更多進階的 workshop，可以玩到更多 Kiro 的深層功能，到時候我一定會再報名！

## Reference

https://github.com/lyhsiang/AWS-Kiro-Workshops

https://github.com/awsdataarchitect/kiro-best-practices/tree/main/.kiro/steering

https://www.nextlink.cloud/what-is-aws-kiro/
