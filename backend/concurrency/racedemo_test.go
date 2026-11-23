//go:build racedemo

package concurrency

import (
	"sync"
	"testing"
)

// Run this ON PURPOSE to see the Race Detector catch a bug:
//
//	go test -race -tags racedemo -run TestRaceDemo ./concurrency/
//
// Expected result: "WARNING: DATA RACE" and the test FAILS. That proves the detector works.
func TestRaceDemo(t *testing.T) {
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
