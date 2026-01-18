# leetcode 226 Invert Binary Tree

給定一棵二元樹的根節點 `root`，請將這棵樹「左右對調」（也就是把每個節點的左子樹和右子樹交換），最後回傳反轉後的樹的根節點

這題要做的就是把整棵樹鏡像翻轉，像是把鏡子放在樹的中間，讓左邊變右邊、右邊變左邊

## 解題思路

這是一題非常經典的 recursive 題目，要反轉一棵樹，就先反轉它的左右子樹，然後把左右子樹交換位置

所以我們可以先處理子樹，再處理自己的遞迴方式來完成：

1. 如果節點是空的（`root` 是 None），直接返回 None
2. 先把原本的左子樹記下來，因為後面會被覆蓋
3. 遞迴反轉右子樹，把結果接給左邊 `root.left = invert(right)`
4. 遞迴反轉原本記下來的左子樹，把結果接給右邊 `root.right = invert(left)`
5. 最後回傳這個已經交換完的節點

假設原始樹是：

```
     4
   /   \
  2     7
 / \   / \
1   3 6   9
```

遞迴過程（從下往上想）：

1. 到達葉節點 1、3、6、9 → 都是 None 的子節點 → 直接返回自己
2. 到達節點 2：
   - left = 1
   - root.left = invert(3) → 3
   - root.right = invert(1) → 1
   - 2 的左右交換完成：變成 3 在左、1 在右
3. 到達節點 7：
   - left = 6
   - root.left = invert(9) → 9
   - root.right = invert(6) → 6
   - 7 的左右交換完成：9 在左、6 在右
4. 最後到達根節點 4：
   - left = 2（還沒交換時的）
   - root.left = invert(7) → 已經變成 [9,6] 的 7
   - root.right = invert(2) → 已經變成 [3,1] 的 2
   - 4 的左右交換完成

最終結果：

```
     4
   /   \
  7     2
 / \   / \
9   6 3   1
```

```python
class Solution:
    def invertTree(self, root: Optional[TreeNode]) -> Optional[TreeNode]:
        if not root:
            return root

        left = root.left

        root.left = self.invertTree(root.right)

        root.right = self.invertTree(left)

        return root
```

這個解法時間複雜度 O(n)，每個節點都走一次，空間複雜度 O(h) 遞迴深度，最壞情況下是 O(n)）
