package concurrency

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TEST 3: CHANNELS
// Send 100 jobs through a channel, receive and process them,
// and verify that all 100 jobs were received exactly once.
func TestChannels_100Jobs(t *testing.T) {
	const total = 100

	jobs := make(chan int, 10) // buffered channel
	var wg sync.WaitGroup

	// producer: sends the jobs, then closes the channel to say "no more jobs"
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(jobs)
		for i := 1; i <= total; i++ {
			jobs <- i
		}
	}()

	// consumer (this goroutine): receives until the channel is closed
	received := 0
	sum := 0
	seen := make(map[int]int)
	for job := range jobs {
		received++
		sum += job
		seen[job]++
	}
	wg.Wait()

	t.Logf("sent=%d received=%d sum=%d", total, received, sum)

	if received != total {
		t.Fatalf("received %d jobs, want %d", received, total)
	}
	if want := total * (total + 1) / 2; sum != want { // 1+2+...+100 = 5050
		t.Fatalf("sum=%d, want %d (a job was lost or duplicated)", sum, want)
	}
	for i := 1; i <= total; i++ {
		if seen[i] != 1 {
			t.Fatalf("job %d was received %d times, want exactly 1", i, seen[i])
		}
	}

	report(t,
		"Verify that channels pass data safely between goroutines: nothing lost, nothing duplicated, and closing the channel ends the consumer loop.",
		"Buffer size: 0 (unbuffered) when you want sender and receiver in lock-step; otherwise about 1-2x the number of consumers "+
			"or the size of an expected burst (our WebSocket hub uses 256). Only the SENDER closes a channel, and only once.")
}

// The result must be the same whatever the buffer size.
func TestChannels_BufferSizes(t *testing.T) {
	const total = 100
	for _, size := range []int{0, 1, 10, 100} {
		t.Run(fmt.Sprintf("buffer=%d", size), func(t *testing.T) {
			ch := make(chan int, size)
			go func() {
				for i := 0; i < total; i++ {
					ch <- i
				}
				close(ch)
			}()
			n := 0
			for range ch {
				n++
			}
			if n != total {
				t.Fatalf("buffer=%d: received %d, want %d", size, n, total)
			}
		})
	}
	report(t,
		"Show that buffer size changes speed/blocking behaviour, never correctness.",
		"Start small (10-100). A huge buffer only hides a slow consumer and uses memory.")
}

// Many producers and many consumers at the same time.
func TestChannels_ManyProducersManyConsumers(t *testing.T) {
	const producers, perProducer, consumers = 10, 10, 5 // 100 jobs

	jobs := make(chan int, 20)
	results := make(chan int, producers*perProducer)

	var pw sync.WaitGroup
	pw.Add(producers)
	for p := 0; p < producers; p++ {
		go func(p int) {
			defer pw.Done()
			for j := 0; j < perProducer; j++ {
				jobs <- p*perProducer + j
			}
		}(p)
	}
	go func() { pw.Wait(); close(jobs) }() // close only after ALL producers finished

	var cw sync.WaitGroup
	cw.Add(consumers)
	for c := 0; c < consumers; c++ {
		go func() {
			defer cw.Done()
			for j := range jobs {
				results <- j
			}
		}()
	}
	cw.Wait()
	close(results)

	seen := map[int]bool{}
	for r := range results {
		if seen[r] {
			t.Fatalf("job %d processed twice", r)
		}
		seen[r] = true
	}
	if len(seen) != 100 {
		t.Fatalf("processed %d unique jobs, want 100", len(seen))
	}
	report(t,
		"Verify fan-in / fan-out: several senders and several receivers share one channel safely.",
		"Close the jobs channel from ONE place (after a WaitGroup for all producers), never from several producers.")
}

// A receive on a channel nobody writes to must not hang the program forever.
func TestChannels_TimeoutInsteadOfDeadlock(t *testing.T) {
	ch := make(chan int) // nobody sends
	select {
	case <-ch:
		t.Fatal("unexpected value")
	case <-time.After(50 * time.Millisecond):
		t.Log("timeout fired as expected, no deadlock")
	}
	report(t,
		"Show the select + time.After pattern that protects handlers from waiting forever.",
		"Always give network/Redis/DB waits a timeout: 3-15 s for API calls (the AI calls here use 15 s).")
}