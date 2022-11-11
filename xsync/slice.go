package xsync

import (
	"sync"
	"sync/atomic"
)

// lockFreeSlice 无锁slice实现。
// 增加容量时，通过grow函数创建一个新的lockFreeSlice对象。
type lockFreeSlice struct {
	// limitedSlices 由多个lockFreeLimitedSlice组成。
	// lockFreeSlice对象内的limitedSlices不会变。变的是lockFreeLimitedSlice内部数据。
	limitedSlices []*lockFreeLimitedSlice

	// limitSlicesNum limitedSlices数量。
	// lockFreeSlice对象内的limitedSlices数量不会发生变化。
	limitSlicesNum int

	// slicesStartIndex g
	slicesStartIndex []int

	// capacity 总容量。
	// lockFreeSlice对象内的容量不会发生变化。
	capacity int
}

// Grow 返回一个新的容量更大的lockFreeSlice对象，新lockFreeSlice对象会拥有原数组和新数组。
func (s *lockFreeSlice) Grow() *lockFreeSlice {
	// 最后一个数组的容量
	var lastCapacity int
	if len(s.limitedSlices) > 0 {
		lastCapacity = s.limitedSlices[len(s.limitedSlices)-1].Capacity()
	}

	// 新数组
	var tailCapacity int
	switch lastCapacity { // 这里，switch 比 if，更能清晰展现逻辑
	case 0:
		tailCapacity = 8
	case 8:
		tailCapacity = 16
	case 16:
		tailCapacity = 32
	case 32:
		tailCapacity = 64
	case 64:
		tailCapacity = 128
	case 128:
		tailCapacity = 256
	case 256:
		tailCapacity = 512
	default:
		tailCapacity = 1024
	}

	tailLimitedSlice := newLockFreeLimitedSlice(tailCapacity)

	// 新slice
	newSlice := &lockFreeSlice{
		limitedSlices:    append(s.limitedSlices, tailLimitedSlice),
		limitSlicesNum:   len(s.limitedSlices) + 1,
		slicesStartIndex: append(s.slicesStartIndex, s.capacity),
		capacity:         s.capacity + tailCapacity,
	}
	return newSlice
}

// slicesPostion 根据下标，计算元素存储在数组切片中的位置。
func slicesPostion(index int) (int, int) {
	var index1d, index2d int
	switch {
	case index < 0:
		panic("index must be non-negative.")
	case index < 8:
		index1d = 0
		index2d = index
	case index < 8+16:
		index1d = 1
		index2d = index - 8
	case index < 8+16+32:
		index1d = 2
		index2d = index - (8 + 16)
	case index < 8+16+32+64:
		index1d = 3
		index2d = index - (8 + 16 + 32)
	case index < 8+16+32+64+128:
		index1d = 4
		index2d = index - (8 + 16 + 32 + 64)
	case index < 8+16+32+64+128+256:
		index1d = 5
		index2d = index - (8 + 16 + 32 + 64 + 128)
	case index < 8+16+32+64+128+256+512:
		index1d = 6
		index2d = index - (8 + 16 + 32 + 64 + 128 + 256)
	default:
		index1d = 7 + (index-(8+16+32+64+128+256+512))/1024
		index2d = index - (8 + 16 + 32 + 64 + 128 + 256 + 512 + (index1d-7)*1024)
	}
	return index1d, index2d
}

// Append 追加新元素。
// 如果成功，返回下标。
// 如果失败，表示该grow了。
func (s *lockFreeSlice) Append(p interface{}) (int, bool) {
	if s.limitSlicesNum == 0 {
		return 0, false
	}
	index2d, ok := s.limitedSlices[s.limitSlicesNum-1].Append(p)
	if !ok {
		return 0, false
	}

	return s.slicesStartIndex[s.limitSlicesNum-1] + index2d, true
}

// Load 根据下标取回一个元素。
func (s *lockFreeSlice) Load(index int) interface{} {
	index1d, index2d := slicesPostion(index)
	return s.limitedSlices[index1d].Load(index2d)
}

// UpdateAt 更新下标位置上的元素，返回旧值。
func (s *lockFreeSlice) UpdateAt(index int, p interface{}) (old interface{}) {
	index1d, index2d := slicesPostion(index)
	return s.limitedSlices[index1d].UpdateAt(index2d, p)
}

// Range 遍历。
func (s *lockFreeSlice) Range(f func(index int, p interface{}) (stopIteration bool)) {
	var stop bool
	for index1d, limitedSlice := range s.limitedSlices {
		if stop {
			break
		}

		limitedSlice.Range(func(index2d int, p interface{}) (stopIteration bool) {
			index := s.slicesStartIndex[index1d] + index2d
			stop := f(index, p)
			return stop
		})
	}
}

// Slice 并发安全的Slice结构。
type Slice struct {
	// mux 锁。
	mu sync.Mutex

	// store 实质存储数据。内部结构为*lockFreeSlice。
	// slice增长时，需要加锁。
	store atomic.Value
}

// Append 在末尾追加一个元素。返回下标。
func (slice *Slice) Append(p interface{}) int {
	store, _ := slice.store.Load().(*lockFreeSlice)
	if store != nil {
		if index, ok := store.Append(p); ok {
			return index
		}
	}

	slice.mu.Lock()
	defer slice.mu.Unlock()

	if store == nil {
		// 初始化
		store, _ = slice.store.Load().(*lockFreeSlice)
		if store == nil {
			store = &lockFreeSlice{}
			slice.store.Store(store)
		}
	} else {
		previous := store
		store = slice.store.Load().(*lockFreeSlice)
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
	slice.store.Store(newStore)

	return index
}

// Load 取得下标位置上的值。
func (slice *Slice) Load(index int) interface{} {
	store, _ := slice.store.Load().(*lockFreeSlice)
	if store == nil {
		panic("empty slice")
	}
	return store.Load(index)
}

// Range 遍历。
func (slice *Slice) Range(f func(index int, p interface{}) (stopIteration bool)) {
	store, _ := slice.store.Load().(*lockFreeSlice)
	if store == nil {
		return
	}
	store.Range(f)
}

// // Length 长度。
// // 在并发Append的场景下，这个长度并不可靠。因为Append时，先增加长度，再存数据。可能长度增加了，对应的数据还没存。
// func (slice *Slice) Length() int {
// 	store, _ := slice.store.Load().(*lockFreeSlice)
// 	if store == nil {
// 		return 0
// 	}
// 	return store.Length()
// }

// UpdateAt 更新下标位置上的值，返回旧值。
func (slice *Slice) UpdateAt(index int, p interface{}) (old interface{}) {
	store, _ := slice.store.Load().(*lockFreeSlice)
	if store == nil {
		panic("empty slice")
	}
	return store.UpdateAt(index, p)
}
