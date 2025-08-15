#!/usr/bin/env python3
"""
Grouped bar chart of RPS by scenario and framework with % labels.

Usage:
    source .venv/bin/activate
    python3 bin/plot_benchmarks.py results/summary_all_n100000000_c256_keep.csv -o results/all_benchmarks.png

Notes:
- Uses matplotlib only (no seaborn), one chart per figure, no explicit colors.
- Keeps the order of scenarios and frameworks as they first appear in the CSV.
- Labels each bar as 100% for the max within its scenario; others are relative %.
"""

import argparse
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
from matplotlib.ticker import FuncFormatter


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("csv_path", help="Path to benchmarks CSV")
    parser.add_argument("-o", "--out", default="grouped_rps.png", help="Output image path")
    args = parser.parse_args()

    # Read CSV (column names must match: scenario,framework,requests_per_sec,...)
    df = pd.read_csv(args.csv_path)

    # Preserve appearance order for scenarios and frameworks
    scenario_order = pd.unique(df["scenario"])
    framework_order = pd.unique(df["framework"])
    df["scenario"] = pd.Categorical(df["scenario"], categories=scenario_order, ordered=True)
    df["framework"] = pd.Categorical(df["framework"], categories=framework_order, ordered=True)

    # Pivot to scenario x framework with values = requests_per_sec
    pivot = (
        df.pivot_table(index="scenario", columns="framework", values="requests_per_sec")
          .loc[scenario_order, framework_order]
    )

    # Prepare grouped bars
    x = np.arange(len(pivot.index))
    nfw = len(framework_order)
    bar_width = 0.8 / max(nfw, 1)  # total group width ~= 0.8
    # offsets centered around 0
    offsets = (np.arange(nfw) - (nfw - 1) / 2.0) * bar_width

    plt.figure(figsize=(13, 6))
    for i, fw in enumerate(framework_order):
        plt.bar(x + offsets[i], pivot[fw].values, width=bar_width, label=fw)

    plt.title("RPS by Scenario and Framework")
    plt.xlabel("Scenario")
    plt.ylabel("Requests per second (RPS)")
    plt.xticks(x, pivot.index, rotation=30, ha="right")
    plt.legend(title="Framework")
    plt.grid(axis="y", linestyle="--", alpha=0.4)
    plt.gca().yaxis.set_major_formatter(FuncFormatter(lambda y, _: f"{int(y):,}"))

    # Percent labels relative to the highest bar in each scenario group
    y_padding = float(pivot.values.max()) * 0.01
    for i, scenario in enumerate(pivot.index):
        group_vals = pivot.loc[scenario].values
        max_val = np.nanmax(group_vals)
        for j, val in enumerate(group_vals):
            if np.isnan(val):
                continue
            pct = (val / max_val * 100.0) if max_val > 0 else 0.0
            label = "100%" if np.isclose(val, max_val) else f"{pct:.0f}%"
            x_pos = x[i] + offsets[j]
            plt.text(x_pos, val + y_padding, label, ha="center", va="bottom", fontsize=9)

    plt.tight_layout()
    plt.savefig(args.out, dpi=220)
    print(f"Saved: {args.out}")

if __name__ == "__main__":
    main()
