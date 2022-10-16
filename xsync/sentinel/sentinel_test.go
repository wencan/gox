package sentinel

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSentinel_Wait(t *testing.T) {
	type Response struct {
		Echo string
	}

	tests := []struct {
		name             string
		goroutines       int
		putResult        interface{}
		putErr           error
		newPullResultPtr func() interface{}
		wantResult       interface{}
		wantErr          bool
	}{
		{
			name:             "put_int",
			goroutines:       500,
			putResult:        1,
			putErr:           nil,
			newPullResultPtr: func() interface{} { var num int; return &num },
			wantResult:       1,
			wantErr:          false,
		},
		{
			name:             "put_struct_response",
			goroutines:       500,
			putResult:        Response{Echo: "nihao"},
			putErr:           nil,
			newPullResultPtr: func() interface{} { return &Response{} },
			wantResult:       Response{Echo: "nihao"},
			wantErr:          false,
		},
		{
			name:             "put_error",
			goroutines:       500,
			putResult:        nil,
			putErr:           errors.New("test"),
			newPullResultPtr: func() interface{} { return &Response{} },
			wantResult:       nil,
			wantErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSentinel()
			defer s.Close()

			var wg sync.WaitGroup
			wg.Add(tt.goroutines)
			for i := 0; i < tt.goroutines; i++ {
				go func() {
					defer wg.Done()

					dest := tt.newPullResultPtr()
					err := s.Wait(context.TODO(), dest)
					if tt.wantErr {
						assert.NotNil(t, err)
					} else {
						if assert.Nil(t, err) {
							assert.Equal(t, tt.wantResult, reflect.ValueOf(dest).Elem().Interface())
						}
					}
				}()
			}

			s.Done(tt.putResult, tt.putErr)

			wg.Wait()
		})
	}
}
