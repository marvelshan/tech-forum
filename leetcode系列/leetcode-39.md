# leetcode 39 Combination Sum

Given an array of distinct integers `candidates` and a target integer `target`, return a list of all unique combinations of `candidates` where the chosen numbers sum to `target`. You may return the combinations in any order.

這邊要注意到題目有說到，同一數字可 unlimited times，而且題目保證 candidates 中的數字都是正整數且互不相同

這題要找出所有「和為 target」的組合，而且數字可以重複使用，順序不影響結果，所以 [2,2,3] 和 [2,3,2] 算同一種，這類問題典型的做法就是 Backtracking，透過「選擇 → 遞迴 → 撤銷選擇」的模式來枚舉所有可能

## 解題思路

我們用一個輔助函數 `backtracking_helper` 來進行 backtrack，主要參數有：

- `temp_list`：目前正在建構的組合
- `remainder`：還剩下的 target
- `index`：從 candidates 的哪個位置開始選（為了避免重複組合）

backtrack 的決策樹長這樣：

- 如果 `remainder == 0` → 找到一個有效組合，把 temp_list 複製一份加進答案
- 如果 `remainder < 0` → 超過了，沒意義，直接返回
- 否則，從 `index` 開始往後選，每選一個數字就：
  1. 把數字加進 temp_list
  2. 剩餘目標減掉這個數字，index 不會變，因為可以重複使用
  3. 遞迴下去繼續選
  4. 遞迴結束後把剛加的數字彈掉（backtrack）

假設 `candidates = [2,3,6,7]`，`target = 7`

```
初始：temp_list = [], remainder = 7, index = 0

選擇 2 (i=0)
  temp_list = [2], remainder = 5
    再選 2 (i=0)
      temp_list = [2,2], remainder = 3
        再選 2 (i=0)
          temp_list = [2,2,2], remainder = 1 → <0 Pruning
        再選 3 (i=1)
          temp_list = [2,2,3], remainder = 0 → 找到 [2,2,3]
      回溯 pop 3
    再選 3 (i=1)
      temp_list = [2,3], remainder = 2 → < 3 無解
    再選 6 (i=2) → 5-6=-1 Pruning
  回溯 pop 2

選擇 3 (i=1)
  temp_list = [3], remainder = 4
    再選 3 (i=1)
      temp_list = [3,3], remainder = 1 → <0 Pruning
    再選 6 (i=2) → 4-6<0 Pruning
  回溯 pop 3

選擇 6 (i=2)
  temp_list = [6], remainder = 1 → 下一層都 <0，無解

選擇 7 (i=3)
  temp_list = [7], remainder = 0 → 找到 [7]
```

所以最終答案會包含：`[[2,2,3], [7]]`

```python
class Solution:
    def combinationSum(self, candidates: List[int], target: int) -> List[List[int]]:
        self.res = []

        def backtracking_helper(candidates, temp_list, remainder, index):
            if remainder == 0:
                self.res.append(temp_list[:])  # 找到解，shallow copy 一個結果
                return
            if remainder < 0:
                return

            for i in range(index, len(candidates)):
                temp_list.append(candidates[i])
                # 注意這裡傳 i 而不是 i+1，因為可以重複使用同一數字
                backtracking_helper(candidates, temp_list, remainder - candidates[i], i)
                #  這裏就繼續 recursive
                temp_list.pop()

        backtracking_helper(candidates, [], target, 0)
        return self.res
```

這個解法時間複雜度是 O(N^(T/M + 1)) N 是 candidates 長度，T 是 target，M 是 candidates 中最小數字，屬於指數級，但因為有 Pruning，在 LeetCode 上通常能通過
