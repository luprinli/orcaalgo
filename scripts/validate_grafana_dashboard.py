#!/usr/bin/env python3
"""Validate Grafana dashboard JSON files for correctness.

Checks that each dashboard file:
  - Is valid JSON.
  - Contains required top-level keys: title, panels, schemaVersion.
  - Every panel with a `targets` array has at least one target containing
    an `expr` (Prometheus) or `query` (SQL/datasource) key.
  - schemaVersion is a positive integer.

Usage:
    python scripts/validate_grafana_dashboard.py configs/grafana/*.json docs/grafana/dashboards/*.json
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

REQUIRED_TOP_KEYS = frozenset({"title", "panels", "schemaVersion"})
TARGET_EXPR_FIELD = frozenset({"expr", "query", "rawSql"})


def validate_dashboard(path: str) -> int:
    """Validate one dashboard JSON file.  Returns 1 on failure, 0 on success."""
    file_path = Path(path)
    if not file_path.exists():
        print(f"FAIL: {path} — file not found")
        return 1

    try:
        data = json.loads(file_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        print(f"FAIL: {path} — invalid JSON: {exc}")
        return 1

    errors: list[str] = []

    # Top-level keys
    for key in REQUIRED_TOP_KEYS:
        if key not in data:
            errors.append(f"missing top-level key: {key!r}")
    if not isinstance(data.get("schemaVersion"), (int, float)) or data.get("schemaVersion", 0) <= 0:
        errors.append(f"schemaVersion must be a positive integer, got {data.get('schemaVersion')!r}")

    # Panel validation
    panels = data.get("panels", [])
    if not isinstance(panels, list):
        errors.append(f"panels must be a list, got {type(panels).__name__}")
    else:
        for idx, panel in enumerate(panels):
            title = panel.get("title", f"panel[{idx}]")
            targets = panel.get("targets")
            if targets is None:
                continue  # stat panels without targets are fine
            if not isinstance(targets, list):
                errors.append(f"{title}: targets must be a list, got {type(targets).__name__}")
                continue
            if len(targets) == 0:
                errors.append(f"{title}: targets is empty")
                continue
            has_query = any(
                field in target
                for target in targets
                for field in TARGET_EXPR_FIELD
            )
            if not has_query:
                errors.append(f"{title}: no expr/query/rawSql found in any target")

    if errors:
        for err in errors:
            print(f"FAIL: {path} — {err}")
        return 1

    print(f"OK: {data.get('title', path)}")
    return 0


def main() -> int:
    if len(sys.argv) < 2:
        print("Usage: validate_grafana_dashboard.py <dashboards...>")
        return 1

    failed = 0
    for path in sys.argv[1:]:
        # Support glob patterns via Path.glob
        expanded = list(Path(".").glob(path)) if "*" in path or "?" in path else [Path(path)]
        if not expanded:
            print(f"FAIL: {path} — no matching files")
            failed += 1
            continue
        for p in expanded:
            failed += validate_dashboard(str(p))

    print()
    print(f"Total: {failed} failure(s)")
    return min(failed, 1)


if __name__ == "__main__":
    sys.exit(main())
