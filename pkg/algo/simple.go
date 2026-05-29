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
	defer close(out)
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

func simpleSieve(limit uint64) []uint64 {
	if limit < 2 {
		return nil
	}
	n := int(limit + 1)
	isPrime := make([]bool, n)
	for i := 2; i < n; i++ {
		isPrime[i] = true
	}
	sqrt := int(math.Sqrt(float64(limit)))
	for i := 2; i <= sqrt; i++ {
		if isPrime[i] {
			step := i
			start := i * i
			for j := start; j < n; j += step {
				isPrime[j] = false
			}
		}
	}
	var count int
	for i := 2; i < n; i++ {
		if isPrime[i] {
			count++
		}
	}
	primes := make([]uint64, 0, count)
	for i := 2; i < n; i++ {
		if isPrime[i] {
			primes = append(primes, uint64(i))
		}
	}
	return primes
}

func estimateUpperBound(n uint64) uint64 {
	if n < 6 {
		return 15
	}
	f := float64(n)
	approx := f * (math.Log(f) + math.Log(math.Log(f)))
	return uint64(math.Ceil(approx))
}
