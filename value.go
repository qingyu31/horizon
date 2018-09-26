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
	New func(ctx context.Context) (interface{}, error)
	//TTL is time to live measured by millisecond. only set before use.
	TTL int64
	//Refresh interval is interval between doing refresh.
	RefreshInterval int64
	//em is a Pointer to elem. use unsafe.Pointer to keep atomic.
	//properties in elem is read only.
	emp unsafe.Pointer
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
}

//value should be closed after use.
func (v *value) Close() {
	valuePool.Put(v)
}

//Get return inner data living and try to refresh.
func (v *value) Get(ctx context.Context) (value interface{}, exists bool) {
	now := timestamp()
	em := v.getElem()
	if nil == v.New {
		if nil == em {
			return nil, false
		}
		return em.Get(), !em.Expired(now)
	}
	if nil == em || em.Expired(now) {
		v.doRefresh(ctx)
		em = v.getElem()
		if em == nil {
			return nil, false
		}
		return em.Get(), !em.Expired(now)
	}
	if em.NeedRefresh(now) || !v.lock.Pending() {
		go v.tryRefresh(ctx)
	}
	return em.Get(), !em.Expired(now)
}

func (v *value) ExpireAt() int64 {
	em := v.getElem()
	if em == nil {
		return 0
	}
	return em.ExpireAt()
}

func (v *value) Expired() bool {
	em := v.getElem()
	if em == nil {
		return true
	}
	return em.Expired(timestamp())
}

//tryRefresh will try to fresh when others are not refreshing.
func (v *value) tryRefresh(ctx context.Context) {
	if !v.lock.Lock() {
		return
	}
	defer v.lock.Unlock()
	v.doRefresh(ctx)
}

//doRefresh begins to refresh until others have done.
func (v *value) doRefresh(ctx context.Context) {
	v.mu.Lock()
	defer v.mu.Unlock()
	now := timestamp()
	if now-v.lastRefresh < MIN_REFRESH_INTVAL {
		return
	}
	em := v.getElem()
	if em != nil && !em.Expired(now) {
		return
	}
	v.lastRefresh = now
	v.Refresh(ctx)
}

//Refresh will refresh immediately.
func (v *value) Refresh(ctx context.Context) error {
	if nil == v.New {
		return nil
	}
	nv, er := v.New(ctx)
	if er != nil {
		return er
	}
	now := timestamp()
	em := elemPool.Get().(*elem)
	em.Init(nv)
	em.expireAt = now + v.TTL
	em.refreshAt = now + v.RefreshInterval
	if v.RefreshInterval <= 0 {
		em.refreshAt = em.expireAt
	}
	v.setElem(em)
	return nil
}

func (v *value) MarshalBinary() ([]byte, error) {
	em := v.getElem()
	if em ==nil {
		return nil,nil
	}
	return em.MarshalBinary()
}

func (v *value) UnmarshalBinary(b []byte) error {
	em := elemPool.Get().(*elem)
	err := em.UnmarshalBinary(b)
	if err != nil {
		return err
	}
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
