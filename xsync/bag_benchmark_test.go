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
