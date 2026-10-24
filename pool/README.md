# Worker pool benchmarks

Nine in-process Go worker pools driven by one harness, 1M tasks per op, recorded at 24, 8 and 4 CPUs: [pool-24cpu](https://benchmarks.goptics.org/?id=pool-24cpu), [pool-8cpu](https://benchmarks.goptics.org/?id=pool-8cpu), [pool-4cpu](https://benchmarks.goptics.org/?id=pool-4cpu).

## Results

Run of 2026-10-24, varmq 415ad73 (unreleased, `replace` in go.mod), `-benchtime=2x`, 495 rows per file. Each file has 55 task/size/workload rows (11 groups like `Loop10W50K` x 5 workloads); the table counts the rows in which each subject has the lowest `ns/op`, and no other subject is fastest in any row.

| CPUs | Rows | VarMQ | BufferedPool | ErrGroup | Ants |
|------|------|-------|--------------|----------|------|
| 24 | 55 | 40 | 13 | 2 | 0 |
| 8 | 55 | 36 | 13 | 6 | 0 |
| 4 | 55 | 20 | 20 | 11 | 4 |

On the five one-worker `Loop10W1` rows of each file, VarMQ's `ns/op` divided by BufferedPool's is 3.26x to 4.22x on the four workloads other than `1Mu-1t` (4.22x on `1u-1Mt` at 24 CPUs) and level on `1Mu-1t` (1.04x at 24 CPUs, 0.96x at 8, 1.08x at 4).

## Subjects

| Name | Version | What it is |
|------|---------|------------|
| VarMQ | 415ad73 (unreleased, `replace` in go.mod) | [goptics/varmq](https://github.com/goptics/varmq), worker bound to a queue |
| Pond | v2.7.2 | [alitto/pond/v2](https://github.com/alitto/pond) |
| Ants | v2.12.1 | [panjf2000/ants/v2](https://github.com/panjf2000/ants) |
| WorkerPool | v1.2.2 | [gammazero/workerpool](https://github.com/gammazero/workerpool) |
| Conc | v0.3.0 | [sourcegraph/conc](https://github.com/sourcegraph/conc) `pool` with a goroutine cap |
| ErrGroup | v0.24.0 | [x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup): one goroutine per task behind a semaphore, not a pool |
| Gopool | v0.1.4 | [bytedance/gopkg](https://github.com/bytedance/gopkg) `util/gopool`: submit never blocks; soft worker cap |
| GoroutinePool | baseline | N goroutines reading from an unbuffered channel |
| BufferedPool | baseline | Same, but the channel holds 1M tasks |

All subjects live in [subjects.go](subjects.go).

## Workloads

| Workload | Producers | Tasks each |
|----------|-----------|------------|
| `1u-1Mt` | 1 | 1,000,000 |
| `100u-10Kt` | 100 | 10,000 |
| `1Ku-1Kt` | 1,000 | 1,000 |
| `10Ku-100t` | 10,000 | 100 |
| `1Mu-1t` | 1,000,000 | 1 |

| Task | Behaviour |
|------|-----------|
| `Sleep10ms` | I/O-like: sleep 10 ms |
| `Loop10` | CPU-like: 10 iterations of trivial math |
| `Sleep5msLoop5` | Mixed: 5 iterations then sleep 5 ms |
| `Loop10K` | CPU-like: 10,000 iterations, about 1 µs. Pool of 1 only. |

| Size | Tasks |
|------|-------|
| 50K workers | `Sleep10ms`, `Loop10`, `Sleep5msLoop5` |
| 100K workers | `Sleep10ms`, `Loop10`, `Sleep5msLoop5` |
| 300K workers | `Sleep10ms`, `Loop10`, `Sleep5msLoop5` |
| 1 worker | `Loop10`, `Loop10K` |

W1 runs CPU tasks only, since one worker sleeping 10 ms per task would take about three hours for 1M tasks.

The matrix is recorded at 24, 8 and 4 CPUs (`-cpu`), one dataset each.

## Method

A subject is a factory that, given `maxWorkers`, returns `submit` and `teardown`. Producers are goroutines: a `100u-10Kt` run starts 100 goroutines, each submitting 10,000 tasks. One op is build the pool, submit 1M tasks, wait for all to finish, tear down, with construction and shutdown inside the timed region.

- `ns/op`: wall time to push all 1M tasks through the pool and drain it
- `ops/s`: tasks completed per second (custom metric, higher is better)
- `B/op` and `allocs/op`: memory cost

## Running

| Task | What it does |
|------|--------------|
| `task pool:test` | `go vet` and `go test` |
| `task pool:bench` | Every benchmark once (`-benchtime=1x`) |
| `task pool:txt` | Full matrix at 24, 8 and 4 CPUs into `bench/pool-24cpu.txt`, `bench/pool-8cpu.txt` and `bench/pool-4cpu.txt` (hours) |

Rows are named `BenchmarkTasks/<Task>W<size>/<workload>/<pool>`, with `W` for workers and sizes in thousands ending in `K`: `BenchmarkTasks/Loop10W50K/100u-10Kt/VarMQ` is the `Loop10` task on 50,000 workers, 100 producers x 10,000 tasks each, subject VarMQ. One slice, the pool of one worker:

```bash
go test -run xxx -bench 'Tasks/W1$' -benchtime=1x
```

The module is `package main` with its own `go.mod`: run from this directory, or as `task pool:<name>` from the root.

## Adding a pool

Append an entry to `subjects` in [subjects.go](subjects.go):

```go
{
    name: "MyPool",
    factory: func(maxWorkers int) (poolSubmit, poolTeardown) {
        p := mypool.New(maxWorkers)
        return p.Submit, p.StopAndWait
    },
},
```

The factory is called with every size in `poolSizes`, including `1`.
