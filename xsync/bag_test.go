package xsync

import (
	"math/rand"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBag(t *testing.T) {
	bag := NewBag()

	getAll := func() []int {
		ints := []int{}
		bag.Range(func(p interface{}) (stopIteration bool) {
			value := p.(int)
			ints = append(ints, value)
			return false
		})
		sort.Ints(ints)
		return ints
	}

	// 放 0-100
	delFuncs := make([]func(), 0, 100)
	for i := 0; i < 100; i++ {
		delFunc := bag.Add(i)
		delFuncs = append(delFuncs, delFunc)
	}

	// 输出全部
	all := getAll()
	want := make([]int, 0, 100)
	for i := 0; i < 100; i++ {
		want = append(want, i)
	}
	sort.Ints(want)
	assert.Equal(t, want, all)

	// 删除中间的0、10、20...
	for i := 0; i < 100; i++ {
		if i%10 == 0 {
			delFuncs[i]()
		}
	}
	// 再对比
	all = getAll()
	want = make([]int, 0, 90)
	for i := 0; i < 100; i++ {
		if i%10 != 0 {
			want = append(want, i)
		}
	}
	sort.Ints(want)
	assert.Equal(t, want, all)

	// 继续添加
	for i := 100; i < 200; i++ {
		bag.Add(i)
	}
	// 再对比
	all = getAll()
	want = make([]int, 0, 190)
	for i := 0; i < 200; i++ {
		if i%10 == 0 && i < 100 {
			continue
		}
		want = append(want, i)
	}
	sort.Ints(want)
	assert.Equal(t, want, all)
}

func TestBagConcurrentlyUpdate(t *testing.T) {
	bag := NewBag()
	big := 50000

	// 并发添加/删除
	var wg sync.WaitGroup
	delFuncChans := make(chan func(), big)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for _, num := range rand.Perm(big) {
				delFunc := bag.Add(num)
				delFuncChans <- delFunc

				delFunc = <-delFuncChans
				delFunc()
			}
		}()
	}

	// 这里添加的不删除
	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, num := range rand.Perm(big) {
			bag.Add(num)
		}
	}()

	wg.Wait()

	// 遍历检查
	all := make([]int, 0, big)
	bag.Range(func(p interface{}) (stopIteration bool) {
		num := p.(int)
		all = append(all, num)
		return false
	})
	sort.Ints(all)

	// 检查剩下的
	want := make([]int, 0, big)
	for i := 0; i < big; i++ {
		want = append(want, i)
	}
	assert.Equal(t, want, all)
}
