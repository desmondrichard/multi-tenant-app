package concurrency

import (
	"math"
	"runtime"
	"testing"
	"time"
)

// report prints the PURPOSE and the RECOMMENDED value at the end of a test.
// Run with -v to see it:  go test -v ./concurrency/
func report(t *testing.T, purpose, recommended string) {
	t.Helper()
	t.Logf("\n      PURPOSE     : %s\n      RECOMMENDED : %s", purpose, recommended)
}

// busy burns a little CPU (used by the CPU-bound benchmarks / pprof).
func busy(n int) float64 {
	x := 0.0
	for i := 1; i <= n; i++ {
		x += math.Sqrt(float64(i))
	}
	return x
}

// waitForGoroutines waits until the number of goroutines drops back to `want`
// (used to prove that nothing leaked). Returns the final count.
func waitForGoroutines(want int, timeout time.Duration) int {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if n := runtime.NumGoroutine(); n <= want {
			return n
		}
		time.Sleep(5 * time.Millisecond)
	}
	return runtime.NumGoroutine()
}