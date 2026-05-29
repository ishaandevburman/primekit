package algo

import (
	"context"
	"math"
)

type PrimeCounter struct{}

func (p *PrimeCounter) Name() string { return "legendre-primecount" }

func (p *PrimeCounter) CountPrimes(ctx context.Context, limit uint64) (uint64, error) {
	return LegendrePhi(ctx, limit)
}

func LegendrePhi(ctx context.Context, x uint64) (uint64, error) {
	if x < 2 {
		return 0, nil
	}
	a := uint64(math.Sqrt(float64(x)))
	basePrimes := simpleSieve(a)
	piSqrt := uint64(len(basePrimes))

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	lp := &legendrePhi{primes: basePrimes, cache: make(map[phiKey]uint64)}
	return lp.phi(x, piSqrt) + piSqrt - 1, nil
}

type phiKey struct {
	x uint64
	a uint64
}

type legendrePhi struct {
	primes []uint64
	cache  map[phiKey]uint64
}

func (lp *legendrePhi) phi(x uint64, a uint64) uint64 {
	if a == 0 {
		return x
	}
	if x == 0 {
		return 0
	}
	pa := lp.primes[a-1]
	if pa > x {
		return 1
	}

	key := phiKey{x, a}
	if v, ok := lp.cache[key]; ok {
		return v
	}

	result := lp.phi(x, a-1) - lp.phi(x/pa, a-1)
	lp.cache[key] = result
	return result
}
