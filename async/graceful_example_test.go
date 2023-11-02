package async

import (
	"context"
	"fmt"
	"time"
)

func ExampleGraceful() {
	go func() {
		DefaultGraceful.Run(func() {
			// nothing
		})
	}()

	branch := DefaultGraceful.NewBranch("branch_1")
	go func() {
		branch.Run(func() {
			time.Sleep(time.Millisecond * 500)
		})
	}()

	// 等待两个goroutine已经运行
	time.Sleep(time.Millisecond * 100)

	// 程序退出时……

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	err := DefaultGraceful.Wait(ctx)
	if err != nil {
		fmt.Println(err)

		busyBranches := DefaultGraceful.BusyBranches()
		fmt.Println(busyBranches)

		// 直接结束进程，不再等待分支Run的过程
	}

	// Output:
	// context deadline exceeded
	// [branch_1]
}
