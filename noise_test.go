package surface

import (
	"math/rand"
	"testing"
)

func TestSampleErrorsExtremes(t *testing.T) {
	l, err := NewLattice(5)
	if err != nil {
		t.Fatalf("NewLattice(5): %v", err)
	}
	tests := []struct {
		name string
		p    float64
		want bool
	}{
		{name: "p=0 flips nothing", p: 0, want: false},
		{name: "p=1 flips everything", p: 1, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := l.SampleErrors(tt.p, rand.New(rand.NewSource(1)))
			if len(errs) != l.NumDataQubits() {
				t.Fatalf("len = %d, want %d", len(errs), l.NumDataQubits())
			}
			for i, e := range errs {
				if e != tt.want {
					t.Fatalf("qubit %d: flipped = %v, want %v", i, e, tt.want)
				}
			}
		})
	}
}

func TestSampleErrorsDeterministic(t *testing.T) {
	l, err := NewLattice(5)
	if err != nil {
		t.Fatalf("NewLattice(5): %v", err)
	}
	a := l.SampleErrors(0.3, rand.New(rand.NewSource(42)))
	b := l.SampleErrors(0.3, rand.New(rand.NewSource(42)))
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("qubit %d differs across identically seeded runs", i)
		}
	}
}

func TestSampleErrorsRate(t *testing.T) {
	l, err := NewLattice(9)
	if err != nil {
		t.Fatalf("NewLattice(9): %v", err)
	}
	rng := rand.New(rand.NewSource(7))
	const trials = 2000
	p := 0.1
	flips := 0
	for i := 0; i < trials; i++ {
		for _, e := range l.SampleErrors(p, rng) {
			if e {
				flips++
			}
		}
	}
	got := float64(flips) / float64(trials*l.NumDataQubits())
	if got < 0.09 || got > 0.11 {
		t.Errorf("empirical flip rate = %v, want within [0.09, 0.11] of p=%v", got, p)
	}
}

func TestSampleErrorsPanics(t *testing.T) {
	l, err := NewLattice(3)
	if err != nil {
		t.Fatalf("NewLattice(3): %v", err)
	}
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "p out of range", fn: func() { l.SampleErrors(1.5, rand.New(rand.NewSource(1))) }},
		{name: "nil rng", fn: func() { l.SampleErrors(0.1, nil) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("want panic, got none")
				}
			}()
			tt.fn()
		})
	}
}
