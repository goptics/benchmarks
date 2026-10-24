# VarMQ Benchmarks

[![Live charts](https://img.shields.io/badge/%F0%9F%93%8A-Live_Charts-blue?style=for-the-badge)](https://benchmarks.goptics.org)

Benchmark suites for [VarMQ](https://github.com/goptics/varmq) against other Go worker pools and message queues. Raw `go test -bench` output is committed under `<suite>/bench/`; the charts at <https://benchmarks.goptics.org> are built from those files.

## Suites

| Suite | Compares | Charts | Details |
|-------|----------|--------|---------|
| Worker pool | VarMQ vs Pond, Ants, WorkerPool, Conc, ErrGroup, Gopool and two channel baselines | [24](https://benchmarks.goptics.org/?id=pool-24cpu), [8](https://benchmarks.goptics.org/?id=pool-8cpu), [4 CPUs](https://benchmarks.goptics.org/?id=pool-4cpu) | [pool/README.md](pool/README.md) |
| Message queue, Redis | VarMQ (redisq) vs Asynq, rmq, taskq, gocraft/work, with one and four consumer processes | [redis](https://benchmarks.goptics.org/?id=redis) | [redis/README.md](redis/README.md) |
| Message queue, SQLite | VarMQ (sqliteq) vs goqite, backlite, River, at WAL checkpoint 1000 and 10000 pages | [sqlite](https://benchmarks.goptics.org/?id=sqlite) | [sqlite/README.md](sqlite/README.md) |

## Results

Pool and Redis runs of 2026-10-24, SQLite run of 2026-10-25, all `-benchtime=2x`. Full tables in the suite READMEs.

- Pool: VarMQ is the fastest subject in 40, 36 and 20 of the 55 rows at 24, 8 and 4 CPUs. It is slower than the buffered-channel baseline on the one-worker `Loop10` rows.
- Redis: with the `Noop` task, VarMQ is first in 2 of the 6 one-process rows and VarMQDistributed in 3 of the 6 four-process rows, every win with 100 producers; gocraft/work leads the single-producer rows. Pickup latency p50: taskq 0.063 ms, VarMQ 0.142 ms, rmq 102.2 ms, Asynq 995.3 ms.
- SQLite: VarMQ (sqliteq) is first in all 16 rows: on `Mixed1Q` at ckpt1000 it runs 3.1x goqite, 4.8x River and 23x backlite, and ten queues in separate files reach 103,728 jobs/s at ckpt10000.

## Environment

Intel Core i7-13700 (24 threads), linux/amd64, Go 1.26.2. Pool suite at GOMAXPROCS 24, 8 and 4. Redis 7.2 (Debian image) pinned to CPU 14, client on the other cores. SQLite databases on ext4 over NVMe.

| Library | Version |
|---------|---------|
| varmq | 415ad73 (unreleased, `replace` in each go.mod) |
| redisq | v1.5.0 |
| sqliteq | v0.3.0 |
| pond | v2.7.2 |
| ants | v2.12.1 |
| workerpool | v1.2.2 |
| conc | v0.3.0 |
| x/sync (errgroup) | v0.24.0 |
| bytedance/gopkg (gopool) | v0.1.4 |
| asynq | v0.26.0 |
| rmq | v5.2.0 |
| taskq | v3.2.9 |
| gocraft/work | v0.5.1 |
| goqite | v0.4.0 |
| backlite | v0.8.0 |
| river | v0.49.0 |

## Running

Needs Go 1.26+ and [Task](https://taskfile.dev); Docker for the Redis suite; a C compiler for the SQLite suite; [vizb](https://github.com/goptics/vizb) and `jq` for `task site:preview`.

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
