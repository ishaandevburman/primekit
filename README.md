# primekit

A modular prime computation library and CLI tool in Go.

## Features

### Interfaces
- **`Generator`** — `NthPrime`, `Primes`, `PrimesInRange`
- **`Counter`** — `CountPrimes` (π(x))
- **`Factorizer`** — `Factor`
- **`PrimalityTester`** — `IsPrime`
- **`StorageBackend`** — `Store`, `Contains`

### Algorithms

| Type | Implementations | File |
|---|---|---|
| Naive iteration | `NaiveIteration` | `naive.go` |
| Sqrt trial division | `SqrtIteration` | `sqrt.go` |
| Simple sieve | `SimpleSieve` | `simple.go` |
| Segmented sieve | `SegmentedSieve` (configurable segment size) | `segmented.go` |
| Wheel-210 sieve | `WheelSegmentedSieve` (mod 210) | `wheel.go` |
| Parallel segmented sieve | `ParallelSegmentedSieve` (configurable workers) | `parallel.go` |
| Sieve of Pritchard | `PritchardSieve` | `pritchard.go` |
| π(x) — Legendre | `PrimeCounter` (φ recursion) | `primecount.go` |
| π(x) — Meissel-Lehmer | `MeisselLehmerCounter` | `meissellehmer.go` |
| Miller-Rabin | `MillerRabin` (deterministic for 64-bit) | `millerrabin.go` |
| Pollard's Rho | `Factorizer` (trial division + Rho) | `factor.go` |
| Bloom filter | `BloomFilter` / `BloomPrimality` | `bloom.go` |
| Prime gaps | `GapFinder` (`MaxGap`, `TwinPrimes`) | `gap.go` |

### Storage
- **BinaryStore** — memory-mapped append-only binary store
- **SQLiteStore** — segment/benchmark metadata
- **DaemonStore** — HTTP client for a remote storage daemon

## Install

```bash
git clone git@github.com:ishaandevburman/primekit.git
cd primekit
make build
```

Requires Go 1.26.2+.

## CLI Reference

### Global flags

All commands accept these flags before the subcommand:

| Flag | Default | Applies to | Description |
|---|---|---|---|
| `--algo <name>` | `segmented` | `nth`, `sieve` | Algorithm to use (see below) |
| `--store <path>` | `primekit.bin` | `nth`, `sieve`, `isprime`, `status` | Binary store for persisted primes |
| `--db <path>` | `primekit.db` | `sieve`, `bench`, `status` | SQLite metadata store |
| `--workers <n>` | `4` | `nth`, `sieve` (with `--algo parallel`) | Number of worker goroutines |
| `--raw` | false | all | Print only the result to stdout; all logging goes to stderr (suppressed) |

**`--algo`** accepts: `naive`, `sqrt`, `simple`, `segmented`, `wheel`, `parallel`.

### Commands

#### `primekit nth <n>`

Compute the **nth prime** (0-indexed: `n=0` → 2).

Uses `NthPrime` which estimates an upper bound via `n(log n + log log n)`, sieves up to that bound to count primes, then sieves again to locate the exact nth. If the estimate is too low, the bound doubles and retries.

```bash
primekit nth 1000000            # use default (segmented, 1MB segments)
primekit --algo parallel nth 10000000   # parallel, 4 workers
primekit --algo wheel --raw nth 100000  # wheel-210, raw output only
```

For large `n` (≥10⁸), prefer `--algo parallel` — it parallelises the sieve across `--workers` goroutines. `wheel` and `segmented` are single-threaded. `naive`/`sqrt` are only practical for tiny `n`.

#### `primekit sieve <limit>`

Sieve all primes up to `limit`. Persists to the binary store (incremental — resumes from the last stored prime). Records the segment in SQLite metadata.

```bash
primekit sieve 1000000           # primes up to 1M, stored to primekit.bin
primekit --algo parallel --workers 8 sieve 100000000  # parallel, 8 workers
primekit --store custom.bin sieve 1000000  # use custom binary store
```

After sieving, future `nth` commands check the store first for a fast answer.

#### `primekit count <limit>`

Count primes ≤ `limit` — the prime-counting function π(x).

Uses Legendre's phi recursion (`PrimeCounter`). For large limits (≥10¹⁰), use the dedicated `MeisselLehmer` function from the library instead — it's orders of magnitude faster.

