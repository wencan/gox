package xcontainer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestListMap_PushBack(t *testing.T) {
	m := NewListMap()

	t1 := time.Now().Nanosecond()
	t2 := time.Now().Nanosecond()
	m.PushBack(1, t1)
	m.PushFront(2, t2)

	v, _ := m.Get(1)
	assert.Equal(t, t1, v)

	v1, _ := m.Back()
	v2, _ := m.Front()
	assert.Equal(t, t1, v1)
	assert.Equal(t, t2, v2)
	assert.Equal(t, 2, m.Len())
}

func TestListMap_PopFront(t *testing.T) {
	m := NewListMap()

	t1 := time.Now().Nanosecond()
	t2 := time.Now().Nanosecond()
	m.PushBack(1, t1)
	m.PushFront(2, t2)

	v, _ := m.PopFront()
	assert.Equal(t, t2, v)

	v, _ = m.Front()
	assert.Equal(t, t1, v)
	assert.Equal(t, 1, m.Len())

	m.PopFront()
	m.PopFront()
	assert.Equal(t, 0, m.Len())
}

func TestListMap_MoveToFront(t *testing.T) {
	m := NewListMap()

	t1 := time.Now().Nanosecond()
	t2 := time.Now().Nanosecond()
	m.PushBack(1, t1)
	m.PushFront(2, t2)

	m.MoveToFront(1)

	v1, _ := m.Back()
	v2, _ := m.Front()
	assert.Equal(t, t2, v1)
	assert.Equal(t, t1, v2)
}

func TestListMap_Range(t *testing.T) {
	m := NewListMap()

	m.PushFront(1, "1")
	m.PushBack(2, "2")
	m.PushFront(0, "0")
	m.PushBack(3, "3")

	keys := []int{}
	values := []string{}
	m.Range(func(key, value interface{}) (stopIteration bool) {
		keys = append(keys, key.(int))
		values = append(values, value.(string))
		return false
	})
	assert.Equal(t, []int{0, 1, 2, 3}, keys)
	assert.Equal(t, []string{"0", "1", "2", "3"}, values)

	keys = []int{}
	values = []string{}
	m.Range(func(key, value interface{}) (stopIteration bool) {
		keys = append(keys, key.(int))
		values = append(values, value.(string))

		if idx := key.(int); idx >= 2 {
			return true
		}
		return false
	})
	assert.Equal(t, []int{0, 1, 2}, keys)
	assert.Equal(t, []string{"0", "1", "2"}, values)
}
