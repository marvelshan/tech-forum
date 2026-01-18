## K8S Lab Day_1

# CNTUG借機器與SSH連線設定

這次的實驗是跟 CNTUG 借了機器，也感謝 Tico 的協助幫我完成審核流程，接著就開始了熟悉介面的地獄。

首先我去到了跟文件不同的 domain https://console.cloudnative.tw/ 雖然也可以使用，但順暢度很明顯就差很多，一整個就很崩潰，還是硬著頭皮把它設定完畢，但是最尷尬的是 ssh 沒辦法連上，試了很多次都是 `ssh: connect to host 103.122.116.＊＊＊ port 22: Operation timed out` 超級崩潰。
<img width="1866" height="734" alt="截圖 2025-09-13 晚上11 17 12" src="https://github.com/user-attachments/assets/c1cb0cea-7a55-47ee-882e-594dcd665cbe" />

但死馬要當活馬醫，就瘋狂的去網路上找資料，最後去翻 email 發現我要去的 domain 是 https://openstack.cloudnative.tw/ 真的是 God Damn 直接把機器重開使用，最重要的地方是在網路卡的部分要一張一張加，開啟後按下手動附加～最後勝利式的成功連上了，今天先完成到這邊吧～
<img width="1415" height="727" alt="截圖 2025-09-13 晚上11 17 42" src="https://github.com/user-attachments/assets/427bd209-e30a-446e-85dc-189ddf37ae79" />

備註：這個單純是我記錄 K8S 的過程，不是鐵人賽的文章，接下來會應該會去寫，但還是要先把最艱難的環境處理完QQ

每天都來推薦一首自己喜歡的日文歌：[香水 / 瑛人](https://youtu.be/9MjAJSoaoSo?si=fcl6s8O6I8kcvVDD)

