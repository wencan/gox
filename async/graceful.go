package async

import (
	"context"
	"runtime"
	"sync/atomic"

	"github.com/wencan/freesync"
)

// DefaultGraceful 默认的Graceful
var DefaultGraceful = Graceful{name: "default"}

// Graceful 运行并graceful退出。
type Graceful struct {
	name string

	branches freesync.Slice

	counter atomic.Uint64
}

// NewBranch 新建一个分支。
func (graceful *Graceful) NewBranch(name string) *Graceful {
	g := &Graceful{
		name: name,
	}
	graceful.branches.Append(g)

	return g
}

// BusyBranches 忙碌中的分支。
func (graceful *Graceful) BusyBranches() []string {
	var names []string

	if graceful.counter.Load() > 0 {
		names = append(names, graceful.name)
	}

	graceful.branches.Range(func(index int, p interface{}) (stopIteration bool) {
		branch := p.(*Graceful)
		names = append(names, branch.BusyBranches()...)
		return false
	})
	return names
}

// Run 新运行一个函数。
func (graceful *Graceful) Run(f func()) {
	graceful.counter.Add(1)
	defer graceful.counter.Add(^uint64(0))

	f()
}

// Wait 等待所有当前Graceful对象运行的函数退出，或者ctx错误。
// 执行Wait时，不应该再执行当前Graceful对象和Branch对象的Run。
func (graceful *Graceful) Wait(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if graceful.counter.Load() == 0 {
			var branchError error
			graceful.branches.Range(func(index int, p interface{}) (stopIteration bool) {
				branch := p.(*Graceful)
				err := branch.Wait(ctx)
				if err != nil {
					branchError = err
					return true
				}
				return false
			})
			if branchError != nil {
				return branchError
			}

			return nil
		}

		runtime.Gosched()
	}
}
