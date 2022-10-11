package xsync

import (
	"testing"

	lru "github.com/wencan/gox/xsync/internal/golang-lru"
)

func BenchmarkLRUMap_Store(b *testing.B) {
	m := NewLRUMap(10000, 10)

	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		var i int
		for p.Next() {
			m.Store(i, i)

			i++
		}
	})
}

func BenchmarkGroupCacheLRU_Store(b *testing.B) {
	m, err := lru.New(10000 * 10)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		var i int
		for p.Next() {
			m.Add(i, i)

			i++
		}
	})
}

func BenchmarkLRUMap_Load(b *testing.B) {
	m := NewLRUMap(10000, 10)
	for i := 0; i < 10000*10; i++ {
		m.Store(i, i)
	}

	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		var i int
		for p.Next() {
			m.Load(i)

			i++
			if i >= 10000 {
				i = 0
			}
		}
	})
}

func BenchmarkGroupCacheLRU_Load(b *testing.B) {
	m, err := lru.New(10000 * 10)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 10000*10; i++ {
		m.Add(i, i)
	}

	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		var i int
		for p.Next() {
			m.Get(i)

			i++
			if i >= 10000 {
				i = 0
			}
		}
	})
}
