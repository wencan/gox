package xsync

import "sync/atomic"

// lockFreeCappedSlice 长度受限的slice。如果空间已满，会覆盖最早的元素。
type lockFreeCappedSlice struct {
	array []atomic.Value

	// capacity 容量。容量不会发生变化。
	capacity uint64

	// nextIndex 下一个元素的位置。
	nextIndex uint64
}

func newLockFreeCappedSlice(capacity int) *lockFreeCappedSlice {
	return &lockFreeCappedSlice{
		array:     make([]atomic.Value, capacity),
		capacity:  uint64(capacity),
		nextIndex: 0,
	}
}

// Append 追加一个新元素。如果空间已满，覆盖最早的元素。返回下标和被覆盖元素。
func (slice *lockFreeCappedSlice) Append(p interface{}) (index int, covered interface{}) {
	for {
		index := atomic.LoadUint64(&slice.nextIndex)

		coveredValue := slice.array[index]

		nextIndex := index + 1
		if nextIndex >= slice.capacity {
			nextIndex = 0
		}

		if atomic.CompareAndSwapUint64(&slice.nextIndex, index, nextIndex) {
			slice.array[index].Store(p)
			covered = coveredValue.Load()

			return int(index), covered
		}
	}
}

// Load 根据下标取回一个元素。
func (slice *lockFreeCappedSlice) Load(index int) interface{} {
	return slice.array[index].Load()
}

// Newbie 最新加入的。如果还没数据，会返回nil。
func (slice *lockFreeCappedSlice) Newbie() interface{} {
	nextIndex := atomic.LoadUint64(&slice.nextIndex)
	var index uint64
	if nextIndex == 0 {
		index = slice.capacity - 1
	} else {
		index = nextIndex - 1
	}
	return slice.array[index].Load()
}
