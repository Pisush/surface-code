package surface

import (
	"reflect"
	"testing"
)

// errsAt builds an error configuration with flips at the given doubled-grid
// coordinates, failing the test if a coordinate is not a data qubit.
func errsAt(t *testing.T, l *Lattice, coords ...Coord) []bool {
	t.Helper()
	errs := make([]bool, l.NumDataQubits())
	for _, c := range coords {
		e, ok := l.EdgeAt(c)
		if !ok {
			t.Fatalf("no data qubit at %v", c)
		}
		errs[e.ID] = !errs[e.ID]
	}
	return errs
}

func TestSyndromeSingleErrorsD3(t *testing.T) {
	l, err := NewLattice(3)
	if err != nil {
		t.Fatalf("NewLattice(3): %v", err)
	}
	// Stabilizer IDs on d=3 (doubled coords): 0:(0,1) 1:(0,3) 2:(2,1)
	// 3:(2,3) 4:(4,1) 5:(4,3).
	tests := []struct {
		name  string
		coord Coord
		want  []int
	}{
		{name: "left boundary horizontal", coord: Coord{0, 0}, want: []int{0}},
		{name: "bulk horizontal top row", coord: Coord{0, 2}, want: []int{0, 1}},
		{name: "right boundary horizontal", coord: Coord{0, 4}, want: []int{1}},
		{name: "vertical top left", coord: Coord{1, 1}, want: []int{0, 2}},
		{name: "vertical bottom right", coord: Coord{3, 3}, want: []int{3, 5}},
		{name: "bulk horizontal bottom row", coord: Coord{4, 2}, want: []int{4, 5}},
		{name: "left boundary bottom row", coord: Coord{4, 0}, want: []int{4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := l.Syndrome(errsAt(t, l, tt.coord))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Syndrome(error at %v) = %v, want %v", tt.coord, got, tt.want)
			}
		})
	}
}

func TestSyndromeMultiErrorParity(t *testing.T) {
	l, err := NewLattice(3)
	if err != nil {
		t.Fatalf("NewLattice(3): %v", err)
	}
	tests := []struct {
		name   string
		coords []Coord
		want   []int
	}{
		{
			name:   "no errors",
			coords: nil,
			want:   nil,
		},
		{
			name:   "adjacent errors cancel the shared stabilizer",
			coords: []Coord{{0, 0}, {0, 2}},
			want:   []int{1},
		},
		{
			name:   "full logical X chain has trivial syndrome",
			coords: []Coord{{0, 0}, {0, 2}, {0, 4}},
			want:   nil,
		},
		{
			name:   "closed loop around an X-plaquette has trivial syndrome",
			coords: []Coord{{0, 2}, {2, 2}, {1, 1}, {1, 3}},
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := l.Syndrome(errsAt(t, l, tt.coords...))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Syndrome(%v) = %v, want %v", tt.coords, got, tt.want)
			}
		})
	}
}

func TestSyndromePanicsOnWrongLength(t *testing.T) {
	l, err := NewLattice(3)
	if err != nil {
		t.Fatalf("NewLattice(3): %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Error("Syndrome with wrong-length errs: want panic, got none")
		}
	}()
	l.Syndrome(make([]bool, 3))
}
