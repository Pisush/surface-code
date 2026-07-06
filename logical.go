package surface

import "fmt"

// ApplyCorrection combines an error configuration with a correction by
// XOR, returning the residual operator as a new slice. It panics if the
// slices have different lengths (programmer error).
func ApplyCorrection(errs, correction []bool) []bool {
	if len(errs) != len(correction) {
		panic(fmt.Sprintf("surface: cannot combine %d errors with %d corrections", len(errs), len(correction)))
	}
	residual := make([]bool, len(errs))
	for i := range errs {
		residual[i] = errs[i] != correction[i]
	}
	return residual
}

// IsLogicalError reports whether a residual operator (errors XOR
// correction) acts as a logical X on the encoded qubit.
//
// It computes the parity of the residual on the left boundary cut: the d
// horizontal data qubits in column 0. Any set of flips with trivial
// syndrome is a disjoint union of stabilizer loops and boundary-to-boundary
// chains; loops and same-side chains cross this cut an even number of
// times, while a left-to-right chain — a logical X — crosses it an odd
// number of times. The result is therefore exactly the homology class of
// the residual, provided the residual has trivial syndrome (which any
// valid decoder guarantees). For a residual with non-trivial syndrome the
// crossing parity is still returned but has no homological meaning.
//
// IsLogicalError panics if residual does not have one entry per data
// qubit (programmer error).
func (l *Lattice) IsLogicalError(residual []bool) bool {
	if len(residual) != len(l.edges) {
		panic(fmt.Sprintf("surface: residual has %d entries, want %d", len(residual), len(l.edges)))
	}
	parity := false
	for r := 0; r < l.d; r++ {
		e, ok := l.EdgeAt(Coord{Row: 2 * r, Col: 0})
		if !ok {
			panic("surface: internal error: missing left-boundary data qubit")
		}
		if residual[e.ID] {
			parity = !parity
		}
	}
	return parity
}
