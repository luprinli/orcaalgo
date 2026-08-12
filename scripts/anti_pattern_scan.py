#!/usr/bin/env python3
"""
Anti-Pattern Scanner — Enforces the 18 Hard Prohibitions from AGENTS.md.

Scans the codebase for violations of the OrcaAlgo hard prohibitions:
  Rule 1:  No reimplementing canonical math in Go/Odin        [CRITICAL]
  Rule 2:  No IEEE 754 float for order prices                  [CRITICAL]
  Rule 3:  No legacy YAML strategy configs                     [MEDIUM]
  Rule 4:  No deployment without pre-flight                    [CRITICAL]
  Rule 5:  No skipped calibration audits                       [HIGH]
  Rule 6:  No full Kelly in production configs                 [CRITICAL]
  Rule 7:  No mutable domain models                            [HIGH]
  Rule 8:  No bypassing kill-switch re-entrancy guard          [CRITICAL]
  Rule 9:  No perfect fill assumption in backtests             [HIGH]
  Rule 10: No panic/throw for recoverable errors               [HIGH]
  Rule 11: No bypassing RiskPipeline (HP #17)                  [CRITICAL]

Usage: python scripts/anti_pattern_scan.py [--changed-only] [--format sarif] [--min-severity HIGH]
Exit:   0 if no CRITICAL+HIGH violations found (default), 1 if any blocking violation detected.
"""

import argparse
import datetime
import json
import os
import re
import subprocess
import sys
from dataclasses import dataclass, field
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

SEVERITY_RANK = {"CRITICAL": 0, "HIGH": 1, "MEDIUM": 2}

RULE_SEVERITY = {
    1: "CRITICAL", 2: "CRITICAL", 3: "MEDIUM", 4: "CRITICAL",
    5: "HIGH", 6: "CRITICAL", 7: "HIGH", 8: "CRITICAL",
    9: "HIGH", 10: "HIGH", 11: "CRITICAL",
}

CANONICAL_MATH_FUNCS = ["kelly", "brier", "platt", "wilson", "ewma"]
PRICE_FIELD_KEYWORDS = [
    "price", "fill", "limitPrice", "stopPrice",
    "avgPrice", "entryPrice", "exitPrice", "markPrice", "settlementPrice",
    "bidPrice", "askPrice", "lastPrice", "highPrice", "lowPrice", "openPrice",
    "closePrice",
]
KILLSWITCH_GUARDS = ["isLocked", "killSwitchReady"]


@dataclass
class Violation:
    rule: int
    file: str
    line: int
    description: str
    severity: str = field(default="")
    spec_ref: str = field(default="")

    def __post_init__(self):
        if not self.severity:
            self.severity = RULE_SEVERITY.get(self.rule, "HIGH")
        if not self.spec_ref:
            refs = {
                1: "§3.1-3.5", 2: "§6.8", 3: "§5.1", 4: "§9.3", 5: "§9.2",
                6: "§3.1.3", 7: "§2.1.2", 8: "§4.2.2", 9: "§9.1.3", 10: "Antipattern #10",
                11: "Antipattern #17",
            }
            self.spec_ref = refs.get(self.rule, "")


def get_changed_files() -> list[str]:
    """Get list of files changed vs origin/main."""
    try:
        out = subprocess.check_output(
            ["git", "diff", "origin/main...HEAD", "--name-only"],
            text=True, stderr=subprocess.DEVNULL,
        )
        return [l.strip() for l in out.splitlines() if l.strip()]
    except subprocess.CalledProcessError:
        return []


# ─── Rule 1: No reimplementing canonical math ──────────────────────────────
def check_rule_1(changed_only: bool = False) -> list[Violation]:
    violations = []
    changed = set(get_changed_files()) if changed_only else None
    for pattern in ["internal/**/*.go", "odin/**/*.odin", "cmd/**/*.go"]:
        for src_file in ROOT.glob(pattern):
            fname = str(src_file)
            if changed_only and not any(fname.replace("\\", "/").endswith(c) for c in changed):
                continue
            try:
                content = src_file.read_text(encoding="utf-8")
            except (OSError, UnicodeDecodeError):
                continue
            for func in CANONICAL_MATH_FUNCS:
                if re.search(rf"\bfunc\s+\w*{func}\w*\b", content, re.IGNORECASE) or \
                   re.search(rf"\b{func}\s*::\s*proc\b", content, re.IGNORECASE):
                    if "os/exec" not in content and "exec.Command" not in content and \
                       "ComputeEWMAVolatility" not in content:
                        for i, line in enumerate(content.splitlines(), 1):
                            if re.search(rf"\b{func}\b", line, re.IGNORECASE):
                                violations.append(Violation(
                                    1, fname, i,
                                    f"Possible reimplementation of canonical '{func}' in Go/Odin. "
                                    "Reference via subprocess (Go) or import (Python)."
                                ))
                                break
    return violations


