package surface

import "fmt"

// Coord addresses a site on the doubled square grid that underlies the
// planar surface code. Doubling the grid gives every object its own
// integer coordinates:
//
//   - horizontal data qubits sit at (even row, even col): (2r, 2c),
//   - vertical data qubits sit at (odd row, odd col): (2r+1, 2c+1),
//   - Z-stabilizers sit at (even row, odd col): (2r, 2c+1).
//
// Row 0 is the top of the lattice; Col 0 is the left ("rough") boundary
// column of horizontal data qubits.
type Coord struct {
	Row, Col int
}

// Edge is a data qubit, i.e. an edge of the decoding graph. Its endpoints
// U and V are node IDs: either Z-stabilizers (IDs in [0, NumStabilizers))
// or virtual boundary nodes (IDs in [NumStabilizers, NumNodes)) that stand
// in for the rough left/right boundaries.
type Edge struct {
	// ID is the data-qubit index in [0, NumDataQubits).
	ID int
	// Coord locates the qubit on the doubled grid.
	Coord Coord
	// U and V are the endpoint node IDs. Exactly one endpoint is a
	// virtual boundary node iff the qubit touches a boundary; interior
	// qubits connect two stabilizers.
	U, V int
}

// Lattice is a distance-d planar surface code layout: data qubits on
// edges, Z-stabilizers on the vertices between them, and virtual nodes for
// the two rough boundaries. A Lattice is immutable after construction and
// safe for concurrent use.
type Lattice struct {
	d              int
	edges          []Edge
	numStabilizers int
	numNodes       int
	incidence      [][]int // node ID -> IDs of incident edges
	edgeByCoord    map[Coord]int
}

// NewLattice constructs the distance-d planar lattice. The distance must
// be odd and at least 3; otherwise an error is returned.
func NewLattice(d int) (*Lattice, error) {
	if d < 3 || d%2 == 0 {
		return nil, fmt.Errorf("surface: distance must be odd and >= 3, got %d", d)
	}

	numStab := d * (d - 1)
	numNodes := numStab + 2*d // one virtual node per boundary data qubit row and side
	l := &Lattice{
		d:              d,
		numStabilizers: numStab,
		numNodes:       numNodes,
		incidence:      make([][]int, numNodes),
		edgeByCoord:    make(map[Coord]int, d*d+(d-1)*(d-1)),
	}

	stab := func(r, c int) int { return r*(d-1) + c }
	virtLeft := func(r int) int { return numStab + r }
	virtRight := func(r int) int { return numStab + d + r }

	addEdge := func(coord Coord, u, v int) {
		e := Edge{ID: len(l.edges), Coord: coord, U: u, V: v}
		l.edges = append(l.edges, e)
		l.incidence[u] = append(l.incidence[u], e.ID)
		l.incidence[v] = append(l.incidence[v], e.ID)
		l.edgeByCoord[coord] = e.ID
	}

	// Horizontal data qubits: d rows by d columns. Column 0 hangs off
	// the left boundary, column d-1 off the right boundary.
	for r := 0; r < d; r++ {
		for c := 0; c < d; c++ {
			u := virtLeft(r)
			if c > 0 {
				u = stab(r, c-1)
			}
			v := virtRight(r)
			if c < d-1 {
				v = stab(r, c)
			}
			addEdge(Coord{Row: 2 * r, Col: 2 * c}, u, v)
		}
	}

	// Vertical data qubits: d-1 rows by d-1 columns, connecting
	// vertically adjacent stabilizers.
	for r := 0; r < d-1; r++ {
		for c := 0; c < d-1; c++ {
			addEdge(Coord{Row: 2*r + 1, Col: 2*c + 1}, stab(r, c), stab(r+1, c))
		}
	}

	return l, nil
}

// Distance returns the code distance d.
func (l *Lattice) Distance() int { return l.d }

// NumDataQubits returns the number of data qubits, d*d + (d-1)*(d-1).
func (l *Lattice) NumDataQubits() int { return len(l.edges) }

// NumStabilizers returns the number of Z-stabilizers, d*(d-1). Stabilizer
// IDs are r*(d-1) + c for row r in [0, d) and column c in [0, d-1).
func (l *Lattice) NumStabilizers() int { return l.numStabilizers }

// NumNodes returns the number of decoding-graph nodes: all Z-stabilizers
// followed by the virtual boundary nodes.
func (l *Lattice) NumNodes() int { return l.numNodes }

// Edges returns all data qubits in ID order. The returned slice is shared
// with the Lattice and must not be modified.
func (l *Lattice) Edges() []Edge { return l.edges }

// Edge returns the data qubit with the given ID. It panics if id is out of
// range (programmer error).
func (l *Lattice) Edge(id int) Edge {
	if id < 0 || id >= len(l.edges) {
		panic(fmt.Sprintf("surface: edge ID %d out of range [0, %d)", id, len(l.edges)))
	}
	return l.edges[id]
}

// EdgeAt looks up a data qubit by its doubled-grid coordinate. The second
// return value reports whether a data qubit exists there.
func (l *Lattice) EdgeAt(c Coord) (Edge, bool) {
	id, ok := l.edgeByCoord[c]
	if !ok {
		return Edge{}, false
	}
	return l.edges[id], true
}

// StabilizerCoord returns the doubled-grid coordinate of a Z-stabilizer.
// It panics if id is out of range (programmer error).
func (l *Lattice) StabilizerCoord(id int) Coord {
	if id < 0 || id >= l.numStabilizers {
		panic(fmt.Sprintf("surface: stabilizer ID %d out of range [0, %d)", id, l.numStabilizers))
	}
	return Coord{Row: 2 * (id / (l.d - 1)), Col: 2*(id%(l.d-1)) + 1}
}

// StabilizerAt looks up a Z-stabilizer ID by its doubled-grid coordinate.
// The second return value reports whether a stabilizer exists there.
func (l *Lattice) StabilizerAt(c Coord) (int, bool) {
	if c.Row < 0 || c.Col < 1 || c.Row%2 != 0 || c.Col%2 != 1 {
		return 0, false
	}
	r, cc := c.Row/2, (c.Col-1)/2
	if r >= l.d || cc >= l.d-1 {
		return 0, false
	}
	return r*(l.d-1) + cc, true
}

// IncidentEdges returns the IDs of the data qubits touching the given
// decoding-graph node (stabilizer or virtual boundary node). The returned
// slice is shared with the Lattice and must not be modified. It panics if
// node is out of range (programmer error).
func (l *Lattice) IncidentEdges(node int) []int {
	if node < 0 || node >= l.numNodes {
		panic(fmt.Sprintf("surface: node ID %d out of range [0, %d)", node, l.numNodes))
	}
	return l.incidence[node]
}

// IsVirtualNode reports whether the given decoding-graph node is a virtual
// boundary node rather than a real Z-stabilizer.
func (l *Lattice) IsVirtualNode(node int) bool {
	return node >= l.numStabilizers && node < l.numNodes
}
