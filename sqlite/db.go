package sqlite

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	"github.com/mattn/go-sqlite3"
)

// dsn carries sqliteq's connection settings, so every subject pays the same
// journal, fsync and lock-wait costs and the chart compares queue code.
func dsn(path string) string {
	return path + "?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000"
}

var (
	driversMu sync.Mutex
	drivers   = map[int]bool{}
)

// driverName registers, once per checkpoint size, a sqlite3 driver whose
// connections start with PRAGMA wal_autocheckpoint, as sqliteq does for its own.
func driverName(ckpt int) string {
	name := fmt.Sprintf("bench_sqlite3_ckpt%d", ckpt)

	driversMu.Lock()
	defer driversMu.Unlock()

	if !drivers[ckpt] {
		pragma := fmt.Sprintf("PRAGMA wal_autocheckpoint=%d", ckpt)
		sql.Register(name, &sqlite3.SQLiteDriver{
			ConnectHook: func(c *sqlite3.SQLiteConn) error {
				_, err := c.Exec(pragma, nil)

				return err
			},
		})
		drivers[ckpt] = true
	}

	return name
}

// openDB opens the database file at path for a rival subject.
func openDB(tb testing.TB, path string, ckpt int) *sql.DB {
	tb.Helper()

	db, err := sql.Open(driverName(ckpt), dsn(path))
	if err != nil {
		tb.Fatalf("open %s: %v", path, err)
	}

	return db
}

// count runs a COUNT(*) query.
func count(db *sql.DB, query string) (int, error) {
	var n int
	err := db.QueryRow(query).Scan(&n)

	return n, err
}
