package sentinel

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// SentinelGroup 哨兵组。
type SentinelGroup struct {
	sentinelMap sync.Map
}

// Do key删除前，不重复执行key相同的逻辑。
// 修改自：https://github.com/wencan/cachex/blob/master/cachex.go
func (sg *SentinelGroup) Do(ctx context.Context, destPtr interface{}, key string, args interface{}, f func(ctx context.Context, destPtr interface{}, args interface{}) error) error {
	sentinel := NewSentinel()
	defer sentinel.Close()
	actual, loaded := sg.sentinelMap.LoadOrStore(key, sentinel) // 这里性能不是很好。尤其是写多时
	if loaded {
		// 由其它过程执行
		// 这里等待其它逻辑的执行结果
		waitSentinel := actual.(*Sentinel)
		err := waitSentinel.Wait(ctx, destPtr)
		return err
	} else {
		// do it
		err := f(ctx, destPtr, args)
		dest := reflect.ValueOf(destPtr).Elem().Interface()
		sentinel.Done(dest, err)
		return err
	}
}

// MDO 处理一批数据。key删除前，不重复执行key相同的逻辑。
// destSlicePtr 是获取结果的切片的指针。
// argsSlice 是给函数f的参数，顺序和长度等于keys的顺序和长度。
// []error表示各个下标位置上的错误。如果没有错误，可以为nil。
// 函数f返回destSlicePtr顺序同keys/argsSlice顺序，destSlicePtr中缺失项必须在返回[]error的相同下标位置有error。
func (sg *SentinelGroup) MDo(ctx context.Context, destSlicePtr interface{}, keys []string, argsSlice interface{}, f func(ctx context.Context, destSlicePtr interface{}, argsSlice interface{}) ([]error, error)) ([]error, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	var waitSentinelMap = make(map[int]*Sentinel)
	var doSentinelMap = make(map[int]*Sentinel)

	var sentinal *Sentinel
	for index, key := range keys {
		if sentinal == nil { // 如果已经使用，创建一个新的
			sentinal = NewSentinel()
		}
		actual, loaded := sg.sentinelMap.LoadOrStore(key, sentinal) // 这里性能不是很好。尤其是写多时
		if loaded {
			// 由其它过程执行
			// 这里等待其它逻辑的执行结果
			waitSentinel := actual.(*Sentinel)
			waitSentinelMap[index] = waitSentinel
		} else {
			// 需要下面执行的
			doSentinelMap[index] = sentinal
			sentinal = nil // 已经使用，下次需要重新创建
		}
	}
	if sentinal != nil { // 多余的
		sentinal.Close()
	}

	// 清理
	defer func() {
		for _, sentinel := range doSentinelMap {
			sentinel.Close() // 如果panic，通知wait的逻辑返回
		}
		for _, sentinel := range waitSentinelMap {
			sentinel.Close()
		}
	}()

	destSliceValue := reflect.ValueOf(destSlicePtr).Elem()
	destElemResultMap := make(map[int]reflect.Value) // 每个下标位置上的结果
	destElemErrorMap := make(map[int]error)          // 每个下标位置上的错误

	// 准备执行函数f
	if len(doSentinelMap) > 0 {
		// 参数
		argsSliceValue := reflect.ValueOf(argsSlice)
		for argsSliceValue.Kind() == reflect.Ptr {
			argsSliceValue = reflect.Indirect(argsSliceValue)
		}
		actualArgsSliceValue := reflect.MakeSlice(argsSliceValue.Type(), 0, len(doSentinelMap))
		indexMap := make(map[int]int, len(doSentinelMap)) // 外部索引 -> 执行函数参数序列中的索引
		var argCount int
		for index := 0; index < argsSliceValue.Len(); index++ { // 保证按照原顺序
			_, ok := doSentinelMap[index]
			if ok {
				actualArgsSliceValue = reflect.Append(actualArgsSliceValue, argsSliceValue.Index(index))
				indexMap[index] = argCount
				argCount++
			}
		}
		// 执行
		actualDestSlicePtr := reflect.New(destSliceValue.Type())
		errs, err := f(ctx, actualDestSlicePtr.Interface(), actualArgsSliceValue.Interface())
		if err != nil {
			for _, sentinel := range doSentinelMap {
				sentinel.Done(nil, err)
			}
			return nil, err
		}
		actualDestSlice := reflect.Indirect(actualDestSlicePtr)

		// 取结果
		var actualDestCount int
		for index := range keys {
			sentinel, ok := doSentinelMap[index]
			if !ok {
				continue
			}

			var result interface{}
			var err error
			doIdx, ok := indexMap[index] // 在执行函数参数序列中的下标
			if !ok {
				panic("wrong index")
			}
			if len(errs) > doIdx { // 允许省去后面的nil
				err = errs[doIdx]
				if err != nil {
					destElemErrorMap[index] = err
				}
			}
			if err == nil {
				if actualDestSlice.Len() <= actualDestCount {
					return nil, errors.New("not enough results")
				}
				value := actualDestSlice.Index(actualDestCount)
				result = value.Interface()
				destElemResultMap[index] = value
				actualDestCount++
			}

			// 通知其它在等待的过程
			sentinel.Done(result, err)
		}
	}

	//  等待其它过程完成
	if len(waitSentinelMap) > 0 {
		for index, sentinal := range waitSentinelMap {
			valuePtr := reflect.New(destSliceValue.Type().Elem())
			err := sentinal.Wait(ctx, valuePtr.Interface())
			if err != nil {
				destElemErrorMap[index] = err
			} else {
				destElemResultMap[index] = valuePtr.Elem()
			}
		}
	}

	// 准备返回的结果
	var errs []error
	for i := 0; i < len(keys); i++ {
		elemValue, ok := destElemResultMap[i]
		elemErr := destElemErrorMap[i]

		if elemErr != nil {
			errs = append(errs, elemErr)
		} else {
			if !ok {
				return nil, fmt.Errorf("not found result by key %s", keys[i])
			}
			destSliceValue.Set(reflect.Append(destSliceValue, elemValue))
			errs = append(errs, nil)
		}
	}

	if len(destElemErrorMap) > 0 {
		if len(destElemErrorMap) == len(keys) {
			// 如果全是同一个错误
			var same bool = true
			for _, e := range errs {
				if e != errs[0] {
					same = false
					break
				}
			}
			if same {
				return nil, errs[0]
			}
		}
		return errs, nil
	}
	return nil, nil
}

// Delete 删除key对应的哨兵。下次需要重新执行该key的逻辑。
func (sg *SentinelGroup) Delete(keys ...string) {
	for _, key := range keys {
		sg.sentinelMap.Delete(key)
	}
}
