DRAFT

# surface-code: watching a quantum error-correction threshold emerge, in Go

**surface-code** is a from-scratch Go implementation of a distance-*d*
planar surface code — quantum error correction's leading practical
scheme — simplified to a classical bit-flip noise model, plus a
near-linear-time **union-find decoder** and a parallel Monte Carlo driver
that produces the one plot that matters in this field: logical error
rate crossing physical error rate at a single point, for every code
distance, at a **threshold**.

## The physics in one paragraph

Qubits are fragile and you can't just copy one to back it up, so
surface codes spread one *logical* qubit across many *physical* data
qubits laid out on a 2D grid. Every *stabilizer* repeatedly checks the
parity of its neighboring qubits; a flipped qubit lights up the
adjacent stabilizers like a tripwire, without revealing or disturbing
the encoded information. This project narrows the scope to only
bit-flip (X) errors decoded from Z-stabilizer parity checks — a closed
classical problem, all bits and booleans, no amplitudes — while
keeping the geometry and the decoding problem real.

## The one clever idea: grow, fuse, peel

The interesting code is the union-find decoder (`unionfind.go`), styled
after Delfosse and Nickerson's algorithm. Every lit stabilizer starts
as its own cluster. Each round, every cluster with an odd number of
defects and no boundary contact grows its frontier edges by half a
step; edges that finish growing fuse their clusters via union-find
(path halving, union by size). A cluster that reaches a boundary goes
neutral — boundaries are legitimate terminals that absorb an unpaired
defect. Once growth stalls, the fully-grown edges form an *erasure*,
which gets *peeled*: a spanning forest rooted at any boundary node,
with leaves stripped one at a time so a defect on a pendant vertex
"hops" inward until it annihilates against a partner or is absorbed by
a boundary root.

Why this matters: neither phase does more than near-constant work per
edge, so the whole decode runs in near-linear time in the number of
data qubits — cheap enough that sweeping tens of thousands of Monte
Carlo trials per data point is a laptop-scale job, not a cluster job.
The tradeoff is accuracy: minimum-weight perfect matching (MWPM) finds
truly optimal pairings but at super-quadratic worst-case cost, which
is why union-find is a leading candidate for real-time decoding
<!-- VERIFY -->.

## Why it's cool

Because the payoff is visible, not asserted. `cmd/threshold` sweeps
code distances d ∈ {3, 5, 7, 9} against physical error rate p, runs
10,000 deterministic trials per point (every point gets its own
derived seed, every trial its own RNG stream, so results are
bit-identical regardless of worker count), and writes a CSV. Plot
logical error rate against p for each distance and the curves cross at
a single point, p ≈ 0.09 for this decoder and noise model — below it,
bigger codes suppress errors exponentially; above it, bigger codes
make things worse. That crossing, reproduced from a few hundred lines
of Go and a laptop's worth of CPU, is the actual empirical signature
that makes scalable quantum computing an engineering target rather
than a hope.
