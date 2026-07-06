package surface

import "fmt"

// UnionFindDecoder is a Delfosse–Nickerson style union-find decoder for
// the planar surface code under bit-flip noise.
//
// It works in two phases:
//
//  1. Growth. Every lit stabilizer starts as its own cluster. Each round,
//     every "active" cluster — odd number of defects and no contact with a
//     rough boundary — grows all edges on its frontier by half an edge.
//     Edges grown to full length fuse the clusters at their endpoints
//     (union-find with path halving and union by size; frontier lists are
//     concatenated shorter-into-longer). A cluster that touches a virtual
//     boundary node becomes neutral: the boundary is a valid terminal that
//     can absorb an unpaired defect. Growth stops when no active cluster
//     remains.
//
//  2. Peeling. The fully grown edges form an "erasure". A spanning forest
//     of the erasure is built, rooting each tree at a virtual boundary
//     node when the cluster touches a boundary. Leaves are peeled one at
//     a time: if the pendant vertex carries a defect, the leaf edge joins
//     the correction and the defect moves to the other endpoint; unpaired
//     defects thus migrate along the forest until they annihilate in
//     pairs or are absorbed by a boundary root.
//
// The returned correction always reproduces the input syndrome exactly.
// Union-find runs in near-linear time in the number of data qubits, which
// is what makes large Monte Carlo sweeps cheap; see the README for how its
// accuracy compares to minimum-weight perfect matching.
//
// A UnionFindDecoder holds no mutable state between calls and is safe for
// concurrent use by multiple goroutines.
type UnionFindDecoder struct {
	l *Lattice
}

// NewUnionFindDecoder returns a union-find decoder for the given lattice.
func NewUnionFindDecoder(l *Lattice) *UnionFindDecoder {
	if l == nil {
		panic("surface: NewUnionFindDecoder requires a non-nil Lattice")
	}
	return &UnionFindDecoder{l: l}
}

// Decode implements Decoder using union-find cluster growth followed by
// peeling. See the type documentation for the algorithm.
func (uf *UnionFindDecoder) Decode(lit []int) []bool {
	l := uf.l
	correction := make([]bool, l.NumDataQubits())
	if len(lit) == 0 {
		return correction
	}

	n := l.NumNodes()
	defect := make([]bool, n)
	for _, s := range lit {
		if s < 0 || s >= l.NumStabilizers() {
			panic(fmt.Sprintf("surface: syndrome contains stabilizer ID %d, valid range [0, %d)", s, l.NumStabilizers()))
		}
		if defect[s] {
			panic(fmt.Sprintf("surface: syndrome contains stabilizer ID %d twice", s))
		}
		defect[s] = true
	}

	growth := uf.grow(defect, lit)
	uf.peel(growth, defect, correction)

	// Every defect must have been annihilated or absorbed by a
	// boundary; anything else is a bug in the decoder, not bad input.
	for s := 0; s < l.NumStabilizers(); s++ {
		if defect[s] {
			panic(fmt.Sprintf("surface: internal error: defect left on stabilizer %d after peeling", s))
		}
	}
	return correction
}

// grow runs the cluster-growth phase and returns the per-edge growth
// stage: 0 (untouched), 1 (half grown), or 2 (fully grown, part of the
// erasure passed to peeling).
func (uf *UnionFindDecoder) grow(defect []bool, lit []int) []uint8 {
	l := uf.l
	n := l.NumNodes()
	growth := make([]uint8, l.NumDataQubits())

	// Union-find state. parity, boundary and frontier are only
	// meaningful at cluster roots.
	parent := make([]int, n)
	size := make([]int, n)
	parity := make([]bool, n)   // odd number of defects in cluster
	boundary := make([]bool, n) // cluster touches a rough boundary
	frontier := make([][]int, n)
	for v := 0; v < n; v++ {
		parent[v] = v
		size[v] = 1
		boundary[v] = l.IsVirtualNode(v)
		frontier[v] = append([]int(nil), l.IncidentEdges(v)...)
	}
	for _, s := range lit {
		parity[s] = true
	}

	find := func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]] // path halving
			x = parent[x]
		}
		return x
	}
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
		// Concatenate frontiers shorter-into-longer so total list
		// work stays near linear.
		if len(frontier[ra]) < len(frontier[rb]) {
			frontier[ra], frontier[rb] = frontier[rb], frontier[ra]
		}
		frontier[ra] = append(frontier[ra], frontier[rb]...)
		frontier[rb] = nil
	}

	seen := make([]bool, n)
	var active, fuse []int
	for {
		// Collect the roots of active clusters: odd parity, not
		// touching a boundary. Only clusters containing defects can
		// be odd, so scanning the lit stabilizers finds them all.
		active = active[:0]
		for _, s := range lit {
			r := find(s)
			if !seen[r] && parity[r] && !boundary[r] {
				seen[r] = true
				active = append(active, r)
			}
		}
		for _, r := range active {
			seen[r] = false
		}
		if len(active) == 0 {
			return growth
		}

		// Phase 1: every active cluster grows its frontier edges by
		// half an edge. Fusions are deferred so that growth within a
		// round is independent of cluster order.
		fuse = fuse[:0]
		for _, r := range active {
			keep := frontier[r][:0]
			for _, id := range frontier[r] {
				if growth[id] == 2 {
					continue // already fused, drop lazily
				}
				e := l.edges[id]
				if find(e.U) == find(e.V) {
					continue // internal edge, drop lazily
				}
				growth[id]++
				if growth[id] == 2 {
					fuse = append(fuse, id)
					continue
				}
				keep = append(keep, id)
			}
			frontier[r] = keep
		}

		// Phase 2: fully grown edges merge their endpoint clusters.
		for _, id := range fuse {
			union(l.edges[id].U, l.edges[id].V)
		}
	}
}

