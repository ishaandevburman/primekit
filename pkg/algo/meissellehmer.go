package algo

import (
	"context"
	"math"
)

type MeisselLehmerCounter struct{}

func (m *MeisselLehmerCounter) Name() string { return "meissel-lehmer" }

func (m *MeisselLehmerCounter) CountPrimes(ctx context.Context, limit uint64) (uint64, error) {
	return MeisselLehmer(ctx, limit)
}

func MeisselLehmer(ctx context.Context, x uint64) (uint64, error) {
	if x < 2 {
		return 0, nil
	}
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	sqrtX := uint64(math.Sqrt(float64(x)))
	primes := simpleSieve(sqrtX)

	piSmall := make([]uint64, sqrtX+1)
	idx := 0
	for i := uint64(2); i <= sqrtX; i++ {
		piSmall[i] = piSmall[i-1]
		if idx < len(primes) && primes[idx] == i {
			piSmall[i]++
			idx++
		}
	}

	ml := &meisselLehmer{
		primes:   primes,
		piSmall:  piSmall,
		phiCache: make(map[phiKey]uint64),
		piCache:  make(map[uint64]uint64),
	}

	return uint64(ml.lehmer(x)), nil
}

type meisselLehmer struct {
	primes   []uint64
	piSmall  []uint64
	phiCache map[phiKey]uint64
	piCache  map[uint64]uint64
}

func (ml *meisselLehmer) pi(x uint64) uint64 {
	if x < 2 {
		return 0
	}
	if x < uint64(len(ml.piSmall)) {
		return ml.piSmall[x]
	}
	if v, ok := ml.piCache[x]; ok {
		return v
	}
	v := ml.lehmer(x)
	ml.piCache[x] = v
	return v
}

func (ml *meisselLehmer) phi(x uint64, a int) uint64 {
	if a == 0 {
		return x
	}
	if x == 0 {
		return 0
	}

	switch a {
	case 1:
		return x - x/2
	case 2:
		return x - x/2 - x/3 + x/6
	case 3:
		return x - x/2 - x/3 - x/5 + x/6 + x/10 + x/15 - x/30
	case 4:
		return x - x/2 - x/3 - x/5 - x/7 +
			x/6 + x/10 + x/14 + x/15 + x/21 + x/35 -
			x/30 - x/42 - x/70 - x/105 +
			x/210
	}

	pa := ml.primes[a-1]
	if pa > x {
		return 1
	}

	if x < pa*pa {
		return 1 + ml.pi(x) - uint64(a)
	}

	key := phiKey{x, uint64(a)}
	if v, ok := ml.phiCache[key]; ok {
		return v
	}

	result := ml.phi(x, a-1) - ml.phi(x/pa, a-1)
	ml.phiCache[key] = result
	return result
}

func (ml *meisselLehmer) lehmer(x uint64) uint64 {
	if x < 2 {
		return 0
	}
	if x < uint64(len(ml.piSmall)) {
		return ml.piSmall[x]
	}
	if v, ok := ml.piCache[x]; ok {
		return v
	}

	a := int(ml.pi(uint64(math.Pow(float64(x), 1.0/4.0))))
	b := int(ml.pi(uint64(math.Sqrt(float64(x)))))
	c := int(ml.pi(uint64(math.Cbrt(float64(x)))))

	result := ml.phi(x, a) + uint64(a) - 1

	// Subtract P2: semiprimes with both primes > p_a
	for i := a + 1; i <= b; i++ {
		w := x / ml.primes[i-1]
		result -= ml.pi(w) - uint64(i-1)
	}

	// Subtract P3: numbers with 3 prime factors all > p_a
	for i := a + 1; i <= c; i++ {
		w := x / ml.primes[i-1]
		maxJ := int(ml.pi(uint64(math.Sqrt(float64(w)))))
		for j := i; j <= maxJ; j++ {
			val := w / ml.primes[j-1]
			result -= ml.pi(val) - uint64(j-1)
		}
	}

	ml.piCache[x] = result
	return result
}
