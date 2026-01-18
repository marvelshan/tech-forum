# leetcode 66 Plus One

給定一個由整數組成的陣列 `digits`，其中每個元素代表一個數字的個位數，且整個陣列代表一個非負整數

## 解題思路

最直觀的方式就是從最低位也就是個位數開始往高位加 1，遇到 9 就變成 0 並繼續進位，直到不進位為止

1. 先把陣列反轉，讓 index 0 變成最低位，這樣比較好從右往左處理
2. 用一個變數 `action` 來表示「是否還需要進位」
3. 用 `traverse_digit` 當作目前處理到的位數
4. 只要還需要進位（action == 1）就繼續處理：
   - 如果還有位數可以處理：
     - 這位是 9 -> 設成 0，繼續進位
     - 這位不是 9 -> 加 1，進位結束（action = 0）
   - 如果已經處理完所有位數（traverse_digit >= len(digits)）：
     - 代表最高位也要進位 -> 直接在最後 append 一個 1
     - 進位結束（action = 0）
5. 最後把陣列再反轉回來，就是正確的結果

```
digits = [1,2,3] -> 123 + 1 = 124

反轉後：[3,2,1]
- traverse_digit=0, digits[0]=3 <9 -> 3+1=4, action=0
- 結束迴圈
反轉回來就是[4,2,1] -> [4,2,1]

digits = [9,9,9] -> 999 + 1 = 1000

反轉後：[9,9,9]
- traverse_digit=0, digits[0]=9 -> 設成 0, action=1
- traverse_digit=1, digits[1]=9 -> 設成 0, action=1
- traverse_digit=2, digits[2]=9 -> 設成 0, action=1
- traverse_digit=3 >= len -> append 1, action=0
目前 digits = [0,0,0,1]
反轉回來：[1,0,0,0] 就完成我們需要的答案
```

```python
class Solution:
    def plusOne(self, digits: List[int]) -> List[int]:
        digits = digits[::-1]
        action = 1
        traverse_digit = 0
        while action:
            if traverse_digit < len(digits):
                if digits[traverse_digit] == 9:
                    digits[traverse_digit] = 0
                else:
                    digits[traverse_digit] += 1
                    action = 0
            else:
                digits.append(1)
                action = 0

            traverse_digit += 1

        return digits[::-1]
```

這個解法時間複雜度 O(n)，空間複雜度 O(1)
