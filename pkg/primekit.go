package primekit

import "context"

type Generator interface {
	Name() string
	NthPrime(ctx context.Context, n uint64) (uint64, error)
	Primes(ctx context.Context, limit uint64, out chan<- uint64) error
	PrimesInRange(ctx context.Context, start, end uint64, out chan<- uint64) error
}

type Counter interface {
	Name() string
	CountPrimes(ctx context.Context, limit uint64) (uint64, error)
}

type Factorizer interface {
	Name() string
	Factor(ctx context.Context, n uint64) ([]uint64, error)
}

type PrimalityTester interface {
	Name() string
	IsPrime(ctx context.Context, n uint64) bool
}

type StorageBackend interface {
	Store(ctx context.Context, primes []uint64) error
	Contains(ctx context.Context, n uint64) (bool, error)
	Close() error
}
