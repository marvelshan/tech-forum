# leetcode 1161 Maximum Level Sum of a Binary Tree

Given the root of a binary tree, the level of its root is 1, the level of its children is 2, and so on.

Return the smallest level x such that the sum of all the values of nodes at level x is maximal.

## 解題思路

一開始設定 level = 0，但題目說到第一層是從 1 開始計算，queue = [root]，max_val = -inf 用來追蹤最大總和，current_level = 0

用 while queue 來做 BFS 層級遍歷

- 每層先準備 next_queue = [] 來裝下一層節點
- current_val = 0 來累加本層總和
- current_level += 1
- 遍歷本層所有節點：
  - current_val += node.val
  - 如果有 left/right 子節點，加進 next_queue，標準的 BFS 作法，用一個新的 [] 來裝後面的節點
- 比較 current_val 是否 > max_val，如果是，更新 level = current_level, max_val = current_val
- 最後 queue = next_queue，繼續下一層

```
while queue:
    next_queue = []
    current_val = 0
    current_level += 1
    for node in queue:
        current_val += node.val
        if node.left: next_queue.append(node.left)
        if node.right: next_queue.append(node.right)
    if max_val < current_val:
        level = current_level
        max_val = current_val
    queue = next_queue
```

所以看到 if max_val < current_val 的時候更新 level 和 max_val，就可以滿足多層一樣大時取最小的，並且是在 while 前有個情況下考慮到，如果 root 是 None，題目保證不是

```python
def maxLevelSum(self, root: Optional[TreeNode]) -> int:
    level = 0
    queue = [root]
    max_val = float("-inf")
    current_level = 0

    # BFS
    while queue:
        next_queue = []
        current_val = 0
        current_level += 1
        for node in queue:
            current_val += node.val
            if node.left:
                next_queue.append(node.left)
            if node.right:
                next_queue.append(node.right)

        if max_val < current_val:
            level = current_level
            max_val = current_val
        queue = next_queue
    return level
```
