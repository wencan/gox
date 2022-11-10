package xsync

import "testing"

func BenchmarkSinglyLinkedList_pushAndPop(b *testing.B) {
	slist := newLockFreeSinglyLinkedList()
	for i := 0; i < 10000; i++ {
		slist.RightPush(i)
	}

	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		var i int
		for p.Next() {
			slist.RightPush(i)
			slist.LeftPop()

			i++
		}
	})
}

func BenchmarkChannel_pushAndPop(b *testing.B) {
	ch := make(chan int, 10000000)
	for i := 0; i < 10000; i++ {
		ch <- i
	}

	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		var i int
		for p.Next() {
			ch <- i
			<-ch

			i++
		}
	})
}
