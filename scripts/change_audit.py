#!/usr/bin/env python3
"""OrcaAlgo Change Audit — Prevents destructive commits by detecting file delta magnitude.

Exit codes: 0 = safe, 1 = violation (blocking), 2 = config error
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
THRESHOLD_PATH = ROOT / "config" / "change-threshold.yaml"
CRITICAL_PATH = ROOT / "config" / "critical-paths.json"


@dataclass(frozen=True)
class AuditResult:
    passed: bool
    check_name: str
    message: str
    severity: str  # "PASS", "BLOCK", "WARN"


def load_thresholds() -> dict:
    if not THRESHOLD_PATH.exists():
        print(f"ERROR: threshold config not found: {THRESHOLD_PATH}")
        sys.exit(2)
    with open(THRESHOLD_PATH) as f:
        return yaml.safe_load(f)


def load_critical_paths() -> dict:
    if not CRITICAL_PATH.exists():
        print(f"WARN: critical paths config not found: {CRITICAL_PATH}")
        return {"paths": {}}
    with open(CRITICAL_PATH) as f:
        return json.load(f)


def get_staged_diff_lines() -> list[str]:
    """Return lines from `git diff --cached --numstat`."""
    try:
        out = subprocess.check_output(
            ["git", "diff", "--cached", "--numstat"],
            text=True, stderr=subprocess.DEVNULL,
        )
    except subprocess.CalledProcessError:
        return []
    return [l.strip() for l in out.splitlines() if l.strip()]


def get_staged_files() -> list[str]:
    """Return list of staged file paths."""
    try:
        out = subprocess.check_output(
            ["git", "diff", "--cached", "--name-only"],
            text=True, stderr=subprocess.DEVNULL,
        )
    except subprocess.CalledProcessError:
        return []
    return [l.strip() for l in out.splitlines() if l.strip()]


def parse_numstat(lines: list[str]) -> dict[str, tuple[int, int]]:
    """Parse git numstat into {filepath: (added, removed)}."""
    result = {}
    for line in lines:
        parts = line.split("\t")
        if len(parts) >= 3:
            try:
                added = int(parts[0]) if parts[0] != "-" else 0
                removed = int(parts[1]) if parts[1] != "-" else 0
                result[parts[2]] = (added, removed)
            except ValueError:
                continue
    return result


def run_audit(staged_only: bool = False) -> list[AuditResult]:
    thresholds = load_thresholds()
    critical = load_critical_paths()
    results: list[AuditResult] = []

    if staged_only:
        files = get_staged_files()
        numstat_lines = get_staged_diff_lines()
        numstat = parse_numstat(numstat_lines)
    else:
        try:
            out = subprocess.check_output(
                ["git", "diff", "HEAD~1", "--numstat"], text=True, stderr=subprocess.DEVNULL
            )
            numstat = parse_numstat(out.splitlines())
            files = [l.split("\t")[2] for l in out.splitlines() if "\t" in l and len(l.split("\t")) >= 3]
        except subprocess.CalledProcessError:
            files = []
            numstat = {}

    if not files:
        results.append(AuditResult(True, "no_changes", "No files changed", "PASS"))
        return results

    # Check 1: File count delta
    max_changed = thresholds.get("max_changed_files", 50)
    changed_count = len(files)
    if changed_count > max_changed:
        results.append(AuditResult(
            False, "file_count_delta",
            f"BLOCKED: {changed_count} files changed (max {max_changed}). "
            f"Split into smaller commits or use --bypass-guard with audit log.",
            "BLOCK",
        ))
    else:
        results.append(AuditResult(
            True, "file_count_delta",
            f"File count {changed_count} OK (max {max_changed})", "PASS",
        ))

    # Check 2: Line deletion percentage
    total_added = sum(a for a, _ in numstat.values())
    total_removed = sum(r for _, r in numstat.values())
    total_changed = total_added + total_removed
    max_del_pct = thresholds.get("max_deletion_pct", 30)

    if total_changed > 0:
        del_pct = (total_removed / total_changed) * 100
        if del_pct > max_del_pct:
            results.append(AuditResult(
                False, "line_deletion_pct",
                f"BLOCKED: {del_pct:.1f}% of changed lines are deletions "
                f"(max {max_del_pct}%). +{total_added}/-{total_removed}. "
                f"Consider restoring deleted functionality first.",
                "BLOCK",
            ))
        else:
            results.append(AuditResult(
                True, "line_deletion_pct",
                f"Line delta OK: +{total_added}/-{total_removed} ({del_pct:.1f}% deletions)", "PASS",
            ))

    # Check 3: Critical file protection
    critical_patterns = {}
    for name, cfg in critical.get("paths", {}).items():
        for pattern in cfg.get("files", []):
            critical_patterns[pattern] = name

    for f in files:
        for pattern, name in critical_patterns.items():
            if f.startswith(pattern.replace("/*", "/")) or f == pattern:
                results.append(AuditResult(
                    False, "critical_file_modified",
                    f"BLOCKED: Critical file modified: {f} ({name}). "
                    f"This file is protected by {CRITICAL_PATH.name}. "
                    f"Review changes carefully and use --bypass-guard if intentional.",
                    "BLOCK",
                ))
                break

    # Check 4: Test file deletion guard
    if thresholds.get("test_file_guard", True):
        deleted_test_files = [
            f for f, (a, r) in numstat.items()
            if ("_test" in f or f.endswith(".spec.cjs") or f.endswith(".spec.js"))
            and a == 0 and r > 0
        ]
        if deleted_test_files:
            results.append(AuditResult(
                False, "test_file_deletion",
                f"BLOCKED: {len(deleted_test_files)} test file(s) deleted without replacement: "
                f"{', '.join(deleted_test_files[:5])}{'...' if len(deleted_test_files) > 5 else ''}. "
                f"Restore tests or add replacement coverage.",
                "BLOCK",
            ))

    return results


def main():
    parser = argparse.ArgumentParser(description="OrcaAlgo Change Audit")
    parser.add_argument("--staged", action="store_true", help="Audit staged changes only")
    parser.add_argument("--json", action="store_true", help="Output JSON")
    args = parser.parse_args()

    results = run_audit(staged_only=args.staged)

    if args.json:
        print(json.dumps([{
            "check": r.check_name,
            "passed": r.passed,
            "severity": r.severity,
            "message": r.message,
        } for r in results], indent=2))
    else:
        blocks = [r for r in results if r.severity == "BLOCK"]
        for r in results:
            icon = "✅" if r.passed else ("⚠️" if r.severity == "WARN" else "🚫")
            print(f"  {icon} [{r.check_name}] {r.message}")

        if blocks:
            print(f"\n🚫 {len(blocks)} blocking violation(s) detected. Commit blocked.")
            sys.exit(1)

    print("\n✅ Change audit passed.")
    sys.exit(0)


if __name__ == "__main__":
    main()
