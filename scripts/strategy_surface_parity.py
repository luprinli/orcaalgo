#!/usr/bin/env python3
"""Strategy-surface parity check (frontend <-> backend).

Verifies that the strategy surface is consistent across the six sources of
truth that must all agree whenever a strategy is added, renamed, or removed:

  1. internal/strategy/registry.go      — Go runner factory table
  2. internal/risk/regime_activation.go — regime activation matrix
  3. configs/universe.json              — active matrix universe
  4. configs/strategies/*.gkr.yaml      — GKR IR configs (HP#3)
  5. internal/db/fixtures.go            — DB seed (strategies table)
  6. web/src/data/strategyCatalog.ts    — frontend catalog (UI)

A strategy is "primary" (user-facing / matrix-swept) when it appears in the
universe or the catalog. Registry aliases (e.g. "orb" -> OrbRunner) are not
primary and are exempt from the .gkr.yaml + catalog requirement.

Exit code 0 = no drift; 1 = drift found.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent

ALIASES = {
    "orb", "breakout", "grid", "trend", "scalp", "macd_rsi", "rsi2",
    "donchian", "keltner", "ichimoku", "stat_arb", "vol_arb",
    "time_series_momentum",
}


def read_registry_factories() -> set[str]:
    text = (PROJECT_ROOT / "internal" / "strategy" / "registry.go").read_text(encoding="utf-8")
    return set(re.findall(r'"([a-z_0-9]+)":\s*func\(\) Strategy', text))


def read_regime_entries() -> set[str]:
    text = (PROJECT_ROOT / "internal" / "risk" / "regime_activation.go").read_text(encoding="utf-8")
    # entries are declared as "strategy_id": { ... StrategyID: "strategy_id", ...
    ids = set(re.findall(r'"([a-z_0-9]+)":\s*\{', text))
    ids.update(re.findall(r'StrategyID:\s*"([a-z_0-9]+)"', text))
    return ids


def read_universe_strategies() -> set[str]:
    data = json.loads((PROJECT_ROOT / "configs" / "universe.json").read_text(encoding="utf-8"))
    return set(data.get("strategies", []))


def read_gkr_strategies() -> set[str]:
    # The .gkr.yaml FILE STEM is the strategy ID used by the Go registry
    # (e.g. rsi2_reversion.gkr.yaml -> "rsi2_reversion"); the embedded `id:`
    # field uses hyphenated display form and is not the registry key.
    out: set[str] = set()
    for p in (PROJECT_ROOT / "configs" / "strategies").glob("*.gkr.yaml"):
        # Path.stem strips only the last suffix (.yaml); drop ".gkr.yaml" too.
        out.add(p.name[:-len(".gkr.yaml")])
    return out


def read_fixture_seed_types() -> set[str]:
    text = (PROJECT_ROOT / "internal" / "db" / "fixtures.go").read_text(encoding="utf-8")
    # StrategySeed literals are `{Name: "...", Type: "...", Parameters: ...}`;
    # BrokerProviderSeed uses `Type: "..."` too but follows with `Driver:`, so
    # anchor on the trailing `Parameters:` field to exclude providers.
    return set(re.findall(r'Type:\s*"([a-z_0-9]+)",\s*Parameters:', text))


def read_catalog() -> dict[str, bool]:
    text = (PROJECT_ROOT / "web" / "src" / "data" / "strategyCatalog.ts").read_text(encoding="utf-8")
    entries: dict[str, bool] = {}
    # Each entry is one object literal: { typeKey: 'x', ..., inEngine: bool, ... }
    for m in re.finditer(r"typeKey:\s*'([a-z_0-9]+)'[^}]*?inEngine:\s*(true|false)", text, re.S):
        entries[m.group(1)] = m.group(2) == "true"
    return entries


def main() -> int:
    parser = argparse.ArgumentParser(description="Strategy-surface parity check")
    parser.add_argument("--json", action="store_true", help="Emit JSON report")
    args = parser.parse_args()

    registry = read_registry_factories()
    regime = read_regime_entries()
    universe = read_universe_strategies()
    gkr = read_gkr_strategies()
    fixtures = read_fixture_seed_types()
    catalog = read_catalog()

    problems: list[str] = []
    warnings: list[str] = []

    # 1. Every catalog entry marked inEngine must have a backend factory.
    for key, in_engine in catalog.items():
        if in_engine and key not in registry:
            problems.append(f"catalog entry '{key}' (inEngine=true) has no registry factory")

    # 2. Every universe strategy needs a factory + .gkr.yaml + catalog entry.
    for s in sorted(universe):
        if s not in registry:
            problems.append(f"universe strategy '{s}' has no registry factory")
        if s not in gkr:
            problems.append(f"universe strategy '{s}' has no .gkr.yaml config")
        if s not in catalog:
            problems.append(f"universe strategy '{s}' missing from strategyCatalog.ts")

    # 3. Every DB seed Type needs a backend factory.
    for s in sorted(fixtures):
        if s not in registry:
            problems.append(f"fixtures seed Type '{s}' has no registry factory")

    # 4. Primary factories (non-alias): missing .gkr.yaml is a warning (HP#3
    #    recommends it); missing regime entry is a warning (permissive default).
    for s in sorted(registry - ALIASES):
        if s not in gkr:
            warnings.append(f"registry factory '{s}' has no .gkr.yaml config")
        if s not in regime:
            warnings.append(f"registry factory '{s}' not in regime matrix (permissive default)")

    report = {
        "registry_factories": len(registry),
        "regime_entries": len(regime),
        "universe_strategies": len(universe),
        "gkr_configs": len(gkr),
        "fixture_seed_types": len(fixtures),
        "catalog_entries": len(catalog),
        "errors": problems,
        "warnings": warnings,
    }

    if args.json:
        print(json.dumps(report, indent=2, sort_keys=True))
        return 1 if problems else 0

    print("=== Strategy-surface parity ===")
    print(f"  registry factories : {len(registry)}")
    print(f"  regime entries     : {len(regime)}")
    print(f"  universe strategies: {len(universe)}")
    print(f"  .gkr.yaml configs  : {len(gkr)}")
    print(f"  fixture seed types : {len(fixtures)}")
    print(f"  catalog entries    : {len(catalog)}")
    print()
    if warnings:
        print(f"WARNINGS ({len(warnings)}):")
        for w in warnings:
            print(f"  - {w}")
        print()
    if not problems:
        print("OK: no hard drift detected across the strategy surface.")
        return 0
    print(f"DRIFT: {len(problems)} error(s):")
    for p in problems:
        print(f"  - {p}")
    return 1


if __name__ == "__main__":
    sys.exit(main())
