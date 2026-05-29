package algo

import (
	"context"
	"math"
	"sync"
)

type ParallelSegmentedSieve struct {
	segmentSize uint64
	workers     int
	OnProgress  ProgressFunc
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

type segmentResult struct {
	index  int
	primes []uint64
	low    uint64
	high   uint64
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

	segCh := make(chan segmentJob, numWorkers*2)
	resultCh := make(chan segmentResult, numWorkers*2)

	var wg sync.WaitGroup

	// Feed segments
	go func() {
		defer close(segCh)
		low := uint64(2)
		high := segSize
		index := 0
		for low <= end {
			if high > end {
				high = end
			}
			select {
			case <-ctx.Done():
				return
			case segCh <- segmentJob{low: low, high: high, index: index}:
			}
			low = high + 1
			high += segSize
			index++
		}
	}()

	// Workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range segCh {
				primes := sieveSegment(job.low, job.high, basePrimes)
				select {
				case <-ctx.Done():
					return
				case resultCh <- segmentResult{index: job.index, primes: primes, low: job.low, high: job.high}:
				}
			}
		}()
	}

	// Close resultCh when all workers finish
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Merger: runs in this goroutine, outputs in segment order
	totalSegments := int((end + segSize - 1) / segSize)
	onProgress := p.OnProgress
	var primesFound uint64
	nextIdx := 0
	pending := make(map[int][]uint64)

	for r := range resultCh {
		if r.index == nextIdx {
			for _, pr := range r.primes {
				if pr >= start {
					out <- pr
					primesFound++
				}
			}
			nextIdx++
			for {
				if pr, ok := pending[nextIdx]; ok {
					for _, p := range pr {
						if p >= start {
							out <- p
							primesFound++
						}
					}
					delete(pending, nextIdx)
					nextIdx++
				} else {
					break
				}
			}
			if onProgress != nil {
				onProgress(Progress{
					SegmentsDone:  nextIdx,
					TotalSegments: totalSegments,
					PrimesFound:   primesFound,
					CurrentLow:    r.low,
					CurrentHigh:   r.high,
					End:           end,
				})
			}
		} else {
			pending[r.index] = r.primes
		}
	}

	return nil
}

type segmentJob struct {
	low   uint64
	high  uint64
	index int
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
