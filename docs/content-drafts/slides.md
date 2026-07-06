---
marp: true
theme: default
paginate: true
---

# surface-code
### Watching a quantum error-correction threshold emerge, in Go

A distance-*d* planar surface code, a near-linear union-find decoder,
and a parallel Monte Carlo driver that shows logical vs. physical error
rate curves crossing at a threshold.

<!-- notes: Cold open. Don't explain the punchline yet — just say "by
the end of this, we'll generate this crossing curve ourselves." -->

---

# The one number that matters

- Quantum error correction spreads one *logical* qubit across many
  *physical* qubits so errors can be caught before they matter
- Below a critical physical error rate *p*_th — the **threshold** —
  bigger codes make the logical qubit exponentially *more* reliable
- Above it, bigger codes make things *worse*
- Whether large-scale quantum computing is possible at all comes down
  to which side of that number your hardware and decoder land on

<!-- notes: State the threshold theorem in one sentence before anything
else. This is the hook the rest of the talk pays off. -->

---

# The classical simplification

- This project only simulates **bit-flip (X) errors** on data qubits,
  detected by **Z-stabilizer** parity checks
- X errors and Z checks commute into a closed *classical* parity
  problem — bits and booleans, no amplitudes anywhere
- Standard pedagogical simplification; Z errors (dual lattice,
  X-stabilizers) are symmetric and omitted for the same reason
- Perfect syndrome measurement, single round — no measurement errors
  (v1 scope)

<!-- notes: This is why the whole simulation can be written as plain Go
booleans and integers instead of stabilizer/Clifford simulation. -->

---

# Decoding is a graph problem

- A flipped qubit lights up its neighboring stabilizers "like a
  tripwire" — the lit set is the **syndrome**
- The decoder never sees the errors directly, only the syndrome
- Lit stabilizers are endpoints of hidden error chains; the decoder's
  job is to guess a correction with the same syndrome
- A "wrong" guess is usually harmless (differs from truth by a closed
  loop); a guess wrong by a lattice-spanning chain is a **logical
  error**

<!-- notes: Reframe decoding as: syndrome in, correction out, checked
against a graph/homology structure, not a physics simulation. -->

---

# Architecture at a glance

- `lattice.go` — `Coord{Row, Col}` on a doubled grid; data qubits,
  Z-stabilizers, boundary nodes
- `noise.go` — iid X-flip sampling, seeded `*rand.Rand`
- `syndrome.go` — Z-stabilizer parity extraction
- `unionfind.go` — Delfosse–Nickerson style union-find decoder
- `logical.go` — logical-error check via a boundary-crossing parity
- `montecarlo.go` + `cmd/threshold` — parallel, deterministic sweep

<!-- notes: Quick map of the codebase before diving into any one file.
-->

---

# The lattice: `Coord{Row, Col}`

- Doubled grid so every object gets integer coordinates:
  - Horizontal data qubits: `(2r, 2c)`
  - Vertical data qubits: `(2r+1, 2c+1)`
  - Z-stabilizers: `(2r, 2c+1)`
- Distance *d* (odd, ≥ 3): `d*d + (d-1)*(d-1)` data qubits,
  `d*(d-1)` Z-stabilizers
- Left/right boundaries are "rough" — virtual nodes that terminate a
  chain without needing a stabilizer

<!-- notes: The coordinate scheme turns "which stabilizers does this
qubit touch" into arithmetic instead of a lookup table. -->

---

# Noise and syndrome

```go
// noise.go
func (l *Lattice) SampleErrors(p float64, rng *rand.Rand) []bool {
    errs := make([]bool, len(l.edges))
    for i := range errs {
        errs[i] = rng.Float64() < p
    }
    return errs
}
```

- iid X-flip per data qubit, probability *p*
- Caller-supplied `*rand.Rand` — every run is reproducible

<!-- notes: Real snippet from noise.go. Emphasize the injected RNG —
it's what makes the whole Monte Carlo sweep deterministic later. -->

---

# The union-find decoder: grow

- Every lit stabilizer starts as its own single-node cluster
- Each round, every **active** cluster (odd defects, no boundary
  contact) grows all frontier edges by half a step
- Edges fully grown from both sides **fuse** their clusters
- A cluster touching a boundary goes **neutral** — the boundary
  absorbs an unpaired defect
- Growth stops when no active cluster remains

<!-- notes: Walk this on a small hand-drawn lattice if presenting live:
2-3 lit stabilizers, grow outward, show two clusters touching. -->

---

# `union()` — real code from `unionfind.go`

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

<!-- notes: Path halving in find(), union by size here. This is the
same union-find you'd learn in an intro algorithms class — the point is
that it's load-bearing for a real decoding scheme. -->

---

# The union-find decoder: peel

- Fully-grown edges form an **erasure** — the region the decoder trusts
  contains the real error chain
- Build a spanning forest of the erasure, rooted at any boundary node
- Repeatedly peel leaves: a pendant defect adds its leaf edge to the
  correction and **hops** to the neighbor
- Defects walk inward until they annihilate in pairs, or are absorbed
  by a boundary root
- The returned correction always reproduces the input syndrome exactly

<!-- notes: Peeling is the second half of grow/fuse/peel. Narrate a
defect "hopping" along the tree as a concrete visual. -->

---

# Why near-linear time matters

- Neither growth nor peeling does more than near-constant work per
  edge, so decoding is **near-linear** in the number of data qubits
- Cheap enough that 10,000+ Monte Carlo trials per data point run on a
  laptop
