package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"primekit/pkg/algo"
	"primekit/pkg/store"
)

type config struct {
	storePath string
	algoName  string
	workers   int
}

func main() {
	cfg := config{}

	flag.StringVar(&cfg.storePath, "store", "primekit.bin", "path to binary prime store")
	flag.StringVar(&cfg.algoName, "algo", "segmented", "algorithm (segmented, simple)")
	flag.IntVar(&cfg.workers, "workers", 4, "number of worker goroutines")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `primekit — prime computation toolkit

Usage:
  primekit [flags] <command> [args...]

Commands:
  nth <n>       Compute the nth prime (1-indexed, n=0 → 2)
  sieve <limit> Sieve all primes up to limit
  isprime <n>   Test if n is prime

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
			fmt.Fprintln(os.Stderr, "usage: primekit nth <n>")
			os.Exit(1)
		}
		n, err := strconv.ParseUint(flag.Arg(1), 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid n: %v\n", err)
			os.Exit(1)
		}
		cmdNth(ctx, cfg, n)

	case "sieve":
		if flag.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "usage: primekit sieve <limit>")
			os.Exit(1)
		}
		limit, err := strconv.ParseUint(flag.Arg(1), 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid limit: %v\n", err)
			os.Exit(1)
		}
		cmdSieve(ctx, cfg, limit)

	case "isprime":
		if flag.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "usage: primekit isprime <n>")
			os.Exit(1)
		}
		n, err := strconv.ParseUint(flag.Arg(1), 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid n: %v\n", err)
			os.Exit(1)
		}
		cmdIsPrime(ctx, cfg, n)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", flag.Arg(0))
		flag.Usage()
		os.Exit(1)
	}
}

func cmdNth(ctx context.Context, cfg config, n uint64) {
	st, err := store.NewBinaryStore(cfg.storePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	if st.Count() > n {
		fmt.Println(st.Data()[n])
		return
	}

	gen := pickAlgo(cfg)
	if n == 0 {
		fmt.Println(2)
		return
	}

	start := time.Now()
	p, err := gen.NthPrime(ctx, n)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compute: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(p)
	fmt.Fprintf(os.Stderr, "elapsed: %v\n", time.Since(start))
}

func cmdSieve(ctx context.Context, cfg config, limit uint64) {
	st, err := store.NewBinaryStore(cfg.storePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	gen := pickAlgo(cfg)
	start := time.Now()

	var allPrimes []uint64
	var mu sync.Mutex
	ch := make(chan uint64, 65536)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for p := range ch {
			mu.Lock()
			allPrimes = append(allPrimes, p)
			mu.Unlock()
		}
	}()

	if err := gen.Primes(ctx, limit, ch); err != nil {
		fmt.Fprintf(os.Stderr, "sieve: %v\n", err)
		os.Exit(1)
	}

	wg.Wait()
	elapsed := time.Since(start)

	if err := st.Store(ctx, allPrimes); err != nil {
		fmt.Fprintf(os.Stderr, "store: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "found %d primes up to %d in %v\n", len(allPrimes), limit, elapsed)
	fmt.Println(allPrimes[len(allPrimes)-1])
}

func cmdIsPrime(ctx context.Context, cfg config, n uint64) {
	st, err := store.NewBinaryStore(cfg.storePath)
	if err == nil {
		defer st.Close()
		found, _ := st.Contains(ctx, n)
		if found {
			fmt.Println("yes (stored)")
			return
		}
	}

	mr := &algo.MillerRabin{}
	if mr.IsPrime(ctx, n) {
		fmt.Println("yes")
	} else {
		fmt.Println("no")
	}
}

func pickAlgo(cfg config) *algo.SegmentedSieve {
	return algo.NewSegmentedSieve(1 << 20)
}
