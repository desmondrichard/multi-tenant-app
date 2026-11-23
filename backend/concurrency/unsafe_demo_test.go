//go:build !race

package concurrency

import (
	"sync"
	"testing"
)

// DEMO ONLY (excluded when you run with -race, because it contains a deliberate data race).
// It shows what happens WITHOUT a mutex: updates are lost, so the total is usually below 100,000.
// The result changes on every run, so this test only logs and never fails.
func TestDemo_CounterWithoutMutex(t *testing.T) {
	counter := 0
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				counter++ // UNSAFE read-modify-write
			}
		}()
	}
	wg.Wait()
	t.Logf("without mutex: final=%d expected=100000 lost updates=%d", counter, 100000-counter)
	report(t,
		"Show why a mutex is needed: this unprotected counter loses updates.",
		"Never share a variable between goroutines without a mutex, atomic or channel. Prove it with: go test -race")
}