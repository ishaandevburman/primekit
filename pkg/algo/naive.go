package algo

import (
	"context"
)

type NaiveIteration struct{}

func (n *NaiveIteration) Name() string { return "naive-iteration" }

func (n *NaiveIteration) NthPrime(ctx context.Context, nth uint64) (uint64, error) {
	if nth == 0 {
		return 2, nil
	}
	count := uint64(0)
	for candidate := uint64(2); ; candidate++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
		if isPrimeNaive(candidate) {
			if count == nth {
				return candidate, nil
			}
			count++
		}
	}
}

func (n *NaiveIteration) Primes(ctx context.Context, limit uint64, out chan<- uint64) error {
	return n.PrimesInRange(ctx, 2, limit, out)
}

func (n *NaiveIteration) PrimesInRange(ctx context.Context, start, end uint64, out chan<- uint64) error {
	defer close(out)
	if end < 2 {
		return nil
	}
	if start < 2 {
		start = 2
	}
	for i := start; i <= end; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if isPrimeNaive(i) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- i:
			}
		}
	}
	return nil
}

func isPrimeNaive(n uint64) bool {
	if n < 2 {
		return false
	}
	if n == 2 {
		return true
	}
	if n%2 == 0 {
		return false
	}
	for i := uint64(3); i < n; i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}
