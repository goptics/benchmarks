# Redis job queue benchmarks

Six subjects from five Go job-queue libraries backed by Redis: [Asynq](https://github.com/hibiken/asynq), [gocraft/work](https://github.com/gocraft/work), [rmq](https://github.com/adjust/rmq), [taskq](https://github.com/vmihailenco/taskq) and VarMQ ([varmq](https://github.com/goptics/varmq) with [redisq](https://github.com/goptics/redisq): a single-instance persistent queue and a distributed queue).
Chart: https://benchmarks.goptics.org/?id=redis

## Workloads

- One op of `BenchmarkDurable` moves 100,000 jobs through freshly built clients. The timed region covers building every consumer instance and, where it is separate, the producer client; the Redis connection dials; every enqueue; draining all 100,000 jobs; and stopping everything. The `FLUSHDB` before each iteration is not timed. Nothing is reused between iterations.
- Payload: JSON `{"id": N, "t": <enqueue time in ns>, "pad": <48 zero bytes as a JSON array>}`, 136 to 140 bytes. Each library wraps it in its own envelope (see [NOTES.md](NOTES.md)).
- `1u-100Kt` is 1 producer goroutine enqueuing 100,000 jobs one after another. `100u-1Kt` is 100 producer goroutines enqueuing 1,000 each, concurrently. All producers share one producer client. For `VarMQ` the producer is consumer instance 0's own queue, because its persistent queue wakes its worker in process.
- `W<n>` is the workers per instance, passed as each library's concurrency setting.
- `C<n>` is the number of library instances inside the same `go test` process, each with its own client and Redis connection pool. `C4` means four instances and 4 x W workers. They are not separate OS processes.
- `Noop` runs nothing in the handler beyond the harness's duplicate check. `Sleep1ms` sleeps 1 ms, so a row is capped at C x W x 1,000 jobs/s before setup and teardown are counted.
- `VarMQ` runs `C1` only, because its persistent queue is single-instance. `Asynq`, `GocraftWork`, `Rmq`, `Taskq` and `VarMQDistributed` run `C1` and `C4`. On the chart the `C1`, Enqueue and Latency groups leave `VarMQDistributed` out; its rows for them are in `bench/redis.txt`.
- `BenchmarkEnqueue` runs the producer only, with no consumers, on the same two workloads and payload. It is timed from building the producer to its close.
- `BenchmarkLatency` measures enqueue-to-handler latency: 1,000 jobs, one at a time, on one instance with 10 workers. Job i+1 is enqueued only after job i reached the handler, with a 10 s limit per job. Each sample is the time from just before the enqueue call to handler entry, so it includes the enqueue round trip. p50, p99 and max are over all samples of the row's iterations: 2,000 samples at `-benchtime=2x`.
- A `Durable` row fails if the 100,000 unique jobs are not all handled within 5 minutes, an enqueue fails, or any job is delivered more than once. Duplicates are counted after every consumer instance has stopped, so a redelivery during shutdown counts too. A `Latency` row fails on any delivery of a job other than the one just enqueued. `TestSubjects` (`task redis:test`) runs the same checks.
- `jobs/s` = jobs x iterations / timed seconds.

## Results

Run of 2026-10-25: varmq v1.10.0, redisq v1.5.0, `-benchtime=2x`, Redis 7.2 pinned to one core, 150 rows in `bench/redis.txt`.

`C1`: jobs/s, `Noop` task, one library instance.

| Row | VarMQ | Asynq | Rmq | Taskq | GocraftWork |
|---|---:|---:|---:|---:|---:|
| NoopW10C1 / 1u-100Kt | 21,968 | 15,806 | 35,129 | 19,031 | 25,428 |
| NoopW10C1 / 100u-1Kt | 41,816 | 17,823 | 29,616 | 34,434 | 48,917 |
| NoopW100C1 / 1u-100Kt | 20,812 | 15,096 | 35,708 | 18,448 | 25,731 |
| NoopW100C1 / 100u-1Kt | 59,898 | 17,744 | 28,521 | 34,499 | 44,775 |
| NoopW1000C1 / 1u-100Kt | 21,157 | 17,710 | 34,621 | 16,351 | 25,147 |
| NoopW1000C1 / 100u-1Kt | 58,308 | 18,954 | 30,010 | 27,197 | 40,255 |

Rmq leads every single-producer row; at 100 producers GocraftWork leads at W10 and VarMQ at W100 and W1000.

`C4`: jobs/s, `Noop` task, four library instances.

| Row | Asynq | VarMQDistributed | Rmq | Taskq | GocraftWork |
|---|---:|---:|---:|---:|---:|
| NoopW10C4 / 1u-100Kt | 12,961 | 22,270 | 31,231 | 11,085 | 25,078 |
| NoopW10C4 / 100u-1Kt | 17,397 | 54,371 | 60,003 | 18,989 | 47,484 |
| NoopW100C4 / 1u-100Kt | 13,246 | 22,371 | 30,557 | 10,603 | 25,473 |
| NoopW100C4 / 100u-1Kt | 17,670 | 64,352 | 59,094 | 15,898 | 42,707 |
| NoopW1000C4 / 1u-100Kt | 12,553 | 22,215 | 27,712 | 7,868 | 23,676 |
| NoopW1000C4 / 100u-1Kt | 19,361 | 64,171 | 50,903 | 11,103 | 36,269 |

Rmq leads every single-producer row; at 100 producers Rmq leads at W10 and VarMQDistributed at W100 and W1000.

`Sleep1ms` rows are capped at C x W x 1,000 jobs/s and are on the chart and in `bench/redis.txt`.

Enqueue only (`BenchmarkEnqueue`, jobs/s, highest first): at one producer VarMQ 34,858, Rmq 31,877, GocraftWork 31,027, VarMQDistributed 28,348, Asynq 25,440, Taskq 24,157; at 100 producers GocraftWork 204,753, Rmq 200,161, Taskq 196,154, VarMQ 195,120, VarMQDistributed 141,995, Asynq 116,778.

Enqueue-to-handler latency (`BenchmarkLatency`, ms, lowest p50 first):

| Subject | p50 | p99 | max |
|---|---:|---:|---:|
| VarMQ | 0.05779 | 0.1332 | 0.3464 |
| Taskq | 0.06198 | 0.1505 | 0.5029 |
| VarMQDistributed | 0.06760 | 0.1620 | 1.161 |
| GocraftWork | 0.08176 | 0.4244 | 10.47 |
| Rmq | 1.251 | 2.440 | 6.231 |
| Asynq | 998.0 | 1491 | 1501 |

Asynq sleeps 0.5 to 1.5 s after an empty poll, and its numbers are that sleep.
Rmq sleeps 1 ms after every fetch pass, which puts its p50 at 1.251 ms.
Taskq's fetcher blocks in `XREADGROUP`, so the producer's `XADD` wakes it directly.
VarMQ's `Add` wakes its worker in the same process, which then pops the job with one script.
VarMQDistributed's enqueue script publishes a wake to one idle consumer.

Every row's setup and teardown are timed. In `C4` rows the four instances are stopped one after another, so Asynq and Taskq pay their idle poll up to four times.

Full run: `bench/redis.txt`.

## Subjects

| Subject (chart name) | Library | Version | Instances | Wake | Consumer round trips per job |
|---|---|---|---|---|---|
| VarMQ | [goptics/varmq](https://github.com/goptics/varmq) with [goptics/redisq](https://github.com/goptics/redisq) | varmq v1.10.0, redisq v1.5.0, [go-redis](https://github.com/redis/go-redis) v9.23.0 | 1 | push in-process | 1 ack pipeline per job + 1 pop script per up to 64 jobs |
| VarMQDistributed | [goptics/varmq](https://github.com/goptics/varmq) with [goptics/redisq](https://github.com/goptics/redisq) | varmq v1.10.0, redisq v1.5.0, [go-redis](https://github.com/redis/go-redis) v9.23.0 | 1 and 4 | push to one idle consumer plus a 1 s nudge | 1 ack pipeline per job + 1 pop script per up to 64 jobs |
| Asynq | [hibiken/asynq](https://github.com/hibiken/asynq) | v0.26.0 with [go-redis](https://github.com/redis/go-redis) v9.23.0 | 1 and 4 | poll 0.5-1.5 s | 2 |
| Rmq | [adjust/rmq](https://github.com/adjust/rmq) | v5.2.0 with [go-redis](https://github.com/redis/go-redis) v9.23.0 | 1 and 4 | poll 1 ms | 2 |
| Taskq | [vmihailenco/taskq](https://github.com/vmihailenco/taskq) | v3.2.9 with [go-redis](https://github.com/redis/go-redis) v8.11.5 | 1 and 4 | blocking XREADGROUP | 2 + 1 per 10 jobs |
| GocraftWork | [gocraft/work](https://github.com/gocraft/work) | v0.5.1 with [redigo](https://github.com/gomodule/redigo) v1.9.3 | 1 and 4 | backoff poll 10 ms to 5 s | 2 |

What each library does per job, read from source: [NOTES.md](NOTES.md).

## Settings

Each subject runs at its library defaults except:

- Worker concurrency is W.
- The Redis pool size is `max(workers, 10 x GOMAXPROCS)` per consumer instance: 220 for W10 and W100 here (GOMAXPROCS 22), 1,000 for W1000. Every producer client gets 10 x GOMAXPROCS (220), go-redis's default.
- rmq polls every 1 ms with a prefetch limit of `max(1000, 2 x workers)`, because rmq has no default for either. The 100 ms of rmq's example capped a connection near prefetch / 0.1 s.
- taskq runs with `MinNumWorker = MaxNumWorker = workers`, which turns off its autotuning.
- gocraft/work's redigo pool has `Wait: true`, so a full pool blocks instead of failing.
- Asynq uses one queue, named `bench`.

Names: `BenchmarkDurable/<Task>W<workers>C<instances>/<workload>/<subject>`, `BenchmarkEnqueue/Enqueue/<workload>/<subject>`, `BenchmarkLatency/LatencyW10C1/1u-1Kt/<subject>`.

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
