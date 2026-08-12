package metrics

import "sync/atomic"

var dbPoolInUse int64

func SetDBPoolInUse(count int32) {
	atomic.StoreInt64(&dbPoolInUse, int64(count))
}

func DBPoolInUse() int64 {
	return atomic.LoadInt64(&dbPoolInUse)
}
