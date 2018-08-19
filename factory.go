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
	//Timeliness means if data is sensitive to timeliness.
	//it should be set before use.
	Timeliness bool
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
func (f *factory) Get(ctx context.Context, key interface{}) interface{} {
	if f.New==nil {
		return nil
	}
	x, _ := f.kv.LoadOrStore(key, f.newValue(key))
	return x
}

func (f *factory) newValue(key interface{}) *value {
	vl := NewValue()
	vl.TTL = f.ttl
	vl.Timeliness = f.Timeliness
	vl.New = func() (interface{}, error) { return f.New(key) }
	return vl
}

//Recycle will recycle value which is idle for a period.
func (f *factory) Recycle() {
	f.kv.Range(func(k,v interface{})bool{
		vl := v.(*value)
		if  vl.Idle(){
			f.kv.Delete(k)
			vl.Close()
		}
		return true
	})
}
