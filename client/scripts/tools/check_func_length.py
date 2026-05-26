#!/usr/bin/env python3
"""Check that no GDScript function exceeds a maximum body length.

Usage:  python3 check_func_length.py [--max N] [--allowlist FILE] <paths...>

Counts lines from the `func` signature to the next `func` or EOF.
Blank lines and comments count (same semantics as Go funlen).

An allowlist file can list known violations as "file:line" entries
(one per line). Allowlisted locations are reported as warnings, not
errors. New violations not in the allowlist cause exit code 1.
"""
import argparse
import sys
from pathlib import Path

DEFAULT_MAX = 60


def check_file(path: Path, max_lines: int) -> list[tuple[str, str, int]]:
    """Returns list of (location, message, body_lines)."""
    results = []
    try:
        lines = path.read_text().splitlines()
    except Exception as e:
        print(f"warning: cannot read {path}: {e}", file=sys.stderr)
        return results

    func_name = None
    func_start = 0

    for i, line in enumerate(lines, 1):
        stripped = line.lstrip()
        is_func = stripped.startswith("func ") or stripped.startswith(
            "static func "
        )
        if is_func:
            if func_name is not None:
                body = i - func_start - 1
                if body > max_lines:
                    loc = f"{path}:{func_start}"
                    msg = f"{func_name} is {body} lines (max {max_lines})"
                    results.append((loc, msg, body))
            sig = stripped.replace("static ", "", 1)
            func_name = sig.split("(")[0]
            func_start = i

    if func_name is not None:
        body = len(lines) + 1 - func_start
        if body > max_lines:
            loc = f"{path}:{func_start}"
            msg = f"{func_name} is {body} lines (max {max_lines})"
            results.append((loc, msg, body))

    return results


def load_allowlist(path: Path) -> set[str]:
    if not path.exists():
        return set()
    entries = set()
    for line in path.read_text().splitlines():
        line = line.strip()
        if line and not line.startswith("#"):
            entries.add(line)
    return entries


def main():
    parser = argparse.ArgumentParser(
        description="GDScript function length checker"
    )
    parser.add_argument("paths", nargs="+", help="Directories or files")
    parser.add_argument(
        "--max", type=int, default=DEFAULT_MAX, dest="max_lines",
        help=f"Max function body lines (default: {DEFAULT_MAX})",
    )
    parser.add_argument(
        "--allowlist", type=Path,
        default=Path("scripts/tools/funlen_allowlist.txt"),
        help="File listing known violations to skip",
    )
    args = parser.parse_args()

    allowed = load_allowlist(args.allowlist)

    files: list[Path] = []
    for p in args.paths:
        path = Path(p)
        if path.is_dir():
            files.extend(
                f for f in sorted(path.rglob("*.gd"))
                if "addons" not in f.parts
            )
        elif path.suffix == ".gd":
            files.append(path)

    new_errors: list[str] = []
    allowed_hits = 0
    for f in files:
        for loc, msg, _ in check_file(f, args.max_lines):
            if loc in allowed:
                allowed_hits += 1
            else:
                new_errors.append(f"{loc}: {msg}")

    if new_errors:
        for e in new_errors:
            print(e, file=sys.stderr)
        print(
            f"\nFailure: {len(new_errors)} new functions exceed "
            f"{args.max_lines} lines",
            file=sys.stderr,
        )
        if allowed_hits:
            print(
                f"({allowed_hits} allowlisted violations skipped)",
                file=sys.stderr,
            )
        sys.exit(1)
    else:
        remaining = len(allowed)
        if allowed_hits:
            print(
                f"OK — {allowed_hits} allowlisted violations remaining "
                f"(shrink the allowlist as you fix them)"
            )
        else:
            print("OK — no function length violations")


if __name__ == "__main__":
    main()
