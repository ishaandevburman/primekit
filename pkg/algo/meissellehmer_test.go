package algo

import (
	"context"
	"math"
	"testing"
)

func TestMeisselLehmerSmall(t *testing.T) {
	known := map[uint64]uint64{
		0:      0,
		1:      0,
		2:      1,
		3:      2,
		4:      2,
		10:     4,
		100:    25,
		1000:   168,
		10000:  1229,
		100000: 9592,
	}
	for x, want := range known {
		got, err := MeisselLehmer(context.Background(), x)
		if err != nil {
			t.Fatalf("MeisselLehmer(%d): %v", x, err)
		}
		if got != want {
			t.Fatalf("MeisselLehmer(%d) = %d, want %d", x, got, want)
		}
	}
}

func TestMeisselLehmerAgainstLegendre(t *testing.T) {
	limits := []uint64{0, 1, 10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000, 500000}
	for _, limit := range limits {
		ml, mlErr := MeisselLehmer(context.Background(), limit)
		lg, lgErr := LegendrePhi(context.Background(), limit)
		if mlErr != nil {
			t.Fatalf("MeisselLehmer(%d): %v", limit, mlErr)
		}
		if lgErr != nil {
			t.Fatalf("LegendrePhi(%d): %v", limit, lgErr)
		}
		if ml != lg {
			t.Fatalf("limit=%d: MeisselLehmer=%d Legendre=%d mismatch", limit, ml, lg)
		}
	}
}

func TestMeisselLehmerMedium(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping medium test in short mode")
	}

	// Known π(x) values from OEIS A000720
	known := map[uint64]uint64{
		1000000:   78498,
		2000000:   148933,
		5000000:   348513,
		10000000:  664579,
		20000000:  1270607,
		50000000:  3001134,
		100000000: 5761455,
	}
	for x, want := range known {
		got, err := MeisselLehmer(context.Background(), x)
		if err != nil {
			t.Fatalf("MeisselLehmer(%d): %v", x, err)
		}
		if got != want {
			t.Fatalf("MeisselLehmer(%d) = %d, want %d", x, got, want)
		}
	}
}

func TestMeisselLehmerLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large test in short mode")
	}

	known := map[uint64]uint64{
		500000000:  26355867,
		1000000000: 50847534,
		2000000000: 98222287,
		5000000000: 234954223,
		10000000000: 455052511,
	}
	for x, want := range known {
		got, err := MeisselLehmer(context.Background(), x)
		if err != nil {
			t.Fatalf("MeisselLehmer(%d): %v", x, err)
		}
		if got != want {
			t.Fatalf("MeisselLehmer(%d) = %d, want %d", x, got, want)
		}
	}
}

func TestMeisselLehmerVeryLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping very large test in short mode")
	}

	// Known π(x) from OEIS
	known := map[uint64]uint64{
		uint64(1e11): 4118054813,
		uint64(1e12): 37607912018,
	}
	for x, want := range known {
		got, err := MeisselLehmer(context.Background(), x)
		if err != nil {
			t.Fatalf("MeisselLehmer(%d): %v", x, err)
		}
		if got != want {
			t.Fatalf("MeisselLehmer(%d) = %d, want %d", x, got, want)
		}
	}
}

func TestMeisselLehmerConsistency(t *testing.T) {
	limits := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 100, 1000, 10000}
	for _, x := range limits {
		ml, mlErr := MeisselLehmer(context.Background(), x)
		if mlErr != nil {
			t.Fatalf("MeisselLehmer(%d): %v", x, mlErr)
		}
		prev, prevErr := MeisselLehmer(context.Background(), x-1)
		if prevErr != nil {
			t.Fatalf("MeisselLehmer(%d): %v", x-1, prevErr)
		}
		diff := ml - prev
		isPrime := diff == 1
		expectedPrime := false
		for _, p := range []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29} {
			if x == p {
				expectedPrime = true
				break
			}
		}
		if expectedPrime && x > 29 {
			// check if x is prime by trial division
			lim := uint64(math.Sqrt(float64(x)))
			expectedPrime = true
			for d := uint64(2); d <= lim; d++ {
				if x%d == 0 {
					expectedPrime = false
					break
				}
			}
		}
		if isPrime != expectedPrime {
			t.Fatalf("π(%d)-π(%d)=%d but %d is %v prime", x, x-1, diff, x, map[bool]string{true: "", false: "not "}[expectedPrime])
		}
	}
}

func TestMeisselLehmerContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := MeisselLehmer(ctx, 1000000)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func BenchmarkMeisselLehmer(b *testing.B) {
	limits := []uint64{uint64(1e8), uint64(1e9), uint64(1e10), uint64(1e11), uint64(1e12)}
	for _, limit := range limits {
		b.Run(sprintML(limit), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				MeisselLehmer(context.Background(), limit)
			}
		})
	}
}

func sprintML(n uint64) string {
	switch {
	case n >= 1e12:
		return "1e12"
	case n >= 1e11:
		return "1e11"
	case n >= 1e10:
		return "1e10"
	case n >= 1e9:
		return "1e9"
	default:
		return "1e8"
	}
}