# ─── Rule 2: No IEEE 754 float for order prices (HARDENED) ──────────────────
def check_rule_2(changed_only: bool = False) -> list[Violation]:
    """Flag float32/float64 used in price-related struct fields (not function params)."""
    violations = []
    changed = set(get_changed_files()) if changed_only else None

    price_pattern = "|".join(PRICE_FIELD_KEYWORDS)

    for go_file in ROOT.glob("internal/**/*.go"):
        if "_test.go" in str(go_file):
            continue
        fname = str(go_file)
        if changed_only and not any(fname.replace("\\", "/").endswith(c) for c in changed):
            continue
        try:
            content = go_file.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            continue
        for i, line in enumerate(content.splitlines(), 1):
            m = re.search(rf"(?i)\b({price_pattern})\b\s+(float32|float64)", line)
            if not m:
                continue
            if re.search(rf"(?i)(?:func|interface)\s", line) and "(" in line:
                continue
            if "(" in line[:m.start()]:
                continue
            violations.append(Violation(
                2, fname, i,
                f"Float type used for price-related field. Use fixed.Fixed (Go) or BIGINT (SQL)."
            ))
    return violations


# ─── Rule 3: No legacy YAML strategy configs ───────────────────────────────
def check_rule_3(changed_only: bool = False) -> list[Violation]:
    violations = []
    strategies_dir = ROOT / "configs" / "strategies"
    if not strategies_dir.exists():
        return violations
    for cfg_file in strategies_dir.glob("*.y*ml"):
        if changed_only:
            continue
        if not cfg_file.name.endswith(".gkr.yaml"):
            violations.append(Violation(
                3, str(cfg_file), 0,
                "Legacy YAML strategy config found. All strategies must use .gkr.yaml format."
            ))
    return violations


# ─── Rule 4: Pre-flight config presence ─────────────────────────────────────
def check_rule_4(changed_only: bool = False) -> list[Violation]:
    violations = []
    preflight_dir = ROOT / "orca" / "preflight"
    if not preflight_dir.exists() or not (preflight_dir / "checklist.py").exists():
        violations.append(Violation(
            4, "orca/preflight/checklist.py", 0,
            "Pre-flight checklist module missing. 'orca preflight' must gate all deployments."
        ))
    return violations


# ─── Rule 5: Calibration audit recency (HARDENED) ───────────────────────────
def check_rule_5(changed_only: bool = False) -> list[Violation]:
    """Check actual calibration audit recency — warn if >90 days."""
    violations = []
    calib_dir = ROOT / "orca" / "calibration"
    if not calib_dir.exists():
        violations.append(Violation(
            5, "orca/calibration/", 0,
            "Calibration audit module missing. Quarterly 'orca calibrate' is mandatory."
        ))
        return violations

    latest_json = ROOT / "reports" / "calibration" / "latest.json"
    if latest_json.exists():
        try:
            data = json.loads(latest_json.read_text())
            ts = data.get("generated_at", "")
            if ts:
                audit_date = datetime.datetime.fromisoformat(ts.replace("Z", "+00:00"))
                days_ago = (datetime.datetime.now(datetime.timezone.utc) - audit_date).days
                if days_ago > 90:
                    violations.append(Violation(
                        5, str(latest_json), 0,
                        f"Calibration audit is {days_ago} days old (>90 day quarterly threshold). "
                        "Run 'orca calibrate' to update."
                    ))
            else:
                violations.append(Violation(
                    5, str(latest_json), 0,
                    "Calibration audit report missing 'generated_at' timestamp."
                ))
        except (json.JSONDecodeError, KeyError, ValueError):
            violations.append(Violation(
                5, str(latest_json), 0,
                "Calibration audit report exists but is malformed or missing timestamp."
            ))
    else:
        violations.append(Violation(
            5, "reports/calibration/latest.json", 0,
            "No calibration audit report found. Quarterly 'orca calibrate' runs are mandatory."
        ))
    return violations


