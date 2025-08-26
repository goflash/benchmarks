#!/usr/bin/env python3
"""
Load all results into memory and build final summary in presentable form.

This script:
1. Loads all benchmark results from CSV files
2. Creates comprehensive comparison tables
3. Generates performance rankings
4. Creates detailed analysis reports
5. Saves results in multiple formats
"""

import json
import os
from datetime import datetime

import numpy as np
import pandas as pd


def parse_duration_to_ms(duration_str):
    """Convert duration string like '0s', '1.5s', '100ms' to milliseconds."""
    if pd.isna(duration_str) or duration_str == '':
        return 0.0

    duration_str = str(duration_str).strip()

    if duration_str.endswith('ms'):
        return float(duration_str[:-2])
    elif duration_str.endswith('s'):
        return float(duration_str[:-1]) * 1000
    elif duration_str.endswith('μs'):
        return float(duration_str[:-2]) / 1000
    else:
        # Try to parse as a number (assume milliseconds)
        try:
            return float(duration_str)
        except:
            return 0.0


def load_all_results(results_dir="results"):
    """Load all CSV results into a single DataFrame."""
    all_data = []

    if not os.path.exists(results_dir):
        print(f"Results directory {results_dir} not found")
        return pd.DataFrame()

    # Find only the main summary.csv files, not processed summaries
    for root, dirs, files in os.walk(results_dir):
        for file in files:
            if file == 'summary.csv':  # Only load main summary files
                file_path = os.path.join(root, file)
                try:
                    df = pd.read_csv(file_path)
                    if not df.empty and 'Framework' in df.columns and 'Scenario' in df.columns:
                        # Add source file information
                        df['source_file'] = file_path
                        all_data.append(df)
                        print(f"Loaded: {file_path}")
                except Exception as e:
                    print(f"Error loading {file_path}: {e}")

    if not all_data:
        return pd.DataFrame()

    # Combine all data
    combined = pd.concat(all_data, ignore_index=True)
    return combined


def create_performance_ranking(data):
    """Create performance ranking by framework."""
    if data.empty:
        return pd.DataFrame()

    # Calculate average RPS per framework
    ranking = data.groupby('Framework').agg({
        'RequestsPerSec': ['mean', 'std', 'min', 'max']
    }).round(2)

    # Flatten column names
    ranking.columns = ['avg_rps', 'std_rps', 'min_rps', 'max_rps']

    # Sort by average RPS
    ranking = ranking.sort_values('avg_rps', ascending=False)

    # Add rank
    ranking['rank'] = range(1, len(ranking) + 1)

    return ranking


def create_scenario_comparison(data):
    """Create detailed comparison by scenario."""
    if data.empty:
        return pd.DataFrame()

    # Check if we have enough data for comparison
    unique_frameworks = data['Framework'].nunique()
    if unique_frameworks < 2:
        print(f"Warning: Only {unique_frameworks} framework(s) found. Scenario comparison requires multiple frameworks.")
        return pd.DataFrame()

    # Pivot data for scenario comparison
    comparison = data.pivot_table(
        index='Scenario',
        columns='Framework',
        values='RequestsPerSec',
        aggfunc='mean'
    ).round(2)

    # Add scenario statistics only if we have numeric data
    if not comparison.empty and comparison.select_dtypes(include=[np.number]).shape[1] > 0:
        # Calculate numeric statistics first (before adding string columns)
        best_rps = comparison.max(axis=1)
        worst_rps = comparison.min(axis=1)
        best_framework = comparison.idxmax(axis=1)

        # Add the calculated statistics as new columns
        comparison['best_framework'] = best_framework
        comparison['best_rps'] = best_rps
        comparison['worst_rps'] = worst_rps
        comparison['rps_range'] = best_rps - worst_rps
        comparison['performance_gap_pct'] = (
            (best_rps - worst_rps) / worst_rps * 100
        ).round(1)

    return comparison


def create_framework_analysis(data):
    """Create detailed analysis for each framework."""
    if data.empty:
        return {}

    analysis = {}

    for framework in data['Framework'].unique():
        fw_data = data[data['Framework'] == framework]

        # Basic stats
        avg_rps = fw_data['RequestsPerSec'].mean()
        std_rps = fw_data['RequestsPerSec'].std()

        # Convert latency strings to numeric values
        if 'LatencyMean' in fw_data.columns:
            latency_mean_ms = fw_data['LatencyMean'].apply(parse_duration_to_ms)
            avg_latency = latency_mean_ms.mean()
        else:
            avg_latency = 0

        if 'LatencyP99' in fw_data.columns:
            latency_p99_ms = fw_data['LatencyP99'].apply(parse_duration_to_ms)
            avg_p99 = latency_p99_ms.mean()
        else:
            avg_p99 = 0

        # Best and worst scenarios
        scenario_performance = fw_data.groupby('Scenario')['RequestsPerSec'].mean()
        best_scenario = scenario_performance.idxmax()
        worst_scenario = scenario_performance.idxmin()

        # Consistency (lower std = more consistent)
        consistency_score = 1 - (std_rps / avg_rps) if avg_rps > 0 else 0

        analysis[framework] = {
            'avg_rps': round(avg_rps, 2),
            'std_rps': round(std_rps, 2),
            'avg_latency_ms': round(avg_latency, 2),
            'avg_p99_ms': round(avg_p99, 2),
            'best_scenario': best_scenario,
            'worst_scenario': worst_scenario,
            'consistency_score': round(consistency_score, 3),
            'total_tests': len(fw_data)
        }

    return analysis


