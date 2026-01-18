# leetcode 53 Maximum Subarray

Given an integer array `nums`，找出一個連續子陣列（至少包含一個元素）使它的和最大，並回傳這個最大和

subarray 必須是連續的，不能跳著選，而且陣列裡可能會有負數，這會讓我們在選擇是否要斷開前面的部分，這題的解法是使用 DP 和 Greedy 的 Kadane's Algorithm

到目前位置為止的最大子陣列和，要嘛是前面累積到現在，要嘛是從現在這個數字重新開始

## 解題思路

主要用兩個變數來存放，`max_numb` 是以目前這個數字結尾的最大子陣列和，這是 DP 的狀態，`max_sum` 是整個陣列中目前找到的最大子陣列和

遍歷陣列時，對每個數字 `nums[i]` 做以下決定：

1. 要不要接上前面？  
   `max_numb + nums[i]`（自己接上） vs `nums[i]`（或是自己重新開始）

2. 然後把比較大的那個更新成新的 `max_numb`

3. 同時檢查這個新的 `max_numb` 有沒有比之前的全局最大 `max_sum` 還大，有就更新

一開始先把第一個數字當作起點，因為至少要選一個元素

```python
max_numb = nums[0]   # 以第一個數字結尾的最大和
max_sum  = nums[0]   # 目前全局最大和
```

接下來從第二個數字開始遍歷：

假設 `nums = [-2,1,-3,4,-1,2,1,-5,4]`

```
i=0: nums[0]=-2
max_numb = -2
max_sum  = -2

i=1: nums[1]=1
max_numb = max(-2+1, 1) = max(-1, 1) = 1
max_sum  = max(-2, 1) = 1

i=2: nums[2]=-3
max_numb = max(1+(-3), -3) = max(-2, -3) = -2
max_sum  = max(1, -2) = 1

i=3: nums[3]=4
max_numb = max(-2+4, 4) = max(2, 4) = 4
max_sum  = max(1, 4) = 4

i=4: nums[4]=-1
max_numb = max(4+(-1), -1) = max(3, -1) = 3
max_sum  = max(4, 3) = 4

i=5: nums[5]=2
max_numb = max(3+2, 2) = max(5, 2) = 5
max_sum  = max(4, 5) = 5

i=6: nums[6]=1
max_numb = max(5+1, 1) = max(6, 1) = 6
max_sum  = max(5, 6) = 6

i=7: nums[7]=-5
max_numb = max(6+(-5), -5) = max(1, -5) = 1
max_sum  = max(6, 1) = 6

i=8: nums[8]=4
max_numb = max(1+4, 4) = max(5, 4) = 5
max_sum  = max(6, 5) = 6
```

最終答案：6（對應子陣列 [4,-1,2,1]）

```python
def maxSubArray(self, nums: List[int]) -> int:
    max_numb = nums[0]
    max_sum = nums[0]

    for i in range(1, len(nums)):
        max_numb = max(max_numb + nums[i], nums[i])
        max_sum = max(max_sum, max_numb)
    return max_sum
```

這個解法時間複雜度 O(n)，空間複雜度 O(1)
