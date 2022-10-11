package xsync

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLRUMap(t *testing.T) {
	m := NewLRUMap(10, 10)

	// 先塞满
	for i := 0; i < 10*10; i++ {
		m.Store(i, i)

		value, ok := m.silentLoad(i)
		if assert.True(t, ok) {
			num := value.(int)
			assert.Equal(t, i, num)
		}
	}

	// 继续塞，应该删除掉最早的前10个
	for i := 10 * 10; i < 10*10+10; i++ {
		m.Store(i, i)

		value, ok := m.silentLoad(i)
		if assert.True(t, ok) {
			num := value.(int)
			assert.Equal(t, i, num)
		}
	}
	// 已经清理的部分
	for i := 0; i < 10; i++ {
		_, ok := m.silentLoad(i)
		assert.False(t, ok)
	}
	// 还在的部分
	for i := 10; i < 10*10+10; i++ {
		value, ok := m.silentLoad(i)
		if assert.True(t, ok, "index: %d", i) {
			num := value.(int)
			assert.Equal(t, i, num)
		}
	}
}

func TestLRUMap_Update(t *testing.T) {
	m := NewLRUMap(10, 10)

	// 先塞满
	for i := 0; i < 10*10; i++ {
		m.Store(i, i)
	}

	// 更新最后一个区块
	// 会清理掉第一个区块
	for i := 10 * 9; i < 10*10; i++ {
		m.Store(i, i*2)
	}

	// 检查
	// 前面10个被清理了
	for i := 10; i < 10*10; i++ {
		value, ok := m.silentLoad(i)
		if assert.True(t, ok, "index: %d", i) {
			num := value.(int)
			if i < 10*9 {
				assert.Equal(t, i, num)
			} else {
				assert.Equal(t, i*2, num)
			}
		}
	}
}

func TestLRUMap_LoadLastChunk(t *testing.T) {
	m := NewLRUMap(10, 10)

	// 先塞满
	for i := 0; i < 10*10; i++ {
		m.Store(i, i)
	}

	// 读取最后添加的10个
	// 最后添加的10个，使用记录已经是最新的，不会更新使用记录
	for i := 10*10 - 10; i < 10*10; i++ {
		value, ok := m.Load(i)
		if assert.True(t, ok) {
			num := value.(int)
			assert.Equal(t, i, num)
		}
	}
	// 检查
	for i := 0; i < 10*10; i++ {
		value, ok := m.silentLoad(i)
		if assert.True(t, ok, "index: %d", i) {
			num := value.(int)
			assert.Equal(t, i, num)
		}
	}
}

func TestLRUMap_LoadFirstChunk(t *testing.T) {
	m := NewLRUMap(10, 10)

	// 先塞满
	for i := 0; i < 10*10; i++ {
		m.Store(i, i)
	}

	// 读取最早添加的10个中的一个
	// 应该删除最早的区块。其中读取的一个，因为已经存到最新的区块，被保留了。
	value, ok := m.Load(1)
	if assert.True(t, ok, "index: 1") {
		num := value.(int)
		assert.Equal(t, 1, num)
	}
	for i := 0; i < 10; i++ {
		if i == 1 {
			continue
		}
		_, ok := m.silentLoad(i)
		assert.False(t, ok)
	}
	for i := 10; i < 10*10; i++ {
		value, ok := m.silentLoad(i)
		if assert.True(t, ok, "index: %d", i) {
			num := value.(int)
			assert.Equal(t, i, num)
		}
	}
}

func TestLRUMap_ConcurrentlyStore(t *testing.T) {
	m := NewLRUMap(10000, 500)

	var wg sync.WaitGroup
	wg.Add(500)
	for i := 0; i < 500; i++ {
		go func(_i int) {
			defer wg.Done()

			for j := 0; j < 10000; j++ {
				m.Store(_i*10000+j, _i*10000+j)
			}
		}(i)
	}
	wg.Wait()

	for i := 0; i < 500*10000; i++ {
		value, ok := m.silentLoad(i)
		if assert.True(t, ok) {
			num := value.(int)
			if num != i {
				assert.Equal(t, i, num, "index: %d", i)
			}
		}
	}
}

func TestLRUMap_ConcurrentlyLoad(t *testing.T) {
	m := NewLRUMap(10000, 500)

	// 先塞满
	for i := 0; i < 500*10000; i++ {
		m.Store(i, i)
	}

	var wg sync.WaitGroup
	wg.Add(500)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < 10; j++ {
				// 只读最后一个区块
				// 前面的会被淘汰掉
				for k := 499 * 10000; k < 500*10000; k++ {
					value, ok := m.Load(k)
					if assert.True(t, ok, "index: %d", k) {
						num := value.(int)
						if num != k {
							assert.Equal(t, k, num)
						}
					}
				}
			}
		}()
	}
	wg.Wait()
}

func TestLRUMap_ConcurrentlyStoreAndLoad(t *testing.T) {
	m := NewLRUMap(100000, 1000)

	var wg sync.WaitGroup
	wg.Add(500)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < 100000; j++ {
				m.Store(j, j)

				// 再读取
				// 最新的一般不会被清理（理论上还是有可能，所以需要足够大的chunkCapacity）
				value, ok := m.Load(j)
				if assert.True(t, ok, "index: %d", j) {
					num := value.(int)
					if num != j {
						assert.Equal(t, j, num)
					}
				}
			}
		}()
	}
	wg.Wait()
}

func TestLRUMap_ConcurrentlyIncrease(t *testing.T) {
	m := NewLRUMap(10000, 1000) // 参数设大点，不然预期的数据可能被覆盖

	var wg sync.WaitGroup
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func(index int) {
			defer wg.Done()

			// 每个位置上自增10000次
			for j := 0; j < 10000; j++ {
				if j == 0 {
					m.Store(index, 1)
				} else {
					value, ok := m.Load(index)
					if assert.True(t, ok) {
						num := value.(int)
						m.Store(index, num+1)
					}
				}
			}
		}(i)
	}
	wg.Wait()

	for i := 0; i < 1000; i++ {
		value, ok := m.Load(i)
		if assert.True(t, ok) {
			num := value.(int)
			if num != 10000 {
				assert.Equal(t, i, num, "index: %d", i)
			}
		}
	}
}
