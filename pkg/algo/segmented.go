package algo

import (
	"context"
	"math"
)

type SegmentedSieve struct {
	segmentSize uint64
	name        string
	OnProgress  ProgressFunc
}

func NewSegmentedSieve(segmentSize uint64) *SegmentedSieve {
	if segmentSize == 0 {
		segmentSize = 1 << 20
	}
	return &SegmentedSieve{segmentSize: segmentSize}
}

func (s *SegmentedSieve) Name() string {
	if s.name != "" {
		return s.name
	}
	return "segmented-sieve"
}

func (s *SegmentedSieve) NthPrime(ctx context.Context, n uint64) (uint64, error) {
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
		var count uint64
		s.PrimesInRange(ctx, 2, upper, newCountWriter(&count))
		if count > n {
			var nth uint64
			var idx uint64
			s.PrimesInRange(ctx, 2, upper, newNthWriter(n, &idx, &nth))
			return nth, nil
		}
		upper *= 2
	}
}

func (s *SegmentedSieve) Primes(ctx context.Context, limit uint64, out chan<- uint64) error {
	return s.PrimesInRange(ctx, 2, limit, out)
}

func (s *SegmentedSieve) PrimesInRange(ctx context.Context, start, end uint64, out chan<- uint64) error {
	defer close(out)
	if end < 2 {
		return nil
	}
	if start < 2 {
		start = 2
	}

	limit := end
	segmentSize := s.segmentSize
	if segmentSize > limit {
		segmentSize = limit
	}
	sqrtLimit := uint64(math.Sqrt(float64(limit)))

	basePrimes := simpleSieve(sqrtLimit)

	totalSegments := int((limit + segmentSize - 1) / segmentSize)
	onProgress := s.OnProgress
	var primesFound uint64
	var segsDone int

	low := uint64(2)
	high := segmentSize

	for low <= limit {
		if high > limit {
			high = limit
		}

		segment := make([]bool, high-low+1)
		for i := range segment {
			segment[i] = true
		}

		for _, p := range basePrimes {
			if p*p > high {
				break
			}
			startVal := ((low + p - 1) / p) * p
			if startVal < p*p {
				startVal = p * p
			}
			for j := startVal; j <= high; j += p {
				segment[j-low] = false
			}
		}

		lastLow := low
		lastHigh := high
		for i, marked := range segment {
			if marked {
				p := low + uint64(i)
				if p >= start {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case out <- p:
					}
					primesFound++
				}
			}
		}

		segsDone++
		if onProgress != nil {
			onProgress(Progress{
				SegmentsDone:  segsDone,
				TotalSegments: totalSegments,
				PrimesFound:   primesFound,
				CurrentLow:    lastLow,
				CurrentHigh:   lastHigh,
				End:           limit,
			})
		}

		low = high + 1
		high += segmentSize
	}
	return nil
}

type countWriter struct {
	count *uint64
}

func newCountWriter(c *uint64) chan<- uint64 {
	* c = 0
	ch := make(chan uint64)
	go func() {
		for range ch {
			(*c)++
		}
	}()
	return ch
}

type nthWriter struct {
	target uint64
	idx    *uint64
	result *uint64
}

func newNthWriter(target uint64, idx *uint64, result *uint64) chan<- uint64 {
	*idx = 0
	*result = 0
	ch := make(chan uint64)
	go func() {
		for p := range ch {
			if *idx == target {
				*result = p
				break
			}
			(*idx)++
		}
		for range ch {
		}
	}()
	return ch
}

func (s *SegmentedSieve) SetSegmentSize(size uint64) {
	if size > 0 {
		s.segmentSize = size
	}
}
