package horizon

import (
	"context"
	"sync"
)

//factory produce,cache and refresh data.
type factory struct {
	id  string
	kv  sync.Map
	ttl int64
	//New is function to produce data.
	//it should be set before use.
	New func(interface{}) (interface{}, error)
	mu  sync.Mutex
}

//NewFactory create a factory with default.
func NewFactory(id string, ttl int64) *factory {
	ft := new(factory)
	ft.id = id
	ft.ttl = ttl
	getCenter().AddFactory(id, ft)
	return ft
}

//Get get data produced or cached.
func (f *factory) Get(ctx context.Context, key interface{}) (interface{}, bool) {
	x, ok := f.kv.Load(key)
	if ok && x != nil {
		return x.(*value).Get(ctx)
	}
	if f.New == nil {
		return nil, false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok = f.kv.Load(key)
	if ok&& x!=nil {
		return x.(*value).Get(ctx)
	}
	f.kv.Store(key, f.newValue(key))
	x, ok = f.kv.Load(key)
	if ok && x!=nil {
		return x.(*value).Get(ctx)
	}
	return nil, false
}

func (f *factory) newValue(key interface{}) *value {
	println("newValue")
	vl := NewValue()
	vl.TTL = f.ttl
	vl.New = func() (interface{}, error) { return f.New(key) }
	return vl
}

//Recycle will recycle value which is idle for a period.
func (f *factory) Recycle() {
	f.kv.Range(func(k, v interface{}) bool {
		vl := v.(*value)
		if vl.Expired() {
			f.kv.Delete(k)
			vl.Close()
		}
		return true
	})
}
