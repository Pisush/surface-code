package surface

import (
	"fmt"
	"math/rand"
)

// SampleErrors draws an iid bit-flip error configuration: each data qubit
// is flipped independently with probability p. The result has one entry
// per data qubit, indexed by qubit ID.
//
// The caller supplies the random source so simulations are deterministic
// and embarrassingly parallel (one *rand.Rand per worker). SampleErrors
// panics if p is outside [0, 1] or rng is nil (programmer errors).
func (l *Lattice) SampleErrors(p float64, rng *rand.Rand) []bool {
	if p < 0 || p > 1 {
		panic(fmt.Sprintf("surface: error probability %v outside [0, 1]", p))
	}
	if rng == nil {
		panic("surface: SampleErrors requires a non-nil *rand.Rand")
	}
	errs := make([]bool, len(l.edges))
	for i := range errs {
		errs[i] = rng.Float64() < p
	}
	return errs
}
