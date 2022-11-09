package xsync

import (
	"sync"
	"testing"
)

func BenchmarkBagWrite(b *testing.B) {
	bag := NewBag()

	ch := make(chan func(), 10000000)

	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		var i int
		for p.Next() {
			delFunc := bag.Add(i)
			ch <- delFunc

			i++

			delFunc = <-ch
			delFunc()
		}
	})
}

func BenchmarkSyncMapWrite(b *testing.B) {
	var mapping sync.Map

	ch := make(chan int, 10000000)

	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		var i int
		for p.Next() {
			mapping.Store(i, i)
			ch <- i

			i++

			delI := <-ch
			mapping.Delete(delI)
		}
	})
}

func BenchmarkBagRange(b *testing.B) {
	bag := NewBag()

	for i := 0; i < 10000; i++ {
		bag.Add(i)
	}

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			bag.Range(func(p interface{}) (stopIteration bool) {
				return false
			})
		}
	})
}

func BenchmarkSyncMapRange(b *testing.B) {
	var mapping sync.Map

	for i := 0; i < 10000; i++ {
		mapping.Store(i, i)
	}

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			mapping.Range(func(key, value any) bool {
				return true
			})
		}
	})
}