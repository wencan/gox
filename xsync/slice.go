package xsync

import (
	"sync"
	"sync/atomic"
)

// lockFreeSlice 无锁slice实现。
// 增加容量时，通过grow函数创建一个新的lockFreeSlice对象。
type lockFreeSlice struct {
	arrays [][]atomic.Value

	// capacity 总容量。
	capacity int

	// length 实际占用的总长度。
	length uint64
}

// Grow 返回一个新的容量更大的slice对象，新slice对象会拥有原数组和新数组。
func (s *lockFreeSlice) Grow() *lockFreeSlice {
	// 最后一个数组的容量
	var lastCapacity int
	if len(s.arrays) > 0 {
		lastCapacity = len(s.arrays[len(s.arrays)-1])
	}

	// 新数组
	var tail []atomic.Value
	switch lastCapacity {
	case 0:
		array := new([8]atomic.Value)
		tail = array[:]
	case 8:
		array := new([16]atomic.Value)
		tail = array[:]
	case 16:
		array := new([32]atomic.Value)
		tail = array[:]
	case 32:
		array := new([64]atomic.Value)
		tail = array[:]
	case 64:
		array := new([128]atomic.Value)
		tail = array[:]
	case 128:
		array := new([256]atomic.Value)
		tail = array[:]
	case 256:
		array := new([512]atomic.Value)
		tail = array[:]
	default:
		array := new([1024]atomic.Value)
		tail = array[:]
	}

	// 新slice
	newSlice := &lockFreeSlice{
		arrays:   make([][]atomic.Value, len(s.arrays)),
		capacity: s.capacity + len(tail),
		length:   atomic.LoadUint64(&s.length),
	}
	copy(newSlice.arrays, s.arrays)
	newSlice.arrays = append(newSlice.arrays, tail)
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
	for {
		index := atomic.LoadUint64(&s.length)
		if index+1 > uint64(s.capacity) {
			return 0, false
		}

		if atomic.CompareAndSwapUint64(&s.length, index, index+1) {
			index1d, index2d := slicesPostion(int(index))
			s.arrays[index1d][index2d].Store(p)
			return int(index), true
		}
	}
}

// Load 根据下标取回一个元素。
func (s *lockFreeSlice) Load(index int) interface{} {
	index1d, index2d := slicesPostion(index)
	return s.arrays[index1d][index2d].Load()
}

// UpdateAt 更新下标位置上的元素。
func (s *lockFreeSlice) UpdateAt(index int, p interface{}) {
	index1d, index2d := slicesPostion(index)
	s.arrays[index1d][index2d].Store(p)
}

// Range 遍历。
func (s *lockFreeSlice) Range(f func(index int, p interface{}) (stopIteration bool)) {
	var index int
	for _, array := range s.arrays {
		for _, value := range array {
			if index >= int(atomic.LoadUint64(&s.length)) {
				return
			}

			p := value.Load()
			stopIteration := f(index, p)
			if stopIteration {
				return
			}

			index++
		}
	}
}

// Length 返回长度。
func (s *lockFreeSlice) Length() int {
	return int(atomic.LoadUint64(&s.length))
}

// Slice 并发安全的Slice结构。
type Slice struct {
	// mux 锁。
	mu sync.Mutex

	// store 实质存储数据。内部结构为*slice。
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
		store = slice.store.Load().(*lockFreeSlice)
		if index, ok := store.Append(p); ok {
			return index
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
func (slice *Slice) Length() int {
	store, _ := slice.store.Load().(*lockFreeSlice)
	if store == nil {
		return 0
	}
	return store.Length()
}

// UpdateAt 更新下标位置上的值。
func (slice *Slice) UpdateAt(index int, p interface{}) {
	store, _ := slice.store.Load().(*lockFreeSlice)
	if store == nil {
		panic("empty slice")
	}
	store.UpdateAt(index, p)
}
