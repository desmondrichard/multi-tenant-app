package concurrency

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TEST 2: WAITGROUP
// Launch 100 goroutines, use wg.Add(100) and defer wg.Done(),
// and verify that the "main" function waits until all of them finish.
func TestWaitGroup_100(t *testing.T) {
	const total = 100
	var finished int64
	var wg sync.WaitGroup

	wg.Add(total) // the counter is set BEFORE any goroutine starts

	start := time.Now()
	for i := 0; i < total; i++ {
		go func(id int) {
			defer wg.Done()                                       // always runs, even if the work panics or returns early
			time.Sleep(time.Duration(10+id%5) * time.Millisecond) // 10..14 ms of "work"
			atomic.AddInt64(&finished, 1)
		}(i)
	}

	// Right now the goroutines are (almost certainly) still working
	t.Logf("before Wait: finished=%d of %d", atomic.LoadInt64(&finished), total)

	wg.Wait() // main blocks here
	elapsed := time.Since(start)

	t.Logf("after  Wait: finished=%d of %d (waited %v)", atomic.LoadInt64(&finished), total, elapsed)

	// 1) Wait returned only after ALL goroutines called Done
	if got := atomic.LoadInt64(&finished); got != total {
		t.Fatalf("Wait returned too early: finished=%d, want %d", got, total)
	}
	// 2) it really blocked: the slowest goroutine needs 14 ms
	if elapsed < 14*time.Millisecond {
		t.Errorf("main did not wait long enough: %v", elapsed)
	}
	// 3) it ran concurrently: sequentially this would be ~1.2 s
	if elapsed > 500*time.Millisecond {
		t.Errorf("goroutines did not run concurrently: %v", elapsed)
	}

	report(t,
		"Make the caller wait until every goroutine is finished, so results are complete before they are used and no goroutine is abandoned.",
		"Call wg.Add(n) BEFORE starting the goroutines (never inside them), call defer wg.Done() as the first line of each goroutine, "+
			"never copy a WaitGroup (pass a pointer), and pair every Add with exactly one Done. For 100 goroutines the overhead is negligible.")
}

// Done must still run when a goroutine panics (that is why we use defer).
func TestWaitGroup_DeferDoneSurvivesPanic(t *testing.T) {
	const total = 100
	var recovered int64
	var wg sync.WaitGroup
	wg.Add(total)

	for i := 0; i < total; i++ {
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt64(&recovered, 1)
				}
			}()
			if id%10 == 0 {
				panic("boom") // 10 goroutines panic
			}
		}(i)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait never returned: a Done() was skipped (deadlock)")
	}
	if recovered != 10 {
		t.Fatalf("expected 10 recovered panics, got %d", recovered)
	}
	report(t,
		"Prove that defer wg.Done() prevents deadlocks even when a goroutine fails.",
		"Always recover() inside long-lived goroutines (HTTP handlers, hub loops) so one panic cannot crash the whole server.")
}