package algo

import (
	"context"
	"math/big"
)

var mrBases = []uint64{2, 325, 9375, 28178, 450775, 9780504, 1795265022}

type MillerRabin struct{}

func (m *MillerRabin) Name() string { return "miller-rabin" }

func (m *MillerRabin) IsPrime(ctx context.Context, n uint64) bool {
	if n < 2 {
		return false
	}
	if n%2 == 0 {
		return n == 2
	}
	if n%3 == 0 {
		return n == 3
	}
	if n%5 == 0 {
		return n == 5
	}
	if n%7 == 0 {
		return n == 7
	}
	if n < 121 {
		return n > 1
	}

	d := n - 1
	s := 0
	for d%2 == 0 {
		d /= 2
		s++
	}

	for _, a := range mrBases {
		if a%n == 0 {
			continue
		}
		x := modPow(a, d, n)
		if x == 1 || x == n-1 {
			continue
		}
		composite := true
		for r := 0; r < s-1; r++ {
			x = modMul(x, x, n)
			if x == n-1 {
				composite = false
				break
			}
		}
		if composite {
			return false
		}
	}
	return true
}

func modPow(base, exp, mod uint64) uint64 {
	if mod == 0 {
		return 0
	}
	result := uint64(1)
	b := base % mod
	e := exp
	for e > 0 {
		if e&1 == 1 {
			result = modMul(result, b, mod)
		}
		b = modMul(b, b, mod)
		e >>= 1
	}
	return result
}

func modMul(a, b, mod uint64) uint64 {
	if mod <= 1<<32 {
		return (a * b) % mod
	}
	biA := new(big.Int).SetUint64(a)
	biB := new(big.Int).SetUint64(b)
	biMod := new(big.Int).SetUint64(mod)
	biA.Mul(biA, biB)
	biA.Mod(biA, biMod)
	return biA.Uint64()
}
