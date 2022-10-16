package sentinel

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSentinelGroup_SimpleDo(t *testing.T) {
	type Response struct {
		Echo string
	}
	var count int
	fDo := func(ctx context.Context, dest, args interface{}) error {
		resp := dest.(*Response)
		num := args.(int)

		resp.Echo = fmt.Sprintf("echo: %d; %d", num, count)
		count++
		return nil
	}

	var sg SentinelGroup

	// 第一次，要执行函数
	var resp1 Response
	err := sg.Do(context.TODO(), &resp1, "123", 123, fDo)
	if !assert.Nil(t, err) {
		t.Fatal()
	}
	if !assert.Equal(t, "echo: 123; 0", resp1.Echo) {
		t.Fatal()
	}

	// 重复，直接取结果
	var resp2 Response
	err = sg.Do(context.TODO(), &resp2, "123", 123, fDo)
	if !assert.Nil(t, err) {
		t.Fatal()
	}
	if !assert.Equal(t, "echo: 123; 0", resp2.Echo) {
		t.Fatal()
	}

	// 删除key，再重复
	// 要重新执行函数
	sg.Delete("123")
	var resp3 Response
	err = sg.Do(context.TODO(), &resp3, "123", 123, fDo)
	if !assert.Nil(t, err) {
		t.Fatal()
	}
	if !assert.Equal(t, "echo: 123; 1", resp3.Echo) {
		t.Fatal()
	}
}

func TestSentinelGroup_ConcurrentlyDo(t *testing.T) {
	var big = 5 * 1000
	var sg SentinelGroup
	var flags = make([]uint64, big)
	var fDo = func(ctx context.Context, dest, args interface{}) error {
		resp := dest.(*string)
		index := args.(int)

		flag := atomic.AddUint64(&flags[index], 1) // 记下每个index被处理的次数

		*resp = fmt.Sprintf("index: %d, flag: %d", index, flag)
		return nil
	}

	rand.Seed(time.Now().UnixNano())

	var wg sync.WaitGroup
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer wg.Done()

			for _, index := range rand.Perm(big) {
				key := fmt.Sprintf("index_%d", index)
				want := fmt.Sprintf("index: %d, flag: %d", index, 1)

				var resp string
				err := sg.Do(context.TODO(), &resp, key, index, fDo)
				if assert.Nil(t, err, "index: %d", index) {
					if want != resp {
						assert.Equal(t, want, resp)
					}
				}
			}
		}()
	}
	wg.Wait()
}

