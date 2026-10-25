package sqlite

import (
	"testing"
	"time"
)

// TestSubjects runs every subject through small mixed, drain, priority and
// two-queue runs and checks that every job is handled exactly once and
// nothing is left in the queue tables.
func TestSubjects(t *testing.T) {
	cases := []scenario{
		{name: "Mixed", queues: 1, producers: 2, total: 200},
		{name: "Drain", queues: 1, producers: 2, total: 200, drain: true},
		{name: "DrainPrio", queues: 1, producers: 2, total: 200, drain: true, priority: true},
		{name: "Mixed2Q", queues: 2, producers: 1, total: 200},
	}

	for _, subj := range subjects {
		for _, sc := range cases {
			if !subj.runs(sc) {
				continue
			}

			t.Run(subj.name+"/"+sc.name, func(t *testing.T) {
				ran := newTally(sc.total)

				c := subj.open(t, t.TempDir(), sc, 1000, ran)

				var failed int64

				if sc.drain {
					failed = produce(c, sc)

					if err := c.start(); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := c.start(); err != nil {
						t.Fatal(err)
					}

					failed = produce(c, sc)
				}

				if err := c.wait(time.Minute); err != nil {
					t.Fatal(err)
				}

				if err := ran.check(); err != nil {
					t.Fatal(err)
				}

				if failed > 0 {
					t.Fatalf("%d adds failed", failed)
				}

				if left, err := c.leftover(); err != nil {
					t.Fatal(err)
				} else if left > 0 {
					t.Fatalf("%d rows left in the queue tables", left)
				}

				if err := c.close(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
