package algo

import (
	"context"
	"math"
)

type SqrtIteration struct{}

func (s *SqrtIteration) Name() string { return "sqrt-iteration" }

func (s *SqrtIteration) NthPrime(ctx context.Context, nth uint64) (uint64, error) {
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
		if isPrimeSqrt(candidate) {
			if count == nth {
				return candidate, nil
			}
			count++
		}
	}
}

func (s *SqrtIteration) Primes(ctx context.Context, limit uint64, out chan<- uint64) error {
	return s.PrimesInRange(ctx, 2, limit, out)
}

func (s *SqrtIteration) PrimesInRange(ctx context.Context, start, end uint64, out chan<- uint64) error {
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
		if isPrimeSqrt(i) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- i:
			}
		}
	}
	return nil
}

func isPrimeSqrt(n uint64) bool {
	if n < 2 {
		return false
	}
	if n == 2 || n == 3 {
		return true
	}
	if n%2 == 0 || n%3 == 0 {
		return false
	}
	sqrt := uint64(math.Sqrt(float64(n)))
	for i := uint64(5); i <= sqrt; i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}
