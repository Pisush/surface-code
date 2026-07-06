DRAFT — podcast script for surface-code (github.com/Pisush/surface-code)

# Episode: Watching a Threshold Cross

*Two hosts. HOST A is a curious generalist — sharp, asks the questions a
smart listener would ask, no quantum computing background. HOST B built
the project and explains it, occasionally geeking out.*

---

**HOST A:** Okay, so today we're talking about quantum error correction,
which — I'll be honest — sounds like a topic where I'm going to nod a lot
and understand very little.

**HOST B:** (laughs) I promise it's not that bad. And actually the project
we're talking about today deliberately strips out most of the quantum part.
It's called surface-code, it's written in Go, and the whole point is you
can understand the interesting bit — the error-correction and decoding —
without ever touching quantum mechanics directly.

**HOST A:** Wait, how do you do quantum error correction without quantum
mechanics?

**HOST B:** So here's the trick. This project only simulates one kind of
error — a bit-flip, they call it an X error — and it only tracks one kind
of check, called a Z-stabilizer. And it turns out when you restrict
yourself to just that pair, the whole problem collapses into something
completely classical. It's bits and booleans the whole way down. No
amplitudes, no superposition math anywhere in the code.

**HOST A:** So it's like — a simplified toy model that still captures the
real difficulty.

**HOST B:** Exactly. And the difficulty is real, because the underlying
problem — why do we even need error correction — is real. Let's back up.
Physical qubits are incredibly fragile. Stray noise flips them. And unlike
a classical bit, you can't just copy a qubit to keep a backup — that's
actually forbidden, the no-cloning theorem. So the workaround the field
converged on is: instead of protecting one physical qubit, you spread one
*logical* qubit — the thing you actually compute with — across a whole
grid of physical qubits.

**HOST A:** So more physical qubits, but they're all working together to
protect one "real" qubit of information.

**HOST B:** Right. And the way you catch errors without ruining the
information is with these things called stabilizers — think of them as
sensors sitting between the data qubits, each one constantly checking
"is the parity of my neighbors even or odd." If a data qubit flips, the
stabilizers next to it light up — like tripwires. Crucially, the
stabilizer tells you *something* changed nearby without telling you what
the encoded qubit's actual state is. That's the whole trick — you can
detect without disturbing.

**HOST A:** Okay so you've got this grid of qubits and tripwires. Where
does Go come in?

**HOST B:** The project builds this layout — a planar surface code,
distance d, d has to be odd — as an explicit coordinate system. Every
data qubit and every stabilizer gets a `Coord`, row and column, on a
"doubled" grid, so which stabilizers a given qubit touches becomes pure
arithmetic instead of a lookup table you'd have to build and maintain.
And "distance d" isn't just the grid size — it's the length of the
shortest error chain that could sneak past the code undetected. Bigger d
means you need a longer, less likely chain of simultaneous errors to
fool it. Which sounds like "bigger is just better" — and that's where it
gets interesting, because it's not unconditionally true.

**HOST A:** This is the threshold thing you mentioned before we started
recording.

**HOST B:** This is the threshold thing. Every combination of a noise
model and a decoder has a critical physical error rate — call it
p-threshold. Below that number, making the code bigger makes your
logical error rate go down, exponentially, as you increase distance.
Above it, bigger makes things *worse* — more qubits just means more
chances for an error chain to sneak all the way across the lattice.

**HOST A:** So there's this knife edge, and which side you're on
completely flips whether "add more qubits" is a good idea or a bad idea.

**HOST B:** Completely flips it. This isn't a minor implementation
detail — it's the number that determines whether large-scale,
fault-tolerant quantum computing is even possible. If your hardware
noise is above threshold for every decoder you can build in real time,
you're stuck; no amount of engineering scale gets you out. Below
threshold, you have a dial you can turn as far as you want.

**HOST A:** Okay so that's the "why does this matter" part. Now — how do
you actually catch the errors? You mentioned a decoder.

