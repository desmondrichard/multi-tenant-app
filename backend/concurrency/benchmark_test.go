package concurrency

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// Run:  go test -bench . -benchmem ./concurrency/
// Purpose: measure execution time (ns/op) and memory (B/op, allocs/op).

// 1,000 goroutines + WaitGroup
func BenchmarkGoroutines_1000(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(1000)
		for g := 0; g < 1000; g++ {
			go func() {
				defer wg.Done()
				busy(10)
			}()
		}
		wg.Wait()
	}
}

// Mutex vs atomic counter under contention
func BenchmarkCounter_Mutex(b *testing.B) {
	var c SafeCounter
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkCounter_Atomic(b *testing.B) {
	var n int64
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic.AddInt64(&n, 1)
		}
	})
}

// 1,000 CPU-bound jobs: sequential vs worker pools of different sizes
func BenchmarkJobs_Sequential(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			busy(2000)
		}
	}
}

func BenchmarkJobs_WorkerPool(b *testing.B) {
	for _, workers := range []int{1, 2, 5, 10} {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				runPool(workers, 1000, func() { busy(2000) })
			}
		})
	}
}

// Channel throughput
func BenchmarkChannel_Buffered100(b *testing.B) {
	b.ReportAllocs()
	ch := make(chan int, 100)
	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()
	for i := 0; i < b.N; i++ {
		ch <- i
	}
	close(ch)
	<-done
}