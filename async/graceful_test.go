package async

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGracefull(t *testing.T) {
	// 不Run
	graceful := DefaultGraceful.NewBranch("test")
	err := graceful.Wait(context.TODO())
	assert.Nil(t, err)
	assert.Empty(t, graceful.BusyBranches())

	// Run一个立即退出的函数
	go graceful.Run(func() {})
	time.Sleep(time.Millisecond * 100) // 等待协程启动
	err = graceful.Wait(context.TODO())
	assert.Nil(t, err)
	assert.Empty(t, graceful.BusyBranches())

	// Run一个0.2秒后退出的函数
	go graceful.Run(func() {
		time.Sleep(time.Millisecond * 200)
	})
	time.Sleep(time.Millisecond * 100) // 等待协程启动
	ctx, _ := context.WithTimeout(context.TODO(), time.Millisecond)
	err = graceful.Wait(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, []string{"test"}, graceful.BusyBranches())

	time.Sleep(time.Millisecond * 100)
	err = graceful.Wait(context.TODO())
	assert.Nil(t, err)
	assert.Empty(t, graceful.BusyBranches())

	// 同时Run两份
	go graceful.Run(func() {})
	go graceful.Run(func() {
		time.Sleep(time.Millisecond * 200)
	})
	time.Sleep(time.Millisecond * 100) // 等待协程启动
	ctx, _ = context.WithTimeout(context.TODO(), time.Millisecond)
	err = graceful.Wait(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, []string{"test"}, graceful.BusyBranches())
	time.Sleep(time.Millisecond * 100)
	err = graceful.Wait(context.TODO())
	assert.Nil(t, err)
	assert.Empty(t, graceful.BusyBranches())

}

func TestGracefull_branch(t *testing.T) {
	graceful := DefaultGraceful.NewBranch("test")
	branch := graceful.NewBranch("branch")
	err := graceful.Wait(context.TODO())
	assert.Nil(t, err)
	assert.Empty(t, graceful.BusyBranches())

	// Run一个立即退出的函数
	go branch.Run(func() {})
	time.Sleep(time.Millisecond * 100) // 等待协程启动
	err = graceful.Wait(context.TODO())
	assert.Nil(t, err)
	assert.Empty(t, graceful.BusyBranches())

	// Run一个0.2秒后退出的函数
	go branch.Run(func() {
		time.Sleep(time.Millisecond * 200)
	})
	time.Sleep(time.Millisecond * 100) // 等待协程启动
	ctx, _ := context.WithTimeout(context.TODO(), time.Millisecond)
	err = graceful.Wait(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, []string{"branch"}, graceful.BusyBranches())

	time.Sleep(time.Millisecond * 100)
	err = graceful.Wait(context.TODO())
	assert.Nil(t, err)
	assert.Empty(t, graceful.BusyBranches())

	// 同时Run两份
	go graceful.Run(func() {
		time.Sleep(time.Millisecond * 200)
	})
	go branch.Run(func() {
		time.Sleep(time.Millisecond * 300)
	})
	time.Sleep(time.Millisecond * 100) // 等待协程启动
	ctx, _ = context.WithTimeout(context.TODO(), time.Millisecond)
	err = graceful.Wait(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, []string{"test", "branch"}, graceful.BusyBranches())
	time.Sleep(time.Millisecond * 100)
	ctx, _ = context.WithTimeout(context.TODO(), time.Millisecond)
	err = graceful.Wait(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, []string{"branch"}, graceful.BusyBranches())
	ctx, _ = context.WithTimeout(context.TODO(), time.Millisecond*150)
	err = graceful.Wait(ctx)
	assert.Nil(t, err)
	assert.Empty(t, graceful.BusyBranches())
}