- Contrast: minimum-weight perfect matching (MWPM / Blossom) finds
  truly optimal pairings but is super-quadratic worst case
- Trade a little accuracy for a lot of speed — why union-find is a
  leading real-time decoding candidate <!-- VERIFY -->

<!-- notes: This is the "one clever idea" slide — the whole reason this
decoder is interesting rather than a from-scratch physics sim. -->

---

# Union-find vs. MWPM

| Decoder | Threshold (this noise model) | Cost |
|---|---|---|
| MWPM (literature) | ≈ 10.3% <!-- VERIFY --> | super-quadratic worst case |
| Weighted-growth union-find (literature) | ≈ 9.9% <!-- VERIFY --> | near-linear |
| This project (uniform growth) | ≈ 9% | near-linear |

- Same correctness guarantee either way: corrects every error of weight
  ≤ ⌊(d−1)/2⌋
- MWPM is deliberately **not implemented** here — v1 scope, keeps
  union-find the center of attention

<!-- notes: Literature figures for MWPM and weighted-growth union-find
are marked VERIFY in the source docs; this project's own measured ~9%
figure is not. -->

---

# Checking for a logical error

```go
// logical.go
func (l *Lattice) IsLogicalError(residual []bool) bool {
    parity := false
    for r := 0; r < l.d; r++ {
        e, _ := l.EdgeAt(Coord{Row: 2 * r, Col: 0})
        if residual[e.ID] {
            parity = !parity
        }
    }
    return parity
}
```

- Parity of the residual (error XOR correction) on the left-boundary
  column stands in for a full homology computation

<!-- notes: One parity check over d qubits instead of tracing the whole
residual operator. This is the trick that makes "did this trial fail"
a cheap O(d) check. -->

---

# The Monte Carlo driver

- `cmd/threshold` sweeps *d* ∈ {3, 5, 7, 9} and *p* ∈ [0.01, 0.16]
- 10,000 trials per (d, p) point by default, spread across all CPUs
- `DeriveSeed` (SplitMix64) gives every point its own seed; every
  trial gets its own RNG stream
- Fully deterministic: identical CSV regardless of worker count or
  goroutine scheduling
- Output: `d,p,logical_error_rate,trials` CSV

<!-- notes: Determinism is load-bearing, not incidental — emphasize
that re-running with -workers 1 vs default produces byte-identical
output. -->

---

# Running the sweep

```sh
go build ./cmd/threshold
./threshold -trials 10000 -seed 1 > threshold.csv
python3 scripts/plot_threshold.py threshold.csv threshold.png
```

```csv
d,p,logical_error_rate,trials
3,0.0300,0.017000,10000
5,0.0300,0.006000,10000
7,0.0300,0.002000,10000
9,0.0300,0.001000,10000
```

<!-- notes: Live-demo beat: actually build and run this if presenting
live. Narrate the -v per-point progress lines on stderr while the CSV
builds on stdout. -->

---

# The crossing point

- At low *p*, larger distance suppresses the logical error rate — d=9
  beats d=3 by an order of magnitude at p=0.03
- At high *p*, larger distance makes it *worse*
- All four curves (d=3,5,7,9) meet at a single point: **p ≈ 0.09**
- That crossing is the threshold, measured — not asserted — from tens
  of thousands of simulated trials

<!-- notes: Describe the plot for an audience that can't see it: four
lines starting far apart, converging to one point, fanning back out in
the opposite order past the crossing. -->

---

# Sample results (10,000 trials/point, seed 1)

| p | d=3 | d=5 | d=7 | d=9 |
|------|-------|-------|-------|-------|
| 0.03 | 0.017 | 0.006 | 0.002 | 0.001 |
| 0.06 | 0.062 | 0.048 | 0.030 | 0.021 |
| 0.09 | 0.113 | 0.117 | 0.117 | 0.117 |
| 0.12 | 0.177 | 0.215 | 0.240 | 0.248 |

- At p=0.09 all four distances land within a few thousandths of each
  other — the empirical crossing
- Straight from the README's own measured sweep, no external numbers

<!-- notes: These are the project's own numbers, not literature values
— safe to state without a VERIFY marker. -->

---

# Limitations (honest v1 scope)

- X errors only; Z errors are symmetric and omitted
- Perfect, single-round syndrome measurement — no measurement errors,
  no repeated rounds (3D matching graph)
- Uniform cluster growth, not Delfosse–Nickerson's smallest-cluster-
  first weighted growth — same correctness, slightly lower threshold
- No MWPM baseline; the accuracy comparison above is from the
  literature <!-- VERIFY -->

<!-- notes: Keep this slide honest — it's what separates "toy demo"
from "engineering project with a clearly scoped v1." -->

---

# Why this is cool

- A few hundred lines of Go, running on a laptop, reproduce the single
  empirical signature that makes scalable quantum computing an
  engineering target rather than a hope
- The interesting idea isn't quantum mechanics — it's a textbook data
  structure (union-find) solving a real decoding problem in
  near-linear time
- The threshold isn't asserted, it's *measured*: run enough trials, and
  the crossing point just appears in your CSV

<!-- notes: Land the thesis of the whole talk/deck here before Q&A. -->

---

# What's next

- Measurement errors + repeated rounds (3D matching graph)
- Delfosse–Nickerson weighted (smallest-cluster-first) growth
- An MWPM baseline for direct accuracy comparison
- Repo: `github.com/Pisush/surface-code`

<!-- notes: Closing slide. Point to the repo link and invite questions.
-->