// peel builds a spanning forest of the erasure (edges with growth 2),
// rooting each tree at a virtual boundary node when one is present, and
// peels leaves: a pendant defect adds its leaf edge to the correction and
// hops to the other endpoint. defect and correction are updated in place.
func (uf *UnionFindDecoder) peel(growth []uint8, defect []bool, correction []bool) {
	l := uf.l
	n := l.NumNodes()

	type arc struct{ edge, to int }
	adj := make([][]arc, n)
	for id := range growth {
		if growth[id] != 2 {
			continue
		}
		e := l.edges[id]
		adj[e.U] = append(adj[e.U], arc{edge: id, to: e.V})
		adj[e.V] = append(adj[e.V], arc{edge: id, to: e.U})
	}

	// Spanning forest by BFS. Virtual nodes are visited first so that
	// any tree containing a boundary is rooted there: the root is peeled
	// last, letting the boundary absorb an unpaired defect.
	visited := make([]bool, n)
	root := make([]int, n)
	deg := make([]int, n) // degree in the forest
	forest := make([][]arc, n)
	inForest := make([]bool, len(growth))
	var queue []int
	bfs := func(start int) {
		if visited[start] || len(adj[start]) == 0 {
			return
		}
		visited[start] = true
		queue = append(queue[:0], start)
		for len(queue) > 0 {
			u := queue[0]
			queue = queue[1:]
			root[u] = start
			for _, a := range adj[u] {
				if visited[a.to] {
					continue
				}
				visited[a.to] = true
				inForest[a.edge] = true
				forest[u] = append(forest[u], arc{edge: a.edge, to: a.to})
				forest[a.to] = append(forest[a.to], arc{edge: a.edge, to: u})
				deg[u]++
				deg[a.to]++
				queue = append(queue, a.to)
			}
		}
	}
	for v := l.NumStabilizers(); v < n; v++ {
		bfs(v)
	}
	for v := 0; v < l.NumStabilizers(); v++ {
		bfs(v)
	}

	// Peel leaves. A node re-enters the queue when its degree drops to
	// one; the root of each tree is never peeled.
	leaves := queue[:0]
	for v := 0; v < n; v++ {
		if visited[v] && deg[v] == 1 && root[v] != v {
			leaves = append(leaves, v)
		}
	}
	for len(leaves) > 0 {
		u := leaves[len(leaves)-1]
		leaves = leaves[:len(leaves)-1]
		if deg[u] != 1 {
			continue // stale entry
		}
		// Find the one remaining forest edge at u.
		var live arc
		found := false
		for _, a := range forest[u] {
			if inForest[a.edge] {
				live = a
				found = true
				break
			}
		}
		if !found {
			panic("surface: internal error: leaf without a live forest edge")
		}
		inForest[live.edge] = false
		deg[u]--
		deg[live.to]--
		if defect[u] {
			correction[live.edge] = true
			defect[u] = false
			defect[live.to] = !defect[live.to]
		}
		if deg[live.to] == 1 && root[live.to] != live.to {
			leaves = append(leaves, live.to)
		}
	}

	// Defects that migrated onto virtual boundary nodes are absorbed.
	for v := l.NumStabilizers(); v < n; v++ {
		defect[v] = false
	}
}
