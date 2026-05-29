package algo

import (
	"context"
	"math"
	"sort"
	"sync"
)

type ParallelSegmentedSieve struct {
	segmentSize uint64
	workers     int
}

func NewParallelSegmentedSieve(segmentSize uint64, workers int) *ParallelSegmentedSieve {
	if segmentSize == 0 {
		segmentSize = 1 << 20
	}
	if workers < 1 {
		workers = 1
	}
	return &ParallelSegmentedSieve{segmentSize: segmentSize, workers: workers}
}

func (p *ParallelSegmentedSieve) Name() string { return "parallel-segmented" }

func (p *ParallelSegmentedSieve) SetWorkers(n int) {
	if n > 0 {
		p.workers = n
	}
}

func (p *ParallelSegmentedSieve) NthPrime(ctx context.Context, n uint64) (uint64, error) {
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
		p.PrimesInRange(ctx, 2, upper, newCountWriter(&count))
		if count > n {
			var nth uint64
			var idx uint64
			p.PrimesInRange(ctx, 2, upper, newNthWriter(n, &idx, &nth))
			return nth, nil
		}
		upper *= 2
	}
}

func (p *ParallelSegmentedSieve) Primes(ctx context.Context, limit uint64, out chan<- uint64) error {
	return p.PrimesInRange(ctx, 2, limit, out)
}

func (p *ParallelSegmentedSieve) PrimesInRange(ctx context.Context, start, end uint64, out chan<- uint64) error {
	defer close(out)
	if end < 2 {
		return nil
	}
	if start < 2 {
		start = 2
	}

	sqrtLimit := uint64(math.Sqrt(float64(end)))
	basePrimes := simpleSieve(sqrtLimit)

	segSize := p.segmentSize
	if segSize > end {
		segSize = end
	}

	numWorkers := p.workers
	if numWorkers > 1 && end > segSize*uint64(numWorkers)*2 {
		numWorkers = p.workers
	} else {
		numWorkers = 1
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	segCh := make(chan segmentJob, numWorkers*2)
	results := make(chan uint64, 65536)

	outputDone := make(chan struct{})
	go func() {
		for p := range results {
			select {
			case <-ctx.Done():
				return
			case out <- p:
			}
		}
		close(outputDone)
	}()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range segCh {
				select {
				case <-ctx.Done():
					return
				default:
				}

				primes := sieveSegment(job.low, job.high, basePrimes)
				mu.Lock()
				for _, p := range primes {
					if p >= start {
						results <- p
					}
				}
				mu.Unlock()
			}
		}()
	}

	low := uint64(2)
	high := segSize
	for low <= end {
		select {
		case <-ctx.Done():
			close(segCh)
			wg.Wait()
			close(results)
			<-outputDone
			return ctx.Err()
		default:
		}
		if high > end {
			high = end
		}
		segCh <- segmentJob{low: low, high: high}
		low = high + 1
		high += segSize
	}

	close(segCh)
	wg.Wait()
	close(results)
	<-outputDone
	return nil
}

type segmentJob struct {
	low  uint64
	high uint64
}

func sieveSegment(low, high uint64, basePrimes []uint64) []uint64 {
	size := int(high - low + 1)
	segment := make([]bool, size)
	for i := range segment {
		segment[i] = true
	}
	for _, p := range basePrimes {
		if p*p > high {
			break
		}
		start := ((low + p - 1) / p) * p
		if start < p*p {
			start = p * p
		}
		for j := start; j <= high; j += p {
			segment[j-low] = false
		}
	}
	primes := make([]uint64, 0, size/10)
	for i, marked := range segment {
		if marked {
			primes = append(primes, low+uint64(i))
		}
	}
	return primes
}

func mergeSortedSlices(slices [][]uint64) []uint64 {
	if len(slices) == 0 {
		return nil
	}
	if len(slices) == 1 {
		return slices[0]
	}

	// Flatten and sort (all slices are already individually sorted)
	var total int
	for _, s := range slices {
		total += len(s)
	}
	result := make([]uint64, 0, total)
	for _, s := range slices {
		result = append(result, s...)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
