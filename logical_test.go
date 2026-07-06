package surface

import "testing"

func TestIsLogicalError(t *testing.T) {
	l, err := NewLattice(3)
	if err != nil {
		t.Fatalf("NewLattice(3): %v", err)
	}
	tests := []struct {
		name   string
		coords []Coord
		want   bool
	}{
		{
			name:   "no residual",
			coords: nil,
			want:   false,
		},
		{
			name:   "logical X chain across row 0",
			coords: []Coord{{0, 0}, {0, 2}, {0, 4}},
			want:   true,
		},
		{
			name:   "logical X chain across row 2",
			coords: []Coord{{2, 0}, {2, 2}, {2, 4}},
			want:   true,
		},
		{
			name:   "staircase logical chain",
			coords: []Coord{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}},
			want:   true,
		},
		{
			name:   "stabilizer loop is trivial",
			coords: []Coord{{0, 2}, {2, 2}, {1, 1}, {1, 3}},
			want:   false,
		},
		{
			name:   "left-to-left boundary chain is trivial",
			coords: []Coord{{0, 0}, {1, 1}, {2, 0}},
			want:   false,
		},
		{
			name:   "right-to-right boundary chain is trivial",
			coords: []Coord{{2, 4}, {3, 3}, {4, 4}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			residual := errsAt(t, l, tt.coords...)
			if syn := l.Syndrome(residual); syn != nil {
				t.Fatalf("test pattern %v has non-trivial syndrome %v; homology class undefined", tt.coords, syn)
			}
			if got := l.IsLogicalError(residual); got != tt.want {
				t.Errorf("IsLogicalError(%v) = %v, want %v", tt.coords, got, tt.want)
			}
		})
	}
}

func TestApplyCorrection(t *testing.T) {
	got := ApplyCorrection([]bool{true, true, false, false}, []bool{true, false, true, false})
	want := []bool{false, true, true, false}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ApplyCorrection[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestApplyCorrectionPanicsOnLengthMismatch(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("want panic, got none")
		}
	}()
	ApplyCorrection(make([]bool, 3), make([]bool, 4))
}
