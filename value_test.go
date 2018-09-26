package horizon

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

const TEST_TTL = 1000
const TEST_CNT = 1000000
const TEST_SPEED_NORMAL int64 = 100
const TEST_SPEED_SLOW int64 = 1000

func TestValue_MarshalBinary(t *testing.T) {
	ctx := context.Background()
	vl := NewValue()
	vl.New = newFuncWithSpeed(TEST_SPEED_NORMAL)
	b, e := vl.MarshalBinary()
	fmt.Println(string(b), e)
	vl.Get(ctx)
	b, e = vl.MarshalBinary()
	fmt.Println(string(b), e)
}

func TestValue_UnmarshalBinary(t *testing.T) {
	ctx := context.Background()
	vl := NewValue()
	data := []byte(`{"data":1537971059611,"expire_at":`+strconv.FormatInt(timestamp(),10)+`}`)
	vl.UnmarshalBinary(data)
	fmt.Println(vl.Get(ctx))
}

func newFuncWithSpeed(speed int64) func(context.Context) (interface{}, error) {
	return func(ctx context.Context) (interface{}, error) {
		rd := rand.Int63n(100)
		//time.Sleep(time.Millisecond * time.Duration(speed+rd))
		if rd%100 == 0 {
			return nil, errors.New("randomError")
		}
		return timestamp(), nil
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
	vl.RefreshInterval = TEST_TTL / 10
	ctx := context.Background()
	//start with warm
	vl.Get(ctx)
	b.ResetTimer()
	var wg sync.WaitGroup
	s := time.Now()
	var max int64 = 0
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := timestamp()
			r, ok := vl.Get(ctx)
			if !ok {
				return
			}
			cost := timestamp() - start
			if cost > max {
				max = cost
			}
			if start-r.(int64) > TEST_TTL+speed*2 {
				fmt.Printf("data too old:%d %d %d\n", i, start, r.(int64))
				b.Fail()
				return
			}
			if cost > speed {
				fmt.Printf("get too slow:%d %d\n", i, cost)
				b.Fail()
				return
			}
		}(i)
		time.Sleep(time.Nanosecond)
	}
	wg.Wait()
	fmt.Printf("average:%v, max:%v\n", time.Now().Sub(s)/time.Duration(b.N), time.Millisecond*time.Duration(max))
}
