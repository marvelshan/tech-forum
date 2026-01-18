# leetcode 121 Best Time to Buy and Sell Stock

```
Input: prices = [7,1,5,3,6,4]
Output: 5
```

這題是給一個 array，讓他去走一遍，然後讓後續的數字去減去前面最低的數字，並獲得這個數字得最大值，

最一開始的想法就是走過 array 每個數字，並且減去 array 裡面每個數字，這樣就可以得到答案，但這個很明顯就是 O(N^2)，

<img width="1460" height="913" alt="image" src="https://github.com/user-attachments/assets/a99cbf7c-8e9c-4438-b812-8eb0cf276aa1" />


所以再繼續往下想，他要先知道誰是最小值，然後讓後面的值去減掉前面最小的值，但後面出現過更小的不算，所以其實假如後面有更小的就直接覆蓋掉也不會影響，

所以開頭就可以看到我們要使用 `min(原本最小,array[i])` 來去比較最小值，然後就是要比較答案，也就是我們目前去跑迴圈得到的值去剪掉現在的最小值，並且去存起來，

然後跟原本存起來的最大值去比大小，也就是 `max(原本減完最大值, maxValue)` 這樣就可以得到我們所需要的最大值，

因為題目有限制說假如都沒有最大值的話就維持是 0，所以一開始就需要把 maxValue 設定為 0，這樣後續減完沒有大於零的話都會是零，所以就會是 return 0 符合題目的需求，這題比較簡單所以就沒有寫詳細的思路了

```python
def maxProfit(self, prices: List[int]) -> int:
    result=0
    minVal=prices[0]
    for digit in prices:
        minVal = min(minVal, digit)
        result = max(result, digit - minVal)
    return result
```
