package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riversqlite"
	"github.com/riverqueue/river/rivermigrate"
)

type riverArgs struct {
	N int `json:"n"`
}

func (riverArgs) Kind() string { return "bench" }

type riverWorker struct {
	river.WorkerDefaults[riverArgs]
	ran *atomic.Int64
}

func (w *riverWorker) Work(context.Context, *river.Job[riverArgs]) error {
	w.ran.Add(1)

	return nil
}

type riverConsumer struct {
	db      *sql.DB
	client  *river.Client[*sql.Tx]
	prio    bool
	ran     *atomic.Int64
	total   int
	started bool
}

// openRiver migrates a fresh database and makes one River client with a
// 16-worker queue per scenario queue.
func openRiver(tb testing.TB, dir string, sc scenario, ckpt int, ran *atomic.Int64) consumer {
	tb.Helper()

	db := openDB(tb, filepath.Join(dir, "bench.db"), ckpt)
	// riversqlite's package doc: one connection, so River's parallel internal
	// operations do not fail with SQLITE_BUSY
	db.SetMaxOpenConns(1)

	driver := riversqlite.New(db)
	ctx := context.Background()

	migrator, err := rivermigrate.New(driver, nil)
	if err != nil {
		tb.Fatalf("river migrator: %v", err)
	}

	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		tb.Fatalf("river migrate: %v", err)
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, &riverWorker{ran: ran})

	queues := make(map[string]river.QueueConfig, sc.queues)
	for i := range sc.queues {
		// Forced: a queue's producer fetches at most MaxWorkers jobs, then waits
		// FetchCooldown before the next fetch, so the default 100 ms caps 16
		// workers at 160 jobs/s whatever the database does. The 1 ms minimum
		// lets SQLite and River's own per-job work set the number. It is set per
		// queue so the client-level value, which also paces River's
		// insert-notification writes, stays at its default.
		queues[fmt.Sprintf("q%d", i)] = river.QueueConfig{MaxWorkers: 16, FetchCooldown: river.FetchCooldownMin}
	}

	client, err := river.NewClient(driver, &river.Config{
		Logger:  slog.New(slog.DiscardHandler),
		Queues:  queues,
		Workers: workers,
	})
	if err != nil {
		tb.Fatalf("river client: %v", err)
	}

	return &riverConsumer{db: db, client: client, prio: sc.priority, ran: ran, total: sc.total}
}

func (c *riverConsumer) add(q, i int) bool {
	opts := &river.InsertOpts{Queue: fmt.Sprintf("q%d", q)}
	if c.prio {
		opts.Priority = i%4 + 1 // 1 is River's highest
	}

	_, err := c.client.Insert(context.Background(), riverArgs{N: i}, opts)

	return err == nil
}

func (c *riverConsumer) start() error {
	c.started = true

	return c.client.Start(context.Background())
}

func (c *riverConsumer) wait(deadline time.Duration) error {
	return waitHandled(c.ran, c.total, c.leftover, deadline)
}

// leftover counts jobs not yet completed; River keeps completed rows in
// river_job for 24 hours by default.
func (c *riverConsumer) leftover() (int, error) {
	return count(c.db, `SELECT COUNT(*) FROM river_job WHERE state <> 'completed'`)
}

func (c *riverConsumer) close() error {
	var err error

	if c.started {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		err = c.client.Stop(ctx)
	}

	return errors.Join(err, c.db.Close())
}