def generate_summary_report(ranking, comparison, analysis):
    """Generate a comprehensive summary report."""
    report = {
        'generated_at': datetime.now().isoformat(),
        'summary': {
            'total_frameworks': len(ranking),
            'total_scenarios': len(comparison),
            'top_performer': ranking.index[0] if not ranking.empty else None,
            'performance_gap': ranking['avg_rps'].iloc[0] - ranking['avg_rps'].iloc[-1] if len(ranking) > 1 else 0
        },
        'ranking': ranking.to_dict('index'),
        'scenario_comparison': comparison.to_dict('index'),
        'framework_analysis': analysis
    }

    return report


def save_results(ranking, comparison, analysis, report, output_dir="results"):
    """Save all results in multiple formats."""
    os.makedirs(output_dir, exist_ok=True)

    # Save ranking
    ranking_path = os.path.join(output_dir, 'performance_ranking.csv')
    ranking.to_csv(ranking_path)
    print(f"Saved ranking: {ranking_path}")

    # Save scenario comparison
    comparison_path = os.path.join(output_dir, 'scenario_comparison.csv')
    comparison.to_csv(comparison_path)
    print(f"Saved comparison: {comparison_path}")

    # Save framework analysis
    analysis_path = os.path.join(output_dir, 'framework_analysis.json')
    with open(analysis_path, 'w') as f:
        json.dump(analysis, f, indent=2)
    print(f"Saved analysis: {analysis_path}")

    # Save comprehensive report
    report_path = os.path.join(output_dir, 'comprehensive_report.json')
    with open(report_path, 'w') as f:
        json.dump(report, f, indent=2)
    print(f"Saved report: {report_path}")


def print_summary(ranking, comparison, analysis):
    """Print a formatted summary to console."""
    print("\n" + "=" * 80)
    print("COMPREHENSIVE BENCHMARK SUMMARY")
    print("=" * 80)

    if ranking.empty:
        print("No data available for summary")
        return

    # Performance ranking
    print("\n🏆 PERFORMANCE RANKING")
    print("-" * 40)
    for i, (framework, row) in enumerate(ranking.iterrows(), 1):
        medal = "🥇" if i == 1 else "🥈" if i == 2 else "🥉" if i == 3 else "📊"
        print(f"{medal} {i}. {framework}: {row['avg_rps']:,.0f} RPS "
              f"(±{row['std_rps']:,.0f})")

    # Top performer analysis
    top_framework = ranking.index[0]
    top_analysis = analysis.get(top_framework, {})
    print(f"\n🏆 TOP PERFORMER: {top_framework}")
    print(f"   Average RPS: {top_analysis.get('avg_rps', 0):,.0f}")
    print(f"   Best Scenario: {top_analysis.get('best_scenario', 'N/A')}")
    print(f"   Consistency Score: {top_analysis.get('consistency_score', 0):.3f}")

    # Scenario insights
    if not comparison.empty:
        print(f"\n📊 SCENARIO INSIGHTS")
        print("-" * 40)
        for scenario, row in comparison.iterrows():
            best_fw = row.get('best_framework', 'N/A')
            best_rps = row.get('best_rps', 0)
            gap_pct = row.get('performance_gap_pct', 0)
            print(f"   {scenario}: {best_fw} ({best_rps:,.0f} RPS, "
                  f"{gap_pct:.1f}% gap)")
    else:
        print(f"\n📊 SCENARIO INSIGHTS")
        print("-" * 40)
        print("   Insufficient data for scenario comparison (need multiple frameworks)")

    print("\n" + "=" * 80)


def main():
    """Main function to build final summary."""
    import sys

    # Accept results directory as command line argument
    results_dir = sys.argv[1] if len(sys.argv) > 1 else "results"

    print(f"Loading all benchmark results from {results_dir}...")

    # Load all data
    data = load_all_results(results_dir)

    if data.empty:
        print("No benchmark data found")
        return

    print(f"Loaded {len(data)} benchmark records")

    # Create analyses
    print("\nCreating performance ranking...")
    ranking = create_performance_ranking(data)

    print("Creating scenario comparison...")
    comparison = create_scenario_comparison(data)

    print("Creating framework analysis...")
    analysis = create_framework_analysis(data)

    print("Generating comprehensive report...")
    report = generate_summary_report(ranking, comparison, analysis)

    # Save results to the specific results directory
    print("\nSaving results...")
    save_results(ranking, comparison, analysis, report, results_dir)

    # Print summary
    print_summary(ranking, comparison, analysis)

    print(f"\nAll results saved to: {results_dir}/")


if __name__ == "__main__":
    main()
