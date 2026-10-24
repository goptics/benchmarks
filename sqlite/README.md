# SQLite message queue benchmarks

VarMQ with [sqliteq](https://github.com/goptics/sqliteq) against [goqite](https://github.com/maragudk/goqite), [backlite](https://github.com/mikestefanello/backlite) and [River](https://github.com/riverqueue/river)'s SQLite driver, end to end in one process.
Chart: https://benchmarks.goptics.org/?id=sqlite

## Results

Run of 2026-10-25: varmq 415ad73 (unreleased, `replace` in go.mod), sqliteq v0.3.0, goqite v0.4.0, backlite v0.8.0, River v0.49.0, `-benchtime=2x`, 32 rows in `bench/sqlite.txt`, ext4 over NVMe. jobs/s, higher is better; walMB is the peak WAL size, lower is better.

| Scenario | Subject | ckpt1000 | ckpt10000 | walMB 1000 / 10000 |
|----------|---------|---------:|----------:|-------------------:|
| `Mixed1Q` | VarMQ | 6,265 | 12,439 | 4.2 / 41.2 |
| `Mixed1Q` | Goqite | 2,031 | 3,848 | 4.1 / 41.2 |
| `Mixed1Q` | Backlite | 267 | 153 | 200.2 / 140.2 |
| `Mixed1Q` | River | 1,302 | 2,404 | 4.2 / 41.3 |
| `Drain1Q` | VarMQ | 15,185 | 24,581 | 4.2 / 41.2 |
| `Drain1Q` | Goqite | 2,754 | 4,051 | 4.1 / 41.2 |
| `Drain1Q` | Backlite | 1,141 | 1,492 | 179.8 / 152.5 |
| `Drain1Q` | River | 3,137 | 3,671 | 4.3 / 41.4 |
| `DrainPrio` | VarMQ | 13,614 | 20,566 | 4.2 / 41.2 |
| `DrainPrio` | Goqite | 2,880 | 3,962 | 4.1 / 41.2 |
| `DrainPrio` | River | 3,101 | 3,554 | 4.5 / 41.4 |
| `Mixed10Q` | VarMQ | 30,560 | 103,728 | 4.1 / 39.3 |
| `Mixed10Q` | VarMQSingleFile | 4,445 | 11,226 | 4.2 / 41.3 |
| `Mixed10Q` | Goqite | 2,152 | 4,449 | 4.1 / 41.2 |
| `Mixed10Q` | Backlite | 4,864 | 5,362 | 367.9 / 369.9 |
| `Mixed10Q` | River | 660 | 1,105 | 4.3 / 41.3 |

VarMQ is first in all 16 rows; on `Mixed1Q` at ckpt1000 it runs 3.1x goqite, 4.8x River and 23x backlite. A 10,000-page checkpoint is 1.1 to 3.4x faster than 1,000 for VarMQ, goqite and River at a WAL about ten times larger (39.3 to 41.4 MB against 4.1 to 4.5 MB); ten queues in separate files (VarMQ) run 6.9x and 9.2x the same ten in one file (VarMQSingleFile). Backlite's WAL peaks at 140 to 370 MB whatever the checkpoint size, and on `Mixed1Q` it is slower at the larger one.
Absolute numbers vary between runs on the same machine; compare ratios within one run.

## Subjects

| Subject | Library | Version | SQLite connections | Consumer |
|---------|---------|---------|--------------------|----------|
| VarMQ | sqliteq through a varmq worker | sqliteq v0.3.0, varmq 415ad73 | pool of 8 per file (sqliteq) | one database file per queue (sqliteq's default), 16 concurrent jobs per queue |
| VarMQSingleFile | same | same | same | every queue in one file (`WithSingleFile`); `Mixed10Q` only |
| Goqite | maragu.dev/goqite with its `jobs` runner | v0.4.0 | one (its README example) | one runner per queue, limit 16, 100 ms poll between empty receives |
| Backlite | mikestefanello/backlite | v0.8.0 | default pool, unbounded | one client, 16 workers per queue, dispatcher woken in-process on every add |
| River | riverqueue/river with riverdriver/riversqlite | v0.49.0 | one (its driver doc) | one client, 16 workers per queue, 1 ms fetch cooldown |

Every subject opens SQLite with WAL, synchronous NORMAL, a 5 s busy timeout and the row's `wal_autocheckpoint`; everything else is the library default. The one forced setting is River's `FetchCooldown`: River fetches at most `MaxWorkers` jobs per cooldown, so its 100 ms default caps 16 workers at 160 jobs/s whatever the database does (measured 159); the 1 ms minimum lets SQLite set the number.

## Scenarios

Every job is an `int`; the handler does nothing.

| Name | Queues | Producers per queue | Jobs | Timed | Subjects |
|------|-------:|--------------------:|-----:|-------|----------|
| `Mixed1Q` | 1 | 4 | 20,000 | first add to last job handled, producing and consuming at once | all |
| `Drain1Q` | 1 | 4 | 20,000 | everything added first, then consumer start to last job handled | all |
| `DrainPrio` | 1 | 4 | 10,000 | as `Drain1Q` on a priority queue, priority `i % 4` | VarMQ, Goqite, River |
| `Mixed10Q` | 10 | 1 | 20,000 | as `Mixed1Q` over ten queues, 16 workers each | all, plus VarMQSingleFile |

- Rows are named `BenchmarkSQLite/<scenario>/ckpt<n>/<subject>`.
- `ckpt<n>` is the WAL checkpoint size in pages, 1000 (SQLite's default) or 10000, set on every connection.
- `jobs/s` is the job count divided by the timed part; setup, teardown and the checks below are untimed.
- `walMB` is the largest size any `-wal` file in the run's directory reached, sampled every 50 ms, untimed phases included.
- A run fails if any add failed, the handler did not run exactly once per job, or rows remain in the queue tables.

## Running

Needs Go 1.26 and a C compiler.

```sh
task sqlite:test    # vet, then every subject through a small job set
task sqlite:bench   # every scenario and subject once
task sqlite:txt     # full run into bench/sqlite.txt (about 20 min)
```

One slice:

```sh
go test -run xxx -bench 'SQLite/Mixed10Q' -benchtime=1x
```

`SQLITEQ_BENCH_DIR` is where the databases are created (default `<user cache dir>/varmq-bench-sqlite`). Use a real disk: durable writes pay for fsync, and `/tmp` is often a tmpfs where that cost vanishes.
