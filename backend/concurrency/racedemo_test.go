package concurrency

import (
	"os"
	"sync"
	"testing"
)

// Run this ON PURPOSE to watch the Race Detector catch a bug.
// It is skipped by default so normal `go test -race` runs stay green.
//
//	PowerShell:  $env:DEMO="1"; go test -race -run TestRaceDemo ./concurrency/
//
// Expected result: "WARNING: DATA RACE" and the test FAILS. That proves the detector works.
func TestRaceDemo(t *testing.T) {
	if os.Getenv("DEMO") != "1" {
		t.Skip("race demo skipped: set DEMO=1 to run it (see the comment above)")
	}

	counter := 0
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter++ // two goroutines write at the same time
		}()
	}
	wg.Wait()
	_ = counter
}