# ─── Rule 6: No full Kelly in production configs ─────────────────────────────
def check_rule_6(changed_only: bool = False) -> list[Violation]:
    violations = []
    strategies_dir = ROOT / "configs" / "strategies"
    if not strategies_dir.exists():
        return violations
    for gkr_file in strategies_dir.glob("*.gkr.yaml"):
        if changed_only:
            continue
        try:
            content = gkr_file.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            continue
        for i, line in enumerate(content.splitlines(), 1):
            if re.search(r"(multiplier|kelly_fraction|sizing_multiplier)\s*:\s*(1\.0|0\.9\d*)", line, re.IGNORECASE):
                violations.append(Violation(
                    6, str(gkr_file), i,
                    "Full Kelly multiplier detected. Fractional Kelly (k=0.25) is mandatory in production."
                ))
            if "full.kelly" in line.lower() or "full_kelly" in line.lower():
                violations.append(Violation(
                    6, str(gkr_file), i,
                    "Full Kelly reference detected. Use fractional Kelly with all three attenuators."
                ))
    return violations


# ─── Rule 7: No mutable domain models (HARDENED) ─────────────────────────────
def check_rule_7(changed_only: bool = False) -> list[Violation]:
    violations = []
    changed = set(get_changed_files()) if changed_only else None

    # Python: check for frozen=True in Pydantic models
    for py_file in ROOT.glob("orca/**/*.py"):
        if "__pycache__" in str(py_file):
            continue
        fname = str(py_file)
        if changed_only and not any(fname.replace("\\", "/").endswith(c) for c in changed):
            continue
        try:
            content = py_file.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            continue
        if "class" in content and ("BaseModel" in content):
            if "frozen=True" not in content:
                for i, line in enumerate(content.splitlines(), 1):
                    if "class " in line and "BaseModel" in line:
                        violations.append(Violation(
                            7, fname, i,
                            "Pydantic BaseModel without frozen=True. All domain models must be immutable."
                        ))
                        break

    # Python: also flag @dataclass with frozen=False
    for py_file in ROOT.glob("orca/**/*.py"):
        if "__pycache__" in str(py_file):
            continue
        fname = str(py_file)
        if changed_only and not any(fname.replace("\\", "/").endswith(c) for c in changed):
            continue
        try:
            content = py_file.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            continue
        for i, line in enumerate(content.splitlines(), 1):
            if "@dataclass(frozen=False)" in line or re.match(r"@dataclass\s*$", line.strip()):
                # Check next few lines for a class declaration
                next_lines = content.splitlines()[i:i+5]
                for j, nl in enumerate(next_lines):
                    if re.match(r"class\s+\w+", nl):
                        violations.append(Violation(
                            7, fname, i + j,
                            "@dataclass without frozen=True. Domain models must use frozen=True."
                        ))
                        break

    # Go: only check domain directories (reduced false-positives)
    for go_file in ROOT.glob("internal/{strategy,risk,broker,backtest,propfirm,types}/**/*.go"):
        if "_test.go" in str(go_file):
            continue
        fname = str(go_file)
        if changed_only and not any(fname.replace("\\", "/").endswith(c) for c in changed):
            continue
        try:
            content = go_file.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            continue
        if "type " in content and "struct" in content:
            has_constructor = bool(re.search(r"func New\w+\(", content))
            for i, line in enumerate(content.splitlines(), 1):
                if re.match(r"^\t[A-Z]\w+\s+", line) and not has_constructor:
                    violations.append(Violation(
                        7, fname, i,
                        "Exported struct field without constructor pattern. "
                        "Go domain structs should use unexported fields with constructor-only initialization."
                    ))
                    break
    return violations


