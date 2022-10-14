package sentinel

import (
	"context"
	"errors"
	"reflect"
)

// Sentinel 哨兵。一个生产者，多个消费者等待生产者完成并提交结果。
// 从https://github.com/wencan/cachex/blob/master/sentinel.go修改来。
type Sentinel struct {
	flag   chan interface{}
	closed bool

	// done 是否已经执行Done()
	done bool

	result interface{}
	err    error
}

// NewSentinel 新建哨兵
func NewSentinel() *Sentinel {
	return &Sentinel{
		flag: make(chan interface{}),
	}
}

// Done 生产者提交结果。
// Wait的resultPtr是指向Done的result的指针。
func (s *Sentinel) Done(result interface{}, err error) {
	s.done = true
	s.result = result
	s.err = err

	close(s.flag)
	s.closed = true
}

// Wait 消费者等待生产者提交结果。
// Wait的resultPtr是指向Done的result的指针。
func (s *Sentinel) Wait(ctx context.Context, resultPtr interface{}) error {
	ptr := reflect.ValueOf(resultPtr)
	if ptr.Kind() != reflect.Ptr || ptr.IsNil() {
		panic("value must is a non-nil pointer")
	}

	select {
	case <-s.flag:
	case <-ctx.Done():
		return ctx.Err()
	}

	if !s.done {
		// 没done，却返回了，说明还没done，s.flag就被close了。
		return errors.New("internal error")
	}

	if s.err != nil {
		return s.err
	}

	if s.result != nil {
		value := reflect.ValueOf(s.result)
		reflect.ValueOf(resultPtr).Elem().Set(value)
	}

	return nil
}

// Close 关闭。
func (s *Sentinel) Close() {
	if !s.closed {
		close(s.flag)
		s.closed = true
	}
}
