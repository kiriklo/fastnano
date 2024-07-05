package fastnano

import (
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkUnixNano(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = time.Now().UnixNano()
	}
}

func BenchmarkFastnano(b *testing.B) {
	t := NewFastNano()
	for i := 0; i < b.N; i++ {
		_ = t.UnixNanoTimestamp()
	}
}

// This idea was taken from https://github.com/VictoriaMetrics/VictoriaMetrics/blob/master/lib/fasttime/fasttime_timing_test.go
// Sink should prevent from code elimination by optimizing compiler
var Sink atomic.Int64

func BenchmarkUnixNano_Parallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var ts int64
		for pb.Next() {
			ts += time.Now().UnixNano()
		}
		Sink.Store(ts)
	})
}

func BenchmarkFastnano_Parallel(b *testing.B) {
	var nt = NewFastNano()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var ts int64
		for pb.Next() {
			ts += nt.UnixNanoTimestamp()
		}
		Sink.Store(ts)
	})
}

func oldFastNano() *FastNano {
	t := time.Now().Add(time.Hour * 24 * 365 * (-1))
	return &FastNano{time: t, nano: t.UnixNano()}
}

func BenchmarkFastnanoOld_Parallel(b *testing.B) {
	var nt = oldFastNano()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var ts int64
		for pb.Next() {
			ts += nt.UnixNanoTimestamp()
		}
		Sink.Store(ts)
	})
}
