package benchmark

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"fmt"
	cache "github.com/patrickmn/go-cache"
	"github.com/qingyu31/horizon"
	"sync/atomic"
)

const N = 1000000
const TTL = 2000

func BenchmarkHorizon(b *testing.B) {
	var refreshCount,getCount int64
	var maxCost time.Duration
	f := horizon.NewFactory("benchmark", TTL)
	f.New = func(i interface{}) (interface{}, error) {
		atomic.AddInt64(&refreshCount, 1)
		return testNew(), nil
	}
	b.N = N
	b.ReportAllocs()
	b.ResetTimer()
	f.Get(context.Background(), "test")
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		func() {
			defer wg.Done()
			start:=time.Now()
			atomic.AddInt64(&getCount, 1)
			f.Get(context.Background(), "test")
			cost := time.Now().Sub(start)
			if cost > maxCost {
				maxCost = cost
			}
		}()
	}
	wg.Wait()
	fmt.Printf("total get count:%d, total refresh count:%d, max cost:%v\n",getCount,refreshCount,maxCost)
}

func BenchmarkSyncMap(b *testing.B) {
	var sm sync.Map
	b.N = N
	b.ReportAllocs()
	b.ResetTimer()
	var wg sync.WaitGroup
	var refreshCount, getCount int64
	var maxCost time.Duration
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			atomic.AddInt64(&getCount, 1)
			v, _ := sm.Load("test")
			if v == nil || v.(*cache.Item).Expired() {
				atomic.AddInt64(&refreshCount, 1)
				sm.Store("test", testNewItem())
			}
			cost := time.Now().Sub(start)
			if cost > maxCost {
				maxCost = cost
			}
		}()
	}
	wg.Wait()
	fmt.Printf("total get count:%d, total refresh count:%d, max cost:%v\n",getCount,refreshCount,maxCost)
}

func testNewItem() *cache.Item {
	return &cache.Item{Object: testNew(), Expiration: time.Now().Add(time.Millisecond * TTL).UnixNano()}
}

func testNew() interface{} {
	time.Sleep(time.Millisecond * 200)
	return rand.Int63()
}

func BenchmarkGoCache(b *testing.B) {
	pc := cache.New(time.Millisecond*TTL, time.Second)
	var refreshCount, getCount int64
	var maxCost time.Duration
	b.N = N
	b.ReportAllocs()
	b.ResetTimer()
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&getCount,1)
			start := time.Now()
			v, exp := pc.Get("test")
			if v == nil || exp {
				atomic.AddInt64(&refreshCount,1)
				pc.Set("test", testNew(), time.Millisecond*TTL)
			}
			cost := time.Now().Sub(start)
			if cost > maxCost {
				maxCost = cost
			}
		}()
	}
	wg.Wait()
	fmt.Printf("total get count:%d, total refresh count:%d, max cost:%v\n",getCount,refreshCount,maxCost)
}
