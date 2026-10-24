package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mikestefanello/backlite"
)

// backlite reads a queue's name from the zero value of its task type, so every
// scenario queue needs a task type of its own: backliteTask[qN].
type qname interface{ name() string }

type (
	q0 struct{}
	q1 struct{}
	q2 struct{}
	q3 struct{}
	q4 struct{}
	q5 struct{}
	q6 struct{}
	q7 struct{}
	q8 struct{}
	q9 struct{}
)

func (q0) name() string { return "q0" }
func (q1) name() string { return "q1" }
func (q2) name() string { return "q2" }
func (q3) name() string { return "q3" }
func (q4) name() string { return "q4" }
func (q5) name() string { return "q5" }
func (q6) name() string { return "q6" }
func (q7) name() string { return "q7" }
func (q8) name() string { return "q8" }
func (q9) name() string { return "q9" }

type backliteTask[Q qname] struct {
	N int
}

func (backliteTask[Q]) Config() backlite.QueueConfig {
	var q Q

	return backlite.QueueConfig{Name: q.name()}
}

// backliteQueue builds the queue for one task type and the function that
// makes its tasks.
func backliteQueue[Q qname](ran *atomic.Int64) (backlite.Queue, func(int) backlite.Task) {
	q := backlite.NewQueue[backliteTask[Q]](func(context.Context, backliteTask[Q]) error {
		ran.Add(1)

		return nil
	})

	return q, func(i int) backlite.Task { return backliteTask[Q]{N: i} }
}

// ponytail: ten task types for the ten-queue scenario; add a type to go wider.
var backliteQueues = []func(*atomic.Int64) (backlite.Queue, func(int) backlite.Task){
	backliteQueue[q0], backliteQueue[q1], backliteQueue[q2], backliteQueue[q3], backliteQueue[q4],
	backliteQueue[q5], backliteQueue[q6], backliteQueue[q7], backliteQueue[q8], backliteQueue[q9],
}

type backliteConsumer struct {
	db      *sql.DB
	client  *backlite.Client
	tasks   []func(int) backlite.Task
	ran     *atomic.Int64
	total   int
	started bool
}

// openBacklite makes one backlite client with 16 workers per scenario queue
// and one registered queue per scenario queue, all in one database file.
func openBacklite(tb testing.TB, dir string, sc scenario, ckpt int, ran *atomic.Int64) consumer {
	tb.Helper()

	if sc.queues > len(backliteQueues) {
		tb.Fatalf("backlite: %d queues, task types for %d", sc.queues, len(backliteQueues))
	}

	db := openDB(tb, filepath.Join(dir, "bench.db"), ckpt)

	client, err := backlite.NewClient(backlite.ClientConfig{
		DB:           db,
		NumWorkers:   16 * sc.queues,
		ReleaseAfter: 10 * time.Minute, // required; requeues a claimed task after this, far beyond one run
	})
	if err != nil {
		tb.Fatalf("backlite: %v", err)
	}

	if err := client.Install(); err != nil {
		tb.Fatalf("backlite schema: %v", err)
	}

	c := &backliteConsumer{db: db, client: client, ran: ran, total: sc.total}

	for i := range sc.queues {
		q, mk := backliteQueues[i](ran)
		client.Register(q)
		c.tasks = append(c.tasks, mk)
	}

	return c
}

func (c *backliteConsumer) add(q, i int) bool {
	_, err := c.client.Add(c.tasks[q](i)).Save()

	return err == nil
}

func (c *backliteConsumer) start() error {
	c.client.Start(context.Background())
	c.started = true

	return nil
}

func (c *backliteConsumer) wait(deadline time.Duration) error {
	return waitHandled(c.ran, c.total, c.leftover, deadline)
}

// leftover counts pending and claimed tasks; without retention a completed
// task is deleted.
func (c *backliteConsumer) leftover() (int, error) {
	return count(c.db, `SELECT COUNT(*) FROM backlite_tasks`)
}

func (c *backliteConsumer) close() error {
	var err error

	if c.started {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		if !c.client.Stop(ctx) {
			err = errors.New("backlite: workers still busy a minute after stop")
		}
	}

	return errors.Join(err, c.db.Close())
}
