// Command threshold estimates logical error rates of the planar surface
// code under a union-find decoder, sweeping the physical error rate for a
// set of code distances, and writes the results as CSV:
//
//	d,p,logical_error_rate,trials
//
// The sweep is embarrassingly parallel and fully deterministic given the
// master seed: every (d, p) point derives its own seed, and within a point
// every trial draws from its own RNG stream, so results do not depend on
// the worker count or goroutine scheduling.
//
// Example:
//
//	threshold -trials 10000 -seed 1 > threshold.csv
//	python3 scripts/plot_threshold.py threshold.csv threshold.png
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	surface "github.com/Pisush/surface-code"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "threshold:", err)
		os.Exit(1)
	}
}

func run(args []string, out *os.File) error {
	fs := flag.NewFlagSet("threshold", flag.ContinueOnError)
	var (
		distances = fs.String("d", "3,5,7,9", "comma-separated odd code distances")
		pMin      = fs.Float64("pmin", 0.01, "smallest physical error rate in the sweep")
		pMax      = fs.Float64("pmax", 0.16, "largest physical error rate in the sweep")
		pStep     = fs.Float64("pstep", 0.01, "sweep step for the physical error rate")
		trials    = fs.Int("trials", 10000, "Monte Carlo trials per (d, p) point")
		workers   = fs.Int("workers", runtime.NumCPU(), "worker goroutines")
		seed      = fs.Int64("seed", 1, "master seed; the full sweep is deterministic given this")
		verbose   = fs.Bool("v", false, "log per-point progress to stderr")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}

	ds, err := parseDistances(*distances)
	if err != nil {
		return err
	}
	ps, err := sweep(*pMin, *pMax, *pStep)
	if err != nil {
		return err
	}
	if *trials <= 0 {
		return fmt.Errorf("-trials must be positive, got %d", *trials)
	}
	if *workers <= 0 {
		return fmt.Errorf("-workers must be positive, got %d", *workers)
	}

	fmt.Fprintln(out, "d,p,logical_error_rate,trials")
	for di, d := range ds {
		lattice, err := surface.NewLattice(d)
		if err != nil {
			return err
		}
		dec := surface.NewUnionFindDecoder(lattice)
		for pi, p := range ps {
			start := time.Now()
			pointSeed := surface.DeriveSeed(*seed, di, pi)
			failures, err := surface.LogicalFailures(lattice, dec, p, *trials, *workers, pointSeed)
			if err != nil {
				return err
			}
			rate := float64(failures) / float64(*trials)
			fmt.Fprintf(out, "%d,%.4f,%.6f,%d\n", d, p, rate, *trials)
			if *verbose {
				fmt.Fprintf(os.Stderr, "d=%d p=%.4f rate=%.6f (%d/%d) in %v\n",
					d, p, rate, failures, *trials, time.Since(start).Round(time.Millisecond))
			}
		}
	}
	return nil
}

// parseDistances parses a comma-separated list of odd distances >= 3.
func parseDistances(s string) ([]int, error) {
	var ds []int
	for _, part := range strings.Split(s, ",") {
		d, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid distance %q: %v", part, err)
		}
		if d < 3 || d%2 == 0 {
			return nil, fmt.Errorf("distance must be odd and >= 3, got %d", d)
		}
		ds = append(ds, d)
	}
	if len(ds) == 0 {
		return nil, fmt.Errorf("no distances given")
	}
	return ds, nil
}

// sweep returns min, min+step, ... up to and including max (within half a
// step of floating-point slack).
func sweep(min, max, step float64) ([]float64, error) {
	if step <= 0 {
		return nil, fmt.Errorf("-pstep must be positive, got %v", step)
	}
	if min < 0 || max > 1 || min > max {
		return nil, fmt.Errorf("sweep [%v, %v] is not inside [0, 1]", min, max)
	}
	var ps []float64
	for i := 0; ; i++ {
		p := min + float64(i)*step
		if p > max+step/2 {
			break
		}
		ps = append(ps, p)
	}
	return ps, nil
}
