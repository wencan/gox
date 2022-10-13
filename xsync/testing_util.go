package xsync

import (
	"math/rand"
	"sort"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

// chaoticInts 将[]int随机打乱。
type chaoticInts []int

// Len 实现sort.Interface。
func (ints *chaoticInts) Len() int {
	return len(*ints)
}

// Less 实现sort.Interface。
func (ints *chaoticInts) Less(i, j int) bool {
	return r.Intn(2) == 0
}

// Swap 实现sort.Interface。
func (ints *chaoticInts) Swap(i, j int) {
	(*ints)[i], (*ints)[j] = (*ints)[j], (*ints)[i]
}

// generateNumberChaoticSequence 生成从0-length的数字序列，再打乱。
func generateNumberChaoticSequence(length int) []int {
	nums := make([]int, 0, length)
	for i := 0; i < length; i++ {
		nums = append(nums, i)
	}

	ints := chaoticInts(nums)
	sort.Sort(&ints)
	return []int(nums)
}

// generateNumberChaoticChannel 生成从0-length的数字序列，再打乱，塞进channel返回。
func generateNumberChaoticChannel(length int) <-chan int {
	nums := generateNumberChaoticSequence(length)
	ch := make(chan int, length)
	for _, num := range nums {
		ch <- num
	}
	close(ch)
	return ch
}
