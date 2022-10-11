package xsync

import (
	"sync"
)

type lruMapEntry struct {
	// key 数据的key。也是存在sync.Map的key。
	key interface{}

	// value 存储的数据。
	value interface{}

	// chunk 所在区块。
	chunk *lockFreeLimitedSlice
}

var lruMapEntryPool = sync.Pool{New: func() interface{} {
	return &lruMapEntry{}
}}

// LRUMap 只持有最近使用元素的map。并发安全。
// 与常见的lru不同，LRUMap按批次清理长期不用的元素，节省查询操作的消耗。
type LRUMap struct {
	// mapping 并发安全的map。
	// value为*lruMapEntry。
	mapping sync.Map

	// chunks 按批次记录最近使用的元素entry，一个区块(chunk)存一批记录。
	chunks *lockFreeCappedSlice

	// chunkCapacity 区块大小。
	chunkCapacity int

	// mu 锁。创建新区块，需要加锁。
	mu sync.Mutex
}

// NewLRUMap 创建LRUMap。
// chunk用来记录一个批次的元素使用记录。每次清理chunkCapacity个最后使用的元素。
// 总容量为chunkCapacity*chunkNum。
func NewLRUMap(chunkCapacity int, chunkNum int) *LRUMap {
	return &LRUMap{
		chunks:        newLockFreeCappedSlice(chunkNum),
		chunkCapacity: chunkCapacity,
	}
}

// Store 存储数据，标记元素为最近使用。
// 如果空间不够，清理chunkCapacity个最后使用的元素。
func (m *LRUMap) Store(key interface{}, value interface{}) {
	entry := lruMapEntryPool.Get().(*lruMapEntry)
	entry.key = key
	entry.value = value
	entry.chunk = nil

	m.mapping.Store(key, entry)

	m.upgradeEntry(entry)
}

// upgradeEntry 存放到最新的区块。如果已经是在最新的区块，什么也不干。
func (m *LRUMap) upgradeEntry(entry *lruMapEntry) {
	topChunk, _ := m.chunks.Newbie().(*lockFreeLimitedSlice)
	if topChunk != nil {
		if topChunk == entry.chunk {
			// 如果已经是最新的区块
			return
		}

		// 尝试存到最新区块
		// 如果失败，表示最新区块满了
		entry.chunk = topChunk
		if _, ok := topChunk.Append(entry); ok {
			return
		}
	}

	// 最新的区块已满，或者还没创建区块。
	// 加锁，然后新增区块。
	var covered interface{}
	func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		// 二次检查
		preTopChunk := topChunk
		topChunk, _ := m.chunks.Newbie().(*lockFreeLimitedSlice)
		if topChunk != nil && topChunk != preTopChunk {
			entry.chunk = topChunk
			if _, ok := topChunk.Append(entry); ok {
				return
			}
		}

		// 创建新的区块，然后再存
		chunk := newLockFreeLimitedSlice(m.chunkCapacity)
		entry.chunk = chunk
		if _, ok := chunk.Append(entry); !ok {
			panic("impossibility")
		}
		_, covered = m.chunks.Append(chunk) // 返回被覆盖的chunk
	}()

	// 清理最后一个区块
	// 这块不需要加锁。
	if covered != nil {
		coveredChunk, _ := covered.(*lockFreeLimitedSlice)
		if coveredChunk != nil {
			for i := 0; i < m.chunkCapacity; i++ {
				entry, _ := coveredChunk.Load(i).(*lruMapEntry)
				if entry != nil {
					v, ok := m.mapping.Load(entry.key)
					if ok { // 如果清理其它区块时，被清理掉了，就会 !ok
						storedEntry := v.(*lruMapEntry)
						if storedEntry.chunk == coveredChunk { // 如果不相等，表示已经更新
							// 这里，storedEntry 和 entry 是同一指针
							m.mapping.Delete(entry.key)
						}
					}
					lruMapEntryPool.Put(entry)
				}
			}
		}
	}
}

// Load 查找元素。如果找到，标记元素为最近使用。
func (m *LRUMap) Load(key interface{}) (value interface{}, ok bool) {
	v, ok := m.mapping.Load(key)
	if !ok {
		return nil, ok
	}

	entry, _ := v.(*lruMapEntry)
	m.upgradeEntry(entry)

	return entry.value, ok
}

// silentLoad 查找元素，不更新使用记录。用于支持测试。
func (m *LRUMap) silentLoad(key interface{}) (value interface{}, ok bool) {
	v, ok := m.mapping.Load(key)
	if !ok {
		return nil, ok
	}

	entry, _ := v.(*lruMapEntry)

	return entry.value, ok
}
