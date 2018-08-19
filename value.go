package horizon

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"
)

const MIN_REFRESH_INTVAL int64 = 10

var valuePool = sync.Pool{New: func() interface{} { return new(value) }}

//value stores data and refreshes itself if needed.
type value struct {
	//New produce new data for value. only set before use.
	New func() (interface{}, error)
	//TTL is time to live measured by millisecond. only set before use.
	TTL int64
	//Timeliness means if data is sensitive to timeliness.
	Timeliness bool
	//em is a Pointer to elem. use unsafe.Pointer to keep atomic.
	//properties in elem is read only.
	emp unsafe.Pointer
	//lastVisited is the time last visited measured by millisecond.
	lastVisited int64
	//cntVisited is the counter of visit.
	cntVisit int64
	//lock is a non-block lock to confirm only one goroutine refreshing.
	lock nonblock
	//lastRefresh record last refresh time to forbidden refresh too often.
	//it is protected by lock. its measurement is millisecond.
	lastRefresh int64
	//mu is a block lock to confirm only one goroutine refreshing
	mu sync.RWMutex
}

func NewValue() *value {
	v := valuePool.Get().(*value)
	v.Init()
	return v
}

//value should be init before use.
func (v *value) Init() {
	v.TTL = 0
	v.New = nil
	v.mu.Lock()
	defer v.mu.Unlock()
	v.setElem(nil)
	v.lastRefresh = 0
	getCenter().AddValue(v)
}

//value should be closed after use.
func (v *value) Close() {
	getCenter().RemoveValue(v)
	valuePool.Put(v)
}

//Idle return if a value not visited for a period.
func (v *value) Idle() bool {
	now := UnixMilli()
	if now < v.Expireat() {
		return false
	}
	return now-atomic.LoadInt64(&v.lastVisited) > v.TTL*4
}

//Get return inner data no matter expired and try to refresh if expired.
func (v *value) Get(ctx context.Context) (value interface{}, exists bool) {
	now := UnixMilli()
	defer atomic.AddInt64(&v.cntVisit, 1)
	defer atomic.StoreInt64(&v.lastVisited, now)
	em := v.getElem()
	if nil == em && nil == v.New {
		return nil, false
	}
	if (v.Idle() || nil == em) && nil != v.New {
		v.doRefresh()
		em = v.getElem()
		if em ==nil{
			return nil,false
		}
		return em.Get(),true
	}
	expireat := v.Expireat()
	if now > expireat && nil != v.New {
		if v.Timeliness {
			v.doRefresh()
		} else if now-expireat < v.TTL*4 && now-expireat < v.TTL+60000 {
			go v.tryRefresh()
		} else {
			v.tryRefresh()
		}
	}
	return em.Get(), true
}

func (v *value) Expireat() int64 {
	em := v.getElem()
	if em == nil {
		return 0
	}
	return em.Expireat()
}

//tryRefresh will try to fresh when others are not refreshing.
func (v *value) tryRefresh() {
	if v.lock.Pending() {
		return
	}
	if !v.lock.Lock() {
		return
	}
	defer v.lock.Unlock()
	v.doRefresh()
}

//doRefresh begins to refresh until others have done.
func (v *value) doRefresh() {
	v.mu.Lock()
	defer v.mu.Unlock()
	now := UnixMilli()
	if now < v.Expireat() {
		return
	}
	if now-v.lastRefresh < MIN_REFRESH_INTVAL {
		return
	}
	v.lastRefresh = now
	v.Refresh()
}

//Refresh will refresh immediately.
func (v *value) Refresh() error {
	if nil == v.New {
		return nil
	}
	nv, er := v.New()
	if er != nil {
		return er
	}
	em := elemPool.Get().(*elem)
	em.Init(nv)
	em.expireat = UnixMilli() + v.TTL
	v.setElem(em)
	return nil
}

//getElem get elem thread safely.
func (v *value) getElem() *elem {
	return (*elem)(atomic.LoadPointer(&v.emp))
}

//setElem set elem thread safely.
func (v *value) setElem(em *elem) {
	old := atomic.SwapPointer(&v.emp, unsafe.Pointer(em))
	if old != nil {
		(*elem)(old).Close()
	}
}
