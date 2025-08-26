#!/usr/bin/env python3
"""
Load all per-framework and per-test CSV files and render them as charts.

This script:
1. Loads all CSV files from results directory
2. Creates individual charts for each scenario
3. Creates a comprehensive comparison chart
4. Saves charts to the results directory
"""

import os

import matplotlib.pyplot as plt
import numpy as np
import pandas as pd


def find_csv_files(results_dir="results"):
    """Find all CSV files in the results directory."""
    csv_files = []

    if not os.path.exists(results_dir):
        print(f"Results directory {results_dir} not found")
        return csv_files

    # Look for CSV files in subdirectories
    for root, dirs, files in os.walk(results_dir):
        for file in files:
            if file.endswith('.csv'):
                csv_files.append(os.path.join(root, file))

    return sorted(csv_files)


def load_csv_data(csv_file):
    """Load data from a CSV file."""
    try:
        df = pd.read_csv(csv_file, dtype={'Framework': str, 'Scenario': str})
        # Filter out rows with invalid Framework values
        df = df.dropna(subset=['Framework'])
        df = df[df['Framework'].str.strip() != '']
        return df
    except Exception as e:
        print(f"Error loading {csv_file}: {e}")
        return None


def create_scenario_chart(data, scenario_name, output_dir):
    """Create a chart for a specific scenario."""
    if data.empty:
        return

    plt.figure(figsize=(10, 6))

    # Create bar chart
    frameworks = data['Framework'].unique()
    # Filter out any non-string framework values
    frameworks = [fw for fw in frameworks if isinstance(fw, str) and fw.strip()]

    if not frameworks:
        print(f"No valid frameworks found for scenario: {scenario_name}")
        plt.close()
        return

    rps_values = []
    for fw in frameworks:
        fw_data = data[data['Framework'] == fw]
        if len(fw_data) > 0:
            rps_values.append(fw_data['RequestsPerSec'].iloc[0])
        else:
            rps_values.append(0)

    colors = ['#4caf50', '#2196f3', '#ff9800', '#9c27b0', '#f44336']
    bars = plt.bar(frameworks, rps_values, color=colors)

    # Add value labels on bars
    for bar, value in zip(bars, rps_values):
        height = bar.get_height()
        plt.text(bar.get_x() + bar.get_width() / 2.,
                 height + height * 0.01,
                 f'{value:,.0f}', ha='center', va='bottom',
                 fontweight='bold')

    plt.title(f'{scenario_name} - Requests per Second', fontsize=14, fontweight='bold')
    plt.ylabel('Requests per Second', fontsize=12)
    plt.xlabel('Framework', fontsize=12)
    plt.grid(axis='y', linestyle='--', alpha=0.7)

    # Rotate x-axis labels if needed
    if len(frameworks) > 4:
        plt.xticks(rotation=45)

    plt.tight_layout()

    # Save chart
    chart_filename = f"{scenario_name.lower().replace(' ', '_').replace('/', '_')}_rps.png"
    chart_path = os.path.join(output_dir, chart_filename)
    plt.savefig(chart_path, dpi=300, bbox_inches='tight')
    plt.close()

    print(f"Created chart: {chart_path}")


def create_comprehensive_chart(all_data, output_dir):
    """Create a comprehensive comparison chart for all scenarios."""
    if all_data.empty:
        return

    # Pivot data for grouped bar chart
    pivot_data = all_data.pivot_table(
        index='Scenario',
        columns='Framework',
        values='RequestsPerSec',
        aggfunc='mean'
    )

    # Create grouped bar chart
    fig, ax = plt.subplots(figsize=(14, 8))

    scenarios = pivot_data.index
    frameworks = pivot_data.columns
    x = np.arange(len(scenarios))
    width = 0.15

    colors = ['#4caf50', '#2196f3', '#ff9800', '#9c27b0', '#f44336']

    for i, framework in enumerate(frameworks):
        values = pivot_data[framework].values
        bars = ax.bar(x + i * width, values, width, label=framework, color=colors[i % len(colors)])

        # Add value labels
        for bar, value in zip(bars, values):
            if not np.isnan(value):
                height = bar.get_height()
                ax.text(bar.get_x() + bar.get_width() / 2., height + height * 0.01,
                        f'{value:,.0f}', ha='center', va='bottom', fontsize=8)

    ax.set_xlabel('Scenarios', fontsize=12)
    ax.set_ylabel('Requests per Second', fontsize=12)
    ax.set_title('Comprehensive Framework Performance Comparison', fontsize=14, fontweight='bold')
    ax.set_xticks(x + width * (len(frameworks) - 1) / 2)
    ax.set_xticklabels(scenarios, rotation=45, ha='right')
    ax.legend(title='Framework')
    ax.grid(axis='y', linestyle='--', alpha=0.7)

    plt.tight_layout()

    # Save chart
    chart_path = os.path.join(output_dir, 'all_benchmarks.png')
    plt.savefig(chart_path, dpi=300, bbox_inches='tight')
    plt.close()

    print(f"Created comprehensive chart: {chart_path}")


