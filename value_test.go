package horizon

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

const TEST_TTL = 1000
const TEST_CNT = 1000000
const TEST_SPEED_NORMAL int64 = 100
const TEST_SPEED_SLOW int64 = 1000

func newFuncWithSpeed(speed int64) func() (interface{}, error) {
	return func() (interface{}, error) {
		rd := rand.Int63n(100)
		//time.Sleep(time.Millisecond * time.Duration(speed+rd))
		if rd%100 == 0 {
			return nil, errors.New("randomError")
		}
		return UnixMilli(), nil
	}
}

func BenchmarkValueWithNormal(b *testing.B) {
	b.N = TEST_CNT
	b.ReportAllocs()
	b.ResetTimer()
	testValueWithSpeed(b, TEST_SPEED_NORMAL)
}

func BenchmarkValueWithSlow(b *testing.B) {
	b.N = TEST_CNT
	b.ReportAllocs()
	b.ResetTimer()
	testValueWithSpeed(b, TEST_SPEED_SLOW)
}

func testValueWithSpeed(b *testing.B, speed int64) {
	vl := NewValue()
	vl.New = newFuncWithSpeed(speed)
	vl.TTL = TEST_TTL
	ctx := context.Background()
	//start with warm
	vl.Get(ctx)
	var wg sync.WaitGroup
	s := time.Now()
	var max int64 = 0
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := UnixMilli()
			r, ok := vl.Get(ctx)
			if !ok {
				return
			}
			cost := UnixMilli() - start
			fmt.Printf("get speed:%d %d\n", i, cost)
			if cost > max {
				max = cost
			}
			if start-r.(int64) > TEST_TTL+speed*2 {
				fmt.Printf("data too old:%d %d %d\n", i, start, r.(int64))
				b.Fail()
			}
			if cost > speed {
				fmt.Printf("get too slow:%d %d\n", i, cost)
				b.Fail()
			}
		}(i)
		time.Sleep(time.Microsecond)
	}
	wg.Wait()
	fmt.Printf("average:%v, max:%v\n", time.Now().Sub(s)/time.Duration(b.N), time.Millisecond*time.Duration(max))
}
