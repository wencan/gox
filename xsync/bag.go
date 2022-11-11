package xsync

import (
	"sync"
	"sync/atomic"
)

var deletedBagEntry = new(interface{})

// Bag 并发安全的容量。
type Bag struct {
	// mux 锁。
	mu sync.Mutex

	// store 实质存储数据。内部结构为*lockFreeSlice。
	// slice增长时，需要加锁。
	store atomic.Value

	// indexPool 删除的索引，等待重用。
	indexPool *lockFreeSinglyLinkedList
}

// NewBag 新建一个Bag。
func NewBag() *Bag {
	bag := &Bag{
		indexPool: newLockFreeSinglyLinkedList(),
	}
	return bag
}

// Add 添加一个元素，返回索引。
// 警告：删除元素的索引会被重用。
func (bag *Bag) Add(p interface{}) int {
	store, _ := bag.store.Load().(*lockFreeSlice)
	if store != nil {
		if index, ok := store.Append(p); ok {
			return index
		}

		// 尝试重用回收的索引
		recycled, _ := bag.indexPool.LeftPop()
		if recycled != nil {
			// 拿到的可能时Grow之后的index
			// 使用最新的lockFreeSlice
			store, _ = bag.store.Load().(*lockFreeSlice)

			index := recycled.(int)
			store.UpdateAt(index, p)
			return index
		}
	}

	bag.mu.Lock()
	defer bag.mu.Unlock()

	if store == nil {
		// 初始化
		store, _ = bag.store.Load().(*lockFreeSlice)
		if store == nil {
			store = &lockFreeSlice{}
			bag.store.Store(store)
		}
	} else {
		previous := store
		store = bag.store.Load().(*lockFreeSlice)
		if store != previous {
			if index, ok := store.Append(p); ok {
				return index
			}
		}
	}

	// 增加容量后再append
	newStore := store.Grow()
	index, ok := newStore.Append(p)
	if !ok {
		panic("impossibility")
	}
	bag.store.Store(newStore)

	return index
}

// DeleteAt 删除指定位置上的元素。
// 警告：删除后，index会被回收重用。
func (bag *Bag) DeleteAt(index int) {
	store, _ := bag.store.Load().(*lockFreeSlice)
	if store == nil {
		panic("empty bag")
	}

	old := store.UpdateAt(index, deletedBagEntry)
	if old != deletedBagEntry {
		bag.indexPool.RightPush(index)
	}
}

// Range 基于索引顺序的遍历。
func (bag *Bag) Range(f func(index int, p interface{}) (stopIteration bool)) {
	store, _ := bag.store.Load().(*lockFreeSlice)
	if store == nil {
		return
	}
	store.Range(func(index int, p interface{}) (stopIteration bool) {
		if p == deletedBagEntry {
			return false
		}
		return f(index, p)
	})
}