def create_summary_table(all_data, output_dir):
    """Create a summary table with all results."""
    if all_data.empty:
        return

    # Group by framework and calculate averages
    summary = all_data.groupby('Framework').agg({
        'RequestsPerSec': ['mean', 'std']
    }).round(2)

    # Flatten column names
    summary.columns = ['avg_rps', 'std_rps']

    # Sort by average RPS
    summary = summary.sort_values('avg_rps', ascending=False)

    # Save summary table
    summary_path = os.path.join(output_dir, 'summary_table.csv')
    summary.to_csv(summary_path)

    print(f"Created summary table: {summary_path}")

    # Print summary to console
    print("\n" + "=" * 80)
    print("PERFORMANCE SUMMARY")
    print("=" * 80)
    print(summary.to_string())
    print("=" * 80)


def create_cumulative_comparison_chart(data, output_dir):
    """Create a cumulative comparison chart with all tests, frameworks as bars."""
    if data.empty:
        return

    # Group by framework and scenario, calculate mean RPS
    grouped = data.groupby(['Framework', 'Scenario'])['RequestsPerSec'].mean().reset_index()

    # Pivot to get frameworks as columns and scenarios as rows
    pivot_data = grouped.pivot(index='Scenario', columns='Framework', values='RequestsPerSec')

    # Create the chart
    plt.figure(figsize=(16, 10))

    # Set up the bar positions
    scenarios = pivot_data.index
    frameworks = pivot_data.columns
    n_scenarios = len(scenarios)
    n_frameworks = len(frameworks)

    # Create color palette
    colors = plt.cm.Set3(np.linspace(0, 1, n_frameworks))

    # Bar width and positions
    bar_width = 0.15
    x_pos = np.arange(n_scenarios)

    # Create bars for each framework
    for i, framework in enumerate(frameworks):
        values = pivot_data[framework].values
        # Handle NaN values
        values = np.nan_to_num(values, nan=0)

        plt.bar(x_pos + i * bar_width, values, bar_width,
                label=framework, color=colors[i], alpha=0.8)

    # Customize the chart
    plt.xlabel('Test Scenarios', fontsize=14, fontweight='bold')
    plt.ylabel('Requests Per Second (RPS)', fontsize=14, fontweight='bold')
    plt.title('🏆 Comprehensive Framework Performance Comparison\nAll Test Scenarios',
              fontsize=16, fontweight='bold', pad=20)

    # Set x-axis labels
    plt.xticks(x_pos + bar_width * (n_frameworks - 1) / 2, scenarios, rotation=45, ha='right')

    # Add legend
    plt.legend(bbox_to_anchor=(1.05, 1), loc='upper left')

    # Add grid
    plt.grid(axis='y', alpha=0.3, linestyle='--')

    # Format y-axis to show values in K format
    ax = plt.gca()
    ax.yaxis.set_major_formatter(plt.FuncFormatter(lambda x, p: f'{x / 1000:.0f}K' if x >= 1000 else f'{x:.0f}'))

    # Adjust layout
    plt.tight_layout()

    # Save the chart
    output_path = os.path.join(output_dir, "all_benchmarks.png")
    plt.savefig(output_path, dpi=300, bbox_inches='tight', facecolor='white')
    plt.close()

    print(f"Cumulative comparison chart saved: {output_path}")


def main():
    """Main function to load CSV files and create charts."""
    import sys

    # Accept results directory as command line argument
    results_dir = sys.argv[1] if len(sys.argv) > 1 else "results"

    print(f"Loading CSV files from {results_dir} and creating charts...")

    # Find all CSV files
    csv_files = find_csv_files(results_dir)

    if not csv_files:
        print(f"No CSV files found in {results_dir}")
        return

    print(f"Found {len(csv_files)} CSV files")

    # Load all data
    all_data = []
    for csv_file in csv_files:
        data = load_csv_data(csv_file)
        if data is not None and not data.empty:
            all_data.append(data)
            print(f"Loaded: {csv_file}")

    if not all_data:
        print("No valid data found")
        return

    # Combine all data
    combined_data = pd.concat(all_data, ignore_index=True)

    # Create output directory for images (changed from charts to images)
    output_dir = os.path.join(results_dir, "images")
    os.makedirs(output_dir, exist_ok=True)

    # Create individual scenario charts
    scenarios = combined_data['Scenario'].unique()
    for scenario in scenarios:
        scenario_data = combined_data[combined_data['Scenario'] == scenario]
        create_scenario_chart(scenario_data, scenario, output_dir)

    # Create comprehensive chart
    create_comprehensive_chart(combined_data, output_dir)

    # Create cumulative comparison chart (new)
    create_cumulative_comparison_chart(combined_data, output_dir)

    # Create summary table
    create_summary_table(combined_data, output_dir)

    print(f"\nAll charts saved to: {output_dir}")


if __name__ == "__main__":
    main()
