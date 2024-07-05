package fastnano

import (
	"testing"
	"time"
)

func TestFastnano(t *testing.T) {
	un := NewFastNano()
	callsCount := 10000
	testFastnanoMultiple(un, callsCount, t)
}

func TestFastnano_Parallel(t *testing.T) {
	un := NewFastNano()
	callsCount := 10000
	const gorotines = 10
	for i := 0; i < gorotines; i++ {
		go func() {
			testFastnanoMultiple(un, callsCount, t)
		}()
	}
}

func testFastnanoMultiple(un *FastNano, c int, t *testing.T) {
	for j := 0; j < c; j++ {
		expected := time.Now().UnixNano()
		nano := un.UnixNanoTimestamp()
		if nano-expected > 25*1000*1000 { // not sure about this value
			t.Fatalf("diffrence between timestamps is greater than 25ms; time.Now().UnixNano() = %d; UnixNanoTimestamp() = %d", nano, expected)
		}
	}
}
