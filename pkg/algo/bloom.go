package algo

import (
	"context"
	"math"
)

type BloomFilter struct {
	bits  []uint64
	k     int
	n     uint64
	m     uint64
	limit uint64
}

func NewBloomFilter(expectedItems uint64, falsePositiveRate float64) *BloomFilter {
	m := uint64(math.Ceil(-float64(expectedItems) * math.Log(falsePositiveRate) / (math.Ln2 * math.Ln2)))
	m = (m + 63) / 64 * 64
	k := int(math.Ceil(math.Ln2 * float64(m) / float64(expectedItems)))
	if k < 1 {
		k = 1
	}
	return &BloomFilter{
		bits: make([]uint64, m/64),
		k:    k,
		m:    m,
	}
}

func (b *BloomFilter) Add(n uint64) {
	b.n++
	for i := 0; i < b.k; i++ {
		h := hash(n, uint64(i)) % b.m
		b.bits[h/64] |= 1 << (h % 64)
	}
}

func (b *BloomFilter) MaybeContains(n uint64) bool {
	for i := 0; i < b.k; i++ {
		h := hash(n, uint64(i)) % b.m
		if b.bits[h/64]&(1<<(h%64)) == 0 {
			return false
		}
	}
	return true
}

func hash(n, seed uint64) uint64 {
	h := n
	h ^= seed
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

type BloomPrimality struct {
	filter *BloomFilter
	base   uint64
}

func NewBloomPrimality(limit uint64) *BloomPrimality {
	approxPrimes := uint64(float64(limit) / math.Log(float64(limit)))
	bp := &BloomPrimality{
		filter: NewBloomFilter(approxPrimes, 0.001),
		base:   uint64(math.Sqrt(float64(limit))),
	}
	// Seed with base primes
	basePrimes := simpleSieve(bp.base)
	for _, p := range basePrimes {
		bp.filter.Add(p)
	}
	return bp
}

func (b *BloomPrimality) IsPrime(ctx context.Context, n uint64) bool {
	if n < 2 {
		return false
	}
	if n <= b.base {
		return isPrimeSqrt(n)
	}
	if !b.filter.MaybeContains(n) {
		return false
	}
	mr := &MillerRabin{}
	return mr.IsPrime(ctx, n)
}
