# leetcode 20 Valid Parentheses

Given a string s containing just the characters '(', ')', '{', '}', '[' and ']', determine if the input string is valid.

這題要判斷說 {} 有沒有對應到相對應自己的括號，但是特別要注意的一點是 {[}] 沒有對應到的話就會是錯誤的，其實這點也就相對應的告訴我們解題的關鍵，就是要使用 Last in First out，也就是 stack，因為這樣使用的話才會是對稱的

<img width="1071" height="432" alt="image" src="https://github.com/user-attachments/assets/3fe328a2-2471-4af9-8f49-bfbd9051d7b7" />

## 解題思路

一開始要先設定一個 array 來裝前面的括號也就是`'(', '{', '['`，不用拿來裝後面的括號，因為我們會用的到的方法是 mapping 的方式來記錄對應的角色`{'(': ')'}`，所以假如我們看到前面，就要對應 mapping 到後面的括號，以下就是最簡單的模式

```
stack = []
mapping = {'(': ')'}
if '(' in mapping
    stack.append('(')
elif mapping[stack.pop()] !== ')'
    return False
```

所以我們看到 elif 的時候假如 stack 最頂端送進來的括號是 '}' 這邊就會是 False，就可以滿足我們所預想中特殊狀況的 `{[}]`，但是在 elif 有個情況下考慮到，有可能有 `]` 的這個狀況會出現程式的 error，因為他不會進入到第一個判斷，會直接進入到第二個判斷，但是 stack 又會沒有東西可以 pop，這時候就會直接 runtime error，所以就直接錯誤了，需要加上 `len(stack) == 0` 就也會是 return False，這樣就可以滿足我們需要的需求

但是按照這個邏輯寫到最後會 return True，這時候會遇到一個 case 是 s = '['，這個情況應該要是 False 卻回覆了 True，因為在我們寫的 case 中，回圈會在 mapping 裡面，並不會 return False，所以我們要在最後判斷說 stack 有沒有確實的被拿取完畢，這樣就可以知道說每個都有沒有對應到，所以就會是 len(stack) == 0 假如有的話就會回覆 True，沒有的話就會回覆 False，就是我們想要達到的邏輯

```python
def isValid(self, s: str) -> bool:
    stack = []
    mapping = {'(': ')', '{': '}', '[': ']'}
    for char in s:
        if char in mapping:
            stack.append(char)
        elif len(stack) == 0 or mapping[stack.pop()] != char:
            return False
    return len(stack) == 0
```
