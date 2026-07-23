"""Best-effort AST-based lint for stateless discipline.

This is a guardrail, not a formal effect system. False positives can be
disabled locally with:

    # qst-lint: disable-next-line -- deterministic test clock stub
"""

from __future__ import annotations

import ast
import sys
from pathlib import Path

FORBIDDEN_CALLS = {
    "time.time",
    "datetime.now",
    "datetime.datetime.now",
    "datetime.utcnow",
    "random.random",
    "random.randint",
    "random.choice",
    "threading.local",
}

WHITELIST_FILES = {
    "tokens\\registry.py",
    "tokens/registry.py",
    "recipes\\registry.py",
    "recipes/registry.py",
}

DISABLE_NEXT_LINE = "qst-lint: disable-next-line"


class StatelessVisitor(ast.NodeVisitor):
    """Collect best-effort stateless discipline violations."""

    def __init__(self, filepath: str):
        self.filepath = filepath
        self.violations: list[tuple[int, str]] = []

    def visit_Global(self, node: ast.Global) -> None:
        self.violations.append((node.lineno, "global keyword forbidden"))

    def visit_Call(self, node: ast.Call) -> None:
        name = self._get_call_name(node.func)
        for forbidden in FORBIDDEN_CALLS:
            if name and name.endswith(forbidden):
                self.violations.append((node.lineno, f"forbidden call: {forbidden}"))
                break
        self.generic_visit(node)

    def _get_call_name(self, func: ast.expr) -> str | None:
        try:
            return ast.unparse(func)
        except Exception:
            return None


def _disabled_lines(source: str) -> set[int]:
    disabled: set[int] = set()
    lines = source.splitlines()
    for i, line in enumerate(lines, start=1):
        if DISABLE_NEXT_LINE in line:
            disabled.add(i + 1)
    return disabled


def lint_file(path: Path) -> list[tuple[int, str]]:
    """Lint one Python file."""

    rel_path = str(path)
    if any(rel_path.endswith(whitelisted) for whitelisted in WHITELIST_FILES):
        return []

    source = path.read_text(encoding="utf-8")
    disabled = _disabled_lines(source)

    tree = ast.parse(source)
    visitor = StatelessVisitor(rel_path)
    visitor.visit(tree)

    return [
        (lineno, msg)
        for lineno, msg in visitor.violations
        if lineno not in disabled
    ]


def main(argv: list[str] | None = None) -> int:
    """CLI entrypoint."""

    args = sys.argv[1:] if argv is None else argv
    if not args:
        print("usage: python -m qst.lint.stateless <package_dir>", file=sys.stderr)
        return 2

    root = Path(args[0])
    total_violations = 0

    files = [root] if root.is_file() else sorted(root.rglob("*.py"))
    for py_file in files:
        violations = lint_file(py_file)
        for lineno, msg in violations:
            print(f"{py_file}:{lineno}: {msg}", file=sys.stderr)
            total_violations += 1

    if total_violations:
        print(f"\n{total_violations} violation(s) found", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
