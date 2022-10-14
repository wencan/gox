package xsync

import (
	"sync/atomic"
)

// lockFreeSinglyLinkedListNode 无锁单链表的节点。
type lockFreeSinglyLinkedListNode struct {
	value atomic.Value

	next atomic.Value // *lockFreeSListNode

	// placeholder 占位节点。
	placeholder bool
}

// lockFreeSinglyLinkedList 无锁的单链表。
type lockFreeSinglyLinkedList struct {
	// leftNode 链表的开始位置。最左边的节点。
	// 最左边节点永远是占位节点。
	leftNode *lockFreeSinglyLinkedListNode

	// rightNode 最右边的节点。并发场景下，未必是最右边的节点。但可以通过next追踪到最右边节点。
	// 初始化时，是一个占位节点。后期随着添加元素，而更新。
	rightNode atomic.Value
}

func newLockFreeSinglyLinkedList() *lockFreeSinglyLinkedList {
	slist := &lockFreeSinglyLinkedList{}
	slist.leftNode = &lockFreeSinglyLinkedListNode{placeholder: true}
	slist.rightNode.Store(&lockFreeSinglyLinkedListNode{placeholder: true})

	return slist
}

// LeftPop 返回并删除最左边的元素。
// 如果slist为空，或者只有一个元素，返回nil。
// 实现限制不能pop唯一的元素。暂时也不需要全部pop。
func (slist *lockFreeSinglyLinkedList) LeftPop() (p interface{}, ok bool) {
	// 最左边节点永远是占位节点。
	// pop出最左边节点的next节点。
	for {
		next, _ := slist.leftNode.next.Load().(*lockFreeSinglyLinkedListNode)
		if next == nil {
			return nil, false
		}
		nextNext := next.next.Load()
		if nextNext == nil {
			// atomic.Value.CompareAndSwap不接受new值为nil
			// 目前逻辑也无法安全移除最右边的节点
			return nil, false
		}
		if slist.leftNode.next.CompareAndSwap(next, nextNext) {
			return next.value.Load(), true
		}
		// 其它过程也在pop。重试
	}
}

// RightPush 添加一个元素到最左边。
func (slist *lockFreeSinglyLinkedList) RightPush(p interface{}) {
	node := &lockFreeSinglyLinkedListNode{}
	node.value.Store(p)

	for {
		rightNode := slist.followRightNode()
		if rightNode.next.CompareAndSwap(nil, node) {
			slist.rightNode.Store(node)
			return
		}
	}
}

// followRightNode 最右边的节点。
func (slist *lockFreeSinglyLinkedList) followRightNode() *lockFreeSinglyLinkedListNode {
	rightNode := slist.rightNode.Load().(*lockFreeSinglyLinkedListNode)
	if rightNode.placeholder {
		// 链表为空
		return slist.leftNode
	}
	for {
		nextNode, _ := rightNode.next.Load().(*lockFreeSinglyLinkedListNode)
		if nextNode != nil {
			rightNode = nextNode
			continue
		}
		return rightNode
	}
}

// RightPeek 返回（不删除）最右边的元素。
func (slist *lockFreeSinglyLinkedList) RightPeek() interface{} {
	right := slist.followRightNode()
	if right == nil {
		return nil
	}
	return right.value.Load()
}
