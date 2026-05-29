package algo

import (
	"context"
	"math"
)

var wheel210 = struct {
	primes   []uint64
	product  uint64
	offsets  []uint64
	offsetSz int
}{
	primes:  []uint64{2, 3, 5, 7},
	product: 210,
}

func init() {
	wheel210.offsets = computeWheelOffsets(wheel210.primes, wheel210.product)
	wheel210.offsetSz = len(wheel210.offsets)
}

func computeWheelOffsets(primes []uint64, product uint64) []uint64 {
	sieve := make([]bool, product)
	for i := uint64(0); i < product; i++ {
		sieve[i] = true
	}
	for _, p := range primes {
		for j := p; j < product; j += p {
			sieve[j] = false
		}
	}
	sieve[1] = false
	var offsets []uint64
	for i := uint64(1); i < product; i++ {
		if sieve[i] {
			offsets = append(offsets, i)
		}
	}
	return offsets
}

type WheelSegmentedSieve struct {
	segmentSize uint64
}

func NewWheelSegmentedSieve(segmentSize uint64) *WheelSegmentedSieve {
	if segmentSize == 0 {
		segmentSize = 1 << 20
	}
	return &WheelSegmentedSieve{segmentSize: segmentSize}
}

func (w *WheelSegmentedSieve) Name() string { return "wheel-segmented-210" }

func (w *WheelSegmentedSieve) SetSegmentSize(size uint64) {
	if size > 0 {
		w.segmentSize = size
	}
}

func (w *WheelSegmentedSieve) NthPrime(ctx context.Context, n uint64) (uint64, error) {
	if n == 0 {
		return 2, nil
	}
	if n <= 4 {
		return []uint64{3, 5, 7, 11}[n-1], nil
	}
	upper := estimateUpperBound(n)
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
		var count uint64
		w.PrimesInRange(ctx, 2, upper, newCountWriter(&count))
		if count > n {
			var nth uint64
			var idx uint64
			w.PrimesInRange(ctx, 2, upper, newNthWriter(n, &idx, &nth))
			return nth, nil
		}
		upper *= 2
	}
}

func (w *WheelSegmentedSieve) Primes(ctx context.Context, limit uint64, out chan<- uint64) error {
	return w.PrimesInRange(ctx, 2, limit, out)
}

func (w *WheelSegmentedSieve) PrimesInRange(ctx context.Context, start, end uint64, out chan<- uint64) error {
	defer close(out)
	if end < 2 {
		return nil
	}
	if start < 2 {
		start = 2
	}

	if end < wheel210.product {
		simple := &SimpleSieve{}
		return simple.PrimesInRange(ctx, start, end, out)
	}

	for _, p := range wheel210.primes {
		if p >= start {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- p:
			}
		}
	}
	if start > wheel210.product {
		goto mainSieve
	}
	start = wheel210.product + 1

mainSieve:
	limit := end
	segSize := w.segmentSize
	if segSize > limit {
		segSize = limit
	}
	sqrtLimit := uint64(math.Sqrt(float64(limit)))
	basePrimes := simpleSieve(sqrtLimit)

	low := start
	high := start + segSize - 1
	if high > limit {
		high = limit
	}

	for low <= limit {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		segSizeActual := high - low + 1
		segment := make([]bool, segSizeActual)
		for i := range segment {
			segment[i] = true
		}

		skipSmall := low/wheel210.product

		for _, bp := range basePrimes {
			if bp <= 7 {
				continue
			}
			if bp*bp > high {
				break
			}

			var first uint64
			r := low % bp
			if r == 0 {
				first = low
			} else {
				first = low + bp - r
			}
			startVal := first
			for startVal < (skipSmall+1)*wheel210.product {
				startVal += bp
			}
			for j := startVal; j <= high; j += bp {
				segment[j-low] = false
			}
		}

		for i, marked := range segment {
			if marked {
				p := low + uint64(i)
				if p >= start && p <= limit {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case out <- p:
					}
				}
			}
		}

		low = high + 1
		high += segSize
		if high > limit {
			high = limit
		}
	}
	return nil
}
