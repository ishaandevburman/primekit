package algo

import (
	"context"
)

type Gap struct {
	PrevPrime uint64
	Prime     uint64
	Size      uint64
}

type GapFinder struct{}

func (g *GapFinder) Name() string { return "prime-gaps" }

func (g *GapFinder) FindGaps(ctx context.Context, limit uint64, out chan<- Gap) error {
	defer close(out)
	sieve := &SegmentedSieve{segmentSize: 1 << 20}
	ch := make(chan uint64, 4096)

	go func() {
		sieve.Primes(ctx, limit, ch)
	}()

	var prev uint64
	first := true
	for p := range ch {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if first {
			prev = p
			first = false
			continue
		}
		gap := p - prev
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- Gap{PrevPrime: prev, Prime: p, Size: gap}:
		}
		prev = p
	}
	return nil
}

func (g *GapFinder) MaxGap(ctx context.Context, limit uint64) (Gap, error) {
	var maxGap Gap
	ch := make(chan Gap, 64)
	errCh := make(chan error, 1)

	go func() {
		errCh <- g.FindGaps(ctx, limit, ch)
	}()

	for gap := range ch {
		if gap.Size > maxGap.Size {
			maxGap = gap
		}
	}

	if err := <-errCh; err != nil {
		return Gap{}, err
	}
	return maxGap, nil
}

func (g *GapFinder) TwinPrimes(ctx context.Context, limit uint64, out chan<- uint64) error {
	defer close(out)
	sieve := &SegmentedSieve{segmentSize: 1 << 20}
	ch := make(chan uint64, 4096)

	go func() {
		sieve.Primes(ctx, limit, ch)
	}()

	var prev uint64
	for p := range ch {
		if prev != 0 && p-prev == 2 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- prev:
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- p:
			}
		}
		prev = p
	}
	return nil
}
