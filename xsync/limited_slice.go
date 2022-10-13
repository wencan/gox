package xsync

import (
	"sync/atomic"
)

// lockFreeLimitedSlice 长度受限的Slice。
type lockFreeLimitedSlice struct {
	array []atomic.Value

	// capacity 容量。
	// 容量不会发生变化。
	capacity uint64

	// nextAppendIndex 下次append元素的位置。无并发场景下，等于长度。
	nextAppendIndex uint64
}

func newLockFreeLimitedSlice(capacity int) *lockFreeLimitedSlice {
	return &lockFreeLimitedSlice{
		array:           make([]atomic.Value, capacity),
		capacity:        uint64(capacity),
		nextAppendIndex: 0,
	}
}

// Append 追加新元素。
// 如果成功，返回下标。
// 如果已满，返回false。
func (slice *lockFreeLimitedSlice) Append(p interface{}) (int, bool) {
	for {
		index := atomic.LoadUint64(&slice.nextAppendIndex)
		if index+1 > slice.capacity {
			return 0, false
		}

		if atomic.CompareAndSwapUint64(&slice.nextAppendIndex, index, index+1) {
			slice.array[index].Store(p)
			return int(index), true
		}
	}
}

// Load 根据下标取回一个元素。
func (slice *lockFreeLimitedSlice) Load(index int) interface{} {
	return slice.array[index].Load()
}
