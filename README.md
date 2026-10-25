# Benchmarks

[![Live charts](https://img.shields.io/badge/%F0%9F%93%8A-Live_Charts-blue?style=for-the-badge)](https://benchmarks.goptics.org)

Benchmarks of Go worker pools and job queues: in-process worker pools, Redis-backed queues and SQLite-backed queues. Raw `go test -bench` output is committed under `<suite>/bench/`, and the charts at <https://benchmarks.goptics.org> are built from it.

## Suites

| Suite | What runs | Libraries | Charts | Details |
|-------|-----------|-----------|--------|---------|
| Worker pool | 9 subjects (7 libraries plus two channel baselines); 5 workload shapes, 4 tasks, 4 pool sizes, 1M tasks per op; recorded at GOMAXPROCS 24, 8 and 4 | [ants](https://github.com/panjf2000/ants), [conc](https://github.com/sourcegraph/conc), errgroup ([x/sync](https://github.com/golang/sync)), gopool ([bytedance/gopkg](https://github.com/bytedance/gopkg)), [pond](https://github.com/alitto/pond), [VarMQ](https://github.com/goptics/varmq), [workerpool](https://github.com/gammazero/workerpool) | [pool-24cpu](https://benchmarks.goptics.org/?id=pool-24cpu), [pool-8cpu](https://benchmarks.goptics.org/?id=pool-8cpu), [pool-4cpu](https://benchmarks.goptics.org/?id=pool-4cpu) | [pool/README.md](pool/README.md) |
| Message queue, Redis | 6 subjects from 5 libraries; 100,000 jobs per op through 1 or 4 library instances in one process, plus enqueue-only and enqueue-to-handler latency | [Asynq](https://github.com/hibiken/asynq), [gocraft/work](https://github.com/gocraft/work), [rmq](https://github.com/adjust/rmq), [taskq](https://github.com/vmihailenco/taskq), [VarMQ](https://github.com/goptics/varmq) with [redisq](https://github.com/goptics/redisq) | [redis](https://benchmarks.goptics.org/?id=redis) | [redis/README.md](redis/README.md) |
| Message queue, SQLite | 5 subjects from 4 libraries; 4 scenarios at WAL checkpoint 1000 and 10000 pages | [backlite](https://github.com/mikestefanello/backlite), [goqite](https://github.com/maragudk/goqite), [River](https://github.com/riverqueue/river), [VarMQ](https://github.com/goptics/varmq) with [sqliteq](https://github.com/goptics/sqliteq) | [sqlite](https://benchmarks.goptics.org/?id=sqlite) | [sqlite/README.md](sqlite/README.md) |

## Environment

Intel Core i7-13700 (24 threads), linux/amd64, Go 1.26, SQLite databases on ext4 over NVMe.

The pool suite sets GOMAXPROCS to 24, 8 and 4 with no core pinning. The SQLite suite runs on all 24 threads. Redis 7.2 (Debian image) is pinned to CPU 14 and the Redis suite's client to CPUs 0-13 and 16-23 (22 threads), so Redis has a whole physical core.

| Library | Version | Suite |
|---------|---------|-------|
| [varmq](https://github.com/goptics/varmq) | v1.10.0 | pool, redis, sqlite |
| [ants](https://github.com/panjf2000/ants) | v2.12.1 | pool |
| [workerpool](https://github.com/gammazero/workerpool) | v1.2.2 | pool |
| [conc](https://github.com/sourcegraph/conc) | v0.3.0 | pool |
| [x/sync](https://github.com/golang/sync) (errgroup) | v0.24.0 | pool |
| [bytedance/gopkg](https://github.com/bytedance/gopkg) (gopool) | v0.1.4 | pool |
| [pond](https://github.com/alitto/pond) | v2.7.2 | pool |
| [redisq](https://github.com/goptics/redisq) | v1.5.0 | redis |
| [Asynq](https://github.com/hibiken/asynq) | v0.26.0 | redis |
| [rmq](https://github.com/adjust/rmq) | v5.2.0 | redis |
| [taskq](https://github.com/vmihailenco/taskq) | v3.2.9 | redis |
| [gocraft/work](https://github.com/gocraft/work) | v0.5.1 | redis |
| [go-redis](https://github.com/redis/go-redis) | v9.23.0 (taskq uses v8.11.5) | redis |
| [redigo](https://github.com/gomodule/redigo) | v1.9.3 | redis |
| [sqliteq](https://github.com/goptics/sqliteq) | v0.3.0 | sqlite |
| [goqite](https://github.com/maragudk/goqite) | v0.4.0 | sqlite |
| [backlite](https://github.com/mikestefanello/backlite) | v0.8.0 | sqlite |
| [River](https://github.com/riverqueue/river) with its [SQLite driver](https://github.com/riverqueue/river/tree/master/riverdriver/riversqlite) | v0.49.0 | sqlite |
| [go-sqlite3](https://github.com/mattn/go-sqlite3) | v1.14.28 | sqlite |

## Running

Needs Go 1.26+, [Task](https://taskfile.dev), Docker for the Redis suite, a C compiler for the SQLite suite, and [vizb](https://github.com/goptics/vizb) and `jq` for `task site:preview`.

varmq v1.10.0 is not tagged yet; each go.mod builds it from a checkout of [goptics/varmq](https://github.com/goptics/varmq) at `../../varmq` through a `replace` directive.

```bash
task --list                          # every task
task pool:bench                      # one iteration of every pool benchmark
task redis:up redis:bench redis:down
task sqlite:bench
task pool:txt                        # full run into pool/bench/ (hours); also redis:txt, sqlite:txt
task report                          # every suite, then the local preview
task site:preview                    # all datasets in one local HTML
```

Each suite README shows how to run one slice with `go test -bench`. Pushes to `main` rebuild and deploy the charts; see [worker/README.md](worker/README.md).

## License

MIT, see [LICENSE](LICENSE). The pool harness started from [alitto/pond-benchmarks](https://github.com/alitto/pond-benchmarks).
