package xsync

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestSlice_Append(t *testing.T) {
	var slice Slice

	for i := 0; i < 102400; i++ {
		index := slice.Append(i)
		if index != i {
			t.Fatalf("want index: %d, got index: %d", i, index)
		}
	}
	length := slice.Length()
	if length != 102400 {
		t.Fatalf("want length: %d, got length: %d", 102400, length)
	}
	for i := 0; i < 102400; i++ {
		if got, _ := slice.Load(i).(int); got != i {
			t.Fatalf("want value: %d, got value: %d", i, got)
		}
	}

	index1 := slice.Append("1")
	if got, _ := slice.Load(index1).(string); got != "1" {
		t.Fatalf("want 1, got %s", got)
	}

	index2 := slice.Append("2")
	if got, _ := slice.Load(index2).(string); got != "2" {
		t.Fatalf("want 2, got %s", got)
	}

	index3 := slice.Append("3")
	if got, _ := slice.Load(index3).(string); got != "3" {
		t.Fatalf("want 3, got %s", got)
	}
}

func TestSlice_ConcurrentlyAppend(t *testing.T) {
	var slice Slice

	var wg sync.WaitGroup
	wg.Add(500)
	letGo := make(chan int)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()

			r := rand.New(rand.NewSource(time.Now().UnixNano()))

			<-letGo

			for j := 0; j < 10000; j++ {
				num := r.Int()
				index := slice.Append(num)
				p := slice.Load(index)
				if p == nil {
					t.Errorf("Failed to load value by index %d", index)
					return
				}
				got, _ := p.(int)
				if got != num {
					t.Errorf("want %d, got %d", num, got)
				}
			}
		}()
	}

	time.Sleep(time.Millisecond * 200)
	close(letGo)

	wg.Wait()

	length := slice.Length()
	if length != 500*10000 {
		t.Errorf("want length: %d, got length: %d", 500*10000, length)
	}
}

func TestSlice_Range(t *testing.T) {
	var slice Slice

	for i := 0; i < 10240; i++ {
		slice.Append(i)
	}
	length := slice.Length()
	if length != 10240 {
		t.Fatalf("want length: %d, got length: %d", 10240, length)
	}

	var count int
	slice.Range(func(index int, p interface{}) (stopIteration bool) {
		if index != count {
			t.Fatalf("want index: %d, got index: %d", count, index)
		}

		num, ok := p.(int)
		if !ok {
			t.Fatalf("Failed to load p by index %d", index)
		}
		if num != count {
			t.Fatalf("want %d, got %d", count, num)
		}

		count++

		return false
	})
}

func TestSlice_UpdateAt(t *testing.T) {
	var slice Slice

	for i := 0; i < 10240; i++ {
		slice.Append(i)
	}

	for i := 0; i < 10240; i++ {
		slice.UpdateAt(i, i*2)
	}

	length := slice.Length()
	if length != 10240 {
		t.Fatalf("want length: %d, got length: %d", 10240, length)
	}

	for i := 0; i < 10240; i++ {
		num, _ := slice.Load(i).(int)
		if num != i*2 {
			t.Fatalf("want %d, got %d", i*2, num)
		}
	}
}

func TestSlice_ConcurrentlyUpdateAt(t *testing.T) {
	var slice Slice

	for i := 0; i < 2000; i++ {
		slice.Append(i)
	}

	var wg sync.WaitGroup
	wg.Add(500)
	letGo := make(chan int)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()

			<-letGo

			// 每个位置并发更新100次
			for j := 0; j < 2000; j++ {
				for k := 1; k <= 100; k++ {
					slice.UpdateAt(j, j*k)
				}
			}
		}()
	}

	time.Sleep(time.Millisecond * 200)
	close(letGo)

	wg.Wait()

	// 检查
	for i := 0; i < 2000; i++ {
		num, _ := slice.Load(i).(int)
		if num != i*100 {
			t.Fatalf("want %d, got %d", i*100, num)
		}
	}
}

func TestSlice_ConcurrentlyAppendAndUpdateAt(t *testing.T) {
	var slice Slice

	var wg sync.WaitGroup
	wg.Add(500)
	letGo := make(chan int)
	indexChann := make(chan int, 1000000)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()

			<-letGo

			for j := 0; j < 10000; j++ {
				// 随便塞个数据
				index := slice.Append(j)

				// 先自己更新一次
				slice.UpdateAt(index, index*2)

				// 请其它goroutine更新一次
				indexChann <- index

				// 更新其它的goroutine的数据一次
				index = <-indexChann
				num, _ := slice.Load(index).(int)
				slice.UpdateAt(index, num*5)
			}
		}()
	}

	time.Sleep(time.Millisecond * 200)
	close(letGo)

	wg.Wait()

	// 检查
	length := slice.Length()
	if length != 500*10000 {
		t.Errorf("want length: %d, got length: %d", 500*10000, length)
	}
	slice.Range(func(index int, p interface{}) (stopIteration bool) {
		num, _ := p.(int)
		if num != index*10 {
			t.Errorf("want %d, got %d", index*10, num)
			return true
		}
		return false
	})
}
