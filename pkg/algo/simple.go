package algo

import (
	"context"
	"math"
)

type SimpleSieve struct{}

func (s *SimpleSieve) Name() string { return "sieve-of-eratosthenes" }

func (s *SimpleSieve) NthPrime(ctx context.Context, n uint64) (uint64, error) {
	if n == 0 {
		return 2, nil
	}
	upper := estimateUpperBound(n)
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
		primes := simpleSieve(upper)
		if uint64(len(primes)) > n {
			return primes[n], nil
		}
		upper *= 2
	}
}

func (s *SimpleSieve) Primes(ctx context.Context, limit uint64, out chan<- uint64) error {
	return s.PrimesInRange(ctx, 2, limit, out)
}

func (s *SimpleSieve) PrimesInRange(ctx context.Context, start, end uint64, out chan<- uint64) error {
	defer close(out)
	if end < 2 {
		return nil
	}
	if start < 2 {
		start = 2
	}
	primes := simpleSieve(end)
	for _, p := range primes {
		if p < start {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- p:
		}
	}
	return nil
}

func estimateUpperBound(n uint64) uint64 {
	if n < 6 {
		return 15
	}
	f := float64(n)
	approx := f * (math.Log(f) + math.Log(math.Log(f)))
	return uint64(math.Ceil(approx))
}
