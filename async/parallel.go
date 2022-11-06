package async

import (
	"sync"
	"sync/atomic"
)

// Parallel 并行执行一组函数，等待执行完成。
// 如果其中一个或多个函数返回错误，未执行的函数不再执行，并优先选择前面函数的错误返回。
// 如果其中一个或多个函数panic，未执行的函数不再执行，并优先选择前面函数的recover()的非nil结果再次panic。
func Parallel(funcs ...func() error) error {
	return ParallelLimit(len(funcs), funcs...)
}

// ParallelLimit 并行执行一组函数，等待执行完成。
// 每次调用最多只执行limit路并发执行。前面的函数优先执行。
// 如果其中一个或多个函数返回错误，未执行的函数不再执行，并优先选择前面函数的错误返回。
// 如果其中一个或多个函数panic，未执行的函数不再执行，并优先选择前面函数的recover()的非nil结果再次panic。
func ParallelLimit(limit int, funcs ...func() error) error {
	var errs = make([]error, len(funcs))
	var recovereds = make([]interface{}, len(funcs))
	var wg sync.WaitGroup
	var interrupted uint32

	var counter uint32
	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			var index int // 第几个函数

			defer func() {
				r := recover()
				if r != nil {
					atomic.SwapUint32(&interrupted, 1)
					recovereds[index] = r
				}
			}()

			for {
				if atomic.LoadUint32(&interrupted) != 0 {
					// 前面的函数错误或者panic
					// 不再执行后面的函数
					break
				}

				index = int(atomic.AddUint32(&counter, 1) - 1)
				if index >= len(funcs) {
					break
				}

				err := funcs[index]()
				if err != nil {
					atomic.SwapUint32(&interrupted, 1)
					errs[index] = err
				}
			}
		}()
	}

	wg.Wait()

	for _, r := range recovereds {
		if r != nil {
			panic(r)
		}
	}
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}
