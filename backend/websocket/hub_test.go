package websocket

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

// These tests exercise the REAL Hub from hub.go with many goroutines at once.
// Run them with the race detector:   go test -race -v ./websocket/

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard) // the Hub logs every connect/disconnect; keep the output readable
	os.Exit(m.Run())
}

func report(t *testing.T, purpose, recommended string) {
	t.Helper()
	t.Logf("\n      PURPOSE     : %s\n      RECOMMENDED : %s", purpose, recommended)
}

// newTestHub creates a fresh Hub (NOT the global one) and starts its loop.
func newTestHub() *Hub {
	h := &Hub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan Event, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
	go h.Run()
	return h
}

func newClient(h *Hub, org, user string, buffer int) *Client {
	return &Client{Hub: h, OrgID: org, UserID: user, Send: make(chan []byte, buffer)}
}

func (h *Hub) clientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.Clients)
}

// waitFor polls a condition (the Hub runs in its own goroutine).
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", what)
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// drain returns every event type currently waiting in the client's channel.
func drain(c *Client) []string {
	var types []string
	for {
		select {
		case raw, ok := <-c.Send:
			if !ok {
				return types
			}
			var e Event
			_ = json.Unmarshal(raw, &e)
			types = append(types, e.Type)
		default:
			return types
		}
	}
}

// registerAll registers the clients from many goroutines at the same time.
func registerAll(h *Hub, clients []*Client) {
	var wg sync.WaitGroup
	wg.Add(len(clients))
	for _, c := range clients {
		go func(c *Client) {
			defer wg.Done()
			h.Register <- c
		}(c)
	}
	wg.Wait()
}

// HUB TEST 1: 1,000 goroutines register 1,000 clients at the same time
func TestHub_ConcurrentRegister_1000(t *testing.T) {
	h := newTestHub()
	clients := make([]*Client, 1000)
	for i := range clients {
		clients[i] = newClient(h, fmt.Sprintf("org-%d", i%4), fmt.Sprintf("user-%d", i), 8)
	}

	start := time.Now()
	registerAll(h, clients)
	waitFor(t, "all 1000 clients registered", func() bool { return h.clientCount() == 1000 })
	t.Logf("registered %d clients concurrently in %v", h.clientCount(), time.Since(start))

	// and 1,000 goroutines unregister them again
	var wg sync.WaitGroup
	wg.Add(len(clients))
	for _, c := range clients {
		go func(c *Client) { defer wg.Done(); h.Unregister <- c }(c)
	}
	wg.Wait()
	waitFor(t, "all clients removed", func() bool { return h.clientCount() == 0 })

	report(t,
		"Prove the Hub's map of connections stays correct (no lost or duplicate entries, no 'concurrent map' crash) when many browsers connect/disconnect at once.",
		"Run with -race. Test with 1,000 simultaneous connections; a single 512 MB server can usually hold several thousand idle WebSockets.")
}

// HUB TEST 2: events reach ONLY the right workspace (tenant isolation)
func TestHub_OrgIsolation(t *testing.T) {
	h := newTestHub()
	const perOrg = 250
	var all []*Client
	byOrg := map[string][]*Client{}
	for _, org := range []string{"org-A", "org-B", "org-C", "org-D"} {
		for i := 0; i < perOrg; i++ {
			c := newClient(h, org, fmt.Sprintf("%s-user-%d", org, i), 8)
			all = append(all, c)
			byOrg[org] = append(byOrg[org], c)
		}
	}
	registerAll(h, all)
	waitFor(t, "clients registered", func() bool { return h.clientCount() == len(all) })

	// event for org-A only, then a "barrier" event for everyone.
	// The Hub handles events in order, so when the barrier has arrived the first event is finished.
	h.Broadcast <- Event{Type: "ORG_A_ONLY", OrgID: "org-A"}
	h.Broadcast <- Event{Type: "BARRIER"} // no OrgID = every workspace

	for _, c := range all {
		c := c
		waitFor(t, "barrier delivered", func() bool { return len(c.Send) >= 2-btoi(c.OrgID != "org-A") })
	}

	gotA, leaked := 0, 0
	for org, list := range byOrg {
		for _, c := range list {
			got := drain(c)
			if org == "org-A" {
				if len(got) == 2 && got[0] == "ORG_A_ONLY" && got[1] == "BARRIER" {
					gotA++
				}
			} else if len(got) != 1 || got[0] != "BARRIER" {
				leaked++
			}
		}
	}
	t.Logf("org-A clients that received the event: %d of %d | clients of other orgs that wrongly received it: %d", gotA, perOrg, leaked)
	if gotA != perOrg {
		t.Fatalf("only %d of %d org-A clients got the event", gotA, perOrg)
	}
	if leaked != 0 {
		t.Fatalf("SECURITY: %d clients of OTHER workspaces received org-A's event", leaked)
	}

	report(t,
		"Verify multi-tenant isolation: a workspace event is delivered to every user of that workspace and to nobody else.",
		"Must be exactly 100% delivered to the right org and 0 leaks. This is a security requirement, keep this test in CI.")
}