# ─── Rule 8: Kill-switch re-entrancy guard (HARDENED) ───────────────────────
def check_rule_8(changed_only: bool = False) -> list[Violation]:
    """Verify BOTH _isLocked AND _killSwitchInFlight appear in Trigger function."""
    violations = []
    kill_switch_file = ROOT / "internal" / "risk" / "kill_switch.go"
    if not kill_switch_file.exists():
        violations.append(Violation(
            8, "internal/risk/kill_switch.go", 0,
            "Kill-switch implementation file not found."
        ))
        return violations

    try:
        content = kill_switch_file.read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError):
        return violations

    has_is_locked = "isLocked" in content
    has_in_flight = "killSwitchReady" in content

    if not has_is_locked:
        violations.append(Violation(
            8, str(kill_switch_file), 0,
            "Kill-switch missing isLocked guard. Required for re-entrancy prevention."
        ))
    if not has_in_flight:
        violations.append(Violation(
            8, str(kill_switch_file), 0,
            "Kill-switch missing killSwitchReady guard. Required for re-entrancy prevention."
        ))

    # Enhanced: verify both appear in the Trigger method specifically
    if has_is_locked and has_in_flight:
        # Extract the Trigger function body
        trigger_match = re.search(
            r"func\s+\(.*\*KillSwitch\)\s+Trigger\s*\([^)]*\)\s*\{",
            content
        )
        if trigger_match:
            start = trigger_match.start()
            brace_count = 0
            trigger_body = ""
            for i, ch in enumerate(content[start:], start):
                if ch == "{":
                    brace_count += 1
                elif ch == "}":
                    brace_count -= 1
                    if brace_count == 0:
                        trigger_body = content[start:i+1]
                        break
            if trigger_body:
                if "isLocked" not in trigger_body:
                    violations.append(Violation(
                        8, str(kill_switch_file), 0,
                        "Kill-switch: isLocked found in file but NOT inside Trigger(). "
                        "The re-entrancy guard must be checked in Trigger."
                    ))
                if "killSwitchReady" not in trigger_body:
                    violations.append(Violation(
                        8, str(kill_switch_file), 0,
                        "Kill-switch: killSwitchReady found in file but NOT inside Trigger(). "
                        "The re-entrancy guard must be checked in Trigger."
                    ))
    return violations


# ─── Rule 9: No perfect fill assumption ─────────────────────────────────────
def check_rule_9(changed_only: bool = False) -> list[Violation]:
    violations = []
    backtest_dir = ROOT / "internal" / "backtest"
    if not backtest_dir.exists():
        return violations

    combined = ""
    for go_file in backtest_dir.glob("*.go"):
        if go_file.name.endswith("_test.go"):
            continue
        try:
            combined += go_file.read_text(encoding="utf-8") + "\n"
        except (OSError, UnicodeDecodeError):
            continue
    fee_file = ROOT / "internal" / "model" / "fee.go"
    if fee_file.exists():
        try:
            combined += fee_file.read_text(encoding="utf-8") + "\n"
        except (OSError, UnicodeDecodeError):
            pass
    fill_file = ROOT / "internal" / "model" / "fill.go"
    if fill_file.exists():
        try:
            combined += fill_file.read_text(encoding="utf-8") + "\n"
        except (OSError, UnicodeDecodeError):
            pass

    checks = {
        "fillProbability": "fill probability modeling",
        "spread": "spread crossing",
        "makerfee": "maker/taker fees",
        "adverse_selection": "adverse selection haircut",
        "slippage": "price slippage",
    }
    missing = set()
    for pattern, description in checks.items():
        if pattern.lower() not in combined.lower():
            missing.add(description)
    if missing:
        violations.append(Violation(
            9, str(ROOT / "internal" / "backtest" / "engine.go"), 0,
            f"Backtest engine may assume perfect fills. Missing: {', '.join(missing)}."
        ))
    return violations


# ─── Rule 10: No panic/throw for recoverable errors ──────────────────────────
def check_rule_10(changed_only: bool = False) -> list[Violation]:
    violations = []
    for go_file in ROOT.glob("internal/**/*.go"):
        if go_file.name == "main.go":
            continue
        try:
            content = go_file.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            continue
        for i, line in enumerate(content.splitlines(), 1):
            if "panic(" in line and "//" not in line.split("panic(")[0]:
                violations.append(Violation(
                    10, str(go_file), i,
                    "panic() call in recoverable context. Return errors instead."
                ))
    return violations


