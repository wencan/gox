package xsync

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_lockFreeCappedSlice(t *testing.T) {
	slice := newLockFreeCappedSlice(10)

	for i := 0; i < 10; i++ {
		index, covered := slice.Append(i)
		assert.Equal(t, i, index)
		assert.Nil(t, covered)

		newbie := slice.Newbie().(int)
		assert.Equal(t, i, newbie)
	}

	for i := 0; i < 10; i++ {
		index, covered := slice.Append(i)
		assert.Equal(t, i, index)

		coveredValue := covered.(int)
		assert.Equal(t, i, coveredValue)
	}

	for i := 0; i < 10; i++ {
		num := slice.Load(i).(int)
		assert.Equal(t, i, num)
	}
}

func Test_lockFreeCappedSlice_ConcurrentlyAppend(t *testing.T) {
	slice := newLockFreeCappedSlice(500 * 10000)

	var wg sync.WaitGroup
	wg.Add(500)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < 10000; j++ {
				_, covered := slice.Append(j)
				assert.Nil(t, covered)
			}
		}()
	}
	wg.Wait()

	wg.Add(500)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < 10000; j++ {
				_, covered := slice.Append(j)
				assert.NotNil(t, covered)
			}
		}()
	}
	wg.Wait()
}

func Test_lockFreeCappedSlice_ConcurrentlyLoad(t *testing.T) {
	slice := newLockFreeCappedSlice(10000)
	for i := 0; i < 10000; i++ {
		slice.Append(i)
	}

	var wg sync.WaitGroup
	wg.Add(500)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < 10000; j++ {
				num := slice.Load(j).(int)
				if num != j {
					assert.Equal(t, j, num)
				}
			}
		}()
	}
	wg.Wait()
}
