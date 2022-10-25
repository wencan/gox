package async

import "sync"

// Parallel 并行执行一组函数，等待执行完成。
// 如果其中一个或多个函数返回错误，优先选择前面函数的错误返回。
// 如果其中一个或多个函数panic，优先选择前面函数的recover()的非nil结果再次panic。
func Parallel(funcs ...func() error) error {
	var errs = make([]error, len(funcs))
	var recovereds = make([]interface{}, len(funcs))

	var wg sync.WaitGroup
	wg.Add(len(funcs))
	for i, f := range funcs {
		i, f := i, f
		go func() {
			defer wg.Done()
			defer func() {
				r := recover()
				recovereds[i] = r
			}()

			err := f()
			errs[i] = err
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
