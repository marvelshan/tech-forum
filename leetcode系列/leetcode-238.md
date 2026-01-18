# leetcode-238 Product of Array Except Self

這題題目的限制是不能用除法，並且時間複雜度要限制在 O(N) 空間複雜度要限制在 O(1)，所以就刪掉了全部乘起來然後再除掉自己的做法，接著下一步想，就是自己不乘進去的話就是自己左邊全部乘上自己右邊全部，但假如這樣硬幹的話就會是 O(N^2)，所以要維持時間複雜度的話就會想到以下的方法

## 解題思路

分為兩個部分，第一個部分是先從 array 的左邊開始走，
nums = [1, 2, 3, 4]
然後將自己走過左邊的數字先乘積然後存下來，等等再從右邊走回來時再做一樣的事情，
就可以實現左右乘積的這個方法了，而這個要怎麼實現會比較好
那我們來看一下假如乘起來會怎樣

```
left = [1, nums[1], nums[1]*nums[2], nums[1]*nums[2]*nums[3]]
```

這樣就發現了一個規律，就是除了第一線左邊沒有值，所以賦值 1，右邊就是左邊乘上 nums[i]，其實也可以看成 1 \* nums[i]，所以就是

```
left[i] = left[i - 1] * nums[i - 1]
```

那右邊反之，

```
right = [1*nums[1]*nums[2]*nums[3],1*nums[2]*nums[3], 1*nums[3], 1]
// 要從左往右看會比較能理解
left = [right[0]*left[0], right[1]*left[1], right[2]*left[2], right[3]*left[3]]
```

不看最右邊的值的話就是

```
所以 right[i] = right[i+1]*nums[i+1]
```

然後再把它結合

```
res[i] = right[i]*left[i]
```

但是這邊再仔細想想，其實在右邊算回來的時候就可以直接結合左邊的就變成答案，然後 right 也可不用是一個 array，只要把它記起來就好，並且由最右邊往回看

```
方向：n-->1

第 n 項時
right=1
res[n] = left[n]*right

第 n-1 項時
right = 1 * nums[n] // 這裏就跟前面寫的 right[i] = right[i+1]*nums[i+1] 相同了
res[n-1] = left[n-1]*right
```

所以最後就可以結合成

```
res[n-1] = res[n-1]*right
```

```python
def productExceptSelf(self, nums: List[int]) -> List[int]:
    n = len(nums)
    res = [1] * n
    for i in range(1, len(nums)):
        res[i] = res[i - 1] * nums[i - 1]
    right = 1
    for i in range(n-1, -1, -1):
        res[i] = res[i]*right
        right = right*nums[i]
    return res
```
