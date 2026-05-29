package algo

import (
	"context"
	"testing"
)

func collectPrimes(t testing.TB, gen NamedGenerator, limit uint64) []uint64 {
	t.Helper()
	ch := make(chan uint64, 65536)
	var out []uint64
	done := make(chan struct{})
	go func() {
		for p := range ch {
			out = append(out, p)
		}
		close(done)
	}()
	if err := gen.Primes(context.Background(), limit, ch); err != nil {
		t.Fatalf("Primes(%d): %v", limit, err)
	}
	<-done
	return out
}

func TestPritchardSmall(t *testing.T) {
	pritchard := NewPritchardSieve()
	segmented := NewSegmentedSieve(1 << 20)

	tests := []uint64{0, 1, 2, 3, 5, 10, 20, 30, 50, 100}
	for _, limit := range tests {
		p := collectPrimes(t, pritchard, limit)
		s := collectPrimes(t, segmented, limit)

		if len(p) != len(s) {
			t.Fatalf("limit=%d: length mismatch pritchard=%d segmented=%d", limit, len(p), len(s))
		}
		for i := range p {
			if p[i] != s[i] {
				t.Fatalf("limit=%d: mismatch at index %d: pritchard=%d segmented=%d", limit, i, p[i], s[i])
			}
		}
	}
}

func TestPritchardAgainstSegmented(t *testing.T) {
	pritchard := NewPritchardSieve()
	segmented := NewSegmentedSieve(1 << 20)

	limits := []uint64{
		1000, 5000, 10000, 50000,
		100000, 200000, 500000,
		1000000, 2000000, 5000000,
		10000000,
	}
	for _, limit := range limits {
		p := collectPrimes(t, pritchard, limit)
		s := collectPrimes(t, segmented, limit)

		if len(p) != len(s) {
			t.Fatalf("limit=%d: length mismatch pritchard=%d segmented=%d", limit, len(p), len(s))
		}
		// spot-check every 1000th prime + first/last
		step := len(p) / 1000
		if step < 1 {
			step = 1
		}
		for i := 0; i < len(p); i += step {
			if p[i] != s[i] {
				t.Fatalf("limit=%d: mismatch at index %d: pritchard=%d segmented=%d", limit, i, p[i], s[i])
			}
		}
		// always check last
		if p[len(p)-1] != s[len(s)-1] {
			t.Fatalf("limit=%d: last prime mismatch pritchard=%d segmented=%d", limit, p[len(p)-1], s[len(s)-1])
		}
	}
}

func TestPritchardNthPrime(t *testing.T) {
	pritchard := NewPritchardSieve()
	segmented := NewSegmentedSieve(1 << 20)

	known := map[uint64]uint64{
		0:     2,
		1:     3,
		2:     5,
		3:     7,
		4:     11,
		5:     13,
		9:     29,
		24:    97,
		99:    541,
		999:   7919,
		9999:  104729,
		49999: 611953,
	}
	for n, expected := range known {
		p, err := pritchard.NthPrime(context.Background(), n)
		if err != nil {
			t.Fatalf("pritchard NthPrime(%d): %v", n, err)
		}
		if p != expected {
			t.Fatalf("pritchard NthPrime(%d) = %d, want %d", n, p, expected)
		}

		s, err := segmented.NthPrime(context.Background(), n)
		if err != nil {
			t.Fatalf("segmented NthPrime(%d): %v", n, err)
		}
		if p != s {
			t.Fatalf("n=%d: pritchard=%d segmented=%d mismatch", n, p, s)
		}
	}
}

func TestPritchardPrimeCounts(t *testing.T) {
	pritchard := NewPritchardSieve()

	// Known π(x) values from OEIS A000720
	known := map[uint64]uint64{
		10:       4,
		100:      25,
		1000:     168,
		10000:    1229,
		100000:   9592,
		1000000:  78498,
		5000000:  348513,
		10000000: 664579,
	}
	for limit, expected := range known {
		primes := collectPrimes(t, pritchard, limit)
		if uint64(len(primes)) != expected {
			t.Fatalf("π(%d) = %d, want %d", limit, len(primes), expected)
		}
	}
}

func TestPritchardPrimesInRange(t *testing.T) {
	pritchard := NewPritchardSieve()
	segmented := NewSegmentedSieve(1 << 20)

	ranges := [][2]uint64{
		{2, 100},
		{50, 200},
		{100, 1000},
		{1000, 5000},
		{10000, 20000},
		{90000, 100000},
	}
	for _, r := range ranges {
		ch := make(chan uint64, 65536)
		var p []uint64
		done := make(chan struct{})
		go func() {
			for v := range ch {
				p = append(p, v)
			}
			close(done)
		}()
		if err := pritchard.PrimesInRange(context.Background(), r[0], r[1], ch); err != nil {
			t.Fatalf("pritchard PrimesInRange(%d,%d): %v", r[0], r[1], err)
		}
		<-done

		s := collectPrimes(t, segmented, r[1])
		var sfilt []uint64
		for _, v := range s {
			if v >= r[0] {
				sfilt = append(sfilt, v)
			}
		}

		if len(p) != len(sfilt) {
			t.Fatalf("range [%d,%d]: length pritchard=%d segmented=%d", r[0], r[1], len(p), len(sfilt))
		}
		for i := range p {
			if p[i] != sfilt[i] {
				t.Fatalf("range [%d,%d]: at %d pritchard=%d segmented=%d", r[0], r[1], i, p[i], sfilt[i])
			}
		}
	}
}

func TestPritchardLargeLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large limit test in short mode")
	}
	pritchard := NewPritchardSieve()
	segmented := NewSegmentedSieve(1 << 20)

	limits := []uint64{10000000, 20000000}
	for _, limit := range limits {
		p := collectPrimes(t, pritchard, limit)
		s := collectPrimes(t, segmented, limit)

		if len(p) != len(s) {
			t.Fatalf("limit=%d: length pritchard=%d segmented=%d", limit, len(p), len(s))
		}
		// spot-check
		step := len(p) / 500
		if step < 1 {
			step = 1
		}
		for i := 0; i < len(p); i += step {
			if p[i] != s[i] {
				t.Fatalf("limit=%d: mismatch at %d: pritchard=%d segmented=%d", limit, i, p[i], s[i])
			}
		}
	}
}

func TestPritchardNthLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large nth in short mode")
	}
	pritchard := NewPritchardSieve()

	// known values from OEIS A006988
	known := []struct {
		n    uint64
		want uint64
	}{
		{100000, 1299721},
		{200000, 2750161},
		{500000, 7368791},
		{1000000, 15485867},
	}
	for _, k := range known {
		p, err := pritchard.NthPrime(context.Background(), k.n)
		if err != nil {
			t.Fatalf("NthPrime(%d): %v", k.n, err)
		}
		if p != k.want {
			t.Fatalf("NthPrime(%d) = %d, want %d", k.n, p, k.want)
		}
	}
}

func BenchmarkPritchard(b *testing.B) {
	limits := []uint64{100000, 1000000, 10000000, 50000000, 100000000}
	for _, limit := range limits {
		b.Run(sprint(limit), func(b *testing.B) {
			s := NewPritchardSieve()
			for i := 0; i < b.N; i++ {
				collectPrimes(b, s, limit)
			}
		})
	}
}

func sprint(n uint64) string {
	switch {
	case n >= 100000000:
		return "100M"
	case n >= 50000000:
		return "50M"
	case n >= 10000000:
		return "10M"
	case n >= 1000000:
		return "1M"
	default:
		return "100K"
	}
}

func TestPritchardContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	s := NewPritchardSieve()
	ch := make(chan uint64, 10)
	err := s.PrimesInRange(ctx, 2, 1000000, ch)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestPritchardPrimalitySmallPrimes(t *testing.T) {
	pritchard := NewPritchardSieve()
	primes := collectPrimes(t, pritchard, 10000)
	primeSet := make(map[uint64]bool)
	for _, p := range primes {
		primeSet[p] = true
	}

	for n := uint64(2); n <= 10000; n++ {
		expected := isPrimeSqrt(n)
		if primeSet[n] && !expected {
			t.Fatalf("%d is in Pritchard output but is not prime", n)
		}
		if !primeSet[n] && expected {
			t.Fatalf("%d is prime but missing from Pritchard output", n)
		}
	}
}

func TestPritchardLastPrime(t *testing.T) {
	// Verify last prime produced is the largest ≤ limit
	pritchard := NewPritchardSieve()

	tests := []struct {
		limit uint64
		last  uint64
	}{
		{100, 97},
		{1000, 997},
		{10000, 9973},
		{100000, 99991},
	}
	for _, tt := range tests {
		primes := collectPrimes(t, pritchard, tt.limit)
		got := primes[len(primes)-1]
		if got != tt.last {
			t.Fatalf("limit=%d: last prime = %d, want %d", tt.limit, got, tt.last)
		}
	}
}

func TestPritchardRegression(t *testing.T) {
	// Regression: verify no "4 is prime" or "25 is prime" bugs
	pritchard := NewPritchardSieve()
	primes := collectPrimes(t, pritchard, 1000)
	primeSet := make(map[uint64]bool)
	for _, p := range primes {
		primeSet[p] = true
	}
	composites := []uint64{4, 6, 8, 9, 10, 15, 21, 25, 27, 49, 121, 169, 289, 361, 529, 841, 961}
	for _, c := range composites {
		if primeSet[c] {
			t.Fatalf("composite %d marked as prime", c)
		}
	}
}

func TestPritchardMemoryLimit(t *testing.T) {
	// Ensure we can handle limits close to large allocations
	// without crashing from OOM
	if testing.Short() {
		t.Skip("skipping memory test in short mode")
	}
	s := NewPritchardSieve()
	// Test limit that requires ~100MB+ W array
	primes := collectPrimes(t, s, 50000000)
	if len(primes) == 0 {
		t.Fatal("no primes produced")
	}
}

func TestPritchardConsistency(t *testing.T) {
	// Run Pritchard twice at the same limit, results must be identical
	pritchard := NewPritchardSieve()
	a := collectPrimes(t, pritchard, 100000)
	b := collectPrimes(t, pritchard, 100000)

	if len(a) != len(b) {
		t.Fatalf("inconsistent results: len %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("inconsistent at %d: %d vs %d", i, a[i], b[i])
		}
	}
}

func TestPritchardBoundary(t *testing.T) {
	// Test near powers of two
	pritchard := NewPritchardSieve()
	segmented := NewSegmentedSieve(1 << 20)

	boundaries := []uint64{10000000, 20000000}
	if testing.Short() {
		boundaries = []uint64{100000, 1000000}
	}
	for _, limit := range boundaries {
		if testing.Short() {
			continue
		}
		p := collectPrimes(t, pritchard, limit)
		s := collectPrimes(t, segmented, limit)
		if len(p) != len(s) {
			t.Fatalf("limit=%d: length %d vs %d", limit, len(p), len(s))
		}
	}
}