// HUB TEST 3: targeted event (FORCE_LOGOUT / ROLE_CHANGED / NOTIFICATION)
func TestHub_TargetedUser(t *testing.T) {
	h := newTestHub()
	target1 := newClient(h, "org-A", "user-1", 8) // two tabs of the same user
	target2 := newClient(h, "org-A", "user-1", 8)
	other := newClient(h, "org-A", "user-2", 8)
	registerAll(h, []*Client{target1, target2, other})
	waitFor(t, "registered", func() bool { return h.clientCount() == 3 })

	h.Broadcast <- Event{Type: "FORCE_LOGOUT", OrgID: "org-A", TargetUser: "user-1"}
	h.Broadcast <- Event{Type: "BARRIER"}
	for _, c := range []*Client{target1, target2, other} {
		c := c
		waitFor(t, "barrier", func() bool { return len(c.Send) >= 1 })
	}

	for name, c := range map[string]*Client{"tab 1": target1, "tab 2": target2} {
		if got := drain(c); len(got) != 2 || got[0] != "FORCE_LOGOUT" {
			t.Fatalf("%s of the target user did not get FORCE_LOGOUT: %v", name, got)
		}
	}
	if got := drain(other); len(got) != 1 || got[0] != "BARRIER" {
		t.Fatalf("a different user received the targeted event: %v", got)
	}

	report(t,
		"Verify that a user-specific event reaches all tabs of that user and nobody else (the removed-user logout depends on this).",
		"Exactly 1 delivery per open tab of the target user, 0 for everyone else.")
}

// HUB TEST 4: 100 goroutines publish 10 events each = 1,000 events, none may be lost
func TestHub_ConcurrentBroadcast_NoLostEvents(t *testing.T) {
	h := newTestHub()
	listener := newClient(h, "org-A", "listener", 2000)
	registerAll(h, []*Client{listener})
	waitFor(t, "registered", func() bool { return h.clientCount() == 1 })

	const publishers, each = 100, 10
	var wg sync.WaitGroup
	wg.Add(publishers)
	start := time.Now()
	for p := 0; p < publishers; p++ {
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				h.Broadcast <- Event{Type: "TASKS_UPDATED", OrgID: "org-A"}
			}
		}()
	}
	wg.Wait()
	waitFor(t, "all events delivered", func() bool { return len(listener.Send) == publishers*each })
	t.Logf("%d events from %d goroutines delivered in %v", len(listener.Send), publishers, time.Since(start))

	report(t,
		"Verify the Broadcast channel + single Run loop handles many concurrent publishers with no lost events.",
		"Broadcast channel buffer: 256 (current). It must be >= the burst size you expect between two Hub loop turns; 100-1,000 events in a burst is typical.")
}

// HUB TEST 5: a slow/stuck browser must not block the others and must be removed safely
func TestHub_SlowClientIsEvictedSafely(t *testing.T) {
	h := newTestHub()
	slow := newClient(h, "org-A", "slow", 1) // buffer of 1 and nobody reads it
	fast := newClient(h, "org-A", "fast", 500)
	registerAll(h, []*Client{slow, fast})
	waitFor(t, "registered", func() bool { return h.clientCount() == 2 })

	for i := 0; i < 50; i++ {
		h.Broadcast <- Event{Type: "TASKS_UPDATED", OrgID: "org-A"}
	}
	waitFor(t, "fast client got everything", func() bool { return len(fast.Send) == 50 })
	waitFor(t, "slow client removed", func() bool { return h.clientCount() == 1 })

	// The slow client's channel must be closed
	closed := false
	for i := 0; i < 3 && !closed; i++ {
		select {
		case _, ok := <-slow.Send:
			closed = !ok
		default:
		}
	}
	if !closed {
		t.Fatal("the evicted client's Send channel was not closed")
	}

	// Its read loop would now call Unregister: that must NOT panic (double close)
	h.Unregister <- slow
	h.Broadcast <- Event{Type: "AFTER", OrgID: "org-A"}
	waitFor(t, "hub still alive", func() bool { return len(fast.Send) == 51 })

	report(t,
		"Verify the Hub protects itself from a stuck connection: it drops that client, keeps serving the others, and never panics.",
		"Per-client Send buffer: 256 (current). Larger hides a stuck client, smaller drops healthy clients during bursts.")
}

// HUB TEST 6: everything at once (register + unregister + broadcast). Its value is under -race.
func TestHub_MixedLoad_Race(t *testing.T) {
	h := newTestHub()
	var wg sync.WaitGroup

	for i := 0; i < 300; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := newClient(h, fmt.Sprintf("org-%d", i%3), fmt.Sprintf("u-%d", i), 64)
			h.Register <- c
			for j := 0; j < 5; j++ {
				h.Broadcast <- Event{Type: "TASKS_UPDATED", OrgID: c.OrgID}
			}
			h.Unregister <- c
		}(i)
	}
	wg.Wait()
	waitFor(t, "all clients gone", func() bool { return h.clientCount() == 0 })

	report(t,
		"Let the race detector inspect the Hub while connections, disconnections and broadcasts all happen simultaneously.",
		"Always run: go test -race ./websocket/ . Zero 'DATA RACE' warnings is the only acceptable result.")
}

// Benchmark: cost of delivering ONE event to 1,000 connected clients
func BenchmarkHub_Broadcast_1000Clients(b *testing.B) {
	h := newTestHub()
	clients := make([]*Client, 1000)
	for i := range clients {
		clients[i] = newClient(h, "org-A", fmt.Sprintf("u-%d", i), 4)
		h.Register <- clients[i]
	}
	// drain everything in the background so the buffers never fill
	stop := make(chan struct{})
	for _, c := range clients {
		go func(c *Client) {
			for {
				select {
				case <-c.Send:
				case <-stop:
					return
				}
			}
		}(c)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Broadcast <- Event{Type: "TASKS_UPDATED", OrgID: "org-A"}
	}
	b.StopTimer()
	close(stop)
}