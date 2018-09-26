package horizon

import (
	"encoding/json"
	"sync"
	"time"
)

var elemPool = sync.Pool{New: func() interface{} { return new(elem) }}

type elem struct {
	data         interface{}
	lastModified int64
	refreshAt    int64
	expireAt     int64
}

func (e *elem) Init(x interface{}) {
	e.data = x
	e.lastModified = time.Now().UnixNano() / int64(time.Millisecond)
}

func (e *elem) Close() {
	e.data = nil
	e.lastModified = 0
	elemPool.Put(e)
}

func (e *elem) Get() interface{} {
	return e.data
}

func (e *elem) LastModified() int64 {
	return e.lastModified
}

func (e *elem) ExpireAt() int64 {
	return e.expireAt
}

func (e *elem) Expired(stamp int64) bool {
	return stamp > e.expireAt
}

func (e *elem) NeedRefresh(stamp int64) bool {
	return stamp <= e.refreshAt
}

func (e *elem) MarshalBinary() ([]byte, error) {
	var exp elemExport
	exp.Data = e.data
	exp.ExpireAt = e.expireAt
	return json.Marshal(exp)
}

func (e *elem) UnmarshalBinary(b []byte) error {
	var exp elemExport
	err := json.Unmarshal(b, &exp)
	if nil != err {
		return err
	}
	e.data = exp.Data
	e.expireAt = exp.ExpireAt
	return nil
}

type elemExport struct {
	Data     interface{} `json:"data"`
	ExpireAt int64       `json:"expire_at"`
}
