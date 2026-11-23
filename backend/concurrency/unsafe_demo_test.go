package concurrency

import (
	"os"
	"sync"
	"testing"
)

// DEMO ONLY: a counter WITHOUT a mutex. It contains a deliberate data race, so it is
// skipped unless you ask for it with the DEMO=1 environment variable.
//
//	PowerShell:  $env:DEMO="1"; go test -run TestDemo_CounterWithoutMutex -v ./concurrency/
//
// Do NOT combine it with -race (the race detector would, correctly, fail it).
// The result changes on every run, so this test only logs and never fails.
func TestDemo_CounterWithoutMutex(t *testing.T) {
	if os.Getenv("DEMO") != "1" {
		t.Skip("demo skipped: set DEMO=1 to run it (see the comment above)")
	}

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
		"Show why a mutex is needed: this unprotected counter can lose updates.",
		"Never share a variable between goroutines without a mutex, atomic or channel. Prove it with: go test -race")
}
