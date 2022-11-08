package xsync

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type bagEntry struct {
	p interface{}

	deleted uint32
}

var bagEntryPool = sync.Pool{New: func() interface{} {
	return &bagEntry{}
}}

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
	entry := bagEntryPool.Get().(*bagEntry)
	entry.p = p
	entry.deleted = 0

	store, _ := bag.store.Load().(*lockFreeSlice)
	if store != nil {
		if index, ok := store.Append(entry); ok {
			return bag.deleteFunc(index)
		}

		// 尝试回收索引，并重用
		recycled, _ := bag.indexPool.LeftPop()
		if recycled != nil {
			// 拿到的可能时Grow之后的index
			// 使用最新的lockFreeSlice
			store, _ = bag.store.Load().(*lockFreeSlice)

			index := recycled.(int)
			oldEntry := store.UpdateAt(index, entry)
			bagEntryPool.Put(oldEntry)
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
			if index, ok := store.Append(entry); ok {
				return bag.deleteFunc(index)
			}
		}
	}

	// 增加容量后再append
	newStore := store.Grow()
	index, ok := newStore.Append(entry)
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

	entry := store.Load(index).(*bagEntry)
	if atomic.CompareAndSwapUint32(&entry.deleted, 0, 1) {
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
		entry := p.(*bagEntry)
		if entry.p == 100 {
			fmt.Println(entry)
		}
		if atomic.LoadUint32(&entry.deleted) == 0 {
			return f(entry.p)
		}
		return false
	})
}
