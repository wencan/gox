package xsync

import (
	"sync"
	"sync/atomic"
)

var deletedBagEntry = new(interface{})

// Bag 并发安全的容量。支持添加、删除、（不保证顺序的）遍历。
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

// Add 添加一个元素，返回删除这个元素的函数。
func (bag *Bag) Add(p interface{}) (delete func()) {
	store, _ := bag.store.Load().(*lockFreeSlice)
	if store != nil {
		if index, ok := store.Append(p); ok {
			return bag.deleteFunc(index)
		}

		// 尝试回收索引，并重用
		recycled, _ := bag.indexPool.LeftPop()
		if recycled != nil {
			// 拿到的可能时Grow之后的index
			// 使用最新的lockFreeSlice
			store, _ = bag.store.Load().(*lockFreeSlice)

			index := recycled.(int)
			store.UpdateAt(index, p)
			return bag.deleteFunc(index)
		}
	}

	bag.mu.Lock()
	defer bag.mu.Unlock()

	if store == nil {
		// 初始化
		store, _ = bag.store.Load().(*lockFreeSlice)
		if store == nil {
			var length uint64
			store = &lockFreeSlice{length: &length}
			bag.store.Store(store)
		}
	} else {
		previous := store
		store = bag.store.Load().(*lockFreeSlice)
		if store != previous {
			if index, ok := store.Append(p); ok {
				return bag.deleteFunc(index)
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

	return bag.deleteFunc(index)
}

func (bag *Bag) deleteFunc(index int) func() {
	return func() {
		bag.deleteAt(index)
	}
}

func (bag *Bag) deleteAt(index int) {
	store, _ := bag.store.Load().(*lockFreeSlice)
	if store == nil {
		panic("empty bag")
	}

	old := store.UpdateAt(index, deletedBagEntry)
	if old != deletedBagEntry {
		bag.indexPool.RightPush(index)
	}
}

// Range 遍历，不保证顺序。
func (bag *Bag) Range(f func(p interface{}) (stopIteration bool)) {
	store, _ := bag.store.Load().(*lockFreeSlice)
	if store == nil {
		return
	}
	store.Range(func(index int, p interface{}) (stopIteration bool) {
		if p == deletedBagEntry {
			return false
		}
		return f(p)
	})
}
