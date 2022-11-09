package xsync

import (
	"sync"
	"sync/atomic"
)

// lockFreeSliceEntry 存在atomic.Value的是*lockFreeSliceEntry。
// lockFreeSliceEntry指针为nil，表示还未初始化；lockFreeSliceEntry.p为nil，表示用户存了一个数据nil。
// atomic.Value不支持存nil，不支持更新不同类型的值，使用lockFreeSliceEntry，绕过了atomic.Value的限制。
type lockFreeSliceEntry struct {
	p interface{}
}

var lockFreeSliceEntryPool = sync.Pool{New: func() interface{} {
	return &lockFreeSliceEntry{}
}}

// lockFreeSlice 无锁slice实现。
// 增加容量时，通过grow函数创建一个新的lockFreeSlice对象。
type lockFreeSlice struct {
	// arrays 存放数据的二维切片，第二维由数组转换来。lockFreeSlice对象的arrays数据不变，变的是atomic.Value内的值。
	// grow时，创建一个新的arrays，拷贝原arrays，追加新数组。
	arrays [][]atomic.Value

	// entries 每次Grow时，预分配一批lockFreeSliceEntry，优化内存。
	entries [][]lockFreeSliceEntry

	// capacity 总容量。
	// lockFreeSlice对象内的容量不会发生变化。
	capacity int

	// length 实际占用的总长度，标示新追加元素的位置。
	length *uint64
}

// Grow 返回一个新的容量更大的lockFreeSlice对象，新lockFreeSlice对象会拥有原数组和新数组。
func (s *lockFreeSlice) Grow() *lockFreeSlice {
	// 最后一个数组的容量
	var lastCapacity int
	if len(s.arrays) > 0 {
		lastCapacity = len(s.arrays[len(s.arrays)-1])
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
	tail := make([]atomic.Value, tailCapacity)
	entries := make([]lockFreeSliceEntry, tailCapacity) // 优化内存

	// 新slice
	newSlice := &lockFreeSlice{
		arrays:   append(s.arrays, tail),
		entries:  append(s.entries, entries),
		capacity: s.capacity + len(tail),
		length:   s.length,
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
	if s.length == nil {
		// 还没初始化
		return 0, false
	}

	for {
		index := atomic.LoadUint64(s.length)
		if index+1 > uint64(s.capacity) {
			return 0, false
		}

		if atomic.CompareAndSwapUint64(s.length, index, index+1) {
			index1d, index2d := slicesPostion(int(index))

			// 这里需要警惕，length增长了，但数据还没存进去。
			// 等到Store完成，才算Append结束。

			entry := &s.entries[index1d][index2d]
			entry.p = p
			s.arrays[index1d][index2d].Store(entry)
			return int(index), true
		}
	}
}

// Load 根据下标取回一个元素。
func (s *lockFreeSlice) Load(index int) interface{} {
	index1d, index2d := slicesPostion(index)
	entry := s.arrays[index1d][index2d].Load().(*lockFreeSliceEntry)
	return entry.p
}

// UpdateAt 更新下标位置上的元素，返回旧值。
func (s *lockFreeSlice) UpdateAt(index int, p interface{}) (old interface{}) {
	index1d, index2d := slicesPostion(index)

	newEntry := lockFreeSliceEntryPool.Get().(*lockFreeSliceEntry)
	newEntry.p = p
	oldVal := s.arrays[index1d][index2d].Swap(newEntry)
	oldEntry := oldVal.(*lockFreeSliceEntry)
	old = oldEntry.p
	lockFreeSliceEntryPool.Put(oldEntry)
	return old
}

// Range 遍历。
func (s *lockFreeSlice) Range(f func(index int, p interface{}) (stopIteration bool)) {
	if s.length == nil {
		return
	}

	var index int
	length := int(atomic.LoadUint64(s.length))
	for _, array := range s.arrays {
		for _, value := range array {
			if index >= length {
				return
			}

			val := value.Load()
			if val == nil {
				// lenght增长了，但数据还没存进去
				break
			}
			entry := val.(*lockFreeSliceEntry)
			stopIteration := f(index, entry.p)
			if stopIteration {
				return
			}

			index++
		}
	}
}

// Length 返回长度。
// 在并发Append的场景下，这个长度并不可靠。因为Append时，先增加长度，再存数据。可能长度增加了，对应的数据还没存。
func (s *lockFreeSlice) Length() int {
	if s.length == nil {
		return 0
	}
	return int(atomic.LoadUint64(s.length))
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
			var length uint64
			store = &lockFreeSlice{length: &length}
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

// Length 长度。
// 在并发Append的场景下，这个长度并不可靠。因为Append时，先增加长度，再存数据。可能长度增加了，对应的数据还没存。
func (slice *Slice) Length() int {
	store, _ := slice.store.Load().(*lockFreeSlice)
	if store == nil {
		return 0
	}
	return store.Length()
}

// UpdateAt 更新下标位置上的值，返回旧值。
func (slice *Slice) UpdateAt(index int, p interface{}) (old interface{}) {
	store, _ := slice.store.Load().(*lockFreeSlice)
	if store == nil {
		panic("empty slice")
	}
	return store.UpdateAt(index, p)
}
