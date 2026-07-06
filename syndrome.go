package surface

import "fmt"

// Syndrome measures every Z-stabilizer against the given error
// configuration and returns the IDs of the "lit" stabilizers — those
// adjacent to an odd number of flipped data qubits — in ascending order.
// Syndrome measurement is perfect: no measurement errors are modeled.
//
// errs must have one entry per data qubit (as produced by SampleErrors);
// Syndrome panics otherwise (programmer error).
func (l *Lattice) Syndrome(errs []bool) []int {
	if len(errs) != len(l.edges) {
		panic(fmt.Sprintf("surface: error configuration has %d entries, want %d", len(errs), len(l.edges)))
	}
	var lit []int
	for s := 0; s < l.numStabilizers; s++ {
		parity := false
		for _, e := range l.incidence[s] {
			if errs[e] {
				parity = !parity
			}
		}
		if parity {
			lit = append(lit, s)
		}
	}
	return lit
}
