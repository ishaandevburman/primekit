package algo

import (
	"context"
	"fmt"
	"sort"
	"time"
)

type BenchmarkResult struct {
	Algorithm   string
	N           uint64
	Limit       uint64
	Elapsed     time.Duration
	PrimesFound uint64
	Output      uint64
	Error       error
}

type BenchmarkSuite struct {
	Algorithms []NamedGenerator
}

type NamedGenerator interface {
	Name() string
	NthPrime(ctx context.Context, n uint64) (uint64, error)
	Primes(ctx context.Context, limit uint64, out chan<- uint64) error
}

func NewBenchmarkSuite() *BenchmarkSuite {
	par := NewParallelSegmentedSieve(1<<20, 4)
	return &BenchmarkSuite{
		Algorithms: []NamedGenerator{
			&SimpleSieve{},
			&SegmentedSieve{segmentSize: 1 << 16, name: "segmented-64k"},
			&SegmentedSieve{segmentSize: 1 << 20, name: "segmented-1m"},
			&SegmentedSieve{segmentSize: 1 << 24, name: "segmented-16m"},
			par,
		},
	}
}

func (b *BenchmarkSuite) RunNthPrime(ctx context.Context, ns []uint64) []BenchmarkResult {
	var results []BenchmarkResult
	for _, algo := range b.Algorithms {
		for _, n := range ns {
			select {
			case <-ctx.Done():
				return results
			default:
			}
			start := time.Now()
			p, err := algo.NthPrime(ctx, n)
			elapsed := time.Since(start)
			results = append(results, BenchmarkResult{
				Algorithm: algo.Name(),
				N:         n,
				Elapsed:   elapsed,
				Output:    p,
				Error:     err,
			})
		}
	}
	return results
}

func (b *BenchmarkSuite) RunSieve(ctx context.Context, limits []uint64) []BenchmarkResult {
	var results []BenchmarkResult
	for _, algo := range b.Algorithms {
		for _, limit := range limits {
			select {
			case <-ctx.Done():
				return results
			default:
			}
			start := time.Now()
			ch := make(chan uint64, 65536)
			var count uint64
			var last uint64
			done := make(chan struct{})
			go func() {
				for p := range ch {
					count++
					last = p
				}
				close(done)
			}()
			err := algo.Primes(ctx, limit, ch)
			<-done
			elapsed := time.Since(start)
			results = append(results, BenchmarkResult{
				Algorithm:   algo.Name(),
				Limit:       limit,
				Elapsed:     elapsed,
				PrimesFound: count,
				Output:      last,
				Error:       err,
			})
		}
	}
	return results
}

func (b *BenchmarkSuite) Summary(results []BenchmarkResult) string {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Algorithm != results[j].Algorithm {
			return results[i].Algorithm < results[j].Algorithm
		}
		return results[i].N+results[i].Limit < results[j].N+results[j].Limit
	})

	s := "\n"
	for _, r := range results {
		label := fmt.Sprintf("n=%-12d", r.N)
		if r.Limit > 0 {
			label = fmt.Sprintf("≤ %-11d", r.Limit)
		}
		s += fmt.Sprintf("  %-28s %s  %12s",
			r.Algorithm, label, r.Elapsed.Round(time.Microsecond))
		if r.Error != nil {
			s += fmt.Sprintf("  ERROR: %v", r.Error)
		}
		s += "\n"
	}
	return s
}
