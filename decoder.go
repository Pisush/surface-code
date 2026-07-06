package surface

// Decoder turns a syndrome into a correction.
type Decoder interface {
	// Decode returns a correction, one entry per data qubit, whose
	// syndrome equals lit: applying the correction on top of the
	// original error leaves every Z-stabilizer unlit. Whether the
	// combined operator is trivial or a logical error is checked
	// separately with Lattice.IsLogicalError.
	//
	// lit must be a set of valid stabilizer IDs (as produced by
	// Lattice.Syndrome); implementations panic on out-of-range or
	// duplicate IDs (programmer errors).
	Decode(lit []int) []bool
}
