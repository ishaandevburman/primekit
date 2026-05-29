package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `primekit - prime computation toolkit

Usage:
  primekit <command> [args...]

Commands:
  nth       Compute the nth prime (1-indexed)
  sieve     Sieve all primes up to a limit
  range     List primes in a range
  count     Count primes up to a limit (π(x))
  isprime   Test if a number is prime
  factor    Factorize a number
  gaps      Find prime gaps up to a limit
  serve     Start storage daemon
  bench     Run benchmarks
  status    Show store statistics
  help      Show this help

Flags:
`)
		flag.PrintDefaults()
	}

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	switch os.Args[1] {
	case "help":
		flag.Usage()
	case "nth":
		cmdNth(ctx, os.Args[2:])
	case "sieve":
		cmdSieve(ctx, os.Args[2:])
	case "range":
		cmdRange(ctx, os.Args[2:])
	case "count":
		cmdCount(ctx, os.Args[2:])
	case "isprime":
		cmdIsPrime(ctx, os.Args[2:])
	case "factor":
		cmdFactor(ctx, os.Args[2:])
	case "gaps":
		cmdGaps(ctx, os.Args[2:])
	case "serve":
		cmdServe(ctx, os.Args[2:])
	case "bench":
		cmdBench(ctx, os.Args[2:])
	case "status":
		cmdStatus(ctx, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		flag.Usage()
		os.Exit(1)
	}
}

func cmdNth(ctx context.Context, args []string) {
	fmt.Fprintln(os.Stderr, "not yet implemented")
	os.Exit(1)
}

func cmdSieve(ctx context.Context, args []string) {
	fmt.Fprintln(os.Stderr, "not yet implemented")
	os.Exit(1)
}

func cmdRange(ctx context.Context, args []string) {
	fmt.Fprintln(os.Stderr, "not yet implemented")
	os.Exit(1)
}

func cmdCount(ctx context.Context, args []string) {
	fmt.Fprintln(os.Stderr, "not yet implemented")
	os.Exit(1)
}

func cmdIsPrime(ctx context.Context, args []string) {
	fmt.Fprintln(os.Stderr, "not yet implemented")
	os.Exit(1)
}

func cmdFactor(ctx context.Context, args []string) {
	fmt.Fprintln(os.Stderr, "not yet implemented")
	os.Exit(1)
}

func cmdGaps(ctx context.Context, args []string) {
	fmt.Fprintln(os.Stderr, "not yet implemented")
	os.Exit(1)
}

func cmdServe(ctx context.Context, args []string) {
	fmt.Fprintln(os.Stderr, "not yet implemented")
	os.Exit(1)
}

func cmdBench(ctx context.Context, args []string) {
	fmt.Fprintln(os.Stderr, "not yet implemented")
	os.Exit(1)
}

func cmdStatus(ctx context.Context, args []string) {
	fmt.Fprintln(os.Stderr, "not yet implemented")
	os.Exit(1)
}
