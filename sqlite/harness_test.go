package sqlite

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var scenarios = []scenario{
	{name: "Mixed1Q", queues: 1, producers: 4, total: 20000},
	{name: "Drain1Q", queues: 1, producers: 4, total: 20000, drain: true},
	{name: "DrainPrio", queues: 1, producers: 4, total: 10000, drain: true, priority: true},
	{name: "Mixed10Q", queues: 10, producers: 1, total: 20000},
}

// benchRoot is where the benchmark databases live. They must be on a real disk:
// on a tmpfs /tmp the fsync cost this benchmark measures would vanish.
func benchRoot(tb testing.TB) string {
	tb.Helper()

	if d := os.Getenv("SQLITEQ_BENCH_DIR"); d != "" {
		return d
	}

	cache, err := os.UserCacheDir()
	if err != nil {
		tb.Fatalf("no SQLITEQ_BENCH_DIR and no user cache dir: %v", err)
	}

	return filepath.Join(cache, "varmq-bench-sqlite")
}

// freshDir makes an empty directory under benchRoot and removes it on cleanup.
func freshDir(tb testing.TB) string {
	tb.Helper()

	root := benchRoot(tb)
	if err := os.MkdirAll(root, 0o755); err != nil {
		tb.Fatalf("mkdir %s: %v", root, err)
	}

	dir, err := os.MkdirTemp(root, "run-")
	if err != nil {
		tb.Fatalf("mkdir temp: %v", err)
	}

	tb.Cleanup(func() { _ = os.RemoveAll(dir) })

	return dir
}

// waitDeadline bounds how long one iteration waits for its consumer to drain,
// so a stuck job fails its row instead of hanging the whole run.
const waitDeadline = 5 * time.Minute

// runScenario runs sc b.N times, each on a fresh database, and reports jobs/s
// over the timed part only: setup, teardown and the leftover check are not
// timed. Mixed times the first add to the last job acknowledged. Drain times
// the consumer start to the last job acknowledged. It also reports walMB, the
// largest total size the run's -wal files reached in any iteration, untimed
// parts included.
func runScenario(b *testing.B, sc scenario, subj subject, ckpt int) {
	b.Helper()

	var walPeak int64

	for range b.N {
		b.StopTimer()
		walPeak = max(walPeak, runOnce(b, sc, subj, ckpt))
	}

	if sec := b.Elapsed().Seconds(); sec > 0 {
		b.ReportMetric(float64(sc.total*b.N)/sec, "jobs/s")
	}

	b.ReportMetric(float64(walPeak)/1e6, "walMB")
}

// walSampleEvery is how often sampleWAL reads the -wal file sizes.
const walSampleEvery = 50 * time.Millisecond

// sampleWAL reads the total size of the -wal files in dir every
// walSampleEvery, on its own goroutine, until the returned stop is called.
// stop takes one last sample and returns the largest total seen; it may be
// called more than once. The total, not the largest file, so a subject with
// one file per queue is measured like one with every queue in one file.
func sampleWAL(dir string) (stop func() int64) {
	var peak int64 // owned by the goroutine until done closes

	sample := func() {
		files, _ := filepath.Glob(filepath.Join(dir, "*-wal"))

		var total int64

		for _, f := range files {
			if fi, err := os.Stat(f); err == nil {
				total += fi.Size()
			}
		}

		peak = max(peak, total)
	}

	quit, done := make(chan struct{}), make(chan struct{})

	go func() {
		defer close(done)

		t := time.NewTicker(walSampleEvery)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				sample()
			case <-quit:
				sample()

				return
			}
		}
	}()

	var once sync.Once

	return func() int64 {
		once.Do(func() { close(quit) })
		<-done

		return peak
	}
}

// runOnce runs one iteration and returns its WAL peak in bytes.
func runOnce(b *testing.B, sc scenario, subj subject, ckpt int) int64 {
	b.Helper()

	dir := freshDir(b)

	stopWAL := sampleWAL(dir)
	defer stopWAL() // a failed iteration still stops the sampler

	var ran atomic.Int64 // handler runs over all queues

	c := subj.open(b, dir, sc, ckpt, &ran)

	closed := false

	defer func() { // a failed iteration still stops its consumer
		if !closed {
			_ = c.close()
		}
	}()

	if sc.drain {
		if failed := produce(c, sc); failed > 0 {
			b.Fatalf("%d adds failed", failed)
		}

		b.StartTimer()

		if err := c.start(); err != nil {
			b.Fatalf("start: %v", err)
		}
	} else {
		if err := c.start(); err != nil {
			b.Fatalf("start: %v", err)
		}

		b.StartTimer()

		if failed := produce(c, sc); failed > 0 {
			b.StopTimer()
			b.Fatalf("%d adds failed", failed)
		}
	}

	if err := c.wait(waitDeadline); err != nil {
		b.StopTimer()
		b.Fatalf("%s: %v: a job is stuck", sc.name, err)
	}

	b.StopTimer()

	if n := ran.Load(); n != int64(sc.total) {
		b.Fatalf("%s: handler ran %d times, want %d: a job was lost or run twice", sc.name, n, sc.total)
	}

	if left, err := c.leftover(); err != nil {
		b.Fatalf("count leftover rows: %v", err)
	} else if left > 0 {
		b.Fatalf("%d rows left in the queue tables", left)
	}

	// Closing checkpoints and removes the WAL, so the sampling stops first
	walPeak := stopWAL()
	closed = true

	if err := c.close(); err != nil {
		b.Fatalf("close: %v", err)
	}

	return walPeak
}
