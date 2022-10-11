package xsync

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_lockFreeLimitedSlice(t *testing.T) {
	slice := newLockFreeLimitedSlice(10)
	for i := 0; i < 10; i++ {
		index, ok := slice.Append(i)
		assert.True(t, ok)
		assert.Equal(t, i, index)

		length := slice.Length()
		assert.Equal(t, i+1, length)
	}

	_, ok := slice.Append(11)
	assert.False(t, ok)

	length := slice.Length()
	assert.Equal(t, 10, length)

	for i := 0; i < 10; i++ {
		num, _ := slice.Load(i).(int)
		assert.Equal(t, i, num)
	}
}

func Test_lockFreeLimitedSlice_ConcurrentlyAppend(t *testing.T) {
	slice := newLockFreeLimitedSlice(500 * 10000)

	var wg sync.WaitGroup
	wg.Add(500)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < 10000; j++ {
				_, ok := slice.Append(123)
				assert.True(t, ok)
			}
		}()
	}
	wg.Wait()

	length := slice.Length()
	assert.Equal(t, 500*10000, length)

	// 空间满后，失败的append
	wg.Add(500)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				_, ok := slice.Append(123)
				assert.False(t, ok)
			}
		}()
	}
	wg.Wait()

	length = slice.Length()
	assert.Equal(t, 500*10000, length)
}

func Test_lockFreeLimitedSlice_ConcurrentlyLoad(t *testing.T) {
	slice := newLockFreeLimitedSlice(10000)
	for i := 0; i < 10000; i++ {
		_, ok := slice.Append(i)
		assert.True(t, ok)
	}

	var wg sync.WaitGroup
	wg.Add(500)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < 10000; j++ {
				num := slice.Load(j).(int)
				if num != j {
					assert.Equal(t, j, num) // assert.Equal较慢
				}
			}
		}()
	}
	wg.Wait()
}
