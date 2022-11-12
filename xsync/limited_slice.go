package xsync

import (
	"sync"
	"sync/atomic"
)

// lockFreeLimitedSliceEntry 包装要保存的数据。
// lockFreeLimitedSliceEntry指针为nil，表示还未初始化；lockFreeLimitedSliceEntry.p为nil，表示用户存了一个数据nil。
// atomic.Value不支持存nil，不支持更新不同类型的值，使用lockFreeLimitedSliceEntry，绕过了atomic.Value的限制。
type lockFreeLimitedSliceEntry struct {
	p interface{}
}

var lockFreeLimitedSliceEntryPool = sync.Pool{New: func() interface{} {
	return &lockFreeLimitedSliceEntry{}
}}

// lockFreeLimitedSlice 长度受限的Slice。
type lockFreeLimitedSlice struct {
	// array 值为*lockFreeLimitedSliceEntry。lockFreeLimitedSliceEntry内p存的是保存的数据。
	array []atomic.Value

	// entites 预分配的lockFreeLimitedSliceEntry
	entites []lockFreeLimitedSliceEntry

	// capacity 容量。
	// 容量不会发生变化。
	capacity int

	// nextAppendIndex 下次append元素的位置。无并发场景下，等于长度。
	nextAppendIndex uint64
}

func newLockFreeLimitedSlice(capacity int) *lockFreeLimitedSlice {
	return &lockFreeLimitedSlice{
		array:           make([]atomic.Value, capacity),
		entites:         make([]lockFreeLimitedSliceEntry, capacity),
		capacity:        capacity,
		nextAppendIndex: 0,
	}
}

// Capacity
func (slice *lockFreeLimitedSlice) Capacity() int {
	return slice.capacity
}

// Append 追加新元素。
// 如果成功，返回下标。
// 如果已满，返回false。
func (slice *lockFreeLimitedSlice) Append(p interface{}) (int, bool) {
	for {
		index := atomic.LoadUint64(&slice.nextAppendIndex)
		if index+1 > uint64(slice.capacity) {
			return 0, false
		}

		if atomic.CompareAndSwapUint64(&slice.nextAppendIndex, index, index+1) {
			// 这里需要警惕，length增长了，但数据还没存进去。
			// 等到Store完成，才算Append结束。
			entry := &slice.entites[index]
			entry.p = p
			slice.array[index].Store(entry)
			return int(index), true
		}
	}
}

// Load 根据下标取回一个元素。
func (slice *lockFreeLimitedSlice) Load(index int) interface{} {
	entry := slice.array[index].Load().(*lockFreeLimitedSliceEntry)
	return entry.p
}

// UpdateAt 更新下标位置上的元素，返回旧值。
func (slice *lockFreeLimitedSlice) UpdateAt(index int, p interface{}) (old interface{}) {
	newEntry := lockFreeLimitedSliceEntryPool.Get().(*lockFreeLimitedSliceEntry)
	newEntry.p = p
	oldVal := slice.array[index].Swap(newEntry)
	oldEntry := oldVal.(*lockFreeLimitedSliceEntry)
	old = oldEntry.p
	lockFreeLimitedSliceEntryPool.Put(oldEntry)
	return old
}

// Range 遍历。
func (slice *lockFreeLimitedSlice) Range(f func(index int, p interface{}) (stopIteration bool)) {
	length := int(atomic.LoadUint64(&slice.nextAppendIndex))
	for index := 0; index < length; index++ {
		val := slice.array[index].Load()
		if val == nil {
			// nextAppendIndex增长了，但数据还没存进去
			continue
		}

		entry := val.(*lockFreeLimitedSliceEntry)
		stopIteration := f(index, entry.p)
		if stopIteration {
			break
		}
	}
}
