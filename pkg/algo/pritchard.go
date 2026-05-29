package algo

import (
	"context"
)

type PritchardSieve struct{}

func NewPritchardSieve() *PritchardSieve {
	return &PritchardSieve{}
}

func (s *PritchardSieve) Name() string {
	return "pritchard"
}

func (s *PritchardSieve) NthPrime(ctx context.Context, n uint64) (uint64, error) {
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
		primes := s.generate(upper)
		if uint64(len(primes)) > n {
			return primes[n], nil
		}
		upper *= 2
	}
}

func (s *PritchardSieve) Primes(ctx context.Context, limit uint64, out chan<- uint64) error {
	return s.PrimesInRange(ctx, 2, limit, out)
}

func (s *PritchardSieve) PrimesInRange(ctx context.Context, start, end uint64, out chan<- uint64) error {
	defer close(out)
	if end < 2 || start > end {
		return nil
	}
	if start < 2 {
		start = 2
	}
	primes := s.generate(end)
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

func (s *PritchardSieve) generate(limit uint64) []uint64 {
	if limit < 2 {
		return nil
	}

	W := make([]bool, limit+1)
	W[1] = true
	maxProd := uint64(1)
	var primes []uint64
	var found []uint64

	for p := uint64(2); p*p <= limit; p++ {
		isPrime := false
		if p > maxProd {
			isPrime = true
			for _, q := range found {
				if p%q == 0 {
					isPrime = false
					break
				}
			}
		} else if W[p] {
			isPrime = true
		}
		if !isPrime {
			continue
		}

		found = append(found, p)

		newMax := p * maxProd
		if newMax > limit {
			newMax = limit
		}

		if newMax > maxProd {
			for x := maxProd + 1; x <= newMax; x++ {
				W[x] = true
			}
			for _, q := range found[:len(found)-1] {
				if q > newMax {
					break
				}
				first := ((maxProd + 1 + q - 1) / q) * q
				for k := first; k <= newMax; k += q {
					W[k] = false
				}
			}
			maxProd = newMax
		}

		for k := p + p; k <= maxProd; k += p {
			W[k] = false
		}
	}

	if maxProd < limit {
		for x := maxProd + 1; x <= limit; x++ {
			W[x] = true
		}
		for _, q := range found {
			if q > limit {
				break
			}
			first := ((maxProd + 1 + q - 1) / q) * q
			for k := first; k <= limit; k += q {
				W[k] = false
			}
		}
	}

	for x := uint64(2); x <= limit; x++ {
		if W[x] {
			primes = append(primes, x)
		}
	}

	return primes
}
