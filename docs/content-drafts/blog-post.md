> DRAFT — technical blog post for surface-code (github.com/Pisush/surface-code)

# Watching a quantum error-correction threshold emerge from a union-find decoder in Go

Quantum error correction has one number that matters more than any
other: the threshold. Below it, adding more physical qubits makes your
logical qubit *more* reliable, exponentially so. Above it, adding more
qubits makes things worse. Everything about whether large-scale quantum
computing is possible at all hinges on which side of that number your
hardware and your decoder land on.

surface-code is a from-scratch Go implementation of the planar surface
code — simplified to the classical case that's actually tractable to
build and understand in an afternoon — plus a union-find decoder and a
parallel Monte Carlo driver that produces exactly the curve you'd hope
to see: logical error rate crossing physical error rate at a single
point, for every code distance, right around p ≈ 0.09-0.10.

## The classical simplification

Full surface-code simulation tracks arbitrary quantum errors via
stabilizer/Clifford simulation. This project narrows the scope
deliberately: only bit-flip (X) errors on data qubits, decoded from
Z-stabilizer syndromes. X errors and Z checks form a closed classical
parity problem — no amplitudes anywhere, just bits — which is the
standard pedagogical simplification and lets the whole thing run as
plain integer and boolean arithmetic. (Z errors, decoded by the
X-stabilizers of the dual lattice, are symmetric and omitted for the same
reason: they'd just be the same code again.)

The lattice itself (`lattice.go`) is a standard unrotated planar layout
at odd distance *d*: data qubits sit on the edges of a square grid
(d² + (d−1)² of them), Z-stabilizers sit on the vertices between them
(d(d−1) of them), and the left/right boundaries are "rough" — they can
absorb a chain endpoint without triggering a stabilizer. Every object
gets an explicit `Coord{Row, Col}` on a doubled grid, so a horizontal
data qubit lands at `(2r, 2c)`, a vertical one at `(2r+1, 2c+1)`, and a
Z-stabilizer at `(2r, 2c+1)` — an indexing scheme that makes "where is
this qubit relative to that stabilizer" a one-line coordinate check
instead of a lookup table.

Noise (`noise.go`) is iid: each data qubit flips independently with
probability *p*, drawn from an injected `*rand.Rand` so every run is
reproducible. `Syndrome` (`syndrome.go`) then measures every Z-stabilizer
against that error pattern and returns the IDs of the "lit" ones — the
ones adjacent to an odd number of flips. Those lit stabilizers are the
only signal a decoder gets: it never sees the errors directly, only
their tripwire-triggered parity.

## The decoder: grow, fuse, peel

The interesting code is the union-find decoder (`unionfind.go`), styled
after Delfosse and Nickerson's near-linear-time algorithm. It runs in two
phases.

**Growth.** Every lit stabilizer starts as its own single-node cluster.
Each round, every *active* cluster — odd number of defects, not yet
touching a boundary — grows every edge on its frontier by half a step.
When an edge finishes growing from both sides it fuses the two clusters
at its endpoints, using union-find with path halving and union-by-size
so the bookkeeping stays cheap:

```go
union := func(a, b int) {
    ra, rb := find(a), find(b)
    if ra == rb {
        return
    }
    if size[ra] < size[rb] {
        ra, rb = rb, ra
    }
    parent[rb] = ra
    size[ra] += size[rb]
    parity[ra] = parity[ra] != parity[rb]
    boundary[ra] = boundary[ra] || boundary[rb]
    ...
}
```

A cluster that reaches a rough boundary becomes *neutral* — the boundary
is a legitimate terminal that can absorb one unpaired defect, the same
way a chain of flips reaching the edge of the lattice doesn't need a
partner defect to explain it. Growth stops the moment no active cluster
remains; every remaining odd-parity cluster has found either a partner
defect or a boundary.

**Peeling.** The edges that finished growing form an *erasure* — a
subgraph the decoder is confident contains the real error chain. Peeling
builds a spanning forest of that erasure, rooting each tree at a virtual
boundary node when one is present, then repeatedly removes leaves: if a
leaf carries a defect, its edge joins the correction and the defect
"hops" to the leaf's neighbor. Defects walk inward along the forest this
way until they annihilate in pairs or are absorbed by a boundary root.
The result always reproduces the original syndrome exactly, and the
whole thing runs in near-linear time in the number of data qubits — the
reason a Monte Carlo sweep of tens of thousands of trials is cheap enough
to run on a laptop.

One correctness note worth calling out for anyone reading the source: the
implementation deliberately grows every active cluster by the same
amount each round, rather than Delfosse-Nickerson's smallest-cluster-
first weighted growth. Both variants correct every error of weight up to
⌊(d−1)/2⌋ — the correctness guarantee doesn't depend on growth order —
but uniform growth measures a slightly lower threshold in practice (see
below).

## Checking for a logical error

A correction can be right (matches the true error up to a stabilizer, no
effect on the encoded qubit) or wrong in a way that spans the lattice
(a logical error). `IsLogicalError` (`logical.go`) tells them apart with
a neat trick: it computes the parity of the residual operator (error XOR
correction) restricted to the left-boundary column of data qubits. Any
set of flips with trivial syndrome decomposes into stabilizer loops and
boundary-to-boundary chains; loops and same-side chains cross that column
an even number of times, while a genuine left-to-right chain — a logical
X — crosses it an odd number of times. One parity check over d qubits
stands in for a full homology computation.

## Union-find vs. minimum-weight perfect matching

The obvious question for anyone who's seen MWPM (Blossom-algorithm)
decoders: why not just match lit stabilizers with true shortest paths?
Accuracy: for this noise model, MWPM reaches a threshold around 10.3%,
weighted-growth union-find around 9.9%, and this implementation's
simpler uniform-growth variant measures around 9% <!-- VERIFY --> (the
first two figures are literature values; the third is the README's own
measured number for this codebase). What union-find buys back is speed —
near-linear versus super-quadratic worst case for matching — which is
why it's a leading real-time-decoding candidate despite being slightly
less accurate, and why MWPM was deliberately left out of this project
rather than added as a slower, more accurate baseline.

## Watching the threshold appear

`cmd/threshold` sweeps distances d ∈ {3, 5, 7, 9} and physical error rates
p ∈ [0.01, 0.16], running (by default) 10,000 trials per point, spread
across all CPUs, and writes `d,p,logical_error_rate,trials` CSV rows.
Determinism is load-bearing here, not incidental: every (d, p) point
derives its own seed from a master seed via a SplitMix64 mixer
(`DeriveSeed`), and every individual trial gets its own RNG stream keyed
by trial index — so results are bit-identical no matter how many workers
you run or how the goroutine scheduler interleaves them.

The payoff is the shape of the output. Below p ≈ 0.09, larger distance
suppresses the logical error rate — d=9 outperforms d=3 by an order of
magnitude at p=0.03 in the README's own sample run. Above p ≈ 0.09,
larger distance makes it *worse* — bigger codes have more chances to
form a spanning chain in a noisier system. Right around that crossing,
every curve for every distance meets at (almost) the same point. That
crossing is the threshold, and it's the whole reason scalable quantum
computing is a real engineering target rather than a physics footnote:
build hardware and decoders below threshold, and you can suppress errors
arbitrarily far just by making the code bigger.

## What's next

This is explicitly a v1: perfect, single-round syndrome measurement (no
measurement errors, no repeated rounds building a 3D matching graph),
X-only noise, and uniform rather than weighted cluster growth. Each of
those is a well-scoped next step, and each would push the measured
threshold a little closer to the MWPM number — at the cost of exactly
the runtime advantage that makes union-find worth using in the first
place.