func TestSentinelGroup_Do(t *testing.T) {
	type Request struct {
		Greetings string
	}
	type Response struct {
		Echo string
	}

	tests := []struct {
		name       string
		goroutines int
		newDestPtr func() interface{}
		key        string
		args       interface{}
		f          func(ctx context.Context, destPtr interface{}, args interface{}) error
		wantDest   interface{}
		wantErr    bool
	}{
		{
			name:       "simple",
			goroutines: 1000,
			newDestPtr: func() interface{} { return new(string) },
			key:        "simple",
			args:       "myargs",
			f: func(ctx context.Context, destPtr, args interface{}) error {
				str := destPtr.(*string)
				*str = fmt.Sprintf("args: %s", args)
				return nil
			},
			wantDest: "args: myargs",
		},
		{
			name:       "struct",
			goroutines: 1000,
			newDestPtr: func() interface{} { return new(Response) },
			key:        "struct",
			args:       Request{Greetings: "你好"},
			f: func(ctx context.Context, destPtr, args interface{}) error {
				resp := destPtr.(*Response)
				req := args.(Request)
				resp.Echo = req.Greetings
				return nil
			},
			wantDest: Response{Echo: "你好"},
		},
		{
			name:       "slice",
			goroutines: 1000,
			newDestPtr: func() interface{} { return &[]*Response{} },
			key:        "slice",
			args:       Request{Greetings: "你好"},
			f: func(ctx context.Context, destPtr, args interface{}) error {
				resp := destPtr.(*[]*Response)
				req := args.(Request)
				for i := 0; i < 3; i++ {
					*resp = append(*resp, &Response{Echo: fmt.Sprintf("%s_%d", req.Greetings, i)})
				}
				return nil
			},
			wantDest: []*Response{{Echo: "你好_0"}, {Echo: "你好_1"}, {Echo: "你好_2"}},
		},
		{
			name:       "args_is_nil",
			goroutines: 1000,
			newDestPtr: func() interface{} { return new(Response) },
			key:        "args_is_nil",
			args:       nil,
			f: func(ctx context.Context, destPtr, args interface{}) error {
				resp := destPtr.(*Response)
				resp.Echo = "你好"
				return nil
			},
			wantDest: Response{Echo: "你好"},
		},
		{
			name:       "error",
			goroutines: 1000,
			newDestPtr: func() interface{} { return new(Response) },
			key:        "error",
			args:       Request{Greetings: "你好"},
			f: func(ctx context.Context, destPtr, args interface{}) error {
				return errors.New("test")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sg SentinelGroup

			var wg sync.WaitGroup
			// tt.goroutines = 1
			wg.Add(tt.goroutines)
			for i := 0; i < tt.goroutines; i++ {
				go func() {
					defer wg.Done()

					destPtr := tt.newDestPtr()
					err := sg.Do(context.TODO(), destPtr, tt.key, tt.args, tt.f)
					if tt.wantErr {
						assert.NotNil(t, err)
					} else {
						if assert.Nil(t, err) {
							assert.Equal(t, tt.wantDest, reflect.ValueOf(destPtr).Elem().Interface())
						}
					}
				}()
			}
			wg.Wait()
		})
	}
}

func TestSentinelGroup_SimpleMDo(t *testing.T) {
	type Request struct {
		Greetings string
	}
	type Response struct {
		Echo string
	}

	var sg SentinelGroup

	var count int
	fDo := func(ctx context.Context, destSlicePtr interface{}, argsSlice interface{}) ([]error, error) {
		resp := destSlicePtr.(*[]*Response)
		req := argsSlice.([]*Request)

		for _, r := range req {
			*resp = append(*resp, &Response{Echo: fmt.Sprintf("echo: %s, count: %d", r.Greetings, count)})
			count++
		}

		return nil, nil
	}

	// 第一次
	var keys1 = []string{"one"}
	var resp1 []*Response
	var args1 = []*Request{{Greetings: "one"}}
	var want1 = []*Response{{Echo: "echo: one, count: 0"}}
	_, err := sg.MDo(context.TODO(), &resp1, keys1, args1, fDo)
	if assert.Nil(t, err) {
		assert.Equal(t, want1, resp1)
	}

	// wait保存的结果
	var keys2 = []string{"one"}
	var resp2 []*Response
	var args2 = []*Request{{Greetings: "one"}}
	var want2 = []*Response{{Echo: "echo: one, count: 0"}}
	_, err = sg.MDo(context.TODO(), &resp2, keys2, args2, fDo)
	if assert.Nil(t, err) {
		assert.Equal(t, want2, resp2)
	}

	// wait第一次保存的结果 + 第二次执行
	var keys3 = []string{"one", "two"}
	var resp3 []*Response
	var args3 = []*Request{{Greetings: "one"}, {Greetings: "two"}}
	var want3 = []*Response{{Echo: "echo: one, count: 0"}, {Echo: "echo: two, count: 1"}}
	_, err = sg.MDo(context.TODO(), &resp3, keys3, args3, fDo)
	if assert.Nil(t, err) {
		assert.Equal(t, want3, resp3)
	}

	// wait 保存的两个结果
	var keys4 = []string{"one", "two"}
	var resp4 []*Response
	var args4 = []*Request{{Greetings: "one"}, {Greetings: "two"}}
	var want4 = []*Response{{Echo: "echo: one, count: 0"}, {Echo: "echo: two, count: 1"}}
	_, err = sg.MDo(context.TODO(), &resp4, keys4, args4, fDo)
	if assert.Nil(t, err) {
		assert.Equal(t, want4, resp4)
	}

	// wait 保存的两个结果
	// 顺序反过来
	var keys5 = []string{"two", "one"}
	var resp5 []*Response
	var args5 = []*Request{{Greetings: "two"}, {Greetings: "one"}}
	var want5 = []*Response{{Echo: "echo: two, count: 1"}, {Echo: "echo: one, count: 0"}}
	_, err = sg.MDo(context.TODO(), &resp5, keys5, args5, fDo)
	if assert.Nil(t, err) {
		assert.Equal(t, want5, resp5)
	}
}

