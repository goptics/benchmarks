# Redis message queue benchmarks

VarMQ (redisq) against Asynq, rmq, taskq and gocraft/work on Redis.
Chart: https://benchmarks.goptics.org/?id=redis

## Results

Run of 2026-10-25: varmq 415ad73 (unreleased, `replace` in go.mod), redisq v1.5.0, `-benchtime=2x`, Redis 7.2 (Debian image) pinned to one core, 150 rows in `bench/redis.txt`.

`C1`: jobs/s, `Noop` task, one consumer process.

| Row | VarMQ | Asynq | Rmq | Taskq | GocraftWork |
|---|---:|---:|---:|---:|---:|
| NoopW10C1 / 1u-100Kt | 21,108 | 15,390 | 33,195 | 18,853 | 24,935 |
| NoopW10C1 / 100u-1Kt | 41,970 | 17,604 | 28,880 | 34,628 | 49,213 |
| NoopW100C1 / 1u-100Kt | 20,665 | 17,048 | 33,439 | 18,352 | 24,990 |
| NoopW100C1 / 100u-1Kt | 60,025 | 19,439 | 29,022 | 34,578 | 44,659 |
| NoopW1000C1 / 1u-100Kt | 19,710 | 16,467 | 32,678 | 16,674 | 24,408 |
| NoopW1000C1 / 100u-1Kt | 58,523 | 17,932 | 27,349 | 28,638 | 40,679 |

`C4`: jobs/s, `Noop` task, four consumer processes.

| Row | VarMQDistributed | Asynq | Rmq | Taskq | GocraftWork |
|---|---:|---:|---:|---:|---:|
| NoopW10C4 / 1u-100Kt | 22,057 | 11,896 | 29,616 | 10,904 | 24,339 |
| NoopW10C4 / 100u-1Kt | 53,754 | 20,889 | 59,649 | 17,323 | 48,722 |
| NoopW100C4 / 1u-100Kt | 22,263 | 13,350 | 29,953 | 10,576 | 23,683 |
| NoopW100C4 / 100u-1Kt | 63,710 | 22,677 | 58,620 | 16,077 | 41,730 |
| NoopW1000C4 / 1u-100Kt | 22,231 | 12,747 | 27,784 | 8,267 | 21,068 |
| NoopW1000C4 / 100u-1Kt | 64,281 | 19,119 | 52,549 | 11,100 | 36,809 |

`Sleep1ms` rows are bound by workers x 1 ms and are only on the chart.

Enqueue only (`BenchmarkEnqueue`, jobs/s): at one producer VarMQ 37,292, Rmq 33,394, GocraftWork 29,720, VarMQDistributed 27,183, Taskq 26,780, Asynq 24,023; at 100 producers VarMQ 207,752, GocraftWork 202,835, Rmq 197,240, Taskq 188,437, VarMQDistributed 144,838, Asynq 121,046.

Pickup latency (`BenchmarkLatency`, 1,000 jobs one at a time, ms):

| Subject | p50 | p99 | max |
|---|---|---|---|
| Taskq | 0.03876 | 0.1623 | 3.195 |
| VarMQDistributed | 0.06492 | 0.2351 | 2.436 |
| VarMQ | 0.09407 | 0.5321 | 3.431 |
| GocraftWork | 0.1274 | 0.5047 | 10.94 |
| Rmq | 1.464 | 2.891 | 6.902 |
| Asynq | 979.7 | 1491 | 1501 |

Asynq's 0.5 to 1.5 s idle sleep shows directly, and rmq's 1 ms poll puts it at 1.464 ms. Taskq's blocking `XREADGROUP` is the fastest pickup at 0.03876 ms; VarMQ's batch pop is at 0.09407 ms and VarMQDistributed's push wake at 0.06492 ms.

In `C4` rows the four instances are stopped one after another inside the timed region, so Asynq and Taskq pay their idle poll up to four times.

Full run: `bench/redis.txt`.
On the chart, the C1, Enqueue and Latency groups show VarMQ and the four rivals; the C4 groups show VarMQDistributed and the four.

## Subjects

| Subject (chart name) | Library | Version | Instances | Wake | Consumer round trips per job |
|---|---|---|---|---|---|
| VarMQ | goptics/varmq with goptics/redisq | varmq 415ad73 (unreleased, `replace` in go.mod), redisq v1.5.0 | 1 | push in-process | 1 ack pipeline per job + 1 pop script per up to 64 jobs |
| VarMQDistributed | goptics/varmq with goptics/redisq | varmq 415ad73 (unreleased, `replace` in go.mod), redisq v1.5.0 | 1 and 4 | push to one idle consumer plus a 1 s nudge | 1 ack pipeline per job + 1 pop script per up to 64 jobs |
| Asynq | hibiken/asynq | v0.26.0 | 1 and 4 | poll 0.5-1.5 s | 2 |
| Rmq | adjust/rmq/v5 | v5.2.0 | 1 and 4 | poll 1 ms | 2 |
| Taskq | vmihailenco/taskq/v3 | v3.2.9 with go-redis v8 v8.11.5 | 1 and 4 | blocking XREADGROUP | 2 + 1 per 10 jobs |
| GocraftWork | gocraft/work | v0.5.1 with redigo v1.9.3 | 1 and 4 | backoff poll 10 ms to 5 s | 2 |

What each library does per job, read from source: [NOTES.md](NOTES.md).

## Matrix

| Benchmark | Varies | Metric |
|---|---|---|
| Durable | tasks `Noop`, `Sleep1ms`; workers 10, 100, 1000; instances 1, 4; workloads `1u-100Kt` (1 producer x 100,000 jobs), `100u-1Kt` (100 x 1,000) | jobs/s |
| Enqueue | workloads `1u-100Kt`, `100u-1Kt` | jobs/s |
| Latency | `1u-1Kt`, W10 C1 | p50-ms, p99-ms, max-ms |

Names: `BenchmarkDurable/<Task>W<workers>C<instances>/<workload>/<subject>`, `BenchmarkEnqueue/Enqueue/<workload>/<subject>`, `BenchmarkLatency/LatencyW10C1/1u-1Kt/<subject>`.

Each subject runs at its library defaults except worker concurrency and the Redis pool size, `max(workers, 10*GOMAXPROCS)`; the settings a library forces are listed in [NOTES.md](NOTES.md). rmq has no default for its poll duration or prefetch limit; the harness uses 1 ms and `max(1000, 2*workers)` (the example's 100 ms capped a connection near prefetch / 0.1 s), see [NOTES.md](NOTES.md).

VarMQ's persistent queue runs one instance, so it has no `C4` rows.

## Running

```sh
task redis:up      # Redis 7.2 (Debian image) on localhost:6390, pinned to CPU 14, no volume
task redis:test    # smoke tests
task redis:bench   # every benchmark once (-benchtime=1x)
task redis:txt     # full run into bench/redis.txt (hours)
task redis:down
```

One slice, after `task redis:up`:

```sh
go test -run xxx -bench 'Durable/NoopW10C1' -benchtime=1x -timeout 0
```

Set `REDIS_URL` to use another Redis (default `redis://localhost:6390/0`). The
tests and benchmarks skip when it isn't reachable. `FLUSHDB` runs before every
iteration, so don't point it at a Redis you care about.

`task redis:txt` runs the client with `taskset -c 0-13,16-23`, so Redis keeps core 14; compare runs on this setup only.
