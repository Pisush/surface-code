#!/usr/bin/env python3
"""Plot surface-code threshold curves from the CSV emitted by cmd/threshold.

Usage:
    threshold -trials 10000 > threshold.csv
    python3 scripts/plot_threshold.py threshold.csv threshold.png

One line per code distance d. Below the threshold, increasing d suppresses
the logical error rate; above it, increasing d makes things worse — the
curves cross at the threshold p_th (about 9% for this uniform-growth
union-find decoder under perfect syndrome measurement).

Only stdlib + matplotlib are required.
"""

import csv
import sys
from collections import defaultdict

import matplotlib.pyplot as plt

# Ordinal one-hue ramp (light -> dark = increasing distance), validated for
# monotone lightness, visible step gaps, and surface contrast.
RAMP = ["#86b6ef", "#5598e7", "#2a78d6", "#184f95"]
SURFACE = "#fcfcfb"
INK = "#0b0b0b"
INK_SECONDARY = "#52514e"
MUTED = "#898781"
GRID = "#e1e0d9"
BASELINE = "#c3c2b7"


def read_points(path):
    """Return {d: [(p, rate), ...]} sorted by p."""
    series = defaultdict(list)
    with open(path, newline="") as f:
        for row in csv.DictReader(f):
            series[int(row["d"])].append(
                (float(row["p"]), float(row["logical_error_rate"]))
            )
    return {d: sorted(pts) for d, pts in sorted(series.items())}


def main():
    if len(sys.argv) != 3:
        sys.exit(__doc__.strip())
    series = read_points(sys.argv[1])
    if not series:
        sys.exit("no data rows found in " + sys.argv[1])
    if len(series) > len(RAMP):
        sys.exit(f"ramp has {len(RAMP)} steps but CSV has {len(series)} distances")

    fig, ax = plt.subplots(figsize=(7.2, 4.8), dpi=160)
    fig.patch.set_facecolor(SURFACE)
    ax.set_facecolor(SURFACE)

    for color, (d, pts) in zip(RAMP, series.items()):
        ps, rates = zip(*pts)
        ax.plot(
            ps,
            rates,
            color=color,
            linewidth=2,
            marker="o",
            markersize=4.5,
            markeredgecolor=SURFACE,
            markeredgewidth=0.8,
            label=f"d = {d}",
        )
        # Direct label at the line end, in ink rather than series color.
        ax.annotate(
            f"d = {d}",
            xy=(ps[-1], rates[-1]),
            xytext=(6, 0),
            textcoords="offset points",
            va="center",
            fontsize=9,
            color=INK_SECONDARY,
        )

    # Identity line p_L = p as a reference for the pseudo-threshold region.
    lo = min(p for pts in series.values() for p, _ in pts)
    hi = max(p for pts in series.values() for p, _ in pts)
    ax.plot([lo, hi], [lo, hi], color=BASELINE, linewidth=1, linestyle=(0, (4, 4)))
    ax.annotate(
        "$p_L = p$",
        xy=(hi, hi),
        xytext=(-2, 10),
        textcoords="offset points",
        ha="right",
        fontsize=9,
        color=MUTED,
    )

    ax.set_xlabel("physical error rate $p$", color=INK)
    ax.set_ylabel("logical error rate $p_L$", color=INK)
    ax.set_title(
        "Surface code under union-find decoding: curves cross at the threshold",
        color=INK,
        fontsize=11,
        pad=12,
    )
    ax.grid(True, color=GRID, linewidth=0.7)
    ax.tick_params(colors=MUTED, labelsize=9)
    for side in ("top", "right"):
        ax.spines[side].set_visible(False)
    for side in ("left", "bottom"):
        ax.spines[side].set_color(BASELINE)
    ax.margins(x=0.02)
    ax.set_ylim(bottom=0)
    legend = ax.legend(
        loc="upper left",
        frameon=False,
        fontsize=9,
        title="code distance",
        title_fontsize=9,
    )
    for text in legend.get_texts():
        text.set_color(INK_SECONDARY)
    legend.get_title().set_color(MUTED)

    fig.tight_layout()
    fig.savefig(sys.argv[2], facecolor=SURFACE, bbox_inches="tight")
    print(f"wrote {sys.argv[2]}")


if __name__ == "__main__":
    main()
