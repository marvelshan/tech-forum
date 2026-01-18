# leetcode 21 Merge Two Sorted Lists

You are given the heads of two sorted linked lists `list1` and `list2`.

Merge the two lists into one sorted list. The list should be made by splicing together the nodes of the first two lists.

Return the head of the merged linked list.

這題是要把兩個已經排序好的 linked list 合併成一個還是排序好的 linked list，而且要直接用原本的節點連接起來，不能新建節點存值。特別要注意的一點是，如果其中一個 list 先用完，剩下的另一個 list 要直接接上去，這樣才能保持排序且高效

其實這很像 merge sort 的 merge 階段，就是兩個指針不斷比較小的值，接上去，然後往前移動

## 解題思路

一開始要先建立一個 dummy node 來當作結果 list 的起點，這樣比較方便處理頭部，然後用一個指針 `ansList` 來追蹤目前要接下一個節點的位置。同時保留 `ansHead` 來最後返回真正的頭，因為 dummy 本身不算是結果

<img width="1600" height="900" alt="image" src="https://github.com/user-attachments/assets/6c114a68-1422-4e53-92ca-c36c434104c6" />

在 while 迴圈裡，只要兩個 list 還有一個有剩`while list1 or list2`，就繼續合併：

- 如果 `list1` 已經沒了，直接把剩下的 `list2` 接上去，然後把 `list2` 往前移
- 如果 `list2` 已經沒了，直接把剩下的 `list1` 接上去，然後把 `list1` 往前移
- 兩個都還有，就比較兩個目前節點的值，誰小就把誰接上去，然後把那個 list 的指針往前移

每次接完一個節點後，都要把 `ansList` 往前移一步，準備接下一個。

這樣迴圈結束後，剩下的部分已經處理好，直接返回 `ansHead.next` 就是合併後的 list 頭了

這種方式特別好處理邊界情況，比如其中一個 list 是空的，或者兩個都空，直接返回空就好

```python
def mergeTwoLists(self, list1: Optional[ListNode], list2: Optional[ListNode]) -> Optional[ListNode]:
    ansList = ListNode()
    ansHead = ansList

    while list1 or list2:
        if not list1:
            ansList.next = list2
            list2 = list2.next
        elif not list2:
            ansList.next = list1
            list1 = list1.next
        else:
            if list1.val > list2.val:
                ansList.next = list2
                list2 = list2.next
            else:
                ansList.next = list1
                list1 = list1.next
        ansList = ansList.next

    return ansHead.next
```

這個解法時間複雜度是 O(m + n)，空間是 O(1)
