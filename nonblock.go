package horizon

import (
	"sync/atomic"
)

const LOCK_FREE = 0
const LOCK_LOCKED = 1

type nonblock struct {
	state int32
}

func (n *nonblock) Lock() bool {
	return atomic.CompareAndSwapInt32(&n.state, LOCK_FREE, LOCK_LOCKED)
}

func (n *nonblock) Unlock() {
	atomic.CompareAndSwapInt32(&n.state, LOCK_LOCKED, LOCK_FREE)
}

func (n *nonblock) Pending() bool {
	return LOCK_LOCKED == atomic.LoadInt32(&n.state)
}
