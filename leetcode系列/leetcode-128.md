# leetcode 128 Longest Consecutive Sequence

Given an unsorted array of integers `nums`，找出其中最長的連續序列，並且必須是連續的數字，例如 1,2,3,4 的長度是 4，並回傳這個長度

這題的要求是時間複雜度必須是 O(n)，不能用排序，因為排序會變成 O(n log n)），這邊要注意的是陣列裡可能有重複數字，但重複不影響連續序列的計算，像是例如 [1,1,1] 還是只有長度 1，這題解題的關鍵就是使用 Hash Set 來達成 O(1) 的查詢，然後只從序列的起點開始往右數

## 解題思路

1. 先把所有數字放進一個 Hash Set
2. 遍歷 set 中的每個數字
   - 什麼是起點？就是前面一個數字（num-1）不存在的數字
   - 如果 num-1 已經在 set 裡，代表這不是起點，直接跳過就好，因為會被前面的數字算到
3. 一旦找到起點，就從這個數字開始往右數（num + 1, num + 2, ...），一直數到斷掉為止
4. 記錄過程中最長的連續長度

假設 `nums = [100, 4, 200, 1, 3, 2]`

轉成 set 後：`{1, 2, 3, 4, 100, 200}`

```
num = 100 -> 99 不在 set -> 是起點
  -> 往右數：100 在 -> 101 不在 -> 長度 = 1
  -> result = max(0, 1) = 1

num = 4 -> 3 在 set -> 不是起點，跳過

num = 200 -> 199 不在 -> 是起點
  -> 往右數：200 在 -> 201 不在 -> 長度 = 1
  -> result = max(1, 1) = 1

num = 1 -> 0 不在 -> 是起點
  -> 往右數：1 在 -> 2 在 -> 3 在 -> 4 在 -> 5 不在
  -> 長度 = 4
  -> result = max(1, 4) = 4

num = 3 -> 2 在 set -> 不是起點，跳過

num = 2 -> 1 在 set -> 不是起點，跳過

最終答案：4（對應序列 [1,2,3,4]）
```

```python
def longestConsecutive(self, nums: List[int]) -> int:
    if not nums:
        return 0
    hash_set = set(nums)
    result = 0
    for num in hash_set:
        if (num - 1) not in hash_set:
            current_len = 0
            while (num + current_len) in hash_set:
                current_len += 1
            result = max(result, current_len)
    return result
```

這個解法時間複雜度 O(n)，空間複雜度 O(n)
