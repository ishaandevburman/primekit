package algo

import (
	"context"
	"math/rand"
	"sort"
)

type Factorizer struct{}

func (f *Factorizer) Name() string { return "pollard-rho" }

func (f *Factorizer) Factor(ctx context.Context, n uint64) ([]uint64, error) {
	if n < 2 {
		return nil, nil
	}
	mr := &MillerRabin{}
	if mr.IsPrime(ctx, n) {
		return []uint64{n}, nil
	}
	factors := trialDivide(ctx, n)
	return append(factors, pollardRho(ctx, n/factorProduct(factors), mr)...), nil
}

func trialDivide(ctx context.Context, n uint64) []uint64 {
	var factors []uint64
	for _, p := range smallPrimes {
		if p*p > n {
			break
		}
		for n%p == 0 {
			factors = append(factors, p)
			n /= p
		}
		select {
		case <-ctx.Done():
			return factors
		default:
		}
	}
	return factors
}

var smallPrimes = []uint64{
	2, 3, 5, 7, 11, 13, 17, 19, 23, 29,
	31, 37, 41, 43, 47, 53, 59, 61, 67, 71,
	73, 79, 83, 89, 97, 101, 103, 107, 109, 113,
	127, 131, 137, 139, 149, 151, 157, 163, 167, 173,
	179, 181, 191, 193, 197, 199, 211, 223, 227, 229,
	233, 239, 241, 251, 257, 263, 269, 271, 277, 281,
	283, 293, 307, 311, 313, 317, 331, 337, 347, 349,
	353, 359, 367, 373, 379, 383, 389, 397, 401, 409,
	419, 421, 431, 433, 439, 443, 449, 457, 461, 463,
	467, 479, 487, 491, 499, 503, 509, 521, 523, 541,
}

func pollardRho(ctx context.Context, n uint64, mr *MillerRabin) []uint64 {
	if n <= 1 {
		return nil
	}
	if mr.IsPrime(ctx, n) {
		return []uint64{n}
	}
	if n%2 == 0 {
		factors := pollardRho(ctx, n/2, mr)
		return append([]uint64{2}, factors...)
	}

	var factors []uint64
	c := uint64(rand.Int63n(int64(n-1)) + 1)
	x := uint64(rand.Int63n(int64(n-2)) + 2)
	y := x
	d := uint64(1)
	f := func(v uint64) uint64 {
		return (modMul(v, v, n) + c) % n
	}

	for d == 1 {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		x = f(x)
		y = f(f(y))
		var diff uint64
		if x > y {
			diff = x - y
		} else {
			diff = y - x
		}
		d = gcd(diff, n)
	}

	if d == n {
		return pollardRho(ctx, n, mr)
	}

	factors = append(factors, pollardRho(ctx, d, mr)...)
	factors = append(factors, pollardRho(ctx, n/d, mr)...)
	sort.Slice(factors, func(i, j int) bool { return factors[i] < factors[j] })
	return factors
}

func gcd(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func factorProduct(factors []uint64) uint64 {
	p := uint64(1)
	for _, f := range factors {
		p *= f
	}
	return p
}

func (f *Factorizer) Factorize(ctx context.Context, n uint64) ([]uint64, error) {
	return f.Factor(ctx, n)
}
