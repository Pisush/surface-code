package surface

import (
	"fmt"
	"math/rand"
	"sync"
)

// splitmix64 is the SplitMix64 finalizer: a bijective mixer used to derive
// well-separated per-trial RNG seeds from a master seed.
func splitmix64(z uint64) uint64 {
	z += 0x9e3779b97f4a7c15
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// DeriveSeed deterministically derives an independent seed from a master
// seed and a sequence of indices (for example distance index, noise-rate
// index). Use it to give every point of a parameter sweep its own RNG
// stream while keeping the whole sweep reproducible from one master seed.
func DeriveSeed(master int64, indices ...int) int64 {
	z := splitmix64(uint64(master))
	for _, idx := range indices {
		z = splitmix64(z ^ uint64(int64(idx)))
	}
	return int64(z)
}

// LogicalFailures runs trials independent Monte Carlo shots on the lattice
// at physical error rate p and counts how many end in a logical error:
// sample iid X-flips, extract the syndrome, decode, and test the residual's
// homology class.
//
// Shots are spread across workers goroutines, but every trial i draws from
// its own RNG seeded by splitmix64(seed, i), so the result is fully
// deterministic given (seed, trials) — independent of the worker count and
// of goroutine scheduling.
//
// LogicalFailures returns an error for invalid trials, workers, or p.
func LogicalFailures(l *Lattice, dec Decoder, p float64, trials, workers int, seed int64) (int, error) {
	if l == nil || dec == nil {
		return 0, fmt.Errorf("surface: LogicalFailures requires a lattice and a decoder")
	}
	if trials <= 0 {
		return 0, fmt.Errorf("surface: trials must be positive, got %d", trials)
	}
	if workers <= 0 {
		return 0, fmt.Errorf("surface: workers must be positive, got %d", workers)
	}
	if p < 0 || p > 1 {
		return 0, fmt.Errorf("surface: physical error rate %v outside [0, 1]", p)
	}
	if workers > trials {
		workers = trials
	}

	counts := make([]int, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		// Contiguous trial ranges; the partition only affects which
		// goroutine runs a trial, never its RNG stream.
		lo := w * trials / workers
		hi := (w + 1) * trials / workers
		wg.Add(1)
		go func(w, lo, hi int) {
			defer wg.Done()
			failures := 0
			for i := lo; i < hi; i++ {
				rng := rand.New(rand.NewSource(int64(splitmix64(uint64(seed) ^ uint64(i)))))
				errs := l.SampleErrors(p, rng)
				correction := dec.Decode(l.Syndrome(errs))
				if l.IsLogicalError(ApplyCorrection(errs, correction)) {
					failures++
				}
			}
			counts[w] = failures
		}(w, lo, hi)
	}
	wg.Wait()

	total := 0
	for _, c := range counts {
		total += c
	}
	return total, nil
}
