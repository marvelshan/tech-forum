# leetcode 2007 Find Original Array From Doubled Array

給一個 array `changed`，裡面每個值都是原本 array 中的元素乘以 2 後的結果，也就是 changed[i] = 2 \* original[j] 的排列）
還原出可能的原始陣列 `original`，並回傳它，如果有多組解，回傳任意一組都可以，如果不可能還原，回傳空陣列

這邊要注意，原本 array 的長度一定是 changed 的一半，並且 changed 中的每個元素都必須被正確對應到 2 \* original 中的某個值

## 解題思路

這題不能單純用 set，因為可能有重複數字，像是 [2,4,4,8] 可以對應 [1,2,2,4]，所以需要用到 Counter 的 hashmap 來記錄每個數字出現的次數，然後按照從小到大依序處理

1. 先檢查 changed 長度是不是偶數，如果不是就不會是答案
2. 把 changed 轉成 Counter，方便 O(1) 查詢與減次數
3. 把 changed 排序，從小到大處理，這可以保證先處理小的數字
4. 遍歷每個數字 num：
   - 如果這個 num 已經被用完（count == 0），直接跳過
   - 檢查 num+num 是否還有剩（find_array[num+num] > 0）
     - 如果沒有 -> 代表這個 num 找不到對應的倍數 -> 回傳 []
   - 把 num 加入結果
   - 減掉 num 一次
   - 減掉 2\*num 一次

這樣從小到大處理，就能確保每個小的數字先被當成原本數字，並留下較大的數字給後面匹配

```
假設 changed = [1,3,4,2,6,8]

排序後：[1,2,3,4,6,8]
Counter 初始：{1:1, 2:1, 3:1, 4:1, 6:1, 8:1}

- num=1 -> count[1]=1 >0, count[2]=1 >0
  -> result=[1], count[1]-=1 ->0, count[2]-=1 ->0

- num=2 -> count[2]=0，跳過

- num=3 -> count[3]=1 >0, count[6]=1 >0
  -> result=[1,3], count[3]-=1 ->0, count[6]-=1 ->0

- num=4 -> count[4]=1 >0, count[8]=1 >0
  -> result=[1,3,4], count[4]-=1 ->0, count[8]-=1 ->0

- num=6 -> count[6]=0，跳過

- num=8 -> count[8]=0，跳過

最終 result = [1,3,4]，驗證：2\*[1,3,4] = [2,6,8] + [4] 剛好湊成原陣列
```

```python
from collections import Counter

class Solution:
    def findOriginalArray(self, changed: List[int]) -> List[int]:
        if len(changed) % 2 == 1:
            return []
        find_array = Counter(changed)
        result = []
        changed.sort()
        for num in changed:
            if find_array[num] == 0:
                continue
            if find_array[num + num] == 0:
                return []
            result.append(num)
            find_array[num] -= 1
            find_array[num + num] -= 1
        return result
```

這個解法時間複雜度 O(n log n)，空間複雜度 O(n)
