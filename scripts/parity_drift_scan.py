#!/usr/bin/env python3
"""
Parity Drift Scanner - Rule 11 (Parity Drift Guard).

Static detectors for the backtest<->live drift root causes identified in
docs/backtest_live_parity_audit_report.md (RC-1 parallel implementations,
RC-2 inert contracts, RC-3 duplicated constants):

  11a  inert *Hash/*Version contract  - field assigned only as a passthrough
  11b  mode-branching in a shared path - if isBacktest / isLive / mode ==
  11c  risk constant hardcoded outside internal/risk/constants.go
  11d  more than one position-sizing entry point (expect a single risk.Size)
  11e  ad-hoc commissionBps fee math outside BrokerageFeeConfig

Report-only by default (exit 0) so it can run in CI as a non-blocking signal while
the R1-R5 refactors land. Pass --strict to fail the build (exit 1 on any finding);
that mode is intended to be enabled per the phased rollout in the audit report (§15)
once the periphery refactors clear today's known findings.

Usage:
    python scripts/parity_drift_scan.py [--strict]
"""

import re
import sys
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

SIZE_FUNC_RE = re.compile(
    r"func\s+(?:\([^)]*\)\s*)?(\w*(?:ComputeSize|PositionSize|CalculatePositionSize)\w*)\s*\("
)
MODE_BRANCH_RE = re.compile(
    r"\bif\b.*\b(isBacktest|backtestMode|isLive|liveMode)\b|\bmode\s*=="
)


@dataclass
class Violation:
    rule: str
    file: str
    line: int
    description: str


def _read(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8", errors="ignore")
    except OSError:
        return ""


def _rel(path: Path) -> str:
    try:
        return str(path.relative_to(ROOT))
    except ValueError:
        return str(path)


def check_11a_inert_contracts() -> list[Violation]:
    """*Hash/*Version field whose only assignment is a passthrough (x.Hash = y.Hash)."""
    out: list[Violation] = []
    for f in ROOT.glob("internal/**/*.go"):
        if "_test.go" in str(f):
            continue
        txt = _read(f)
        for m in re.finditer(r"^\s*(\w*(?:Hash|Version))\s+string", txt, re.M):
            name = m.group(1)
            assigns = re.findall(rf"\b{name}\s*=(?!=)", txt)
            passthrough = bool(re.search(rf"{name}\s*=\s*\w+\.{name}\b", txt))
            if assigns and passthrough and len(assigns) == 1:
                out.append(Violation("11a", _rel(f), 0,
                                     f"{name}: only ever assigned as a passthrough (inert parity contract)"))
    return out


def check_11b_mode_branch() -> list[Violation]:
    """Mode-branching inside strategy runners or the two engine orchestrators."""
    out: list[Violation] = []
    patterns = ["internal/strategy/**/*.go",
                "internal/backtest/engine.go",
                "internal/engine/live_engine.go"]
    for pat in patterns:
        for f in ROOT.glob(pat):
            if "_test.go" in str(f):
                continue
            for i, line in enumerate(_read(f).splitlines(), 1):
                if MODE_BRANCH_RE.search(line):
                    out.append(Violation("11b", _rel(f), i,
                                         f"mode-branch in shared path: {line.strip()}"))
    return out


def check_11c_risk_literals() -> list[Violation]:
    """Risk multipliers/thresholds hardcoded outside internal/risk/constants.go.

    Scoped to scalar multiplier/assignment sites (`x *= 0.75`, `cap := 0.02`); composite
    literals like `[4]float64{...}` (e.g. HMM probability vectors) are excluded to keep
    the signal focused on duplicated *sizing* constants (RC-3).
    """
    out: list[Violation] = []
    for f in ROOT.glob("internal/risk/*.go"):
        if f.name == "constants.go" or "_test.go" in str(f):
            continue
        for i, line in enumerate(_read(f).splitlines(), 1):
            if "float64{" in line or "[]float64" in line:
                continue  # composite literal, not a scalar risk constant
            if re.search(r"\*=\s*0\.(50|75|90|25)\b", line) or re.search(
                r"[:=]\s*0\.(02|30|002|25)\b", line
            ):
                out.append(Violation("11c", _rel(f), i,
                                     f"risk literal outside constants.go: {line.strip()}"))
    return out


def check_11d_sizing_entry_points() -> list[Violation]:
    """More than one distinct position-sizing entry point across internal/.

    Constructors (New*) and accessors are excluded; only functions that actually
    compute a size are counted, so the check asserts a single risk.Size kernel.
    """
    out: list[Violation] = []
    found: list[tuple[str, str]] = []
    for f in ROOT.glob("internal/**/*.go"):
        if "_test.go" in str(f):
            continue
        for n in SIZE_FUNC_RE.findall(_read(f)):
            if n.startswith("New") or n.endswith("Sizer"):
                continue  # constructor / type, not a sizing computation
            found.append((_rel(f), n))
    if len({n for _, n in found}) > 1:
        for path, n in sorted(set(found)):
            out.append(Violation("11d", path, 0,
                                 f"sizing entry point '{n}' (expect a single risk.Size)"))
    return out


def check_11e_flat_fee_math() -> list[Violation]:
    """Flat commissionBps fee math outside BrokerageFeeConfig."""
    out: list[Violation] = []
    for f in ROOT.glob("internal/**/*.go"):
        if f.name == "engine.go" or "_test.go" in str(f):
            continue
        for i, line in enumerate(_read(f).splitlines(), 1):
            if "commissionBps" in line and "/ 10000" in line:
                out.append(Violation("11e", _rel(f), i,
                                     f"ad-hoc fee math outside BrokerageFeeConfig: {line.strip()}"))
    return out


def main() -> None:
    strict = "--strict" in sys.argv[1:]

    violations: list[Violation] = []
    violations.extend(check_11a_inert_contracts())
    violations.extend(check_11b_mode_branch())
    violations.extend(check_11c_risk_literals())
    violations.extend(check_11d_sizing_entry_points())
    violations.extend(check_11e_flat_fee_math())

    if not violations:
        print("Parity Drift Guard (Rule 11): no findings.")
        sys.exit(0)

    label = "VIOLATION" if strict else "finding"
    print(f"\nParity Drift Guard (Rule 11): {len(violations)} {label}(s)"
          f"{'' if strict else ' (report-only)'}:\n")
    for v in sorted(violations, key=lambda v: (v.rule, v.file, v.line)):
        loc = f"{v.file}:{v.line}" if v.line else v.file
        print(f"  [{v.rule}] {loc}")
        print(f"          {v.description}")
        print()

    by_rule: dict[str, int] = {}
    for v in violations:
        by_rule[v.rule] = by_rule.get(v.rule, 0) + 1
    print("Summary by rule: " + ", ".join(f"{k}={by_rule[k]}" for k in sorted(by_rule)))

    if strict:
        sys.exit(1)
    print("\nReport-only mode (exit 0). Run with --strict to enforce as a gate.")
    sys.exit(0)


if __name__ == "__main__":
    main()
