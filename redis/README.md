# Redis message queue benchmarks

VarMQ (redisq) against Asynq, rmq, taskq and gocraft/work on Redis.
Chart: https://benchmarks.goptics.org/?id=redis

## Results

Run of 2026-10-24: varmq 415ad73 (unreleased, `replace` in go.mod), redisq v1.5.0, `-benchtime=2x`, Redis 7.2 (Debian image) pinned to one core, 150 rows in `bench/redis.txt`.

`C1`: jobs/s, `Noop` task, one consumer process.

| Row | VarMQ | Asynq | Rmq | Taskq | GocraftWork |
|---|---:|---:|---:|---:|---:|
| NoopW10C1 / 1u-100Kt | 21,459 | 15,988 | 6,195 | 18,963 | 25,057 |
| NoopW10C1 / 100u-1Kt | 41,972 | 18,030 | 5,933 | 35,648 | 49,503 |
| NoopW100C1 / 1u-100Kt | 20,859 | 16,978 | 6,129 | 18,439 | 25,315 |
| NoopW100C1 / 100u-1Kt | 60,551 | 19,828 | 5,859 | 35,560 | 45,110 |
| NoopW1000C1 / 1u-100Kt | 21,027 | 15,484 | 10,271 | 16,939 | 24,912 |
| NoopW1000C1 / 100u-1Kt | 59,182 | 18,693 | 9,317 | 28,531 | 40,710 |

`C4`: jobs/s, `Noop` task, four consumer processes.

| Row | VarMQDistributed | Asynq | Rmq | Taskq | GocraftWork |
|---|---:|---:|---:|---:|---:|
| NoopW10C4 / 1u-100Kt | 21,946 | 13,057 | 21,816 | 10,968 | 25,650 |
| NoopW10C4 / 100u-1Kt | 55,294 | 19,377 | 16,459 | 17,191 | 48,839 |
| NoopW100C4 / 1u-100Kt | 22,484 | 13,135 | 22,400 | 10,629 | 25,512 |
| NoopW100C4 / 100u-1Kt | 65,606 | 21,198 | 16,889 | 16,018 | 43,263 |
| NoopW1000C4 / 1u-100Kt | 22,347 | 14,519 | 23,041 | 7,663 | 19,542 |
| NoopW1000C4 / 100u-1Kt | 64,993 | 16,488 | 25,137 | 11,377 | 32,672 |

`Sleep1ms` rows are bound by workers x 1 ms and are only on the chart.

Enqueue only (`BenchmarkEnqueue`, jobs/s): VarMQ 42,867 at one producer, the best of the six; at 100 producers Rmq 202,068, GocraftWork 201,297 and VarMQ 201,078 tie, and VarMQDistributed trails at 143,183.

Pickup latency (`BenchmarkLatency`, 1,000 jobs one at a time, ms):

| Subject | p50 | p99 | max |
|---|---|---|---|
| Taskq | 0.06330 | 0.1897 | 0.6119 |
| GocraftWork | 0.09234 | 0.4743 | 10.80 |
| VarMQ | 0.1425 | 0.3843 | 3.583 |
| VarMQDistributed | 0.1457 | 0.5407 | 2.183 |
| Rmq | 102.2 | 112.3 | 113.3 |
| Asynq | 995.3 | 1492 | 1507 |

Rmq's 100 ms poll and Asynq's 0.5 to 1.5 s idle sleep show directly. VarMQ's
batch pop p50 is 0.1425 ms against Taskq's 0.06330 ms for its blocking `XREADGROUP`.

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

Each subject runs at its library defaults except worker concurrency and the Redis pool size, `max(workers, 10*GOMAXPROCS)`; the settings a library forces are listed in [NOTES.md](NOTES.md).

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
