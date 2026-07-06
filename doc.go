// Package surface implements a distance-d planar surface code playground
// under a simplified, fully classical noise model.
//
// # Scope
//
// Only bit-flip (X) errors on data qubits are simulated, and they are
// detected by Z-stabilizer parity checks. Because X errors and Z checks
// commute into a purely classical parity problem, the whole simulation is
// bits, not amplitudes: an error configuration is a []bool over data qubits,
// a syndrome is the set of Z-stabilizers adjacent to an odd number of
// errors. Syndrome measurement is perfect (no measurement errors) in v1.
//
// # Layout
//
// The lattice is the standard (unrotated) planar surface code of odd
// distance d, with data qubits on the edges of a square grid:
//
//   - Z-stabilizers form a grid of d rows by d-1 columns.
//   - Horizontal data qubits: d rows by d columns; the leftmost and
//     rightmost columns touch the "rough" left/right boundaries.
//   - Vertical data qubits: d-1 rows by d-1 columns.
//
// This gives d*d + (d-1)*(d-1) data qubits and d*(d-1) Z-stabilizers, the
// textbook planar-code counts. Every site has an explicit Coord{Row, Col}
// on a doubled grid (see Coord) so tests and tools can address qubits and
// stabilizers geometrically.
//
// A logical X operator is any left-to-right chain of horizontal data
// qubits; the shortest one has weight d, which is what "distance d" means.
package surface