func TestSentinelGroup_ConcurrentlyMDo(t *testing.T) {
	type Request struct {
		Index int
	}
	type Response struct {
		Echo string
	}

	var big = 10000
	var sg SentinelGroup
	var flags = make([]uint64, big)
	var fDo = func(ctx context.Context, destSlicePtr interface{}, argsSlice interface{}) ([]error, error) {
		resp := destSlicePtr.(*[]*Response)
		req := argsSlice.([]*Request)

		for _, r := range req {
			flag := atomic.AddUint64(&flags[r.Index], 1) // 记下每个index被处理的次数
			*resp = append(*resp, &Response{
				Echo: fmt.Sprintf("index: %d, flag: %d", r.Index, flag),
			})
		}
		return nil, nil
	}

	rand.Seed(time.Now().UnixNano())

	var wg sync.WaitGroup
	wg.Add(500)
	for i := 0; i < 500; i++ {
		go func() {
			defer wg.Done()

			all := rand.Perm(big)
			var count int
			for {
				var keys []string
				var args []*Request
				var want []*Response

				for i := 0; i < rand.Intn(9)+1; i++ {
					if count+1 > len(all) {
						break
					}
					index := all[count]
					name := fmt.Sprintf("index_%d", index)
					keys = append(keys, name)
					args = append(args, &Request{Index: index})
					want = append(want, &Response{Echo: fmt.Sprintf("index: %d, flag: %d", index, 1)})

					count++
				}
				if len(keys) == 0 {
					break // end
				}

				var resp []*Response
				_, err := sg.MDo(context.TODO(), &resp, keys, args, fDo)
				if assert.Nil(t, err) {
					if !assert.Equal(t, want, resp) {
						t.Logf("%+v, %+v", want, resp)
					}
				}
			}
		}()
	}
	wg.Wait()
}

