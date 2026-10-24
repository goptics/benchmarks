package sqlite

import (
	"fmt"
	"testing"
)

// checkpoints are the WAL checkpoint page counts every row runs at.
var checkpoints = []int{1000, 10000}

// BenchmarkSQLite enqueues and drains jobs through every subject. Names are
// <scenario>/ckpt<n>/<subject>.
func BenchmarkSQLite(b *testing.B) {
	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			for _, ckpt := range checkpoints {
				b.Run(fmt.Sprintf("ckpt%d", ckpt), func(b *testing.B) {
					for _, subj := range subjects {
						if !subj.runs(sc) {
							continue
						}

						b.Run(subj.name, func(b *testing.B) {
							runScenario(b, sc, subj, ckpt)
						})
					}
				})
			}
		})
	}
}
