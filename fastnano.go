package fastnano

import (
	"time"
)

// FastNano is a time.Time wrapper for nanoseconds.
type FastNano struct {
	time time.Time
	nano int64
	_    int64 // hack
}

// NewFastNano returns new FastNano.
func NewFastNano() *FastNano {
	t := time.Now()
	return &FastNano{time: t, nano: t.UnixNano()}
}

// UnixNanoTimeNano returns new unix timestamp in nanoseconds.
//
// It is faster than time.Now().UnixNano().
// It is safe calling this function from concurrent goroutines.
func (t *FastNano) UnixNanoTimestamp() int64 {
	return time.Since(t.time).Nanoseconds() + t.nano
}
