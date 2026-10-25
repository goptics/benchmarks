# Worker pool benchmarks

Nine Go worker pools run by one harness, 1M tasks per op: [ants](https://github.com/panjf2000/ants), [bytedance/gopkg](https://github.com/bytedance/gopkg) gopool, [conc](https://github.com/sourcegraph/conc), [pond](https://github.com/alitto/pond), [varmq](https://github.com/goptics/varmq), [workerpool](https://github.com/gammazero/workerpool), [x/sync](https://github.com/golang/sync) errgroup and two channel baselines.
Charts at 24, 8 and 4 CPUs: [pool-24cpu](https://benchmarks.goptics.org/?id=pool-24cpu), [pool-8cpu](https://benchmarks.goptics.org/?id=pool-8cpu), [pool-4cpu](https://benchmarks.goptics.org/?id=pool-4cpu).

## Results

Run of 2026-10-24, varmq v1.10.0, `-benchtime=2x`, 495 rows per file. Each file has 55 task/size/workload rows (11 groups such as `Loop10W50K` x 5 workloads), each with 9 subjects. The table counts the rows in which each subject has the lowest `ns/op`; no other subject is fastest in any row.

| CPUs | Rows | VarMQ | Ants | ErrGroup | BufferedPool |
|------|------|-------|------|----------|--------------|
| 24 | 55 | 40 | 0 | 2 | 13 |
| 8 | 55 | 36 | 0 | 6 | 13 |
| 4 | 55 | 20 | 4 | 11 | 20 |

On the 15 one-worker `Loop10W1` rows (5 workloads x 3 files), BufferedPool is fastest in 14. On the four workloads other than `1Mu-1t`, the next subject (Gopool on `1u-1Mt`, VarMQ on the other three) takes 2.02x to 3.81x BufferedPool's `ns/op`. On `1Mu-1t`, BufferedPool and VarMQ are the two fastest, and the slower of them takes 1.04x, 1.04x and 1.08x the faster one's `ns/op` at 24, 8 and 4 CPUs. BufferedPool is fastest at 24 and 4 CPUs, VarMQ at 8.

## Subjects

| Name | Version | What it is | Submit when the pool is full | Teardown |
|------|---------|------------|------------------------------|----------|
| VarMQ | v1.10.0 | [varmq](https://github.com/goptics/varmq) worker bound to an in-memory queue (`BindQueue`) | Never blocks: `q.Add` appends to an unbounded queue and signals the worker with a non-blocking send | `WaitAndStop`: waits until no job is queued or running, then signals its workers to stop and returns without waiting for them to exit |
| Ants | v2.12.1 | [ants](https://github.com/panjf2000/ants) `NewPool` with default options | `Submit` blocks until a worker is free | `Release`, returns without waiting for its workers to exit |
| WorkerPool | v1.2.2 | [workerpool](https://github.com/gammazero/workerpool) | Never blocks: `Submit` enqueues the task | `StopWait`, waits for its workers |
| Conc | v0.3.0 | [conc](https://github.com/sourcegraph/conc) `pool` with a goroutine cap | `p.Go` blocks at the cap | `Wait`, waits for its goroutines |
| ErrGroup | v0.24.0 | [x/sync](https://github.com/golang/sync) `errgroup` with `SetLimit`: one goroutine per task, not a pool | `g.Go` blocks at the limit | `Wait`, waits for its goroutines |
| Gopool | v0.1.4 | [bytedance/gopkg](https://github.com/bytedance/gopkg) `util/gopool`, soft worker cap | Never blocks: unbounded task list | None; idle workers exit on their own |
| GoroutinePool | baseline | N goroutines reading one unbuffered channel | Blocks: the send waits for a worker to receive | Closes the channel, waits for all N workers to exit |
| BufferedPool | baseline | Same, with a channel that holds 1M tasks | Never blocks: the channel holds all 1M | Closes the channel, waits for all N workers to exit |
| Pond | v2.7.2 | [pond](https://github.com/alitto/pond) v2 `NewPool` | Never blocks: `DefaultQueueSize` is `Unbounded` | `StopAndWait`, waits for its workers |

Ants, Gopool and VarMQ do not wait for their workers to exit in teardown, so they skip that cost inside the timed region.

All subjects live in [subjects.go](subjects.go).

## Workloads

| Workload | Producers | Tasks each |
|----------|-----------|------------|
| `1u-1Mt` | 1 | 1,000,000 |
| `100u-10Kt` | 100 | 10,000 |
| `1Ku-1Kt` | 1,000 | 1,000 |
| `10Ku-100t` | 10,000 | 100 |
| `1Mu-1t` | 1,000,000 | 1 |

A producer is a goroutine. The harness spawns the producers one after another in a loop, with no start barrier. Each producer submits its share of tasks and exits. Producers are not joined; the run waits for task completions. `1Mu-1t` therefore spawns one million goroutines inside the timed region, each submitting one task.

| Task | What it runs |
|------|--------------|
| `Sleep10ms` | `time.Sleep(10ms)`: a timer park, not I/O |
| `Loop10` | A counted loop of 10 iterations |
| `Sleep5msLoop5` | A 5-iteration loop, then a 5 ms sleep |
| `Loop10K` | A counted loop of 10,000 iterations; pool of one only |

The loop body is `_ = i * i`, so the compiler removes the multiplication and the CPU tasks are empty counted loops. `Loop10K` takes about 1 µs on this machine: BufferedPool's `Loop10KW1` rows come to 1.02 to 1.18 µs per task.

| Size | Tasks |
|------|-------|
| 50K workers | `Sleep10ms`, `Loop10`, `Sleep5msLoop5` |
| 100K workers | `Sleep10ms`, `Loop10`, `Sleep5msLoop5` |
| 300K workers | `Sleep10ms`, `Loop10`, `Sleep5msLoop5` |
| 1 worker | `Loop10`, `Loop10K` |

GoroutinePool and BufferedPool start all N workers when the pool is built, so 300K goroutines start inside the timed region at `W300K`. BufferedPool's channel buffer holds all 1M tasks. W1 runs CPU tasks only, since one worker sleeping 10 ms per task would need about 2.8 hours.

The matrix is recorded with `-cpu` 24, 8 and 4, which sets GOMAXPROCS without pinning cores; rows carry a `-24`, `-8` or `-4` suffix.

## Method

The timed region of one op covers building the pool, spawning the producers, every submit, the wait until all 1M tasks have completed, and teardown. Completions are counted by the harness's own WaitGroup around each task, not by any pool signal.

- `ns/op`: wall time of that whole region
- `ops/s`: the same measurement as a rate, 1M x `b.N` / elapsed
- `B/op` and `allocs/op`: allocations of the whole process during the op, including any per-task closure an adapter allocates

`-benchtime=2x` means two ops per row, with no warm-up op.

## Running

| Task | What it does |
|------|--------------|
| `task pool:test` | `go vet ./...` and `go test ./...` |
| `task pool:bench` | Every benchmark once: `-benchtime=1x -benchmem -timeout 0` |
| `task pool:txt` | Full matrix with `-benchtime=2x -benchmem -timeout 0` at `-cpu` 24, 8 and 4 into `bench/pool-24cpu.txt`, `bench/pool-8cpu.txt` and `bench/pool-4cpu.txt`; the committed files took 943.059 s, 724.893 s and 711.582 s |

Rows are named `BenchmarkTasks/<Task>W<size>/<workload>/<pool>-<cpus>`, with `W` for workers and sizes in thousands ending in `K`. `BenchmarkTasks/Loop10W50K/100u-10Kt/VarMQ-24` is the `Loop10` task on 50,000 workers, 100 producers x 10,000 tasks each, subject VarMQ, at `-cpu 24`. One slice, the pool of one worker:

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
