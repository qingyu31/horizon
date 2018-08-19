package horizon

import (
	"sync"
	"time"
)

var gcenter *center
var glock sync.Mutex

func getCenter() *center {
	if gcenter == nil {
		glock.Lock()
		defer glock.Unlock()
		if gcenter == nil {
			gcenter = new(center)
			go gcenter.daemon()
		}
	}
	return gcenter
}

type center struct {
	mu       sync.Mutex
	factorys sync.Map
	values   sync.Map
}

func (c *center) AddFactory(k string, f *factory) {
	c.factorys.Store(k, f)
}

func (c *center) AddValue(v *value) {
	c.values.Store(v, v)
}

func (c *center) RemoveValue(v *value) {
	c.values.Delete(v)
}

func (c *center) daemon() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			/*c.factorys.Range(func(k, v interface{}) bool {
				go v.(*factory).Recycle()
				return true
			})*/
			c.values.Range(func(k, v interface{}) bool {
				vl, ok := v.(*value)
				if !ok {
					c.values.Delete(k)
					return true
				}
				if !vl.Idle() {
					go vl.tryRefresh()
				}
				return true
			})
		}
	}
}
