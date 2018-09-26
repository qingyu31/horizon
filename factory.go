package horizon

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
)

//Factory produce,cache and refresh data.
type Factory struct {
	//KeyType declare type of key for serialization using.
	KeyType         reflect.Type
	kv              sync.Map
	TTL             int64
	RefreshInterval int64
	//New is function to produce data.
	//it should be set before use.
	New func(context.Context, interface{}) (interface{}, error)
	mu  sync.Mutex
}

//Get get data produced or cached.
func (f *Factory) Get(ctx context.Context, key interface{}) (interface{}, bool) {
	vl, ok := f.GetValue(ctx, key)
	if !ok {
		return nil, false
	}
	return vl.Get(ctx)
}

func (f *Factory) GetValue(ctx context.Context, key interface{}) (*value, bool) {
	x, ok := f.kv.Load(key)
	if ok && x != nil {
		return x.(*value), true
	}
	if f.New == nil {
		return nil, false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok = f.kv.Load(key)
	if ok && x != nil {
		return x.(*value), true
	}
	f.kv.Store(key, f.newValue(key))
	x, ok = f.kv.Load(key)
	if ok && x != nil {
		return x.(*value), true
	}
	return nil, false
}

func (f *Factory) newValue(key interface{}) *value {
	vl := NewValue()
	vl.TTL = f.TTL
	vl.RefreshInterval = f.RefreshInterval
	if vl.RefreshInterval <= 0 {
		vl.RefreshInterval = vl.TTL
	}
	vl.New = func(ctx context.Context) (interface{}, error) { return f.New(ctx, key) }
	return vl
}

//Recycle will recycle value which is idle for a period.
func (f *Factory) Recycle() {
	f.kv.Range(func(k, v interface{}) bool {
		vl := v.(*value)
		if vl.Expired() {
			f.kv.Delete(k)
			vl.Close()
		}
		return true
	})
}

func (f *Factory) MarshalBinary() ([]byte, error) {
	kvm := make(map[interface{}]*value)
	f.kv.Range(func(k, v interface{}) bool {
		vl := v.(*value)
		if vl == nil || vl.Expired() {
			return true
		}
		kvm[k] = vl
		return true
	})
	result := make(map[string]*json.RawMessage)
	for k, v := range kvm {
		kb, e := json.Marshal(k)
		if e != nil {
			continue
		}
		vb, e := v.MarshalBinary()
		if e != nil {
			continue
		}
		vrm := json.RawMessage(vb)
		result[string(kb)] = &vrm
	}
	return json.Marshal(result)
}

func (f *Factory) UnmarshalBinary(b []byte) error {
	switch f.KeyType.Kind() {
	case reflect.String:
	case reflect.Int:
	case reflect.Int64:
	case reflect.Struct:
	default:
		return errors.New("unsupport key type")
	}
	result := make(map[string]*json.RawMessage)
	for k, v := range result {
		if v == nil {
			continue
		}
		ki := reflect.New(f.KeyType).Interface()
		e := json.Unmarshal([]byte(k), ki)
		if e != nil {
			continue
		}
		vl := NewValue()
		e = vl.UnmarshalBinary([]byte(*v))
		if e != nil {
			continue
		}
		f.kv.Store(ki, vl)
	}
	return nil
}
