package async

import (
	"errors"
	"testing"
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