**HOST B:** Right, so remember — all you get to see, as a decoder, is
which stabilizers lit up. You never see the actual errors, just the
tripwires. Your job is to guess a correction that's consistent with what
you saw. And the decoder in this project is a union-find decoder, in the
style of a paper by Delfosse and Nickerson.

**HOST A:** Union-find — like the data structure from an intro algorithms
class? Disjoint sets, path compression?

**HOST B:** That exact one. It's kind of delightful that this very
standard, very teachable data structure turns out to be load-bearing for
a leading-edge quantum decoding scheme. Walk me through it — actually,
let me walk you through it.

Picture the lattice with a few stabilizers lit up —
those are your "defects." Each lit stabilizer starts life as its own tiny
cluster. Then you run this in rounds: every cluster that's still "active" —
meaning it has an odd number of defects in it and hasn't touched a
boundary yet — grows outward. Specifically every edge on its frontier
grows by half a step each round.

**HOST A:** Half a step — why half?

**HOST B:** Because an edge is shared by two clusters. If both clusters
growing toward each other each contribute half, the edge finishes
exactly when both sides have reached it, so growth stays even regardless
of scheduling. When an edge finishes, it fuses the two clusters at its
endpoints — and that's where union-find earns its keep: merging
clusters and tracking which one has an odd defect count is exactly the
union-by-size, path-halving bookkeeping from an algorithms course.

**HOST A:** And you said something about boundaries acting weird.

**HOST B:** Right — the left and right edges of the lattice are "rough"
boundaries, and they act like a free pass: a chain of errors that runs
off the edge doesn't need a partner defect to explain it, the boundary
itself can absorb it. So a cluster that grows enough to touch a boundary
node immediately goes "neutral" and stops growing. Boundaries are
terminals — basically free defects sitting at the edge of the world.
Growth stops entirely the moment no cluster is still active — every
odd-parity region has either found a partner or reached a boundary. What's
left is a bunch of fully-grown edges, called the erasure: the region the
decoder is now confident contains the real error chain, even without
knowing the exact path.

**HOST A:** And then the "peel" part.

**HOST B:** You take that erasure and build a spanning forest out of it,
rooting each tree at a boundary node if one's present. Then you
repeatedly strip off leaves: whenever a leaf is carrying a defect, its
edge joins the correction, and the defect "hops" to the leaf's neighbor.
Defects walk inward, leaf after leaf, until pairs meet and annihilate, or
one walks all the way to a boundary root and gets absorbed.

**HOST A:** So — grow until you're confident you've bracketed the error,
then peel inward to actually solve for the correction.

**HOST B:** Exactly. Grow, fuse, peel. And the beautiful part is the
complexity: neither phase does more than roughly constant work per edge,
so the whole decode runs in near-linear time in the number of data
qubits. That matters for two reasons. One — a real-time decoder,
correcting a physical quantum computer as it runs, has a hard latency
budget dictated by how long a qubit stays coherent, so near-linear is a
requirement, not a nice-to-have. Two — even for research, you want tens
of thousands of trials per data point across multiple distances and error
rates, and if decoding were quadratic that sweep stops being a
laptop-scale job.

**HOST A:** Is there a more accurate decoder that's slower?

**HOST B:** Minimum-weight perfect matching, MWPM — sometimes called
Blossom matching. It finds the true optimal pairing of defects, so it's
more accurate, but super-quadratic in the worst case. So there's a real
tradeoff: union-find gives up a bit of accuracy for a lot of speed, which
is why it's a leading candidate for real-time decoding <!-- VERIFY -->.
This project deliberately leaves MWPM out — out of scope by design, so
union-find's tradeoffs stay the center of attention.

**HOST A:** Okay, the decoder spits out a correction. How do you know if
it actually worked?

