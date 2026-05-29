package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"primekit/internal/daemon"
	"primekit/pkg/algo"
	"primekit/pkg/store"
)

type config struct {
	storePath  string
	dbPath     string
	algoName   string
	workers    int
	raw        bool
}

func main() {
	cfg := config{}

	flag.StringVar(&cfg.storePath, "store", "primekit.bin", "path to binary prime store")
	flag.StringVar(&cfg.dbPath, "db", "primekit.db", "path to SQLite metadata store")
	flag.StringVar(&cfg.algoName, "algo", "segmented", "algorithm (naive, sqrt, simple, segmented, parallel)")
	flag.IntVar(&cfg.workers, "workers", 4, "number of worker goroutines")
	flag.BoolVar(&cfg.raw, "raw", false, "raw output (no stderr)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `primekit — prime computation toolkit

Usage:
  primekit [flags] <command> [args...]

Commands:
  nth <n>       Compute the nth prime (0-indexed, n=0 → 2)
  sieve <limit> Sieve all primes up to limit
  isprime <n>   Test if n is prime
  count <limit> Count primes up to limit (π(x))
  factor <n>    Factorize n into primes
  gaps <limit>  List prime gaps up to limit
  bench         Run benchmark suite
  status        Show store statistics
  help          Show this help

Flags:
`)
		flag.PrintDefaults()
	}

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	switch flag.Arg(0) {
	case "nth":
		if flag.NArg() < 2 {
			fail("usage: primekit nth <n>")
		}
		n := parseUint(flag.Arg(1))
		cmdNth(ctx, cfg, n)
	case "sieve":
		if flag.NArg() < 2 {
			fail("usage: primekit sieve <limit>")
		}
		limit := parseUint(flag.Arg(1))
		cmdSieve(ctx, cfg, limit)
	case "isprime":
		if flag.NArg() < 2 {
			fail("usage: primekit isprime <n>")
		}
		n := parseUint(flag.Arg(1))
		cmdIsPrime(ctx, cfg, n)
	case "count":
		if flag.NArg() < 2 {
			fail("usage: primekit count <limit>")
		}
		limit := parseUint(flag.Arg(1))
		cmdCount(ctx, cfg, limit)
	case "factor":
		if flag.NArg() < 2 {
			fail("usage: primekit factor <n>")
		}
		n := parseUint(flag.Arg(1))
		cmdFactor(ctx, cfg, n)
	case "gaps":
		if flag.NArg() < 2 {
			fail("usage: primekit gaps <limit>")
		}
		limit := parseUint(flag.Arg(1))
		cmdGaps(ctx, cfg, limit)
	case "bench":
		cmdBench(ctx, cfg)
	case "serve":
		cmdServe(ctx, cfg)
	case "status":
		cmdStatus(ctx, cfg)
	case "help":
		flag.Usage()
	default:
		fail("unknown command: " + flag.Arg(0))
	}
}

func cmdNth(ctx context.Context, cfg config, n uint64) {
	st, err := store.NewBinaryStore(cfg.storePath)
	if err != nil {
		msg("store: %v", err)
		goto compute
	}
	defer st.Close()

	if st.Count() > n {
		out(st.Data()[n])
		return
	}

	st.Close()

compute:
	gen := pickGenerator(cfg)

	if n == 0 {
		out(2)
		return
	}

	start := time.Now()
	p, err := gen.NthPrime(ctx, n)
	if err != nil {
		fail("compute: %v", err)
	}
	out(p)
	msg("elapsed: %v", time.Since(start))
}

func cmdSieve(ctx context.Context, cfg config, limit uint64) {
	gen := pickGenerator(cfg)
	start := time.Now()

	st, err := store.NewBinaryStore(cfg.storePath)
	if err != nil {
		msg("store: %v (will not persist)", err)
	}

	startFrom := uint64(2)
	var existing []uint64
	if st != nil {
		existing = st.Data()
		if uint64(len(existing)) > 0 {
			startFrom = existing[len(existing)-1] + 1
		}
	}
	if startFrom > limit {
		msg("already have primes up to %d", limit)
		if st != nil {
			st.Close()
		}
		if len(existing) > 0 {
			out(existing[len(existing)-1])
		}
		return
	}

	var newPrimes []uint64
	var mu sync.Mutex
	ch := make(chan uint64, 65536)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for p := range ch {
			mu.Lock()
			newPrimes = append(newPrimes, p)
			mu.Unlock()
		}
	}()

	if err := gen.PrimesInRange(ctx, startFrom, limit, ch); err != nil {
		fail("sieve: %v", err)
	}
	wg.Wait()
	elapsed := time.Since(start)

	total := uint64(len(existing) + len(newPrimes))

	if st != nil {
		st.Store(ctx, newPrimes)
		st.Close()
	}
	if sq, sqErr := store.NewSQLiteStore(cfg.dbPath); sqErr == nil {
		sq.RecordSegment(ctx, startFrom, limit, uint64(len(newPrimes)), gen.Name(), elapsed)
		sq.Close()
	}

	msg("found %d new primes from %d to %d in %v (total stored: %d)",
		len(newPrimes), startFrom, limit, elapsed, total)
	if len(newPrimes) > 0 {
		out(newPrimes[len(newPrimes)-1])
	} else if len(existing) > 0 {
		out(existing[len(existing)-1])
	}
}

func cmdIsPrime(ctx context.Context, cfg config, n uint64) {
	st, err := store.NewBinaryStore(cfg.storePath)
	if err == nil {
		defer st.Close()
		found, _ := st.Contains(ctx, n)
		if found {
			out("yes (stored)")
			return
		}
	}
	mr := &algo.MillerRabin{}
	if mr.IsPrime(ctx, n) {
		out("yes")
	} else {
		out("no")
	}
}

func cmdCount(ctx context.Context, cfg config, limit uint64) {
	pc := &algo.PrimeCounter{}
	start := time.Now()
	count, err := pc.CountPrimes(ctx, limit)
	if err != nil {
		fail("count: %v", err)
	}
	msg("π(%d) = %d  (%v)", limit, count, time.Since(start))
	out(count)
}

func cmdFactor(ctx context.Context, cfg config, n uint64) {
	f := &algo.Factorizer{}
	start := time.Now()
	factors, err := f.Factor(ctx, n)
	if err != nil {
		fail("factor: %v", err)
	}
	msg("elapsed: %v", time.Since(start))
	outFactors(n, factors)
}

func outFactors(n uint64, factors []uint64) {
	if len(factors) == 0 {
		out(n)
		return
	}
	s := fmt.Sprintf("%d = %d", n, factors[0])
	for _, f := range factors[1:] {
		s += fmt.Sprintf(" × %d", f)
	}
	out(s)
}

func cmdGaps(ctx context.Context, cfg config, limit uint64) {
	gf := &algo.GapFinder{}
	start := time.Now()
	maxGap, err := gf.MaxGap(ctx, limit)
	if err != nil {
		fail("gaps: %v", err)
	}
	msg("max gap up to %d: %d (between %d and %d) — %v",
		limit, maxGap.Size, maxGap.PrevPrime, maxGap.Prime, time.Since(start))
	out(maxGap.Size)
}

func cmdBench(ctx context.Context, cfg config) {
	suite := algo.NewBenchmarkSuite()

	msg("=== nth prime benchmarks ===\n")
	ns := []uint64{100, 1000, 10000, 100000}
	results := suite.RunNthPrime(ctx, ns)
	msg(suite.Summary(results))

	for _, r := range results {
		if r.Error != nil {
			continue
		}
		sq, err := store.NewSQLiteStore(cfg.dbPath)
		if err == nil {
			sq.RecordBenchmark(ctx, r.Algorithm, r.N, 0, uint64(r.Elapsed.Milliseconds()), 0)
			sq.Close()
		}
	}

	msg("=== sieve benchmarks ===\n")
	limits := []uint64{100000, 1000000, 10000000}
	sieveResults := suite.RunSieve(ctx, limits)
	msg(suite.Summary(sieveResults))

	for _, r := range sieveResults {
		sq, err := store.NewSQLiteStore(cfg.dbPath)
		if err == nil {
			sq.RecordBenchmark(ctx, r.Algorithm, 0, r.Limit, uint64(r.Elapsed.Milliseconds()), r.PrimesFound)
			sq.Close()
		}
	}
}

func cmdStatus(ctx context.Context, cfg config) {
	st, err := store.NewBinaryStore(cfg.storePath)
	if err == nil {
		msg("binary store: %s", cfg.storePath)
		msg("  primes stored: %d", st.Count())
		msg("  max prime:     %d", st.MaxPrime())
		st.Close()
	} else {
		msg("binary store: %s (not found)", cfg.storePath)
	}

	sq, sqErr := store.NewSQLiteStore(cfg.dbPath)
	if sqErr == nil {
		defer sq.Close()
		msg("SQLite store: %s", cfg.dbPath)

		segs, _ := sq.ListSegments(ctx)
		if len(segs) > 0 {
			msg("  segments: %d", len(segs))
			last := segs[len(segs)-1]
			msg("  last range: %d – %d (%d primes via %s, %dms)",
				last.Start, last.End, last.Count, last.Algorithm, last.ElapsedMs)
		}

		bencs, _ := sq.ListBenchmarks(ctx)
		if len(bencs) > 0 {
			msg("  recent benchmarks: %d", len(bencs))
			for _, b := range bencs {
				var label string
				if b.NValue != nil && *b.NValue > 0 {
					label = fmt.Sprintf("n=%d", *b.NValue)
				} else if b.LimitValue != nil && *b.LimitValue > 0 {
					label = fmt.Sprintf("≤%d", *b.LimitValue)
				} else {
					label = "?"
				}
				msg("    %s  %s  %dms  %d primes",
					b.Algorithm, label, b.ElapsedMs, b.PrimesFound)
			}
		}
	} else {
		msg("SQLite store: %s (%v)", cfg.dbPath, sqErr)
	}
}

func pickGenerator(cfg config) algo.NamedGenerator {
	switch cfg.algoName {
	case "naive":
		return &algo.NaiveIteration{}
	case "sqrt":
		return &algo.SqrtIteration{}
	case "simple":
		return &algo.SimpleSieve{}
	case "segmented":
		return algo.NewSegmentedSieve(1 << 20)
	// case "wheel":
	// 	wheel sieve needs a rewrite; use "segmented" or "parallel" instead
	case "parallel":
		return algo.NewParallelSegmentedSieve(1<<20, cfg.workers)
	default:
		fail("unknown algorithm: %s", cfg.algoName)
		return nil
	}
}

func cmdServe(ctx context.Context, cfg config) {
	srv, err := daemon.NewServer(cfg.storePath, cfg.dbPath)
	if err != nil {
		fail("daemon: %v", err)
	}
	addr := ":8080"
	if flag.NArg() > 1 {
		addr = flag.Arg(1)
	}
	msg("listening on %s", addr)
	if err := srv.ListenAndServe(addr); err != nil {
		fail("serve: %v", err)
	}
}

func parseUint(s string) uint64 {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		fail("invalid number: %s", s)
	}
	return v
}

func out(a ...interface{}) {
	fmt.Println(a...)
}

func msg(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

func fail(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
