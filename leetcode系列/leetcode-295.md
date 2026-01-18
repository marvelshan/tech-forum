# leetcode 295 Find Median from Data Stream

Design a data structure that supports the following two operations:

- addNum(int num) - Add an integer number from the data stream to the data structure.
- findMedian() - Return the median of all elements so far.

這題是要設計一個資料結構，能夠從 data flow 中加入數字，並且隨時可以找到目前所有數字的中位數，如果總數是奇數，中位數就是中間那個，如果是偶數，就是中間兩個的平均

特別要注意的一點是，假如效率要高，不能每次加數字都排序整個列表，那樣太慢，解題的關鍵就是使用兩個堆 priority queue：一個 max-heap 來存較小的那一半數字（左半邊），一個 min-heap 來存較大那一半（右半邊）。這樣可以保持平衡，讓 addNum 是 O(log n)，findMedian 是 O(1)

其實這就像維持兩個堆，讓 max-heap 的頂端是左半邊的最大值，min-heap 的頂端是右半邊的最小值，這樣中位數就在這兩個頂端之間

## 解題思路

一開始在 **init** 設定兩個空列表：max_heap 用來模擬 max-heap（存負值，因為 Python 的 heapq 是 min-heap），min_heap 是 min-heap。

在 addNum：

- 先把 -num push 進 max_heap（這樣頂端是負的最大值，實際是左半邊的最大）
- 然後從 max_heap pop 出頂端（負的），取負 push 進 min_heap（這樣把可能太大或太小的調整到右邊）
- 如果 min_heap 的長度 > max_heap 的長度，就從 min_heap pop 出頂端，取負 push 回 max_heap，保持 max_heap 的大小 >= min_heap

這樣可以確保兩個堆平衡，且所有左邊的數字 <= 右邊的數字

在 findMedian：

- 如果 max_heap 的長度 > min_heap，返回 -max_heap[0] 左半邊最大，就是中位數
- 否則，返回 (-max_heap[0] + min_heap[0]) / 2 兩個頂端的平均

需要注意邊界：如果總數是 0，findMedian 不會被呼叫，加第一個數字會進 max_heap

這種方式可以處理特殊情況，比如連續加相同數字，或是負數都可以解決

以下是最簡單的模式：

```
heapq.heappush(self.max_heap, -num)
heapq.heappush(self.min_heap, -heapq.heappop(self.max_heap))

if len(self.min_heap) > len(self.max_heap):
    heapq.heappush(self.max_heap, -heapq.heappop(self.min_heap))
```

所以在 if 的時候確保平衡，findMedian 就直接看頂端

```python
import heapq

class MedianFinder:

    def __init__(self):
        self.max_heap = []
        self.min_heap = []

    def addNum(self, num: int) -> None:
        heapq.heappush(self.max_heap, -num)
        heapq.heappush(self.min_heap, -heapq.heappop(self.max_heap))

        if len(self.min_heap) > len(self.max_heap):
            heapq.heappush(self.max_heap, -heapq.heappop(self.min_heap))

    def findMedian(self) -> float:
        if len(self.max_heap) > len(self.min_heap):
            return -self.max_heap[0]
        return (-self.max_heap[0] + self.min_heap[0]) / 2
```

```
Assuming input stream: nums = [1, 2, 3, 4, 5]

Initial:
max_heap = [], min_heap = []

After add 1:
heappush(max_heap, -1) => max_heap = [-1]
heappush(min_heap, -heappop(max_heap)) => min_heap = [1], max_heap = []
Since len(min_heap) > len(max_heap): heappush(max_heap, -heappop(min_heap)) => max_heap = [-1], min_heap = []
findMedian: since len(max_heap) > len(min_heap), -max_heap[0] = 1

max_heap = [-1], min_heap = []
median = 1

After add 2:
heappush(max_heap, -2) => max_heap = [-2, -1]
heappush(min_heap, -heappop(max_heap)) => min_heap = [2], max_heap = [-1]
No balance needed
findMedian: (-max_heap[0] + min_heap[0]) / 2 = (1 + 2) / 2 = 1.5

max_heap = [-1], min_heap = [2]
median = 1.5

After add 3:
heappush(max_heap, -3) => max_heap = [-3, -1]
heappush(min_heap, -heappop(max_heap)) => min_heap = [2, 3], max_heap = [-1]
Since len(min_heap) > len(max_heap): heappush(max_heap, -heappop(min_heap)) => max_heap = [-2, -1], min_heap = [3]
findMedian: since len(max_heap) > len(min_heap), -max_heap[0] = 2

max_heap = [-2, -1], min_heap = [3]
median = 2

After add 4:
heappush(max_heap, -4) => max_heap = [-4, -1, -2]
heappush(min_heap, -heappop(max_heap)) => min_heap = [3, 4], max_heap = [-2, -1]
No balance needed
findMedian: (-max_heap[0] + min_heap[0]) / 2 = (2 + 3) / 2 = 2.5

max_heap = [-2, -1], min_heap = [3, 4]
median = 2.5

After add 5:
heappush(max_heap, -5) => max_heap = [-5, -1, -2]
heappush(min_heap, -heappop(max_heap)) => min_heap = [3, 4, 5], max_heap = [-2, -1]
Since len(min_heap) > len(max_heap): heappush(max_heap, -heappop(min_heap)) => max_heap = [-3, -1, -2], min_heap = [4, 5]
findMedian: since len(max_heap) > len(min_heap), -max_heap[0] = 3

max_heap = [-3, -1, -2], min_heap = [4, 5]
median = 3
```
