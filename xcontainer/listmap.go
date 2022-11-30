package xcontainer

import (
	"container/list"
	"sync"
)

type listEntry struct {
	key   interface{}
	value interface{}
}

var listEntryPool = sync.Pool{New: func() interface{} {
	return &listEntry{}
}}

// ListMap 同时具备List和Map的特性。
// 非并发安全。
// 来自：https://github.com/wencan/cachex。
type ListMap struct {
	sequence *list.List
	mapping  map[interface{}]*list.Element
}

// NewListMap 新建一个NewListMap。
func NewListMap() *ListMap {
	return &ListMap{
		sequence: list.New(),
		mapping:  make(map[interface{}]*list.Element),
	}
}

// Get 根据key获取一个值。
func (m *ListMap) Get(key interface{}) (value interface{}, ok bool) {
	elem, ok := m.mapping[key]
	if ok {
		entry := elem.Value.(*listEntry)
		return entry.value, ok
	}
	return nil, false
}

// Pop 根据key获取并删除一个值。
func (m *ListMap) Pop(key interface{}) (value interface{}, ok bool) {
	elem, ok := m.mapping[key]
	if ok {
		delete(m.mapping, key)
		m.sequence.Remove(elem)

		entry := elem.Value.(*listEntry)
		value := entry.value

		listEntryPool.Put(entry)

		return value, true
	}
	return nil, false
}

// Len 长度。
func (m *ListMap) Len() int {
	return m.sequence.Len()
}

// Front 最前的值。
func (m *ListMap) Front() (value interface{}, ok bool) {
	elem := m.sequence.Front()
	if elem != nil {
		entry := elem.Value.(*listEntry)
		return entry.value, true
	}
	return nil, false
}

// Back 最后的值。
func (m *ListMap) Back() (value interface{}, ok bool) {
	elem := m.sequence.Back()
	if elem != nil {
		entry := elem.Value.(*listEntry)
		return entry.value, true
	}
	return nil, false
}

// PopFront 获取并删除最前的值。
func (m *ListMap) PopFront() (value interface{}, ok bool) {
	elem := m.sequence.Front()
	if elem != nil {
		entry := elem.Value.(*listEntry)

		m.sequence.Remove(elem)
		delete(m.mapping, entry.key)
		value = entry.value

		listEntryPool.Put(entry)

		return value, true
	}
	return nil, false
}

// PopBack 获取并删除最后的值。
func (m *ListMap) PopBack() (value interface{}, ok bool) {
	elem := m.sequence.Back()
	if elem != nil {
		entry := elem.Value.(*listEntry)

		m.sequence.Remove(elem)
		delete(m.mapping, entry.key)
		value = entry.value

		listEntryPool.Put(entry)

		return value, true
	}
	return nil, false
}

// PushFront 新增最前的值。
func (m *ListMap) PushFront(key interface{}, value interface{}) {
	entry := listEntryPool.Get().(*listEntry)
	entry.key = key
	entry.value = value

	elem := m.sequence.PushFront(entry)
	m.mapping[key] = elem
}

// PushBack 新增最后的值。
func (m *ListMap) PushBack(key interface{}, value interface{}) {
	entry := listEntryPool.Get().(*listEntry)
	entry.key = key
	entry.value = value

	elem := m.sequence.PushBack(entry)
	m.mapping[key] = elem
}

// MoveToFront 将kv对移到最前。
func (m *ListMap) MoveToFront(key interface{}) {
	elem, ok := m.mapping[key]
	if ok {
		m.sequence.MoveToFront(elem)
	}
}

// MoveToBack 将kv对移到最后。
func (m *ListMap) MoveToBack(key interface{}) {
	elem, ok := m.mapping[key]
	if ok {
		m.sequence.MoveToBack(elem)
	}
}

// Range 从前面开始，遍历所有元素。
func (m *ListMap) Range(f func(key, value interface{}) (stopIteration bool)) {
	elem := m.sequence.Front()
	for elem != nil {
		entry := elem.Value.(*listEntry)
		stopIteration := f(entry.key, entry.value)
		if stopIteration {
			return
		}

		elem = elem.Next()
	}
}

// Clear 清理全部数据。
func (m *ListMap) Clear() {
	for _, elem := range m.mapping {
		listEntryPool.Put(elem.Value)
	}

	m.mapping = make(map[interface{}]*list.Element)
	m.sequence = list.New()
}
