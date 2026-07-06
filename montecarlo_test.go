package surface

import "testing"

func TestLogicalFailuresDeterministicAcrossWorkerCounts(t *testing.T) {
	l, err := NewLattice(5)
	if err != nil {
		t.Fatalf("NewLattice(5): %v", err)
	}
	dec := NewUnionFindDecoder(l)
	const trials = 500
	var want int
	for i, workers := range []int{1, 2, 7, 16, 1000} {
		got, err := LogicalFailures(l, dec, 0.12, trials, workers, 42)
		if err != nil {
			t.Fatalf("LogicalFailures(workers=%d): %v", workers, err)
		}
		if i == 0 {
			want = got
			if want == 0 || want == trials {
				t.Fatalf("degenerate failure count %d/%d at p=0.12; test needs a mid-range point", want, trials)
			}
			continue
		}
		if got != want {
			t.Errorf("workers=%d: failures = %d, want %d (result must not depend on worker count)", workers, got, want)
		}
	}
}

func TestLogicalFailuresSeedSensitivity(t *testing.T) {
	l, err := NewLattice(3)
	if err != nil {
		t.Fatalf("NewLattice(3): %v", err)
	}
	dec := NewUnionFindDecoder(l)
	a, err := LogicalFailures(l, dec, 0.12, 400, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := LogicalFailures(l, dec, 0.12, 400, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Errorf("same seed gave %d then %d failures", a, b)
	}
}

func TestLogicalFailuresZeroNoise(t *testing.T) {
	l, err := NewLattice(3)
	if err != nil {
		t.Fatalf("NewLattice(3): %v", err)
	}
	got, err := LogicalFailures(l, NewUnionFindDecoder(l), 0, 100, 4, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Errorf("p=0: failures = %d, want 0", got)
	}
}

// TestLogicalFailuresThresholdOrdering is the physics payoff: below
// threshold larger distance suppresses logical errors, above threshold it
// amplifies them. Trial counts and rates are chosen so the gaps are far
// wider than statistical noise for these fixed seeds.
func TestLogicalFailuresThresholdOrdering(t *testing.T) {
	const trials = 4000
	rate := func(d int, p float64, seed int64) float64 {
		t.Helper()
		l, err := NewLattice(d)
		if err != nil {
			t.Fatalf("NewLattice(%d): %v", d, err)
		}
		n, err := LogicalFailures(l, NewUnionFindDecoder(l), p, trials, 8, seed)
		if err != nil {
			t.Fatal(err)
		}
		return float64(n) / trials
	}
	if r3, r7 := rate(3, 0.03, 11), rate(7, 0.03, 12); r7 >= r3 {
		t.Errorf("below threshold: rate(d=7)=%v should be well below rate(d=3)=%v", r7, r3)
	}
	if r3, r7 := rate(3, 0.16, 13), rate(7, 0.16, 14); r7 <= r3 {
		t.Errorf("above threshold: rate(d=7)=%v should exceed rate(d=3)=%v", r7, r3)
	}
}

func TestLogicalFailuresInputValidation(t *testing.T) {
	l, err := NewLattice(3)
	if err != nil {
		t.Fatalf("NewLattice(3): %v", err)
	}
	dec := NewUnionFindDecoder(l)
	tests := []struct {
		name    string
		lattice *Lattice
		dec     Decoder
		p       float64
		trials  int
		workers int
	}{
		{name: "nil lattice", lattice: nil, dec: dec, p: 0.1, trials: 10, workers: 1},
		{name: "nil decoder", lattice: l, dec: nil, p: 0.1, trials: 10, workers: 1},
		{name: "zero trials", lattice: l, dec: dec, p: 0.1, trials: 0, workers: 1},
		{name: "negative workers", lattice: l, dec: dec, p: 0.1, trials: 10, workers: -1},
		{name: "p above 1", lattice: l, dec: dec, p: 1.5, trials: 10, workers: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := LogicalFailures(tt.lattice, tt.dec, tt.p, tt.trials, tt.workers, 1); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}

func TestDeriveSeedSeparatesIndices(t *testing.T) {
	seen := map[int64][2]int{}
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			s := DeriveSeed(1, i, j)
			if prev, dup := seen[s]; dup {
				t.Fatalf("DeriveSeed(1, %d, %d) collides with DeriveSeed(1, %d, %d)", i, j, prev[0], prev[1])
			}
			seen[s] = [2]int{i, j}
		}
	}
	if DeriveSeed(1, 2, 3) == DeriveSeed(2, 2, 3) {
		t.Error("different master seeds should give different derived seeds")
	}
}