func TestSentinelGroup_MDo(t *testing.T) {
	type Request struct {
		Greetings string
	}
	type Response struct {
		Echo string
	}

	tests := []struct {
		name            string
		goroutines      int
		newDestSlicePtr func() interface{}
		keys            []string
		argsSlice       interface{}
		fDo             func(ctx context.Context, destSlicePtr interface{}, argsSlice interface{}) ([]error, error)
		want            interface{}
		wantErrs        []bool
		wantErr         bool
	}{
		{
			name:            "1",
			goroutines:      1000,
			newDestSlicePtr: func() interface{} { return new([]string) },
			keys:            []string{"1"},
			argsSlice:       []int{1},
			fDo: func(ctx context.Context, destSlicePtr, argsSlice interface{}) ([]error, error) {
				resp := destSlicePtr.(*[]string)
				req := argsSlice.([]int)

				for _, r := range req {
					*resp = append(*resp, fmt.Sprintf("hi_%d", r))
				}
				return nil, nil
			},
			want: []string{"hi_1"},
		},
		{
			name:            "1_2",
			goroutines:      1000,
			newDestSlicePtr: func() interface{} { return new([]string) },
			keys:            []string{"1", "2"},
			argsSlice:       []int{1, 2},
			fDo: func(ctx context.Context, destSlicePtr, argsSlice interface{}) ([]error, error) {
				resp := destSlicePtr.(*[]string)
				req := argsSlice.([]int)

				for _, r := range req {
					*resp = append(*resp, fmt.Sprintf("hi_%d", r))
				}
				return nil, nil
			},
			want: []string{"hi_1", "hi_2"},
		},
		{
			name:            "struct_one",
			goroutines:      1000,
			newDestSlicePtr: func() interface{} { return new([]*Response) },
			keys:            []string{"struct_one"},
			argsSlice:       []*Request{{Greetings: "one"}},
			fDo: func(ctx context.Context, destSlicePtr, argsSlice interface{}) ([]error, error) {
				req := argsSlice.([]*Request)
				resp := destSlicePtr.(*[]*Response)

				for _, r := range req {
					*resp = append(*resp, &Response{Echo: "echo: " + r.Greetings})
				}
				return nil, nil
			},
			want: []*Response{{Echo: "echo: one"}},
		},
		{
			name:            "struct_one_two",
			goroutines:      1000,
			newDestSlicePtr: func() interface{} { return new([]*Response) },
			keys:            []string{"struct_one", "struct_two"},
			argsSlice:       []*Request{{Greetings: "one"}, {Greetings: "two"}},
			fDo: func(ctx context.Context, destSlicePtr, argsSlice interface{}) ([]error, error) {
				req := argsSlice.([]*Request)
				resp := destSlicePtr.(*[]*Response)

				for _, r := range req {
					*resp = append(*resp, &Response{Echo: "echo: " + r.Greetings})
				}
				return nil, nil
			},
			want: []*Response{{Echo: "echo: one"}, {Echo: "echo: two"}},
		},
		{
			name:            "struct_one_two_three",
			goroutines:      1000,
			newDestSlicePtr: func() interface{} { return new([]*Response) },
			keys:            []string{"struct_one", "struct_two", "struct_three"},
			argsSlice:       []*Request{{Greetings: "one"}, {Greetings: "two"}, {Greetings: "three"}},
			fDo: func(ctx context.Context, destSlicePtr, argsSlice interface{}) ([]error, error) {
				req := argsSlice.([]*Request)
				resp := destSlicePtr.(*[]*Response)

				for _, r := range req {
					*resp = append(*resp, &Response{Echo: "echo: " + r.Greetings})
				}
				return nil, nil
			},
			want: []*Response{{Echo: "echo: one"}, {Echo: "echo: two"}, {Echo: "echo: three"}},
		},
		{
			name:            "struct_one_2_tee",
			goroutines:      1000,
			newDestSlicePtr: func() interface{} { return new([]*Response) },
			keys:            []string{"struct_one", "struct_two", "struct_three", "struct_four", "struct_five", "struct_six", "struct_seven", "struct_eight", "struct_nine", "struct_ten"},
			argsSlice:       []*Request{{Greetings: "one"}, {Greetings: "two"}, {Greetings: "three"}, {Greetings: "four"}, {Greetings: "five"}, {Greetings: "six"}, {Greetings: "seven"}, {Greetings: "eight"}, {Greetings: "nine"}, {Greetings: "ten"}},
			fDo: func(ctx context.Context, destSlicePtr, argsSlice interface{}) ([]error, error) {
				req := argsSlice.([]*Request)
				resp := destSlicePtr.(*[]*Response)

				for _, r := range req {
					*resp = append(*resp, &Response{Echo: "echo: " + r.Greetings})
				}
				return nil, nil
			},
			want: []*Response{{Echo: "echo: one"}, {Echo: "echo: two"}, {Echo: "echo: three"}, {Echo: "echo: four"}, {Echo: "echo: five"}, {Echo: "echo: six"}, {Echo: "echo: seven"}, {Echo: "echo: eight"}, {Echo: "echo: nine"}, {Echo: "echo: ten"}},
		},
		{
			name:            "struct_one_notfound_three",
			goroutines:      1000,
			newDestSlicePtr: func() interface{} { return new([]*Response) },
			keys:            []string{"struct_one", "struct_notfound", "struct_three"},
			argsSlice:       []*Request{{Greetings: "one"}, {Greetings: "notfound"}, {Greetings: "three"}},
			fDo: func(ctx context.Context, destSlicePtr, argsSlice interface{}) ([]error, error) {
				req := argsSlice.([]*Request)
				resp := destSlicePtr.(*[]*Response)

				var errs []error
				for _, r := range req {
					if strings.HasPrefix(r.Greetings, "notfound") {
						errs = append(errs, errors.New("notfound"))
					} else {
						*resp = append(*resp, &Response{Echo: "echo: " + r.Greetings})
						errs = append(errs, nil)
					}
				}
				return errs, nil
			},
			want:     []*Response{{Echo: "echo: one"}, {Echo: "echo: three"}},
			wantErrs: []bool{false, true, false},
		},
		{
			name:            "do_error",
			goroutines:      1000,
			newDestSlicePtr: func() interface{} { return new([]*Response) },
			keys:            []string{"do_error"},
			argsSlice:       []*Request{{Greetings: "one"}},
			fDo: func(ctx context.Context, destSlicePtr, argsSlice interface{}) ([]error, error) {
				return nil, errors.New("wow")
			},
			want:     *(new([]*Response)),
			wantErr:  false,
			wantErrs: []bool{true},
		},
		{
			name:            "do_partial_error",
			goroutines:      1000,
			newDestSlicePtr: func() interface{} { return new([]*Response) },
			keys:            []string{"do_partial_error_one", "do_partial_error1", "do_partial_error_two", "do_partial_error2"},
			argsSlice:       []*Request{{Greetings: "one"}, {Greetings: ""}, {Greetings: "two"}, {Greetings: ""}},
			fDo: func(ctx context.Context, destSlicePtr, argsSlice interface{}) ([]error, error) {
				req := argsSlice.([]*Request)
				resp := destSlicePtr.(*[]*Response)

				var errs []error
				for _, r := range req {
					switch r.Greetings {
					case "":
						errs = append(errs, errors.New("wow"))
					default:
						*resp = append(*resp, &Response{Echo: "echo: " + r.Greetings})
						errs = append(errs, nil)
					}
				}
				return errs, nil
			},
			want:     []*Response{{Echo: "echo: one"}, {Echo: "echo: two"}},
			wantErr:  false,
			wantErrs: []bool{false, true, false, true},
		},
		{
			name:            "empty",
			goroutines:      1000,
			newDestSlicePtr: func() interface{} { return new([]*Response) },
			keys:            []string{},
			argsSlice:       []*Request{},
			fDo:             nil,
			want:            *(new([]*Response)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sg SentinelGroup

			var wg sync.WaitGroup
			tt.goroutines = 1
			wg.Add(tt.goroutines)
			for i := 0; i < tt.goroutines; i++ {
				go func() {
					defer wg.Done()

					destSlicePtr := tt.newDestSlicePtr()
					errs, err := sg.MDo(context.TODO(), destSlicePtr, tt.keys, tt.argsSlice, tt.fDo)

					if tt.wantErr {
						assert.NotNil(t, err)
					} else {
						if assert.Nil(t, err) {
							if !assert.Equal(t, tt.want, reflect.ValueOf(destSlicePtr).Elem().Interface()) {
								t.Log(tt.want == nil)
								t.Log(reflect.ValueOf(destSlicePtr).Elem().Interface() == nil)
								t.Log(reflect.ValueOf(destSlicePtr).Interface() == destSlicePtr)
								t.Log(reflect.ValueOf(destSlicePtr).Elem().Len())
							}
						}
					}

					var gotErrs []bool
					for _, err := range errs {
						if err != nil {
							gotErrs = append(gotErrs, true)
						} else {
							gotErrs = append(gotErrs, false)
						}
					}
					assert.Equal(t, tt.wantErrs, gotErrs)
				}()
			}
			wg.Wait()
		})
	}
}
