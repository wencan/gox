package async

import (
	"errors"
	"testing"
	"time"
)

func TestParallel(t *testing.T) {
	type args struct {
		funcs []func() error
	}
	tests := []struct {
		name      string
		args      args
		wantPanic bool
		wantErr   bool
	}{
		{
			name: "ok",
			args: args{
				funcs: []func() error{
					func() error {
						return nil
					},
				},
			},
		},
		{
			name: "two",
			args: args{
				funcs: []func() error{
					func() error {
						return nil
					},
					func() error {
						time.Sleep(time.Millisecond * 10)
						return nil
					},
				},
			},
		},
		{
			name: "three",
			args: args{
				funcs: []func() error{
					func() error {
						return nil
					},
					func() error {
						time.Sleep(time.Millisecond * 10)
						return nil
					},
					func() error {
						time.Sleep(time.Millisecond * 20)
						return nil
					},
				},
			},
		},
		{
			name: "error",
			args: args{
				funcs: []func() error{
					func() error {
						return errors.New("wow")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "partial_error",
			args: args{
				funcs: []func() error{
					func() error {
						return errors.New("wow")
					},
					func() error {
						return nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "panic",
			args: args{
				funcs: []func() error{
					func() error {
						panic("wow")
					},
				},
			},
			wantPanic: true,
		},
		{
			name: "partial_panic",
			args: args{
				funcs: []func() error{
					func() error {
						return nil
					},
					func() error {
						panic("wow")
					},
				},
			},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			func() {
				defer func() {
					r := recover()
					if r == nil && tt.wantPanic {
						t.Errorf("recover() = %v, want panic: %t", r, tt.wantPanic)
					}
				}()

				if err := Parallel(tt.args.funcs...); (err != nil) != tt.wantErr {
					t.Errorf("Parallel() error = %v, wantErr %v", err, tt.wantErr)
				}
			}()
		})
	}
}

func TestParallelLimit(t *testing.T) {
	type args struct {
		limit int
		funcs []func() error
	}
	tests := []struct {
		name      string
		args      args
		wantPanic bool
		wantErr   bool
	}{
		{
			name: "ok",
			args: args{
				limit: 2,
				funcs: []func() error{
					func() error {
						return nil
					},
				},
			},
		},
		{
			name: "two",
			args: args{
				limit: 2,
				funcs: []func() error{
					func() error {
						return nil
					},
					func() error {
						time.Sleep(time.Millisecond * 10)
						return nil
					},
				},
			},
		},
		{
			name: "three",
			args: args{
				limit: 2,
				funcs: []func() error{
					func() error {
						return nil
					},
					func() error {
						time.Sleep(time.Millisecond * 10)
						return nil
					},
					func() error {
						time.Sleep(time.Millisecond * 20)
						return nil
					},
				},
			},
		},
		{
			name: "error",
			args: args{
				limit: 2,
				funcs: []func() error{
					func() error {
						return errors.New("wow")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "partial_error",
			args: args{
				limit: 2,
				funcs: []func() error{
					func() error {
						return errors.New("wow")
					},
					func() error {
						return nil
					},
					func() error {
						return errors.New("wow")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "panic",
			args: args{
				limit: 2,
				funcs: []func() error{
					func() error {
						panic("wow")
					},
					func() error {
						panic("wow")
					},
				},
			},
			wantPanic: true,
		},
		{
			name: "partial_panic",
			args: args{
				limit: 2,
				funcs: []func() error{
					func() error {
						return nil
					},
					func() error {
						panic("wow")
					},
				},
			},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			func() {
				defer func() {
					r := recover()
					if r == nil && tt.wantPanic {
						t.Errorf("recover() = %v, want panic: %t", r, tt.wantPanic)
					}
				}()

				if err := ParallelLimit(tt.args.limit, tt.args.funcs...); (err != nil) != tt.wantErr {
					t.Errorf("Parallel() error = %v, wantErr %v", err, tt.wantErr)
				}
			}()
		})
	}
}
