package async

// Series 串行执行一组函数，直至出错或者执行完。
func Series(funcs ...func() error) error {
	var err error
	for _, f := range funcs {
		err = f()
		if err != nil {
			break
		}
	}
	return err
}
