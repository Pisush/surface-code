> DRAFT — conference talk outline for surface-code (github.com/Pisush/surface-code)

# Talk outline: Watching a quantum error-correction threshold emerge from a union-find decoder in Go

**Format:** 25-30 min conference talk
**Audience:** Go engineers; no quantum computing background assumed, but
comfortable with union-find, graphs, and Monte Carlo-style benchmarking.

## CFP abstract (150-200 words)

Quantum error correction lives or dies on one number: the threshold.
Below it, bigger error-correcting codes make your logical qubit
exponentially more reliable. Above it, bigger makes things worse. Whether
large-scale quantum computing is even possible depends on which side of
that line your hardware and decoder land on.

This talk builds a distance-d surface code from scratch in Go —
simplified to classical bit-flip errors and parity checks, so it's just
booleans and integers, no amplitudes — and a union-find decoder in the
style of Delfosse and Nickerson: grow clusters around triggered parity
checks, fuse them with union-find, peel the result into a correction.
We'll trace the algorithm on a small lattice, then run a parallel,
fully-deterministic Monte Carlo sweep across code distances and error
rates live, and watch the logical-error curves for every distance
converge at a single crossing point — the threshold, empirically, on
stage.

We'll close with the honest tradeoff: union-find is near-linear time but
slightly less accurate than the "proper" minimum-weight-matching
decoders, and why that tradeoff is exactly what makes it useful in
practice.

## Section breakdown with timings

**1. Cold open — the one number that decides if this is possible (3 min)**
- State the threshold theorem in one sentence: below p_th, bigger codes
  suppress errors exponentially; above it, bigger codes amplify them.
- Show the punchline chart first (the crossing curves) with no
  explanation yet — "by the end of this talk, we'll have generated this
  ourselves."

**2. The classical simplification (4 min)**
- Why this project only models bit-flip (X) errors + Z-stabilizer
  syndromes: it turns a quantum problem into a closed classical parity
  problem — bits, not amplitudes.
- Quick tour of the lattice: `Coord{Row, Col}` on a doubled grid, data
  qubits on edges, Z-stabilizers on vertices, rough boundaries left and
  right that can absorb a chain endpoint.
- One-line mental model: "a flipped qubit lights up its neighboring
  stabilizers like a tripwire."

**3. The union-find decoder, traced by hand (8 min)**
- Draw a small (d=3 or d=5) lattice on a slide with 2-3 lit stabilizers.
- Growth phase: clusters grow from each lit stabilizer, frontier edges
  extend, two clusters touch and fuse — narrate `union()`'s path-halving
  and union-by-size in the source as the mechanism, not the point; the
  point is *why* growth stops (neutral clusters: even parity or hit a
  boundary).
- Peeling phase: the grown edges form an erasure; walk the spanning-
  forest-then-peel-leaves process by hand on the same drawing, showing a
  defect "hop" and eventual annihilation or boundary absorption.
- Land the complexity claim: near-linear in the number of data qubits —
  why that matters for running 10,000+ trials per data point.

**4. Checking for a real logical error (3 min)**
- The subtlety: a "wrong" correction is usually harmless (it differs from
  the truth by a closed loop); only a boundary-to-boundary chain flips
  the encoded qubit.
- Show `IsLogicalError`'s trick: parity of the residual on one boundary
  column stands in for a full homology check.

**5. Union-find vs. MWPM — the tradeoff, honestly (3 min)**
- MWPM (Blossom matching) is more accurate — cite the literature
  thresholds (~10.3% MWPM vs ~9.9% weighted union-find) — but
  super-quadratic worst case.
- This project's uniform-growth union-find measures ~9%: same
  correctness guarantee (corrects every error up to weight ⌊(d-1)/2⌋),
  cheaper, slightly less accurate.
- Why that tradeoff is the right one for real-time decoding — decoders
  have a hard latency budget dictated by the physical qubit's coherence
  time.

**6. Live demo: the threshold sweep (6 min — see demo plan below)**

**7. Close / Q&A (1-2 min)**
- What v2 looks like: measurement errors + repeated rounds (3D matching
  graph), weighted growth, maybe an MWPM baseline for comparison.
- Repo link, one-line pitch: "a threshold is just a crossing point in a
  CSV file, if you're willing to run enough trials."

## Live-demo plan

**Setup (before the talk):** repo built (`go build ./cmd/threshold`);
Python 3 + matplotlib available for `scripts/plot_threshold.py`; a
pre-generated `threshold.csv` and `threshold.png` on hand as a fallback;
terminal font large; the `docs/threshold.png` from the README open in a
tab as a "sample results" comparison point.

**Demo 1 — decode one syndrome by hand, then in code (during section 3,
~2 min):**
- Build a small lattice in a Go playground / REPL-style script:
  `l, _ := surface.NewLattice(5)`, sample one noisy error configuration,
  print the syndrome, decode it, print the correction. Compare the
  printed correction against the hand-drawn trace from the slide.

**Demo 2 — the threshold sweep, live (section 6, ~6 min):**
```sh
./threshold -d 3,5,7,9 -trials 2000 -seed 1 -v > threshold.csv
```
- Use a reduced trial count (2,000 rather than the README's 10,000) so
  the sweep finishes in the talk's time budget; note out loud that this
  is a real accuracy/runtime tradeoff, the same theme as the union-find
  vs. MWPM discussion.
- Narrate the `-v` per-point progress lines scrolling on stderr while the
  CSV builds on stdout.
- Plot it:
  ```sh
  python3 scripts/plot_threshold.py threshold.csv threshold.png
  ```
  and display the resulting PNG — the crossing point should be visibly
  close to the README's own ~0.09-0.10, even at reduced trial count.
- Explicitly call out determinism: re-run with `-workers 1` vs. the
  default `runtime.NumCPU()` and diff the two CSVs live to show they're
  byte-identical — a good "here's what taking concurrency seriously buys
  you" beat.
- Fallback if live plotting misbehaves: the pre-generated
  `threshold.png` plus a pre-run terminal transcript, narrated
  identically.

**Time budget safety valve:** if running short, cut the `-workers 1` vs.
default determinism diff (demo 2's last beat) rather than the plot itself
— the crossing curve is the payoff and must survive any cut.
