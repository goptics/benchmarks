# SQLite job queue benchmarks

Five subjects from four Go job-queue libraries backed by SQLite: [backlite](https://github.com/mikestefanello/backlite), [goqite](https://github.com/maragudk/goqite), [River](https://github.com/riverqueue/river) through its SQLite driver [riversqlite](https://github.com/riverqueue/river/tree/master/riverdriver/riversqlite), and [varmq](https://github.com/goptics/varmq) with [sqliteq](https://github.com/goptics/sqliteq), run as two subjects: VarMQ, with one database file per queue, and VarMQSingleFile, with every queue in one file.
Every subject runs end to end in one process, and all of them reach SQLite through [go-sqlite3](https://github.com/mattn/go-sqlite3).
Chart: https://benchmarks.goptics.org/?id=sqlite

## Results

Run of 2026-10-25: backlite v0.8.0, go-sqlite3 v1.14.28, goqite v0.4.0, River v0.49.0, sqliteq v0.3.0, varmq v1.10.0, `-benchtime=2x`, 32 rows in `bench/sqlite.txt`, ext4 over NVMe.
Higher jobs/s is better; lower walMB (the peak total size of the run's WAL files) is better.

| Scenario | Subject | ckpt1000 | ckpt10000 | walMB 1000 / 10000 |
|----------|---------|---------:|----------:|-------------------:|
| `Mixed1Q` | VarMQ | 8,575 | 11,860 | 4.2 / 41.2 |
| `Mixed1Q` | Goqite | 3,032 | 7,091 | 4.1 / 41.2 |
| `Mixed1Q` | Backlite | 924 | 294 | 197.4 / 204.9 |
| `Mixed1Q` | River | 1,375 | 2,115 | 4.2 / 41.3 |
| `Drain1Q` | VarMQ | 14,725 | 21,225 | 4.1 / 41.2 |
| `Drain1Q` | Goqite | 2,933 | 4,921 | 4.2 / 41.2 |
| `Drain1Q` | Backlite | 1,119 | 1,016 | 182.2 / 162.4 |
| `Drain1Q` | River | 3,230 | 3,719 | 4.2 / 41.3 |
| `DrainPrio` | VarMQ | 12,937 | 25,608 | 4.2 / 41.2 |
| `DrainPrio` | Goqite | 2,908 | 4,828 | 4.1 / 41.2 |
| `DrainPrio` | River | 3,240 | 3,625 | 4.5 / 41.4 |
| `Mixed10Q` | VarMQ | 29,624 | 104,471 | 41.4 / 386.8 |
| `Mixed10Q` | VarMQSingleFile | 4,553 | 14,110 | 4.2 / 41.3 |
| `Mixed10Q` | Goqite | 2,131 | 4,668 | 4.1 / 41.2 |
| `Mixed10Q` | Backlite | 5,326 | 5,419 | 246.5 / 369.4 |
| `Mixed10Q` | River | 1,431 | 2,175 | 4.4 / 41.2 |

VarMQ is the fastest subject in all 8 scenario and checkpoint groups.
On `Mixed1Q` the rest follow in the order Goqite, River, Backlite at both checkpoint sizes.
On `Drain1Q` and `DrainPrio`, River is ahead of Goqite at ckpt1000 and Goqite is ahead of River at ckpt10000; Backlite is last on `Drain1Q` at both.
On `Mixed10Q` the order after VarMQ is Backlite, VarMQSingleFile, Goqite, River at ckpt1000 and VarMQSingleFile, Backlite, Goqite, River at ckpt10000.
At ckpt10000, jobs/s is 1.1 to 3.5 times the ckpt1000 figure for Goqite, River, VarMQ and VarMQSingleFile in every scenario.
For Backlite, jobs/s falls from 924 to 294 on `Mixed1Q` and from 1,119 to 1,016 on `Drain1Q`, and rises from 5,326 to 5,419 on `Mixed10Q`.
In every row with one database file except Backlite's, walMB is 4.1 to 4.5 MB at ckpt1000 and 41.2 to 41.4 MB at ckpt10000.
Backlite's WAL peaks at 182.2 to 246.5 MB at ckpt1000 and 162.4 to 369.4 MB at ckpt10000.
On `Mixed10Q`, ten queues in ten files (VarMQ) reach 6.5 times the jobs/s of the same ten queues in one file (VarMQSingleFile) at ckpt1000 and 7.4 times at ckpt10000.
The ten WAL files total 41.4 and 386.8 MB, against 4.2 and 41.3 MB for the single file.
Absolute numbers vary between runs on the same machine; compare ratios within one run.

## Subjects

| Subject | Library | Version | Database files | SQLite connections | Stored payload | Consumer |
|---------|---------|---------|----------------|--------------------|----------------|----------|
| VarMQ | [varmq](https://github.com/goptics/varmq) with [sqliteq](https://github.com/goptics/sqliteq) | varmq v1.10.0, sqliteq v0.3.0 | one per queue (sqliteq's default) | pool of 8 per file | varmq's job JSON (`id`, `status`, `data`) | one varmq worker per queue, 16 concurrent jobs |
| VarMQSingleFile | [varmq](https://github.com/goptics/varmq) with [sqliteq](https://github.com/goptics/sqliteq) | varmq v1.10.0, sqliteq v0.3.0 | one `bench.db` for every queue (`WithSingleFile`) | pool of 8 | as VarMQ | as VarMQ; `Mixed10Q` only |
| Goqite | [goqite](https://github.com/maragudk/goqite) with its `jobs` runner | v0.4.0 | one `bench.db` | one (its README's setup) | the int's decimal bytes inside goqite's gob job envelope | one runner per queue, limit 16; goqite's default 100 ms poll between empty receives and while all 16 slots are busy |
| Backlite | [backlite](https://github.com/mikestefanello/backlite) | v0.8.0 | one `bench.db` | default pool, unbounded | JSON `{"N": i}` | one client with 16 x queues workers shared by every queue; dispatcher woken in process on every add |
| River | [River](https://github.com/riverqueue/river) with [riversqlite](https://github.com/riverqueue/river/tree/master/riverdriver/riversqlite) | v0.49.0 | one `bench.db` | one (its driver doc) | JSON `{"n": i}` | one client, 16 workers per queue, per-queue `FetchCooldown` 1 ms; completions written in batches by its completer |

Every subject opens SQLite with WAL, synchronous NORMAL, a 5 s busy timeout and the row's `wal_autocheckpoint`, through the same DSN and connect hook as sqliteq.
Beyond the table, everything is the library default except three settings.
Backlite's `ReleaseAfter` is 10 minutes, since it has no default.
River's per-queue `FetchCooldown` is 1 ms: a queue fetches at most `MaxWorkers` jobs per cooldown, so the 100 ms default caps 16 workers at 160 jobs/s whatever the database does.
River's client-level `FetchCooldown`, which also paces its insert notifications, stays at its default.
River's logger is discarded.

## Scenarios

| Name | Queues | Producers per queue | Jobs (total) | Per queue | Timed | Subjects |
|------|-------:|--------------------:|-------------:|-----------|-------|----------|
| `Mixed1Q` | 1 | 4 | 20,000 | 20,000 (5,000 per producer) | mixed | all but VarMQSingleFile |
| `Drain1Q` | 1 | 4 | 20,000 | 20,000 (5,000 per producer) | drain | all but VarMQSingleFile |
| `DrainPrio` | 1 | 4 | 10,000 | 10,000 (2,500 per producer) | drain | VarMQ, Goqite, River |
| `Mixed10Q` | 10 | 1 | 20,000 | 2,000 | mixed | all five |

- Mixed: the consumers start before the timer, which runs from the first add to the last job acknowledged, producing and consuming at once.
- Drain: every job is added first, then the timer runs from consumer start to the last job acknowledged, so each subject's start-up is timed (backlite `Start`, goqite's runner goroutines, River `client.Start`, varmq `Worker.Start`).
- Every job is an `int` id `i`, unique across all queues: queue x per-queue + producer x per-producer + j, with queue, producer and that producer's add j all counted from 0.
- Every subject's handler only records `i` in a shared tally, one counter per job.
- Priority (`DrainPrio`): Goqite, River and VarMQ run it, and each computes a job's priority from its id, mapping `i % 4 == 0` to its highest priority: goqite takes higher numbers first (`3 - i % 4`), River takes 1 as highest (`i % 4 + 1`), varmq takes lower numbers first (`i % 4`). The order in which jobs are served is not checked.
- Rows are named `BenchmarkSQLite/<scenario>/ckpt<n>/<subject>`; `ckpt<n>` is the WAL checkpoint size in pages, 1000 (SQLite's default) or 10000, set on every connection.
- jobs/s = jobs x iterations / timed seconds.
- walMB = the largest total size of all `-wal` files in the run's directory, sampled every 50 ms from before the store opens until just before it closes, the maximum over the row's iterations. SQLite does not shrink a WAL file after a checkpoint, so this is the files' high-water size.
- A row fails on an open or schema error, a failed add, a start error, the jobs not all acknowledged within 5 minutes, rows left in the queue tables (River counts jobs not in state `completed`, since it keeps completed rows), or a close error.
- A row also fails if any job is delivered twice, an id the run never added arrives, or any job is never handled.
- The tally is checked after the last job and again after close, so a redelivery while stopping counts too.

## Running

Needs Go 1.26 and a C compiler.

```sh
task sqlite:test    # vet, then every subject through a small job set
task sqlite:bench   # every scenario and subject once
task sqlite:txt     # full run into bench/sqlite.txt
```

One slice:

```sh
go test -run xxx -bench 'SQLite/Mixed10Q' -benchtime=1x
```

`SQLITEQ_BENCH_DIR` is where the databases are created (default `<user cache dir>/varmq-bench-sqlite`).
Use a real disk: durable writes pay for fsync, and `/tmp` is often a tmpfs where that cost vanishes.