```bash
primekit count 1000000000        # π(10⁹) = 50847534
```

Note: `count` does **not** use `--algo`; it always uses Legendre's formula.

#### `primekit isprime <n>`

Test if `n` is prime. Checks the binary store first (fast if stored), then falls back to deterministic Miller-Rabin (64-bit).

```bash
primekit isprime 982451653       # yes
primekit isprime 1000000000000061  # probably not
```

#### `primekit factor <n>`

Factorize `n` into its prime factors using trial division (first 55 primes) + Pollard's Rho.

```bash
primekit factor 123456789012345
# 123456789012345 = 3 × 5 × 1153 × 7138631591
primekit factor 1000000000000000003
```

#### `primekit gaps <limit>`

Find the **maximum prime gap** up to `limit`, plus the twin-prime count.

```bash
primekit gaps 1000000            # max gap up to 1M
```

#### `primekit bench`

Run the full benchmark suite across 6 sieve algorithms: `simple`, `segmented-64k`, `segmented-1m`, `wheel-210`, `pritchard`, `parallel-segmented`.

Tests `NthPrime` for n = {100, 1000, 10000, 100000} and sieving for limits = {10⁵, 10⁶, 10⁷}. Results are recorded in the SQLite store.

```bash
primekit bench
primekit --db mybench.db bench   # custom SQLite output
```

#### `primekit serve [addr]`

Start the HTTP storage daemon. Accepts `GET /isprime/{n}`, `POST /store`, `GET /status`. Default addr is `:8080`.

```bash
primekit serve :8080
```

Once running, other instances can use `DaemonStore` to query/store remotely.

#### `primekit status`

Show store statistics: binary store count/max, SQLite segments and recent benchmarks.

```bash
primekit status
primekit --store custom.bin --db custom.db status
```

#### `primekit help`

Print the full usage message.

### Practical examples

```bash
# Fastest way to get the millionth prime
primekit --algo parallel --workers 8 nth 1000000

# Count primes up to 10^12 (use the library for better perf)
# (CLI uses Legendre; for 10^12 call algo.MeisselLehmer directly)

# Persist primes up to 10^8, then look up nth prime from store
primekit --algo parallel sieve 100000000
primekit nth 5000000             # reads from store, instant

# Factor a large semiprime
primekit factor 18446744073709551617

# Run benchmarks and view results
primekit bench
primekit status
```

## Library Usage

```go
import "github.com/ishaandevburman/primekit/pkg/algo"

// Count primes up to 10^12 — fastest method in the library
count, err := algo.MeisselLehmer(ctx, 1e12)

// Generate primes up to 10^6
gen := algo.NewParallelSegmentedSieve(1<<20, 4)
ch := make(chan uint64, 1024)
go gen.Primes(ctx, 1_000_000, ch)
for p := range ch {
    // ...
}

// Test primality
mr := &algo.MillerRabin{}
if mr.IsPrime(ctx, 982451653) {
    // ...
}

// Factorize
f := &algo.Factorizer{}
factors, _ := f.Factor(ctx, 123456789012345)
```

## Project Structure

```
primekit/
├── cmd/primekit/        — CLI entry point
├── internal/daemon/     — HTTP storage daemon
├── pkg/
│   ├── algo/            — All algorithm implementations
│   │   ├── bench.go     — Benchmark harness
│   │   ├── naive.go, sqrt.go, simple.go
│   │   ├── segmented.go, wheel.go, parallel.go
│   │   ├── pritchard.go
│   │   ├── primecount.go, meissellehmer.go
│   │   ├── millerrabin.go, factor.go, bloom.go, gap.go
│   │   └── sieve.go     — Shared simpleSieve helper
│   ├── primekit.go      — Core interfaces
│   └── store/           — Storage backends
├── Makefile
├── PLAN.md              — Design plan & milestones
├── go.mod / go.sum
└── LICENSE              — Apache 2.0
```

## Benchmarks

```bash
make bench
```

The benchmark suite runs six sieve algorithms (`simple`, `segmented-64k`, `segmented-1m`, `wheel-210`, `pritchard`, `parallel-segmented`) across nth-prime (100, 1K, 10K, 100K) and sieve (10⁵, 10⁶, 10⁷) workloads.

## License

Apache 2.0
