package concurrency

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// SafeCounter is a shared counter protected by a mutex.
type SafeCounter struct {
	mu sync.Mutex
	n  int
}

func (c *SafeCounter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}

func (c *SafeCounter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// TEST 4: MUTEX
// Shared counter, incremented by 100 goroutines. The final value must be exact.
func TestMutex_Counter(t *testing.T) {
	cases := []struct{ goroutines, perGoroutine int }{
		{100, 1},    // expected exactly 100
		{100, 1000}, // expected exactly 100,000
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%dx%d", tc.goroutines, tc.perGoroutine), func(t *testing.T) {
			var c SafeCounter
			var wg sync.WaitGroup
			wg.Add(tc.goroutines)
			for i := 0; i < tc.goroutines; i++ {
				go func() {
					defer wg.Done()
					for j := 0; j < tc.perGoroutine; j++ {
						c.Inc()
					}
				}()
			}
			wg.Wait()

			want := tc.goroutines * tc.perGoroutine
			t.Logf("final=%d expected=%d", c.Value(), want)
			if got := c.Value(); got != want {
				t.Fatalf("final counter=%d, want exactly %d (lost updates = data race)", got, want)
			}
		})
	}
	report(t,
		"Prove that a mutex makes a read-modify-write (counter++) atomic, so the final value is exactly goroutines x increments with no lost updates.",
		"Use a Mutex around SMALL critical sections only and never do I/O (DB, HTTP, Redis) while holding it. "+
			"For a simple counter prefer sync/atomic (faster). Use RWMutex when reads are much more frequent than writes. "+
			"Always run this test with -race.")
}

// atomic gives the same guarantee without a lock.
func TestAtomic_Counter(t *testing.T) {
	var n int64
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				atomic.AddInt64(&n, 1)
			}
		}()
	}
	wg.Wait()
	if n != 100000 {
		t.Fatalf("atomic counter=%d, want 100000", n)
	}
	report(t,
		"Show the lock-free alternative for simple counters.",
		"Use atomic for counters and flags; use a Mutex when several fields must change together.")
}

// A plain map is NOT safe: concurrent writes crash Go. RWMutex protects it.
type SafeMap struct {
	mu sync.RWMutex
	m  map[int]int
}

func (s *SafeMap) Set(k, v int) {
	s.mu.Lock()
	s.m[k] = v
	s.mu.Unlock()
}

func (s *SafeMap) Get(k int) (int, bool) {
	s.mu.RLock()
	v, ok := s.m[k]
	s.mu.RUnlock()
	return v, ok
}

func TestRWMutex_Map(t *testing.T) {
	s := &SafeMap{m: make(map[int]int)}
	var wg sync.WaitGroup

	// 50 writers
	for w := 0; w < 50; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				s.Set(w*100+i, i)
			}
		}(w)
	}
	// 50 readers at the same time
	for r := 0; r < 50; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				s.Get(i)
			}
		}()
	}
	wg.Wait()

	if len(s.m) != 5000 {
		t.Fatalf("map has %d entries, want 5000", len(s.m))
	}
	report(t,
		"Verify that a map shared by many goroutines is safe when guarded by RWMutex (a plain map would crash with 'concurrent map writes').",
		"RWMutex for read-heavy data such as caches and the connection list; keep the locked section tiny. "+
			"Our WebSocket Hub guards its client map with a Mutex for the same reason.")
}