**HOST B:** This is a subtlety I like a lot. A "wrong" correction is
usually harmless — if it differs from the truth by a closed loop, or a
chain that stays on one side of the lattice, the encoded logical qubit
never changes. The only kind of "wrong" that matters is a chain spanning
the lattice left to right. And you can check for that without tracing
anything: take the residual — original error XOR'd with the correction —
and check its parity along one boundary column of qubits. Any residual
with a trivial syndrome decomposes into loops and same-side chains, which
always cross that column an even number of times; a genuine logical error
crosses it an odd number of times. One parity check over d qubits stands
in for what would otherwise be a full topological computation.

**HOST A:** That's satisfying. Okay — I want to actually see this
threshold thing you keep talking about. You said there's a command that
produces it?

**HOST B:** Yeah, let's do it. There's a command, `cmd/threshold`. Let me
actually run it right now.

**HOST A:** Let's do it.

**HOST B:** Okay, I'm building it — `go build ./cmd/threshold` — and now
I'll run it. By default it sweeps code distances 3, 5, 7, and 9, physical
error rate from about 1% up to 16% in steps, and it runs ten thousand
trials for every point on that grid, fully parallel across every CPU
core I've got.

**HOST A:** And this doesn't take forever?

**HOST B:** It doesn't — that's the whole point of the near-linear
decoder, this finishes on a laptop. And here's the thing that's kind of
wild: even spread across a bunch of goroutines with no control over
which one runs which trial first, the output is bit-for-bit identical
every time you run it with the same seed. Every data point derives its
own seed from a master seed, and every trial inside that point gets its
own independent random stream, so concurrency introduces zero
nondeterminism — that's exactly the kind of thing that's easy to get
subtly wrong, a shared RNG or a result that depends on CPU count, so it
was worth the effort up front. Anyway — it writes a CSV, four columns:
distance, physical error rate, measured logical error rate, trial count.
Then a little Python script turns that CSV into a chart.

**HOST A:** Okay, describe the chart to people who can't see it.

**HOST B:** So you've got physical error rate along the x-axis, logical
error rate on the y-axis, and one line per code distance — four lines,
for three, five, seven, nine. On the left side of the chart, at low
physical error rates, the lines are stacked with the biggest distance on
the bottom — meaning bigger codes have dramatically lower logical error
rates, sometimes an order of magnitude lower comparing d=9 to d=3 at the
low end. Then as you sweep rightward toward higher physical error rates,
all four lines converge and cross each other at almost exactly the same
point, right around eight or nine percent physical error rate. And past
that crossing point, the order flips completely — now the biggest
distance code has the *worst* logical error rate.

**HOST A:** And that crossing point is the threshold.

**HOST B:** That crossing point is the threshold, yeah. Visually it looks
like — imagine four lines that start far apart, converge to basically one
point, and then fan back out on the other side in the opposite order.
That single crossing point is the empirical signature of everything we
talked about earlier — it's not a theoretical claim anymore, it's a
number you actually measured by running tens of thousands of simulated
noisy qubits through a decoder and counting failures.

**HOST A:** And why does seeing that crossing matter so much, beyond just
being a cool chart?

**HOST B:** Because it's the difference between "quantum error correction
is a nice mathematical idea" and "quantum error correction is an
engineering path to a working large-scale quantum computer." Below that
crossing point, you have a knob — code distance — that suppresses your
logical error rate arbitrarily far, just by building a bigger code. Above
it, no amount of scaling saves you; you need better qubits or a better
decoder first. That crossing point is quite literally the dividing line
the whole field is trying to get its hardware under.

**HOST A:** That's a great place to land this. So, quick summary for
anyone who tuned out during the algorithm part — build a grid of qubits,
watch for parity tripwires, use union-find to grow-fuse-peel your way to
a guess at the error, check one boundary column for the real logical
error, and if you sweep enough trials, you'll watch the curves cross.

**HOST B:** That's it. That's the whole show.

**HOST A:** Great episode. Thanks for building this and then explaining
it to me like I'm five.

**HOST B:** Anytime. Go clone the repo, it's called surface-code, it's
all Go, it's small enough to read in an evening.

---

*[END OF EPISODE]*
