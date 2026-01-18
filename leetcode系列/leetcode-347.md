# Leetcode 347 Top K Frequent Elements

有鑒於在這陣子有遇到需要考 leetcode 的題目，發現自己在這方面荒廢蠻久的，這陣子也有時間，要好好的的把這塊補起來，過去其實對於資料結構和演算法都略有研究，但刷題這件事真的是時間一久真的就忘的差不多，為了要好好的補起來，這次就來用 python 來完成這個系列，也好好把自己練習的過程記錄下來！

## 題目

```text
Input: nums = [1,1,1,2,2,3], k = 2
Output: [1,2]
```

我的主要想法是先統計每個數字出現的次數，然後從這些數字中找出出現次數最多的前 k 個

## Heap

<img width="1000" height="599" alt="image" src="https://github.com/user-attachments/assets/d2b27202-25cf-4704-aa32-be6583a81003" />

這時候就要先來看 heap 這個 data structure，這邊就要先看到 Python 的 heapq 提供的是 Min Heap，並且只保證 heap 最頂端的元素是最小值

## 解法

```python
from collections import defaultdict
class Solution:
    def topKFrequent(self, nums: List[int], k: int) -> List[int]:
        freq = defaultdict(int)
        for num in nums:
            freq[num] += 1
        ans = []
        for key, value in freq.items():
            heapq.heappush(ans, [value, key])
            if len(ans)>k:
                heapq.heappop(ans)
        return [key for value, key in ans]
```

## 解題思路

這次使用的是 hashmap + heap 的解法，首先是使用 haspmap 的方式來記錄每個數字出現的次數，然後再裝到 heap 裡面，然後 heap 只保留題目要求最 freq 的 k 的數量，也就是當 heap 的大小超過 k 時，就把 heap 頂端的值 pop 掉，最後留下來的值就會是題目所需要 k 的數量

## 複雜度

時間複雜度：O(Nlogk)
空間複雜度：O(N)
