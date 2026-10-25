// Package sqlite benchmarks SQLite-backed job queues end to end: sqliteq
// through a varmq worker against goqite, backlite and River's SQLite driver.
package sqlite

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// scenario is one group of the benchmark matrix.
type scenario struct {
	name      string
	queues    int
	producers int  // per queue
	total     int  // jobs over all queues
	drain     bool // fill everything first, then time the consumer from start to empty
	priority  bool // job i gets priority i%4; only subjects with a priority queue run it
}

func (sc scenario) perQueue() int    { return sc.total / sc.queues }
func (sc scenario) perProducer() int { return sc.perQueue() / sc.producers }

// tally records which jobs every subject's handler ran. Job ids run from 0 to
// total-1 across all queues, so each one must be seen exactly once.
type tally struct {
	seen    []atomic.Uint32
	handled atomic.Int64 // distinct jobs seen
	bad     atomic.Int64 // repeats and ids out of range
}

func newTally(total int) *tally { return &tally{seen: make([]atomic.Uint32, total)} }

// handle is every subject's whole handler.
func (t *tally) handle(id int) {
	if id < 0 || id >= len(t.seen) {
		t.bad.Add(1)

		return
	}

	if t.seen[id].Add(1) == 1 {
		t.handled.Add(1)
	} else {
		t.bad.Add(1)
	}
}

// check fails unless every job was handled exactly once.
func (t *tally) check() error {
	if n := t.bad.Load(); n > 0 {
		return fmt.Errorf("%d duplicate or unknown deliveries", n)
	}

	if n := t.handled.Load(); n != int64(len(t.seen)) {
		return fmt.Errorf("%d of %d jobs handled", n, len(t.seen))
	}

	return nil
}

// consumer is one subject bound to a scenario's queues on a fresh database.
// Every subject's handler is ran.handle(job id).
type consumer interface {
	add(queue, i int) bool             // enqueue job i on queue
	start() error                      // begin consuming: after filling on drain scenarios, before on mixed
	wait(deadline time.Duration) error // until every job is handled and acknowledged
	leftover() (int, error)            // rows still in the queue tables
	close() error
}

type subject struct {
	name     string
	priority bool // has a priority queue, so it runs DrainPrio
	only10Q  bool // identical to another subject with one queue, so it runs only with more
	open     func(tb testing.TB, dir string, sc scenario, ckpt int, ran *tally) consumer
}

// subjects in chart order: vizb lays out the x axis by first appearance.
var subjects = []subject{
	{name: "VarMQ", priority: true, open: openVarMQ(folder)},
	{name: "VarMQSingleFile", only10Q: true, open: openVarMQ(singleFile)},
	{name: "Goqite", priority: true, open: openGoqite},
	{name: "Backlite", open: openBacklite},
	{name: "River", priority: true, open: openRiver},
}

func (s subject) runs(sc scenario) bool {
	return (!sc.priority || s.priority) && (!s.only10Q || sc.queues > 1)
}

// produce runs sc.producers goroutines per queue, each adding perProducer
// jobs, and returns how many adds failed.
func produce(c consumer, sc scenario) int64 {
	var (
		wg     sync.WaitGroup
		failed atomic.Int64
	)

	for q := range sc.queues {
		for p := range sc.producers {
			wg.Go(func() {
				for j := range sc.perProducer() {
					if !c.add(q, q*sc.perQueue()+p*sc.perProducer()+j) {
						failed.Add(1)
					}
				}
			})
		}
	}

	wg.Wait()

	return failed.Load()
}

// waitHandled is wait for the subjects without a Wait of their own: first
// until the handler ran total times, then until the last acknowledgement
// cleared the queue table, all within deadline.
func waitHandled(ran *tally, total int, leftover func() (int, error), deadline time.Duration) error {
	timeout := time.NewTimer(deadline)
	defer timeout.Stop()

	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()

	for ran.handled.Load() < int64(total) {
		select {
		case <-tick.C:
		case <-timeout.C:
			return fmt.Errorf("%d of %d jobs handled after %s", ran.handled.Load(), total, deadline)
		}
	}

	// leftover is a table scan for some subjects, on a pool of one connection
	// that the last acknowledgements need too, so it is polled more gently
	slow := time.NewTicker(10 * time.Millisecond)
	defer slow.Stop()

	for {
		n, err := leftover()
		if err != nil {
			return err
		}

		if n == 0 {
			return nil
		}

		select {
		case <-slow.C:
		case <-timeout.C:
			return fmt.Errorf("%d rows still queued %s after the last job", n, deadline)
		}
	}
}
