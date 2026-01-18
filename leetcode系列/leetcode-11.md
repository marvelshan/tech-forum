# 11. Container With Most Water

這題是來計算面積的題目，想當初最一開始練習面試就遇到這題，直接不知道怎麼搞，只能暴力出 O(N^2) 的解法，現在回來真是懷念～這是經典的 two pointer 的題目，老實說最早能想出這個方法的人真的蠻天才的，雖然現在會用了就覺得還行，但是完全不會的話，是真的是很難想到

<img width="801" height="383" alt="image" src="https://github.com/user-attachments/assets/5ead1d14-cfb5-451d-bfc5-00deda450076" />

## 解題思路

我把這題換句話說有點像是左右逼近，因為最終要算得方式會是 min(arr[left], arr[right]) \* (right - left)，所以左右誰最小就是關鍵了，假如 l < r 代表 r 應該要去找更好的人來去搭配他，相反的 l > r 的話，r 就要去找更好的人來搭配他，然後我們就要把最大的值都記錄下來，然後在變換的過程中去跟最大的比較，這樣就能找到最大的面積

```python
def maxArea(self, height: List[int]) -> int:
    left, right = 0, len(height) - 1
    area = 0
    while left < right:
        area = max(area, min(height[left], height[right]) * (right - left))
        if height[left] > height[right]:
            right -= 1
        else:
            left += 1
    return area
```

```
height = [1,8,6,2,5,4,8,3,7]


left = 0
right = 8
min(1,7) * (8-0)= 8
area = 8

1<7
left = 1
right = 8
min(8,7) * (8-1)=49
area = max(8,42) = 49

8>7
left = 1
right = 7
min(8,3) * (7-1)=18
area = max(18,42) = 49

8>3
left = 1
right = 6
min(8,8) * (6-1)=40
area = max(40,42) = 49

8=8
left = 2
right = 6
min(6,8) * (6-2)=24
area = max(24,42) = 49

6<8
left = 3
right = 6
min(2,8) * (6-3)=6
area = max(6,42) = 49

2<8
left = 4
right = 6
min(5,8) * (6-4)=10
area = max(10,42) = 49

5<8
left = 5
right = 6
min(4,8) * (6-5)=4
area = max(4,42) = 49

return 49
```
