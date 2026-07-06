package surface

import (
	"math/rand"
	"testing"
)

// decodeAndCheck runs the union-find decoder on the syndrome of errs and
// asserts the correction reproduces the syndrome exactly (trivial residual
// syndrome). It returns whether the residual is a logical error.
func decodeAndCheck(t *testing.T, l *Lattice, errs []bool) bool {
	t.Helper()
	dec := NewUnionFindDecoder(l)
	correction := dec.Decode(l.Syndrome(errs))
	residual := ApplyCorrection(errs, correction)
	if syn := l.Syndrome(residual); syn != nil {
		t.Fatalf("correction does not reproduce the syndrome; residual syndrome %v", syn)
	}
	return l.IsLogicalError(residual)
}

func TestDecodeEmptySyndrome(t *testing.T) {
	l, err := NewLattice(3)
	if err != nil {
		t.Fatalf("NewLattice(3): %v", err)
	}
	correction := NewUnionFindDecoder(l).Decode(nil)
	for i, c := range correction {
		if c {
			t.Errorf("qubit %d corrected on an empty syndrome", i)
		}
	}
}

// TestDecodeAllSingleErrors checks that every possible single-qubit error
// is corrected without a logical error — the code must handle all
// weight-1 errors at any distance.
func TestDecodeAllSingleErrors(t *testing.T) {
	for _, d := range []int{3, 5, 7} {
		l, err := NewLattice(d)
		if err != nil {
			t.Fatalf("NewLattice(%d): %v", d, err)
		}
		for _, e := range l.Edges() {
			errs := make([]bool, l.NumDataQubits())
			errs[e.ID] = true
			if decodeAndCheck(t, l, errs) {
				t.Errorf("d=%d: single error at %v caused a logical error", d, e.Coord)
			}
		}
	}
}

// TestDecodeKnownPatternsD3 pins down exact corrections for hand-checked
// error patterns on d=3 where the minimal correction is unique.
func TestDecodeKnownPatternsD3(t *testing.T) {
	l, err := NewLattice(3)
	if err != nil {
		t.Fatalf("NewLattice(3): %v", err)
	}
	tests := []struct {
		name string
		errs []Coord
		want []Coord // exact expected correction
	}{
		{
			name: "single boundary error corrected in place",
			errs: []Coord{{0, 0}},
			want: []Coord{{0, 0}},
		},
		{
			name: "single bulk vertical error corrected in place",
			errs: []Coord{{1, 1}},
			want: []Coord{{1, 1}},
		},
		{
			// The four lit stabilizers fuse into one cluster, but
			// peeling still recovers both flips exactly.
			name: "two single errors both corrected in place",
			errs: []Coord{{1, 1}, {3, 3}},
			want: []Coord{{1, 1}, {3, 3}},
		},
	}
	dec := NewUnionFindDecoder(l)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dec.Decode(l.Syndrome(errsAt(t, l, tt.errs...)))
			want := errsAt(t, l, tt.want...)
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("correction[%d] (%v) = %v, want %v", i, l.Edge(i).Coord, got[i], want[i])
				}
			}
		})
	}
}

// TestWeightHalfDChainCausesLogicalFailure demonstrates the code-distance
// bound: an error chain of weight ceil(d/2) reaching in from a boundary
// makes the decoder complete the chain the short way, producing a logical
// error. This is expected decoder behavior, not a bug.
func TestWeightHalfDChainCausesLogicalFailure(t *testing.T) {
	tests := []struct {
		name string
		d    int
		errs []Coord
	}{
		{
			// Both boundary qubits of row 0 flipped; the decoder
			// pairs the two lit stabilizers through the middle,
			// completing a logical X chain.
			name: "d=3 weight-2 chain",
			d:    3,
			errs: []Coord{{0, 0}, {0, 4}},
		},
		{
			// Three errors reaching past the middle of row 0; the
			// decoder connects the remaining lit stabilizer to the
			// nearer (right) boundary.
			name: "d=5 weight-3 chain",
			d:    5,
			errs: []Coord{{0, 0}, {0, 2}, {0, 4}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if want := (tt.d + 1) / 2; len(tt.errs) != want {
				t.Fatalf("test pattern has weight %d, want ceil(d/2) = %d", len(tt.errs), want)
			}
			l, err := NewLattice(tt.d)
			if err != nil {
				t.Fatalf("NewLattice(%d): %v", tt.d, err)
			}
			if !decodeAndCheck(t, l, errsAt(t, l, tt.errs...)) {
				t.Errorf("weight-%d chain on d=%d did not cause a logical error", len(tt.errs), tt.d)
			}
		})
	}
}

// TestDecodeBelowHalfDistance checks the guarantee that any error of
// weight at most floor((d-1)/2) is corrected without logical error, on
// random error patterns of exactly that weight.
func TestDecodeBelowHalfDistance(t *testing.T) {
	for _, d := range []int{3, 5, 7} {
		l, err := NewLattice(d)
		if err != nil {
			t.Fatalf("NewLattice(%d): %v", d, err)
		}
		rng := rand.New(rand.NewSource(int64(100 + d)))
		weight := (d - 1) / 2
		for trial := 0; trial < 200; trial++ {
			errs := make([]bool, l.NumDataQubits())
			for placed := 0; placed < weight; {
				q := rng.Intn(l.NumDataQubits())
				if !errs[q] {
					errs[q] = true
					placed++
				}
			}
			if decodeAndCheck(t, l, errs) {
				t.Fatalf("d=%d trial %d: weight-%d error caused a logical error (errs=%v)", d, trial, weight, errs)
			}
		}
	}
}

// TestDecodeRandomizedSyndromeConsistency stresses the decoder invariant
// that the correction always reproduces the input syndrome, across noise
// strengths well above threshold.
func TestDecodeRandomizedSyndromeConsistency(t *testing.T) {
	for _, d := range []int{3, 5, 7} {
		l, err := NewLattice(d)
		if err != nil {
			t.Fatalf("NewLattice(%d): %v", d, err)
		}
		for _, p := range []float64{0.02, 0.10, 0.25} {
			rng := rand.New(rand.NewSource(int64(d*1000) + int64(p*100)))
			for trial := 0; trial < 300; trial++ {
				decodeAndCheck(t, l, l.SampleErrors(p, rng))
			}
		}
	}
}

// TestDecodeLowNoiseLogicalRate is a coarse end-to-end sanity check: well
// below threshold, logical errors must be rare.
func TestDecodeLowNoiseLogicalRate(t *testing.T) {
	l, err := NewLattice(5)
	if err != nil {
		t.Fatalf("NewLattice(5): %v", err)
	}
	rng := rand.New(rand.NewSource(2026))
	const trials = 2000
	failures := 0
	for i := 0; i < trials; i++ {
		if decodeAndCheck(t, l, l.SampleErrors(0.01, rng)) {
			failures++
		}
	}
	if rate := float64(failures) / trials; rate > 0.02 {
		t.Errorf("logical error rate at d=5, p=0.01 is %v; want < 0.02", rate)
	}
}

func TestDecodePanics(t *testing.T) {
	l, err := NewLattice(3)
	if err != nil {
		t.Fatalf("NewLattice(3): %v", err)
	}
	dec := NewUnionFindDecoder(l)
	tests := []struct {
		name string
		lit  []int
	}{
		{name: "stabilizer ID out of range", lit: []int{6}},
		{name: "negative stabilizer ID", lit: []int{-1}},
		{name: "duplicate stabilizer ID", lit: []int{2, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("want panic, got none")
				}
			}()
			dec.Decode(tt.lit)
		})
	}
}
