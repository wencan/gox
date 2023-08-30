package xsync

import (
	"sync"
	"sync/atomic"

	"github.com/wencan/freesync/lockfree"
)

type lruMapEntry struct {
	// key 数据的key。也是存在sync.Map的key。
	key interface{}

	// value 存储的数据。
	value interface{}

	// chunk 所在区块。
	chunk atomic.Value //*lockfree.LimitedSlice
}

// LRUMap 只持有最近使用元素的map。并发安全。
// 按批次（区块）清理长期不用的元素。
// 总是提升Store/Load的元素到最新区块。空间不足时，清理最后一个区块和区块内的元素。
type LRUMap struct {
	// mapping 并发安全的map。
	// value为*lruMapEntry。
	mapping sync.Map

	// chunks 按批次记录最近使用的元素entry，一个区块(chunk)存一批记录。
	chunks *lockfree.SinglyLinkedList

	// chunkCapacity 区块大小。
	chunkCapacity int

	// chunkNumLimit 区块数量上限。
	chunkNumLimit int

	// chunkCount 区块计数。
	chunkCount int

	// chunksLock 创建新区块，需要加锁。
	chunksLock sync.Mutex

	// rwLock 存新数据时，用可重入的读锁；清理数据时，用不可重入的写锁。
	rwLock sync.RWMutex

	// onEvicted 被执行时，清理的数据可能正被使用。删除onEvicted。只能让资源等待被回收。
	// onEvicted 元素被清理时，回调该函数。可用于回收资源。
	// onEvicted func(k, v interface{})
}

// NewLRUMap 创建LRUMap。
// 总容量为chunkCapacity*chunkNum。
func NewLRUMap(chunkCapacity int, chunkNum int) *LRUMap {
	// return NewLRUMapWithEvict(chunkCapacity, chunkNum, nil)
	return &LRUMap{
		chunks:        lockfree.NewSinglyLinkedList(),
		chunkCapacity: chunkCapacity,
		chunkNumLimit: chunkNum,
	}
}

// // NewLRUMapWithEvict 创建LRUMap。
// // 总容量为chunkCapacity*chunkNum。
// // 当最近不用的元素和被覆盖的元素被清理时，onEvicted函数将被回调。
// func NewLRUMapWithEvict(chunkCapacity int, chunkNum int, onEvicted func(key, value interface{})) *LRUMap {
// 	return &LRUMap{
// 		chunks:        lockfree.NewSinglyLinkedList(),
// 		chunkCapacity: chunkCapacity,
// 		chunkNumLimit: chunkNum,
// 		onEvicted:     onEvicted,
// 	}
// }

// Store 存储数据。记录元素到最新区块。
// 被覆盖的value会跟随最近不用的value，等待被清理。
func (m *LRUMap) Store(key interface{}, value interface{}) {
	entry := &lruMapEntry{
		key:   key,
		value: value,
	}

	m.rwLock.RLock() // 可重入的锁
	m.mapping.Store(key, entry)
	m.rwLock.RUnlock()

	m.upgradeEntry(entry)
}

// upgradeEntry 存放到最新的区块，可能删除最老的区块。如果已经是在最新的区块，什么也不干。
func (m *LRUMap) upgradeEntry(entry *lruMapEntry) {
	coveredChunk := m.putInTopChunk(entry)
	if coveredChunk != nil {
		m.deleteCoveredChunk(coveredChunk)
	}
}

// putInTopChunk  存放到最新的区块。如果最新区块已满，创建新的区块。
func (m *LRUMap) putInTopChunk(entry *lruMapEntry) (coveredChunk *lockfree.LimitedSlice) {
	topChunk, _ := m.chunks.RightPeek().(*lockfree.LimitedSlice)
	if topChunk != nil {
		currentChunk, _ := entry.chunk.Load().(*lockfree.LimitedSlice)
		if topChunk == currentChunk {
			// 如果已经是最新的区块
			return
		}

		// 尝试存到最新区块
		var old interface{}
		if currentChunk != nil {
			old = currentChunk
		}
		if !entry.chunk.CompareAndSwap(old, topChunk) {
			// 有另一路并发过程在更新entry
			return
		}
		// 如果失败，表示最新区块满了
		if _, ok := topChunk.Append(entry); ok {
			return
		}
	}

	// 最新的区块已满，或者还没创建区块。
	// 加锁，然后新增区块。
	m.chunksLock.Lock()
	defer m.chunksLock.Unlock()

	// 二次检查
	preTopChunk := topChunk
	topChunk, _ = m.chunks.RightPeek().(*lockfree.LimitedSlice)
	if topChunk != nil && topChunk != preTopChunk {
		currentChunk, _ := entry.chunk.Load().(*lockfree.LimitedSlice)
		if currentChunk == topChunk {
			return
		}
		entry.chunk.Store(topChunk)
		if _, ok := topChunk.Append(entry); ok {
			return
		}
	}

	// 创建新的区块，然后再存
	chunk := lockfree.NewLimitedSlice(m.chunkCapacity)
	entry.chunk.Store(chunk)
	if _, ok := chunk.Append(entry); !ok {
		panic("impossibility")
	}
	m.chunks.RightPush(chunk)
	m.chunkCount++

	if m.chunkCount > m.chunkNumLimit {
		p, _ := m.chunks.LeftPop()
		coveredChunk, _ = p.(*lockfree.LimitedSlice)
		m.chunkCount--
	}

	return coveredChunk
}

// deleteCoveredChunk 删除最老的区块。
func (m *LRUMap) deleteCoveredChunk(coveredChunk *lockfree.LimitedSlice) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()

	coveredChunk.Range(func(index int, p interface{}) (stopIteration bool) {
		entry, _ := p.(*lruMapEntry)
		if entry == nil {
			return false
		}

		v, ok := m.mapping.Load(entry.key)
		if !ok {
			// 清理其它区块时，被清理掉了
			return false
		}

		storedEntry := v.(*lruMapEntry)
		currentChunk := storedEntry.chunk.Load().(*lockfree.LimitedSlice)
		covered := currentChunk != coveredChunk // 如果不相等，表示已经被覆盖
		if !covered {
			// 仅当没被覆盖时删除key，不然会删除最新的kv对。
			// 这里，storedEntry 和 entry 是同一指针
			// 从mapping.Load后到mapping.Delete这个区间要加锁。
			// 如果mapping.Load后到mapping.Delete前，key被重新Store，删除的就会是新的kv对
			m.mapping.Delete(entry.key)
		}
		// // 回调通知
		// if m.onEvicted != nil {
		// 	m.onEvicted(entry.key, entry.value)
		// }

		// 清理的entry可能正被LRUMap.Load程序加载。
		// 否则，可以回收entry。

		return false
	})
}

// Load 查找元素。如果找到，记录元素到最新区块。
func (m *LRUMap) Load(key interface{}) (value interface{}, ok bool) {
	v, ok := m.mapping.Load(key)
	if !ok {
		return nil, false
	}

	entry, _ := v.(*lruMapEntry)
	m.upgradeEntry(entry)

	return entry.value, ok
}

// SilentLoad 查找元素。如果找到，返回一个记录元素到最新区块的函数。
func (m *LRUMap) SilentLoad(key interface{}) (value interface{}, upgrade func(), ok bool) {
	v, ok := m.mapping.Load(key)
	if !ok {
		return nil, nil, false
	}

	entry, _ := v.(*lruMapEntry)

	return entry.value, func() {
		m.upgradeEntry(entry)
	}, true
}
