package concurrency

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// A job simulates I/O work (like a DB or API call): it waits 1 ms.
const jobDuration = time.Millisecond

type poolResult struct {
	processed int64
	perWorker []int64
	seen      []int32 // how many times each job id was processed
	elapsed   time.Duration
}

// runPool processes `jobsCount` jobs with `workers` workers.
func runPool(workers, jobsCount int, work func()) poolResult {
	jobs := make(chan int, workers*2) // small buffer
	res := poolResult{
		perWorker: make([]int64, workers),
		seen:      make([]int32, jobsCount),
	}

	var wg sync.WaitGroup
	start := time.Now()

	// start the fixed number of workers
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(w int) {
			defer wg.Done()
			for id := range jobs { // each worker takes jobs until the channel is closed
				work()
				atomic.AddInt32(&res.seen[id], 1)
				atomic.AddInt64(&res.perWorker[w], 1)
				atomic.AddInt64(&res.processed, 1)
			}
		}(w)
	}

	// send the jobs
	for i := 0; i < jobsCount; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	res.elapsed = time.Since(start)
	return res
}

// runSequential processes the same jobs one by one in a single goroutine.
func runSequential(jobsCount int, work func()) (int, time.Duration) {
	start := time.Now()
	done := 0
	for i := 0; i < jobsCount; i++ {
		work()
		done++
	}
	return done, time.Since(start)
}

// TEST 5: WORKER POOL
// 5 workers, 1,000 jobs. Verify every job is processed exactly once and
// compare the time with processing the jobs one after another.
func TestWorkerPool_5Workers_1000Jobs(t *testing.T) {
	const workers, jobsCount = 5, 1000
	work := func() { time.Sleep(jobDuration) }

	pool := runPool(workers, jobsCount, work)
	seqDone, seqTime := runSequential(jobsCount, work)

	speedup := float64(seqTime) / float64(pool.elapsed)
	t.Logf("worker pool : %d jobs, %d workers -> %v", pool.processed, workers, pool.elapsed)
	t.Logf("sequential  : %d jobs            -> %v", seqDone, seqTime)
	t.Logf("speed-up    : %.2fx faster (ideal = %dx)", speedup, workers)
	t.Logf("jobs per worker: %v", pool.perWorker)

	// 1) all jobs processed
	if pool.processed != jobsCount {
		t.Fatalf("processed %d jobs, want %d", pool.processed, jobsCount)
	}
	// 2) each job exactly once (no loss, no duplicates)
	for id, n := range pool.seen {
		if n != 1 {
			t.Fatalf("job %d was processed %d times, want 1", id, n)
		}
	}
	// 3) the work was shared between the workers
	var total int64
	for w, n := range pool.perWorker {
		if n == 0 {
			t.Errorf("worker %d did no work", w)
		}
		total += n
	}
	if total != jobsCount {
		t.Fatalf("workers handled %d jobs in total, want %d", total, jobsCount)
	}
	// 4) the pool is clearly faster than sequential (ideal 5x; we only require 2x so slow CI machines pass)
	if speedup < 2 {
		t.Errorf("pool was only %.2fx faster than sequential, expected at least 2x", speedup)
	}

	report(t,
		"Show that a FIXED number of workers can process a large number of jobs safely (all done, none twice) and much faster than one-by-one, "+
			"without starting 1,000 goroutines.",
		fmt.Sprintf("I/O-bound jobs (DB, Redis, HTTP): 5-20 workers, or the size of the downstream connection pool. "+
			"CPU-bound jobs: workers = number of CPU cores (this machine: %d). "+
			"Job queue (buffer): 2x the worker count up to ~100. Expect close to a %dx speed-up for I/O-bound work.",
			runtime.NumCPU(), workers))
}

// How the speed changes with the number of workers.
func TestWorkerPool_Scaling(t *testing.T) {
	const jobsCount = 1000
	work := func() { time.Sleep(jobDuration) }

	var previous time.Duration
	for _, workers := range []int{1, 2, 5, 10, 50} {
		res := runPool(workers, jobsCount, work)
		t.Logf("workers=%-3d processed=%-5d time=%v", workers, res.processed, res.elapsed)
		if res.processed != jobsCount {
			t.Fatalf("workers=%d processed %d, want %d", workers, res.processed, jobsCount)
		}
		if previous != 0 && res.elapsed > previous {
			t.Logf("   (note: %d workers were not faster than the previous setting; timing noise or limit reached)", workers)
		}
		previous = res.elapsed
	}
	report(t,
		"Find the point where adding more workers stops helping.",
		"Start with 5-10 workers and load-test (k6) to tune. More workers than the database/API can handle only causes queueing and errors.")
}

// Workers must stop cleanly: no goroutine may be left behind.
func TestWorkerPool_NoLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	runPool(5, 200, func() {})
	if after := waitForGoroutines(before, time.Second); after > before {
		t.Fatalf("worker goroutines leaked: before=%d after=%d", before, after)
	}
	report(t,
		"Prove that closing the jobs channel + WaitGroup stops every worker.",
		"Every worker needs an exit path: close(jobs) or context cancellation.")
}