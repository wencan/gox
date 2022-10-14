package sentinel

import (
	"context"
	"errors"
	"fmt"
)

func ExampleSentinelGroup_Do() {
	var sg SentinelGroup

	f := func(ctx context.Context, destPtr interface{}, args interface{}) error {
		resp := destPtr.(*string)
		req := args.(string)

		*resp = "echo: " + req

		return nil
	}

	var key string = "hello"
	var resp string
	var args = "Hello"
	err := sg.Do(context.TODO(), &resp, key, args, f)
	defer sg.Delete(key)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(resp)
	// Output: echo: Hello
}

func ExampleSentinelGroup_MDo() {
	var sg SentinelGroup

	f := func(ctx context.Context, destSlicePtr interface{}, argsSlice interface{}) ([]error, error) {
		resp := destSlicePtr.(*[]string)
		req := argsSlice.([]string)

		for _, r := range req {
			*resp = append(*resp, "echo: "+r)
		}
		return nil, nil
	}

	var keys = []string{"one", "two", "three"}
	var args = []string{"one", "two", "three"}
	var resp []string
	errs, err := sg.MDo(context.TODO(), &resp, keys, args, f)
	defer sg.Delete(keys...)

	if err != nil {
		fmt.Println(err)
		return
	}
	for _, res := range resp {
		fmt.Println(res)
	}
	for _, e := range errs {
		if e != nil {
			fmt.Println("partial failure:", e)
		}
	}

	// Output: echo: one
	// echo: two
	// echo: three
}

func ExampleSentinelGroup_partialFailure() {
	var sg SentinelGroup

	f := func(ctx context.Context, destSlicePtr interface{}, argsSlice interface{}) ([]error, error) {
		resp := destSlicePtr.(*[]string)
		req := argsSlice.([]string)

		var errs []error
		for _, r := range req {
			if r == "" {
				errs = append(errs, errors.New("skip"))
			} else {
				*resp = append(*resp, "echo: "+r)
				errs = append(errs, nil)
			}
		}
		return errs, nil
	}

	var keys = []string{"one", "skip", "three"}
	var args = []string{"one", "", "three"}
	var resp []string
	errs, err := sg.MDo(context.TODO(), &resp, keys, args, f)
	defer sg.Delete(keys...)

	if err != nil {
		fmt.Println(err)
		return
	}
	for _, res := range resp {
		fmt.Println(res)
	}
	for _, e := range errs {
		if e != nil {
			fmt.Println("partial failure:", e)
		}
	}

	// Output: echo: one
	// echo: three
	// partial failure: skip
}
