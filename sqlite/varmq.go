package sqlite

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/goptics/sqliteq"
	"github.com/goptics/varmq"
)

// layout is how sqliteq maps queues onto database files.
type layout int

const (
	// folder is one database file per queue, all in the directory path: sqliteq's default.
	folder layout = iota
	// singleFile is every queue in the one database file path.
	singleFile
)

// openStore opens one sqliteq database set at path with the given WAL
// checkpoint page count.
func openStore(tb testing.TB, path string, l layout, ckpt int) sqliteq.Queues {
	tb.Helper()

	opts := []sqliteq.StoreOption{sqliteq.WithCheckpointPages(ckpt)}
	if l == singleFile {
		opts = append(opts, sqliteq.WithSingleFile())
	}

	qs, err := sqliteq.New(path, opts...)
	if err != nil {
		tb.Fatalf("sqliteq.New(%s): %v", path, err)
	}

	return qs
}

// bound is one queue with its worker.
type bound struct {
	worker varmq.Worker
	add    func(i int) bool // enqueues job number i
}

func bind(tb testing.TB, qs sqliteq.Queues, sc scenario, name string, ran *tally) bound {
	tb.Helper()

	configs := []any{varmq.WithConcurrency(16)}
	if sc.drain {
		configs = append(configs, varmq.WithAutoRun(false)) // start() runs it after the fill
	}

	w := varmq.NewWorker(func(j varmq.Job[int]) { ran.handle(j.Data()) }, configs...)

	if sc.priority {
		q, err := qs.NewPriorityQueue(name)
		if err != nil {
			tb.Fatal(err)
		}

		pq := w.WithPersistentPriorityQueue(q)

		return bound{worker: pq.Worker(), add: func(i int) bool { return pq.Add(i, i%4) }}
	}

	q, err := qs.NewQueue(name)
	if err != nil {
		tb.Fatal(err)
	}

	pq := w.WithPersistentQueue(q)

	return bound{worker: pq.Worker(), add: func(i int) bool { return pq.Add(i) }}
}

type varmqConsumer struct {
	qs     sqliteq.Queues
	bounds []bound
	path   string
	l      layout
	drain  bool
}

func openVarMQ(l layout) func(testing.TB, string, scenario, int, *tally) consumer {
	return func(tb testing.TB, dir string, sc scenario, ckpt int, ran *tally) consumer {
		tb.Helper()

		path := dir
		if l == singleFile {
			path = filepath.Join(dir, "bench.db")
		}

		c := &varmqConsumer{qs: openStore(tb, path, l, ckpt), path: path, l: l, drain: sc.drain}

		for i := range sc.queues {
			c.bounds = append(c.bounds, bind(tb, c.qs, sc, fmt.Sprintf("q%d", i), ran))
		}

		return c
	}
}

func (c *varmqConsumer) add(q, i int) bool { return c.bounds[q].add(i) }

// start runs the workers of a drain run; a mixed run's workers have been
// running since NewWorker.
func (c *varmqConsumer) start() error {
	if !c.drain {
		return nil
	}

	for _, bd := range c.bounds {
		if err := bd.worker.Start(); err != nil {
			return err
		}
	}

	return nil
}

func (c *varmqConsumer) wait(deadline time.Duration) error {
	waited := make(chan struct{})

	go func() {
		defer close(waited)

		for _, bd := range c.bounds {
			bd.worker.Wait()
		}
	}()

	select {
	case <-waited:
		return nil
	case <-time.After(deadline):
		return fmt.Errorf("workers still busy %s after the last add", deadline)
	}
}

func (c *varmqConsumer) leftover() (int, error) { return rowsLeft(c.path, c.l, len(c.bounds)) }

func (c *varmqConsumer) close() error {
	for _, bd := range c.bounds {
		if err := bd.worker.WaitAndStop(); err != nil {
			return err
		}
	}

	return c.qs.Close()
}

// rowsLeft counts the rows in every table of the store's database files, read
// with database/sql directly. It fails when it finds fewer than queues tables,
// so a check that looked at nothing cannot pass.
func rowsLeft(path string, l layout, queues int) (rows int, err error) {
	files := []string{path}

	if l == folder {
		if files, err = filepath.Glob(filepath.Join(path, "*.db")); err != nil {
			return 0, err
		}
	}

	tables := 0

	for _, f := range files {
		db, err := sql.Open("sqlite3", f)
		if err != nil {
			return 0, err
		}

		names, err := tableNames(db)
		if err != nil {
			db.Close()

			return 0, err
		}

		for _, t := range names {
			var n int
			if err := db.QueryRow(`SELECT COUNT(*) FROM "` + t + `"`).Scan(&n); err != nil {
				db.Close()

				return 0, err
			}

			rows += n
			tables++
		}

		db.Close()
	}

	if tables < queues {
		return 0, fmt.Errorf("found %d queue tables, want at least %d", tables, queues)
	}

	return rows, nil
}

func tableNames(db *sql.DB) ([]string, error) {
	r, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var names []string

	for r.Next() {
		var n string
		if err := r.Scan(&n); err != nil {
			return nil, err
		}

		names = append(names, n)
	}

	return names, r.Err()
}
