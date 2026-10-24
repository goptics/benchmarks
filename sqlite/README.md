# SQLite message queue benchmarks

VarMQ with [sqliteq](https://github.com/goptics/sqliteq) against [goqite](https://github.com/maragudk/goqite), [backlite](https://github.com/mikestefanello/backlite) and [River](https://github.com/riverqueue/river)'s SQLite driver, end to end in one process.
Chart: https://benchmarks.goptics.org/?id=sqlite

## Results

Run of 2026-10-25: varmq 415ad73 (unreleased, `replace` in go.mod), sqliteq v0.3.0, goqite v0.4.0, backlite v0.8.0, River v0.49.0, `-benchtime=2x`, 32 rows in `bench/sqlite.txt`, ext4 over NVMe. jobs/s, higher is better; walMB is the peak total size of the run's WAL files, lower is better.

| Scenario | Subject | ckpt1000 | ckpt10000 | walMB 1000 / 10000 |
|----------|---------|---------:|----------:|-------------------:|
| `Mixed1Q` | VarMQ | 6,245 | 13,906 | 4.2 / 41.2 |
| `Mixed1Q` | Goqite | 2,039 | 3,682 | 4.1 / 41.2 |
| `Mixed1Q` | Backlite | 266 | 926 | 113.8 / 182.3 |
| `Mixed1Q` | River | 1,523 | 2,283 | 4.2 / 41.3 |
| `Drain1Q` | VarMQ | 14,923 | 25,420 | 4.1 / 41.2 |
| `Drain1Q` | Goqite | 2,834 | 4,229 | 4.2 / 41.2 |
| `Drain1Q` | Backlite | 1,286 | 1,168 | 126.3 / 166.9 |
| `Drain1Q` | River | 3,252 | 3,774 | 4.3 / 41.2 |
| `DrainPrio` | VarMQ | 14,044 | 30,925 | 4.2 / 41.2 |
| `DrainPrio` | Goqite | 2,862 | 4,052 | 4.1 / 41.2 |
| `DrainPrio` | River | 3,083 | 3,584 | 4.7 / 41.2 |
| `Mixed10Q` | VarMQ | 28,971 | 107,459 | 41.3 / 385.4 |
| `Mixed10Q` | VarMQSingleFile | 4,759 | 10,902 | 4.2 / 41.3 |
| `Mixed10Q` | Goqite | 2,187 | 5,048 | 4.1 / 41.2 |
| `Mixed10Q` | Backlite | 4,807 | 5,781 | 352.9 / 369.6 |
| `Mixed10Q` | River | 1,383 | 2,173 | 4.3 / 41.2 |

VarMQ is first in all 8 scenario and checkpoint groups; on `Mixed1Q` at ckpt1000 it runs 3.1x goqite, 4.1x River and 23x backlite. A 10,000-page checkpoint is 1.2 to 3.7x faster than 1,000 for VarMQ, goqite and River and holds a WAL 9 to 10 times larger (4.2 MB against 41.2 MB per file). Ten queues in separate files (VarMQ) run 6.1x and 9.9x the same ten in one file (VarMQSingleFile), with ten WAL files instead of one: 41.3 and 385.4 MB in total against 4.2 and 41.3. Backlite's WAL peaks at 114 to 370 MB whatever the checkpoint size and on `Drain1Q` it is slower at the larger one.
Absolute numbers vary between runs on the same machine; compare ratios within one run.

## Subjects

| Subject | Library | Version | SQLite connections | Consumer |
|---------|---------|---------|--------------------|----------|
| VarMQ | sqliteq through a varmq worker | sqliteq v0.3.0, varmq 415ad73 | pool of 8 per file (sqliteq) | one database file per queue (sqliteq's default), 16 concurrent jobs per queue |
| VarMQSingleFile | same | same | same | every queue in one file (`WithSingleFile`); `Mixed10Q` only |
| Goqite | maragu.dev/goqite with its `jobs` runner | v0.4.0 | one (its README example) | one runner per queue, limit 16, 100 ms poll between empty receives |
| Backlite | mikestefanello/backlite | v0.8.0 | default pool, unbounded | one client with 16 x queues workers shared by every queue, dispatcher woken in-process on every add |
| River | riverqueue/river with riverdriver/riversqlite | v0.49.0 | one (its driver doc) | one client, 16 workers per queue, 1 ms fetch cooldown per queue, completions written in batches by its completer |

Every subject opens SQLite with WAL, synchronous NORMAL, a 5 s busy timeout and the row's `wal_autocheckpoint`. Everything else is the library default, apart from three settings: River's per-queue `FetchCooldown` is its 1 ms minimum, since a queue's producer fetches at most `MaxWorkers` jobs per cooldown and the 100 ms default caps 16 workers at 160 jobs/s whatever the database does (the client-level value, which also paces River's insert notifications, stays at the default); River's logger is discarded; backlite's `ReleaseAfter`, which has no default, is 10 minutes.

## Scenarios

Every job is an `int`; the handler does nothing.

| Name | Queues | Producers per queue | Jobs | Timed | Subjects |
|------|-------:|--------------------:|-----:|-------|----------|
| `Mixed1Q` | 1 | 4 | 20,000 | first add to last job acknowledged, producing and consuming at once | all |
| `Drain1Q` | 1 | 4 | 20,000 | everything added first, then consumer start (River's client start-up included) to last job acknowledged | all |
| `DrainPrio` | 1 | 4 | 10,000 | as `Drain1Q` on a priority queue, priority `i % 4` | VarMQ, Goqite, River |
| `Mixed10Q` | 10 | 1 | 20,000 | as `Mixed1Q` over ten queues, 16 workers each | all, plus VarMQSingleFile |

- Rows are named `BenchmarkSQLite/<scenario>/ckpt<n>/<subject>`.
- `ckpt<n>` is the WAL checkpoint size in pages, 1000 (SQLite's default) or 10000, set on every connection.
- `jobs/s` is the job count divided by the timed part; setup, teardown and the checks below are untimed.
- `walMB` is the largest total size the `-wal` files in the run's directory reached, sampled every 50 ms from before the store opens, untimed phases included.
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
