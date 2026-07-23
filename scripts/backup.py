#!/usr/bin/env python3
"""OrcaAlgo project backup script.
Creates compressed zip archives with timestamped names, excludes heavy
dependency/build directories, and ensures the backup destination is in
.gitignore.
"""

import os
import sys
import zipfile
import datetime
from pathlib import Path
from typing import Set

PROJECT_ROOT = Path(__file__).resolve().parent
BACKUP_DIR = PROJECT_ROOT / "backups"
RESEARCH_DIR = PROJECT_ROOT / ".research"

EXCLUDE_DIRS: Set[str] = {
    "node_modules",
    ".venv",
    "venv",
    "build",
    "bin",
    "generated",
    "__pycache__",
    ".cache",
    ".git",
    ".pnp",
    "backups",
    ".research",
    "playwright-report",
    "test-results",
    "data",
}

EXCLUDE_FILES: Set[str] = {
    ".DS_Store",
    ".env",
    "Thumbs.db",
    "orca-server.exe",
}

SEPARATOR = "=" * 56


def _ensure_dir() -> None:
    BACKUP_DIR.mkdir(parents=True, exist_ok=True)


def _add_to_gitignore() -> bool:
    gitignore = PROJECT_ROOT / ".gitignore"
    entries = ("backups/", ".research/")

    try:
        lines = gitignore.read_text(encoding="utf-8").splitlines()
    except FileNotFoundError:
        print(f"WARNING: .gitignore not found at {gitignore}")
        return False

    added = False
    stripped = {line.strip() for line in lines}
    try:
        content = gitignore.read_text(encoding="utf-8")
        needs_newline = content and not content.endswith("\n")
        with open(gitignore, "a", encoding="utf-8") as f:
            for entry in entries:
                if entry not in stripped:
                    if needs_newline:
                        f.write("\n")
                        needs_newline = False
                    f.write(f"{entry}\n")
                    print(f"[OK] Added '{entry}' to .gitignore")
                    added = True
    except OSError as e:
        print(f"[FAIL] Could not update .gitignore: {e}")
    return added


def _create_archive() -> Path:
    _ensure_dir()

    timestamp = datetime.datetime.now().strftime("%Y-%m-%d_%H%M%S")
    archive_name = f"orca_backup_{timestamp}"
    archive_path = BACKUP_DIR / f"{archive_name}.zip"

    print(f"Source  : {PROJECT_ROOT}")
    print(f"Target  : {archive_path}")
    sys.stdout.flush()

    file_count = 0
    total_bytes = 0
    ext_counts: dict[str, int] = {}

    with zipfile.ZipFile(archive_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for root, dirs, files in os.walk(PROJECT_ROOT, topdown=True):
            dirs[:] = [d for d in dirs if d not in EXCLUDE_DIRS]

            rel_root = os.path.relpath(root, PROJECT_ROOT)

            for filename in files:
                if filename in EXCLUDE_FILES:
                    continue
                if filename.endswith((".exe", ".dll", ".so", ".dylib")):
                    continue

                filepath = os.path.join(root, filename)
                arcname = filename if rel_root == "." else os.path.join(rel_root, filename)

                try:
                    zf.write(filepath, arcname)
                    file_count += 1
                    total_bytes += os.path.getsize(filepath)
                    ext = os.path.splitext(filename)[1] or "(none)"
                    ext_counts[ext] = ext_counts.get(ext, 0) + 1
                except OSError as e:
                    print(f"  SKIP  {arcname}: {e}")

    archive_size_mb = total_bytes / (1024 * 1024)

    print(f"\n  Files   : {file_count:,}")
    print(f"  Size    : {archive_size_mb:,.1f} MB")

    if ext_counts:
        top_exts = sorted(ext_counts.items(), key=lambda x: x[1], reverse=True)[:6]
        ext_summary = "  ".join(f".{e[0] if e[0] != '(none)' else 'no-ext'}:{e[1]}" for e in top_exts)
        print(f"  Top ext : {ext_summary}")

    print(f"[OK] Backup created successfully")
    return archive_path


def _prune_old_backups(keep: int = 5) -> int:
    backups = sorted(BACKUP_DIR.glob("orca_backup_*.zip"), reverse=True)
    removed = 0
    for old in backups[keep:]:
        try:
            old.unlink()
            removed += 1
        except OSError:
            pass
    if removed:
        print(f"[OK] Pruned {removed} old backup(s), keeping latest {keep}")
    return removed


def main() -> int:
    print(SEPARATOR)
    print("OrcaAlgo Project Backup")
    print(SEPARATOR)

    _add_to_gitignore()

    try:
        _create_archive()
        _prune_old_backups(keep=5)
        print(SEPARATOR)
        return 0
    except Exception as exc:
        print(f"[FAIL] {exc}")
        print(SEPARATOR)
        return 1


if __name__ == "__main__":
    sys.exit(main())
