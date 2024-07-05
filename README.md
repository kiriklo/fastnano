# fastnano
## Table of Contents

- [Intro](#intro)
- [Important note](#important-note)
- [10 lines of code](#10-lines-of-code) 
- [Is it faster?](#is-it-faster)
- [How is this possible?](#how-is-this-possible)
- [Profiles](#profiles)
- [Concurent safe?](#concurent-safe)
- [Why not update time in struct?](#why-not-update-time-in-struct)
- [Hack](#hack)

## Intro
Recently, I was exploring the [VictoriaMetrics][1] library and came across a really interesting one - [fasttime][2]. This package allows you to get the current Unix timestamp in seconds, and it's faster than time.Now().Unix(). So what it does is, instead of calling every time time.Now().Unix() creates a separate goroutine and time.NewTicker(time.Second) and stores timestamp in atomic variable. And each time you call UnixTimestamp() it's just a simple atomic Load().
This approach is interesting, but it only allows us to get the current timestamp in seconds. Let's say we want to get the timestamp in nanoseconds. So let's try to figure out how we can do it faster than just calling every time time.Now().UnixNano().

## Important note
All of the following only applies to Linux amd64 architecture. Because this is low level calls, this trick may not work on some other platforms. So make benchmarks before adding this to your code.

## 10 lines of code
```go
type FastNano struct {
	time time.Time
	nano int64
	_    int64
}

func NewFastNano() *FastNano {
	t := time.Now()
	return &FastNano{time: t, nano: t.UnixNano()}
}

func (t *FastNano) UnixNanoTimestamp() int64 {
	return time.Since(t.time).Nanoseconds() + t.nano
}
```
Yes, these 10 lines of code are actually the whole fastnano package. So first, let's see what this package consists of. We have a structure called FastNano, which has two fields: time.Time for time and int64 for unix timestamp (actually 3 fields, but I will explain this later).
Then we have function NewFastNano(), which returns FastNano with the current time. And finally, function UnixNanoTimestamp() which returns the actual timestamp in nanoseconds.

## Is it faster?
What is the point of using time.Since() instead of just calling time.Now()? We still need to get the current timestamp in nanoseconds to calculate the difference, isn't it the same? It should be, but it's not.
```
GOMAXPROCS=4 go test -bench=. -benchmem -benchtime=10s
goos: linux
goarch: amd64
pkg: fastnano
BenchmarkUnixNano-4             265343518               45.20 ns/op            0 B/op           0 allocs/op
BenchmarkFastnano-4             469715604               25.52 ns/op            0 B/op           0 allocs/op
BenchmarkUnixNano_Parallel-4    1000000000              11.50 ns/op            0 B/op           0 allocs/op
BenchmarkFastnano_Parallel-4    1000000000               6.469 ns/op           0 B/op           0 allocs/op
```

## How is this possible?
To understand why time.Since() is faster, let's dive into the [time][3] package.
When we call time.Now() we actually call now() function:
```go
// Now returns the current local time.
func Now() Time {
	sec, nsec, mono := now()
	mono -= startNano
	sec += unixToInternal - minWall
	if uint64(sec)>>33 != 0 {
		// Seconds field overflowed the 33 bits available when
		// storing a monotonic time. This will be true after
		// March 16, 2157.
		return Time{uint64(nsec), sec + minWall, Local}
	}
	return Time{hasMonotonic | uint64(sec)<<nsecShift | uint64(nsec), mono, Local}
}
```
Function now():
```go
// Provided by package runtime.
func now() (sec int64, nsec int32, mono int64)
```
Function now() in [runtime][4]:
```go
//go:linkname time_now time.now
func time_now() (sec int64, nsec int32, mono int64)
```
So we actually call an assembly function [time_now][5].

Now let's see what's happening when we call time.Since() function:
```go
// Since returns the time elapsed since t.
// It is shorthand for time.Now().Sub(t).
func Since(t Time) Duration {
	var now Time
	if t.wall&hasMonotonic != 0 {
		// Common case optimization: if t has monotonic time, then Sub will use only it.
		now = Time{hasMonotonic, runtimeNano() - startNano, nil}
	} else {
		now = Now()
	}
	return now.Sub(t)
}
```
Because we already called time.Now() we have t.wall and optimization take place. So instead of calling time.Now() and then calculate the difference, we call runtimeNano().
```go
// runtimeNano returns the current value of the runtime clock in nanoseconds.
//
//go:linkname runtimeNano runtime.nanotime
func runtimeNano() int64
```
Here is [nanotime][6] function in runtime package:
```go
//go:nosplit
func nanotime() int64 {
	return nanotime1()
}
```
Function nanotime1 is an assebmbly function located in [sys_linux_amd64][7] file (for linux amd64 architecture).

We will not compare these two assembly functions or try to write the new one because we want our code to be simple and safe. So let's just consider that because of returning only one integer value instead of three, it is more optimized and thus faster.

## Profiles
Let's take a look at CPU profiles to prove that we are right.
``` 
Profile collected with the following options: 
export ver=fastnano && go test -run . -bench="BenchmarkFastnano_Parallel" -benchtime 10s -count 5 -cpu 4 -benchmem -memprofile=${ver}.mem.pprof -cpuprofile=${ver}.cpu.pprof

/opt/go-1.21/src/runtime/time_nofake.go

  Total:     122.08s    122.08s (flat, cum) 84.84%
     13            .          .           // 
     14            .          .           // Zero means not to use faketime. 
     15            .          .           var faketime int64 
     16            .          .            
     17            .          .           //go:nosplit 
     18        930ms      930ms           func nanotime() int64 { 
     19      121.15s    121.15s           	return nanotime1() 
     20            .          .           } 
     21            .          .            
     22            .          .           var overrideWrite func(fd uintptr, p unsafe.Pointer, n int32) int32 
     23            .          .            
     24            .          .           // write must be nosplit on Windows (see write1) 
```
``` 
Profile collected with the following options: 
export ver=unixnano && go test -run . -bench="BenchmarkUnixNano_Parallel" -benchtime 10s -count 5 -cpu 4 -benchmem -memprofile=${ver}.mem.pprof -cpuprofile=${ver}.cpu.pprof

/opt/go-1.21/src/time/time.go

  Total:     244.32s    244.33s (flat, cum) 96.38%
   1105            .          .           // we avoid ever reporting a monotonic time of 0. 
   1106            .          .           // (Callers may want to use 0 as "time not set".) 
   1107            .          .           var startNano int64 = runtimeNano() - 1 
   1108            .          .            
   1109            .          .           // Now returns the current local time. 
   1110        1.42s      1.43s           func Now() Time { 
   1111      239.35s    239.35s           	sec, nsec, mono := now() 
   1112            .          .           	mono -= startNano 
   1113        110ms      110ms           	sec += unixToInternal - minWall 
   1114        1.33s      1.33s           	if uint64(sec)>>33 != 0 { 
   1115            .          .           		// Seconds field overflowed the 33 bits available when 
   1116            .          .           		// storing a monotonic time. This will be true after 
   1117            .          .           		// March 16, 2157. 
   1118            .          .           		return Time{uint64(nsec), sec + minWall, Local} 
   1119            .          .           	} 
   1120        2.11s      2.11s           	return Time{hasMonotonic | uint64(sec)<<nsecShift | uint64(nsec), mono, Local} 
   1121            .          .           } 

```

From the above, we can clearly see that we spend almost all our time calling assembly functions, and nanotime1() is almost 2x faster than now().

## Concurent safe?
Yes, we modify the struct only once when calling NewFastNano(). That's why we don't need mutex or store data in atomic variables.

## Why not update time in struct?
There is no point in doing it. We spend almost all time on calling assembly functions, not on math operations. To prove this, let's assume that our application runs for one year straight without stopping. To emulate this, let's chanage our init funcion a little bit:
```go
func oldFastNano() *FastNano {
	t := time.Now().Add(time.Hour * 24 * 365 * (-1))
	return &FastNano{time: t, nano: t.UnixNano()}
}
```
So now we are calculating how many nanoseconds have passed since now and now-1 year. As you can see, the speed is the same:
```
BenchmarkFastnanoOld_Parallel-8   	278496669	         4.506 ns/op	       0 B/op	       0 allocs/op
```
## Hack
Hack is just a simple memory padding we want to add to avoid [false sharing][8] because of [cache cohesion][9].

[//]: <> (Links in the order of appearance)
[1]: https://github.com/VictoriaMetrics/VictoriaMetrics                             "VictoriaMetrics"
[2]: https://github.com/VictoriaMetrics/VictoriaMetrics/tree/master/lib/fasttime    "fasttime"
[3]: https://go.dev/src/time/                                                       "time"
[4]: https://go.dev/src/runtime/timeasm.go                                          "runtime"
[5]: https://go.dev/src/runtime/time_linux_amd64.s                                  "time_now"
[6]: https://go.dev/src/runtime/time_nofake.go                                      "nanotime"
[7]: https://go.dev/src/runtime/sys_linux_amd64.s                                   "sys_linux_amd64"
[8]: https://en.wikipedia.org/wiki/False_sharing                                    "sharing"
[9]: https://en.wikipedia.org/wiki/Cache_coherence                                  "cohesion"