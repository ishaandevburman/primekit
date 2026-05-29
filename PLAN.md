# primekit — Plan

A fast library for prime computation, storage, and querying, inspired by [QueenJewels](https://github.com/SheafificationOfG/QueenJewels) (hand-written LLVM IR nth-prime algorithms).

---

## 1. Goals

- Provide a **pluggable set of prime-finding algorithms** (like QueenJewels' multiple strategies)
- Store discovered primes in **tiered backends** — binary file (fast), SQLite (queryable), and optionally a separate storage daemon for zero-contention writes
- Offer both a **reusable Go library** and a **CLI tool**
- Go **beyond nth-prime** to include prime counting, factorization, gap analysis, and streaming queries

---

## 2. Storage Architecture (Tiered)

```
┌──────────────┐     channel     ┌──────────────────┐     async     ┌──────────────┐
│  Sieve Engine│ ──────────────> │  Storage Worker   │ ───────────> │  SQLite/DB   │
│  (goroutine) │                 │  (batched writes) │              │  (durable)   │
└──────────────┘                 └──────────────────┘              └──────────────┘
                                         │
                                         │ sync
                                         ▼
                                 ┌──────────────┐
                                 │  Binary File  │
                                 │  (mmap, fast) │
                                 └──────────────┘
```

### Layers

| Tier | Backend | Latency | Durability | Purpose |
|------|---------|---------|------------|---------|
| L1 | In-memory ring buffer | ns | None | Hot cache for recent queries |
| L2 | Memory-mapped binary file (uint64 LE) | ~μs | Medium | Fast persistent store |
| L3 | SQLite / PostgreSQL | ~ms | Full | Queryable, indexed, metadata |
| L4 | Storage daemon (optional) | varies | Configurable | Accepts primes over Unix socket / TCP for zero-blocking writes |

### Why a storage daemon?
- Parallel segmented sieves on N cores all produce primes concurrently
- Without serialization, N goroutines would contend for file I/O
- A standalone daemon serializes writes, buffers them, and flushes in batches — the sieve never waits

---

## 3. Algorithms (Pluggable)

Each algorithm implements a common interface:

```go
type PrimeSource interface {
    Name() string
    NthPrime(n uint64) (uint64, error)
    Primes(ctx context.Context, limit uint64, out chan<- uint64) error
    PrimesInRange(ctx context.Context, start, end uint64, out chan<- uint64) error
}
```

### QueenJewels-inspired implementations

| Algorithm | Complexity | Notes |
|-----------|------------|-------|
| Naïve iteration | O(n²) | Baseline, for testing |
| Square-root optimization | O(n^{3/2}) | Stop at sqrt(candidate) |
| Miller-Rabin | O(n log n) | Probabilistic, fast per-candidate check |
| Sieve of Eratosthenes | O(n log log n) | Classic, good up to ~10⁸ |
| Segmented sieve | O(n log log n) | Cache-friendly, scales to 10¹²+ |
| Wheel-factorised segmented sieve | O(n log log n) | ~2-4x faster with 2·3·5·7 wheel |
| Sieve of Pritchard | O(n log n / log log n) | Theoretically fastest comparison-based sieve |

### Go-specific additions

| Algorithm | Purpose |
|-----------|---------|
| Parallel segmented sieve | Split range across N goroutines, merge results |
| Concurrent Miller-Rabin | Parallel primality testing |
| Baillie-PSW | Deterministic for 64-bit, no false positives |
| Pollard's Rho | Factorization of large composites |
| Meissel-Lehmer / Deleglise-Rivat | π(x) prime counting |

---

## 4. Core Library API

```go
package primekit

// Core interfaces
type PrimeSource interface { ... }
type StorageBackend interface {
    Store(ctx context.Context, primes []uint64) error
    Lookup(ctx context.Context) (PrimeIterator, error)
    Contains(ctx context.Context, n uint64) (bool, error)
    Close() error
}

// High-level API
func NthPrime(n uint64, opts ...Option) (uint64, error)
func IsPrime(n uint64) bool
func Primes(limit uint64) <-chan uint64
func PrimesInRange(start, end uint64) <-chan uint64
func CountPrimes(limit uint64) (uint64, error)  // π(x)
func Factor(n uint64) ([]uint64, error)
func NextPrime(n uint64) (uint64, error)
func PrevPrime(n uint64) (uint64, error)
func PrimeGaps(limit uint64) <-chan Gap
```

---

## 5. CLI Tool

```
primekit nth 1000000          # Compute 1,000,000th prime
primekit sieve 1000000000     # Sieve all primes up to 10^9
primekit range 10^12 10^12+1000
primekit count 10^12          # π(10^12)
primekit isprime 9876543211
primekit factor 123456789012345
primekit gaps 10^9            # Find prime gaps up to 10^9
primekit serve                # Start storage daemon
primekit bench                # Run benchmarks across all algos
primekit status               # Show stats (primes stored, ranges, etc.)
```

### Flags

| Flag | Description |
|------|-------------|
| `--algo` | Which algorithm to use |
| `--store` | Storage backend (bin, sqlite, daemon) |
| `--db` | SQLite/Postgres DSN |
| `--workers` | Parallelism (for parallel/segmented sieves) |
| `--progress` | Show progress bar |
| `--json` | Output as JSON |

---

## 6. Extra Features (Beyond QueenJewels)

| Feature | Why |
|---------|-----|
| **π(x) prime counting** | Major use case, Meissel-Lehmer is non-trivial |
| **Factorization** | Common need — Pollard's Rho + trial division |
| **Prime gaps** | Records gaps, twin primes, constellations |
| **Concurrent generation** | Leverage multi-core for segmented sieve |
| **Bloom filter** | Fast "probably prime" check before expensive primality test |
| **Bloom filter-backed IsPrime** | Pre-filter before Miller-Rabin |
| **REST/gRPC server** | Expose as a service (separate binary) |
| **Export/Import** | Binary, CSV, JSON, Parquet |
| **Metadata tracking** | Store when/how a range was computed, which algo, how long |
| **Incremental computation** | Extend existing store instead of recomputing |
| **Lazy evaluation** | Compute primes on-demand up to N, cache as you go |
| **TUI dashboard** | `primekit monitor` — live stats, throughput, ETA |
| **Benchmark suite** | Reproduce QueenJewels "1 second to find billionth prime" |
| **WebAssembly** | Run sieve in browser |

---

## 7. Project Structure

```
primekit/
├── cmd/
│   └── primekit/         # CLI entry point
├── pkg/
│   ├── algo/             # Algorithm implementations
│   │   ├── naive.go
│   │   ├── sqrt.go
│   │   ├── millerrabin.go
│   │   ├── sieve.go
│   │   ├── segmented.go
│   │   ├── wheel.go
│   │   ├── pritchard.go
│   │   ├── parallel.go
│   │   └── ...
│   ├── store/            # Storage backends
│   │   ├── binary.go     # mmap'd binary file
│   │   ├── sqlite.go
│   │   ├── postgres.go
│   │   └── daemon.go     # Remote storage client
│   ├── primekit.go       # Public API, top-level functions
│   ├── primecount.go     # π(x) implementations
│   ├── factor.go         # Factorization
│   ├── gap.go            # Prime gap analysis
│   └── bloom.go          # Bloom filter
├── internal/
│   ├── daemon/           # Storage daemon server
│   └── ui/               # TUI components
├── bench/                # Benchmark scripts
├── scripts/              # Python helper scripts
├── PLAN.md
└── README.md
```

---

## 8. Milestones

| Phase | Status | What |
|-------|--------|------|
| **P0** | ✅ | Core interface, segmented sieve, binary store, CLI: `nth`, `sieve`, `isprime` |
| **P1** | ✅ | All QueenJewels algos ported, benchmark suite |
| **P2** | ✅ | SQLite store, `count` (π(x)), `factor`, `gaps` |
| **P3** | ✅ | Parallel segmented sieve, Bloom filter, incremental computation |
| **P4** | ✅ | Storage daemon, REST API, Makefile with safe targets |
| **P5** | 🚧 | WebAssembly, Pritchard's sieve, Meissel-Lehmer π(x) |

---

## 9. Open Questions (To Decide)

- **Binary format**: Straight uint64 LE array, or index blocks + delta encoding (like EWAH/COMPAX)?
Recommendation (P0–P2):

Plain uint64 little-endian array
Memory-mapped (mmap)

Optional header:

MAGIC
VERSION
PRIME_COUNT
MAX_PRIME

Why

Simplest implementation
Direct random access to nth prime
Easy mmap support
Easy interoperability with other tools

Later (P4+)

Add chunked compression:
Store first prime in block
Delta encode remaining primes
Varint compression

The gap between primes grows slowly (~log n), so delta encoding compresses extremely well.

Decision: Start with raw uint64 array.
- **SQLite schema**: Single `primes` table with range metadata? Or chunked segments?
I would avoid storing one row per prime.

Bad:

CREATE TABLE primes(
    value INTEGER PRIMARY KEY
);

At billions of primes this becomes huge.

Better:

CREATE TABLE prime_segments(
    id INTEGER PRIMARY KEY,
    start_prime INTEGER,
    end_prime INTEGER,
    count INTEGER,
    file_offset INTEGER,
    algorithm TEXT,
    created_at TIMESTAMP
);

Actual primes remain in binary storage.

SQLite stores:

metadata
computed ranges
statistics
benchmark history

Think of SQLite as an index/catalog, not the primary store.

Decision: Chunked segments + metadata. Keep primes in binary files.
- **Language of storage daemon**: Go (same binary) or C/Rust for max speed?
Go.

Reasons:

Goroutines already solve concurrency
Easier deployment
Easier maintenance

A Rust/C daemon might gain only a few percent while increasing complexity significantly.

A single writer goroutine can already handle massive throughput:

Workers
   ↓
channel
   ↓
Writer goroutine
   ↓
batch flush

For primes, disk bandwidth becomes the bottleneck before Go runtime overhead.

Decision: Go daemon.
- **Wheel size**: Which primorial for the wheel? 30 (2·3·5), 210 (2·3·5·7), or 2310 (2·3·5·7·11)?
Wheel	Size	Recommendation
30	2×3×5	Good
210	2×3×5×7	Best balance
2310	2×3×5×7×11	Usually too large

210 is the sweet spot.

Benefits:

Removes most composites
Fits cache nicely
Commonly used in high-performance sieves

2310 adds complexity and cache pressure for diminishing returns.

Decision: Default wheel = 210. or 2310 if 2.08 composite elimination feels good enough, you say, 2310 - 3840bytes Consumes up to 12% of the entire L1 cache just for table logistics, triggering massive cache thrashing.
- **License**: MIT, Apache 2, or GPL?
For maximum adoption:

Apache 2.0

Advantages:

Permissive
Explicit patent protection
Enterprise-friendly
Compatible with most ecosystems

MIT is also fine, but Apache 2.0 tends to be preferred for infrastructure libraries.

Decision: Apache 2.0.

- **Others**
Remove PostgreSQL from early phases

You already have:

mmap binary store
SQLite metadata

Postgres adds operational complexity with little benefit.

Move it to P5 or make it a community extension.

Separate interfaces

Instead of:

type PrimeSource interface {
    NthPrime(...)
    Primes(...)
    PrimesInRange(...)
}

Split into:

type Generator interface {}
type Counter interface {}
type Factorizer interface {}
type PrimalityTester interface {}

Not every algorithm supports every operation efficiently.

Add cancellation everywhere

Every long-running operation should accept:

context.Context

including:

NthPrime
CountPrimes
Factor
PrimeGaps
Add checkpoints

For multi-day sieves:

checkpoint/
  segment_001.chk
  segment_002.chk

Crash recovery becomes trivial.