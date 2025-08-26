#!/usr/bin/env python3
"""
Generate README.md from template with actual benchmark results.
"""

import glob
import os
import re
from datetime import datetime


def get_latest_results_date():
    """Get the most recent results directory date."""
    results_dir = "results"
    if not os.path.exists(results_dir):
        return None

    dates = []
    for item in os.listdir(results_dir):
        if os.path.isdir(os.path.join(results_dir, item)) and re.match(r'\d{4}-\d{2}-\d{2}', item):
            dates.append(item)

    if not dates:
        return None

    return sorted(dates)[-1]


def get_framework_results(date):
    """Get framework results from the latest run."""
    # Try multiple possible file locations
    possible_files = [
        f"results/{date}/summary.csv",
        f"results/{date}/performance_ranking.csv",
        f"results/performance_ranking.csv"
    ]

    summary_file = None
    for file_path in possible_files:
        if os.path.exists(file_path):
            summary_file = file_path
            break

    if not summary_file:
        return {}

    results = {}
    try:
        with open(summary_file, 'r') as f:
            lines = f.readlines()
            if len(lines) < 2:
                return {}

            # Skip header
            for line in lines[1:]:
                parts = line.strip().split(',')
                if len(parts) >= 3:  # At least framework, avg_rps, rank
                    framework = parts[0]
                    try:
                        rps = float(parts[1]) if parts[1] else 0  # avg_rps column
                        if framework not in results:
                            results[framework] = {}
                        # Store average RPS for the framework
                        results[framework]['average'] = rps
                    except (ValueError, IndexError):
                        continue
    except Exception as e:
        print(f"Error reading {summary_file}: {e}")
        return {}

    return results


def calculate_average_rps(results, framework):
    """Calculate average RPS for a framework."""
    if framework not in results:
        return 0

    # If we have an average stored directly
    if 'average' in results[framework]:
        return results[framework]['average']

    # Otherwise calculate from individual scenarios
    rps_values = list(results[framework].values())
    if not rps_values:
        return 0

    return sum(rps_values) / len(rps_values)


def generate_performance_summary(results):
    """Generate a performance summary table."""
    if not results:
        return "No benchmark results available yet."

    frameworks = list(results.keys())
    if not frameworks:
        return "No benchmark results available yet."

    # Calculate averages
    framework_averages = {}
    for framework in frameworks:
        avg_rps = calculate_average_rps(results, framework)
        framework_averages[framework] = avg_rps

    # Sort by performance
    sorted_frameworks = sorted(framework_averages.items(), key=lambda x: x[1], reverse=True)

    summary = "### 🏆 Performance Summary\n\n"
    summary += "| Rank | Framework | Avg RPS | Performance |\n"
    summary += "|------|-----------|---------|-------------|\n"

    for i, (framework, avg_rps) in enumerate(sorted_frameworks, 1):
        if i == 1:
            performance = "🥇 **Best**"
        elif i == 2:
            performance = "🥈 **Second**"
        elif i == 3:
            performance = "🥉 **Third**"
        else:
            performance = "📊 **Good**"

        summary += f"| {i} | **{framework}** | {avg_rps:,.0f} | {performance} |\n"

    return summary


def main():
    """Generate README.md from template."""
    # Read template
    with open("README.template.md", "r") as f:
        template = f.read()

    # Get latest results date
    latest_date = get_latest_results_date()
    if not latest_date:
        print("No results found. Using template as-is.")
        with open("README.md", "w") as f:
            f.write(template)
        return

    print(f"Using results from: {latest_date}")

    # Get results
    results = get_framework_results(latest_date)

    # Replace placeholders
    template = template.replace("{{DATE}}", latest_date)

    # Add performance summary if we have results
    if results:
        performance_summary = generate_performance_summary(results)
        # Insert after the overview section
        template = template.replace(
            "### 🏆 Frameworks Under Test",
            f"### 🏆 Frameworks Under Test\n\n{performance_summary}"
        )

    # Write final README
    with open("README.md", "w") as f:
        f.write(template)

    print("README.md generated successfully!")


if __name__ == "__main__":
    main()
