package algo

import (
	"context"
	"math"
)

const (
	wheelMod     = 210
	wheelProdStr = "2·3·5·7"
)

var (
	wheelPrimes   = []uint64{2, 3, 5, 7}
	wheelResidues = computeResidues()
	wheelResIndex [wheelMod]int // wheelResIndex[r] = position in wheelResidues, -1 if not coprime
)

func init() {
	for i := range wheelResIndex {
		wheelResIndex[i] = -1
	}
	for i, r := range wheelResidues {
		wheelResIndex[r] = i
	}
}

func computeResidues() []uint64 {
	sieve := make([]bool, wheelMod)
	for i := uint64(2); i < wheelMod; i++ {
		sieve[i] = true
	}
	sieve[0] = false // 0 is divisible by all primes
	sieve[1] = true  // 1 is coprime to 210
	for _, p := range wheelPrimes {
		for j := p; j < wheelMod; j += p {
			sieve[j] = false
		}
	}
	var res []uint64
	for i := uint64(1); i < wheelMod; i++ {
		if sieve[i] {
			res = append(res, i)
		}
	}
	return res
}

type WheelSegmentedSieve struct {
	segmentSize uint64
}

func NewWheelSegmentedSieve(segmentSize uint64) *WheelSegmentedSieve {
	if segmentSize == 0 {
		segmentSize = 1 << 20
	}
	// round segment size down to multiple of wheelMod
	segmentSize = (segmentSize / wheelMod) * wheelMod
	if segmentSize < wheelMod {
		segmentSize = wheelMod
	}
	return &WheelSegmentedSieve{segmentSize: segmentSize}
}

func (w *WheelSegmentedSieve) Name() string { return "wheel-210" }

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
	if end < wheelMod {
		simple := &SimpleSieve{}
		return simple.PrimesInRange(ctx, start, end, out)
	}

	limit := end
	sqrtLimit := uint64(math.Sqrt(float64(limit)))
	basePrimes := simpleSieve(sqrtLimit)

	// Handle the pre-wheel region (< wheelMod) with the simple sieve
	if start < wheelMod {
		preEnd := uint64(wheelMod - 1)
		if preEnd > limit {
			preEnd = limit
		}
		for i := start; i <= preEnd; i++ {
			if isPrimeSqrt(i) {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case out <- i:
				}
			}
		}
		if limit < wheelMod {
			return nil
		}
	}

	// Align low to the next wheel boundary
	low := ((start + wheelMod - 1) / wheelMod) * wheelMod
	if low < wheelMod {
		low = wheelMod
	}
	residueCnt := len(wheelResidues)

	for low <= limit {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		high := low + w.segmentSize - 1
		if high > limit {
			high = limit
		}

		numBlocks := (high-low)/wheelMod + 1
		segLen := int(numBlocks) * residueCnt
		segment := make([]bool, segLen)
		for i := range segment {
			segment[i] = true
		}

		for _, bp := range basePrimes {
			if bp <= 7 {
				continue
			}
			if bp*bp > high {
				break
			}

			first := ((low + bp - 1) / bp) * bp
			for j := first; j <= high; j += bp {
				r := j % wheelMod
				pos := wheelResIndex[r]
				if pos >= 0 {
					block := (j - low) / wheelMod
					segment[int(block)*residueCnt+pos] = false
				}
			}
		}

		for k, marked := range segment {
			if marked {
				block := k / residueCnt
				ridx := k % residueCnt
				p := low + uint64(block)*wheelMod + wheelResidues[ridx]
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
	}
	return nil
}
