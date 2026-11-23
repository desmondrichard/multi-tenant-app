package concurrency

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TEST 1: GOROUTINES
// Launch 1,000 goroutines, count how many finish successfully, measure the time.
func TestGoroutines_1000(t *testing.T) {
	const total = 1000

	before := runtime.NumGoroutine()
	var success int64
	var wg sync.WaitGroup // only used so the test can wait for the goroutines

	start := time.Now()
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = busy(100)                // a tiny piece of work
			atomic.AddInt64(&success, 1) // count a successful run
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("launched=%d  succeeded=%d  time=%v  (%.1f µs per goroutine)",
		total, success, elapsed, float64(elapsed.Microseconds())/total)

	// 1) every goroutine must have executed
	if got := atomic.LoadInt64(&success); got != total {
		t.Fatalf("expected %d successful goroutines, got %d", total, got)
	}

	// 2) generous time guard (it normally takes a few milliseconds)
	if elapsed > 2*time.Second {
		t.Errorf("1,000 goroutines took too long: %v", elapsed)
	}

	// 3) no goroutine leak: the count must return to the starting value
	if after := waitForGoroutines(before, time.Second); after > before {
		t.Errorf("goroutine leak: before=%d after=%d", before, after)
	}

	report(t,
		"Prove that goroutines are cheap and that every launched goroutine really runs (no lost work, no leak).",
		"Test with 1,000 (stress up to 10,000-100,000). Success must be exactly 100%. "+
			"A goroutine should take only microseconds to start (< 100 ms total for 1,000). "+
			"In the real API never start unbounded goroutines per request: cap them with a worker pool.")
}

// Same test but as a table: shows the behaviour as the number grows.
func TestGoroutines_Scaling(t *testing.T) {
	for _, n := range []int{10, 100, 1000, 10000} {
		var success int64
		var wg sync.WaitGroup
		start := time.Now()
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				atomic.AddInt64(&success, 1)
			}()
		}
		wg.Wait()
		t.Logf("n=%-6d succeeded=%-6d time=%v", n, success, time.Since(start))
		if int(success) != n {
			t.Fatalf("n=%d: only %d goroutines ran", n, success)
		}
	}
	report(t,
		"Show that time grows roughly linearly with the number of goroutines and none are lost.",
		"Keep concurrency proportional to what the downstream system can take (e.g. MongoDB connections), not to how many goroutines Go can start.")
}