# ─── Rule 11: No bypassing RiskPipeline (HP #17) ──────────────────────────
def check_rule_11(changed_only: bool = False) -> list[Violation]:
    """Flag NewEngine() followed by Run() without intervening WirePipeline() in Go files.

    HP #17: Engine.generateSignal and LiveEngine.ProcessTick both route through
    RiskPipeline.ProcessSignal/ReconcileFill. New risk checks must be added to the
    pipeline, not duplicated in each engine.
    """
    violations = []
    changed = set(get_changed_files()) if changed_only else None
    targets = ["internal/backtest/**/*.go", "internal/engine/**/*.go", "cmd/**/*.go"]

    for pattern in targets:
        for go_file in ROOT.glob(pattern):
            if go_file.name.endswith("_test.go"):
                continue
            fname = str(go_file)
            if changed_only and not any(fname.replace("\\", "/").endswith(c) for c in changed):
                continue
            try:
                content = go_file.read_text(encoding="utf-8")
            except (OSError, UnicodeDecodeError):
                continue

            lines = content.splitlines()

            for i, line in enumerate(lines):
                stripped = line.strip()
                if not stripped:
                    continue
                if "//" in stripped and stripped.strip().startswith("//"):
                    continue

                if "NewEngine(" in stripped or "NewEngineBuilder(" in stripped:
                    needle = "WirePipeline"
                    has_pipeline = False
                    for j in range(i + 1, min(i + 15, len(lines))):
                        if needle in lines[j] and "//" not in lines[j].split(needle)[0]:
                            has_pipeline = True
                            break
                        if re.match(rf"^\s*[a-zA-Z_]\w*\.Run\b|^\s*Run\(", lines[j]):
                            break
                    if not has_pipeline:
                        found_run = False
                        for j in range(i + 1, min(i + 20, len(lines))):
                            if re.match(rf"^\s*[a-zA-Z_]\w*\.Run\b|^\s*Run\(", lines[j]):
                                found_run = True
                                break
                        if found_run:
                            violations.append(Violation(
                                11, fname, i + 1,
                                "NewEngine() followed by Run() without WirePipeline(). "
                                "Engine.generateSignal must route through RiskPipeline.ProcessSignal. "
                                "Call engine.WirePipeline() before engine.Run()."
                            ))
    return violations


# ─── Output formatters ───────────────────────────────────────────────────────
def format_text(violations: list[Violation], min_severity: str = "HIGH") -> str:
    min_rank = SEVERITY_RANK.get(min_severity, 1)
    filtered = [v for v in violations if SEVERITY_RANK.get(v.severity, 1) <= min_rank]
    if not filtered:
        return "All 10 hard prohibitions: PASSED"
    lines = [f"\n{len(filtered)} HARD PROHIBITION VIOLATION(S) FOUND (severity >= {min_severity}):\n"]
    for v in sorted(filtered, key=lambda v: (SEVERITY_RANK.get(v.severity, 99), v.rule, v.file)):
        lines.append(f"  [{v.severity}] [RULE {v.rule}] {v.file}:{v.line}  (§{v.spec_ref})")
        lines.append(f"              {v.description}")
        lines.append("")
    return "\n".join(lines)


def format_sarif(violations: list[Violation]) -> str:
    """Output SARIF v2.1.0 format for CI dashboard integration."""
    results = []
    for v in violations:
        results.append({
            "ruleId": f"orca/anti-pattern/R{v.rule}",
            "level": "error" if v.severity == "CRITICAL" else "warning",
            "message": {"text": v.description},
            "locations": [{
                "physicalLocation": {
                    "artifactLocation": {"uri": v.file.replace("\\", "/")},
                    "region": {"startLine": v.line} if v.line > 0 else {},
                }
            }],
            "properties": {"severity": v.severity, "spec_ref": v.spec_ref},
        })
    sarif = {
        "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
        "version": "2.1.0",
        "runs": [{"tool": {"driver": {"name": "OrcaAlgo Anti-Pattern Scanner"}}, "results": results}],
    }
    return json.dumps(sarif, indent=2)


# ─── Main ────────────────────────────────────────────────────────────────────
def main():
    parser = argparse.ArgumentParser(description="OrcaAlgo Anti-Pattern Scanner")
    parser.add_argument("--changed-only", action="store_true", help="Scan only changed files vs origin/main")
    parser.add_argument("--format", choices=["text", "sarif"], default="text", help="Output format")
    parser.add_argument("--min-severity", choices=["CRITICAL", "HIGH", "MEDIUM"], default="HIGH",
                        help="Minimum severity to report (default: HIGH)")
    args = parser.parse_args()

    checks = [
        check_rule_1, check_rule_2, check_rule_3, check_rule_4, check_rule_5,
        check_rule_6, check_rule_7, check_rule_8, check_rule_9, check_rule_10,
        check_rule_11,
    ]

    violations: list[Violation] = []
    for check_fn in checks:
        violations.extend(check_fn(changed_only=args.changed_only))

    if args.format == "sarif":
        print(format_sarif(violations))
    else:
        print(format_text(violations, min_severity=args.min_severity))

    min_rank = SEVERITY_RANK.get(args.min_severity, 1)
    blocking = [v for v in violations if SEVERITY_RANK.get(v.severity, 1) <= min_rank]
    sys.exit(1 if blocking else 0)


if __name__ == "__main__":
    main()
