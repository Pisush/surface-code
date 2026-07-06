package surface

import "testing"

func TestNewLatticeRejectsBadDistance(t *testing.T) {
	for _, d := range []int{-3, 0, 1, 2, 4, 10} {
		if _, err := NewLattice(d); err == nil {
			t.Errorf("NewLattice(%d): want error, got nil", d)
		}
	}
}

func TestLatticeCounts(t *testing.T) {
	tests := []struct {
		d               int
		wantQubits      int
		wantStabilizers int
	}{
		{d: 3, wantQubits: 13, wantStabilizers: 6},
		{d: 5, wantQubits: 41, wantStabilizers: 20},
		{d: 7, wantQubits: 85, wantStabilizers: 42},
		{d: 9, wantQubits: 145, wantStabilizers: 72},
	}
	for _, tt := range tests {
		l, err := NewLattice(tt.d)
		if err != nil {
			t.Fatalf("NewLattice(%d): %v", tt.d, err)
		}
		if got := l.NumDataQubits(); got != tt.wantQubits {
			t.Errorf("d=%d: NumDataQubits() = %d, want %d", tt.d, got, tt.wantQubits)
		}
		if got := l.NumStabilizers(); got != tt.wantStabilizers {
			t.Errorf("d=%d: NumStabilizers() = %d, want %d", tt.d, got, tt.wantStabilizers)
		}
		if got := l.NumNodes(); got != tt.wantStabilizers+2*tt.d {
			t.Errorf("d=%d: NumNodes() = %d, want %d", tt.d, got, tt.wantStabilizers+2*tt.d)
		}
		if got := l.Distance(); got != tt.d {
			t.Errorf("d=%d: Distance() = %d", tt.d, got)
		}
	}
}

func TestEdgeCoordRoundTrip(t *testing.T) {
	for _, d := range []int{3, 5, 7} {
		l, err := NewLattice(d)
		if err != nil {
			t.Fatalf("NewLattice(%d): %v", d, err)
		}
		for _, e := range l.Edges() {
			got, ok := l.EdgeAt(e.Coord)
			if !ok {
				t.Fatalf("d=%d: EdgeAt(%v) not found for edge %d", d, e.Coord, e.ID)
			}
			if got.ID != e.ID {
				t.Errorf("d=%d: EdgeAt(%v).ID = %d, want %d", d, e.Coord, got.ID, e.ID)
			}
		}
		if _, ok := l.EdgeAt(Coord{Row: 0, Col: 1}); ok {
			t.Errorf("d=%d: EdgeAt found a data qubit at a stabilizer site", d)
		}
	}
}

func TestStabilizerCoordRoundTrip(t *testing.T) {
	for _, d := range []int{3, 5} {
		l, err := NewLattice(d)
		if err != nil {
			t.Fatalf("NewLattice(%d): %v", d, err)
		}
		for s := 0; s < l.NumStabilizers(); s++ {
			c := l.StabilizerCoord(s)
			got, ok := l.StabilizerAt(c)
			if !ok || got != s {
				t.Errorf("d=%d: StabilizerAt(%v) = (%d, %v), want (%d, true)", d, c, got, ok, s)
			}
		}
		if _, ok := l.StabilizerAt(Coord{Row: 0, Col: 0}); ok {
			t.Errorf("d=%d: StabilizerAt found a stabilizer at a data-qubit site", d)
		}
	}
}

func TestBoundaryStructure(t *testing.T) {
	l, err := NewLattice(3)
	if err != nil {
		t.Fatalf("NewLattice(3): %v", err)
	}
	virtualEdges := 0
	for _, e := range l.Edges() {
		u, v := l.IsVirtualNode(e.U), l.IsVirtualNode(e.V)
		if u && v {
			t.Errorf("edge %d at %v connects two virtual nodes", e.ID, e.Coord)
		}
		if u || v {
			virtualEdges++
			if e.Coord.Col != 0 && e.Coord.Col != 2*(l.Distance()-1) {
				t.Errorf("edge %d at %v touches a boundary but is not in a boundary column", e.ID, e.Coord)
			}
		}
	}
	// d boundary qubits per rough side.
	if want := 2 * l.Distance(); virtualEdges != want {
		t.Errorf("boundary qubits = %d, want %d", virtualEdges, want)
	}
	// Every stabilizer has degree 3 (top/bottom rows) or 4 (bulk).
	for s := 0; s < l.NumStabilizers(); s++ {
		deg := len(l.IncidentEdges(s))
		row := l.StabilizerCoord(s).Row
		want := 4
		if row == 0 || row == 2*(l.Distance()-1) {
			want = 3
		}
		if deg != want {
			t.Errorf("stabilizer %d (coord %v): degree = %d, want %d", s, l.StabilizerCoord(s), deg, want)
		}
	}
}
