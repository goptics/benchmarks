package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"maragu.dev/goqite"
	"maragu.dev/goqite/jobs"
)

// goqiteSchema is schema_sqlite.sql from maragu.dev/goqite v0.4.0 (MIT): the
// module ships it as a file, not as Go code.
const goqiteSchema = `
create table goqite (
  id text primary key default ('m_' || lower(hex(randomblob(16)))),
  created text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  updated text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  queue text not null,
  body blob not null,
  timeout text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  received integer not null default 0,
  priority integer not null default 0
) strict;

create trigger goqite_updated_timestamp after update on goqite begin
  update goqite set updated = strftime('%Y-%m-%dT%H:%M:%fZ') where id = old.id;
end;

create index goqite_queue_priority_created_idx on goqite (queue, priority desc, created);
`

type goqiteConsumer struct {
	db      *sql.DB
	queues  []*goqite.Queue
	runners []*jobs.Runner
	prio    bool
	ran     *atomic.Int64
	total   int
	cancel  context.CancelFunc
	stopped sync.WaitGroup
}

// openGoqite makes one goqite queue and one job runner (16 jobs at a time,
// the default 100 ms poll) per scenario queue, all in one database file.
func openGoqite(tb testing.TB, dir string, sc scenario, ckpt int, ran *atomic.Int64) consumer {
	tb.Helper()

	db := openDB(tb, filepath.Join(dir, "bench.db"), ckpt)
	// goqite's README setup: one connection, since SQLite allows one writer at a time
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec(goqiteSchema); err != nil {
		tb.Fatalf("goqite schema: %v", err)
	}

	c := &goqiteConsumer{db: db, prio: sc.priority, ran: ran, total: sc.total}

	for i := range sc.queues {
		q := goqite.New(goqite.NewOpts{DB: db, Name: fmt.Sprintf("q%d", i)})
		r := jobs.NewRunner(jobs.NewRunnerOpts{Limit: 16, Queue: q})
		r.Register("bench", func(context.Context, []byte) error {
			ran.Add(1)

			return nil
		})

		c.queues = append(c.queues, q)
		c.runners = append(c.runners, r)
	}

	return c
}

func (c *goqiteConsumer) add(q, i int) bool {
	m := goqite.Message{Body: strconv.AppendInt(nil, int64(i), 10)}
	if c.prio {
		m.Priority = i % 4 // higher first
	}

	_, err := jobs.Create(context.Background(), c.queues[q], "bench", m)

	return err == nil
}

func (c *goqiteConsumer) start() error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	for _, r := range c.runners {
		c.stopped.Go(func() { r.Start(ctx) })
	}

	return nil
}

func (c *goqiteConsumer) wait(deadline time.Duration) error {
	return waitHandled(c.ran, c.total, c.leftover, deadline)
}

// leftover counts messages; the runner deletes a message after its job returns.
func (c *goqiteConsumer) leftover() (int, error) { return count(c.db, `SELECT COUNT(*) FROM goqite`) }

func (c *goqiteConsumer) close() error {
	if c.cancel != nil {
		c.cancel()
		c.stopped.Wait()
	}

	return c.db.Close()
}
