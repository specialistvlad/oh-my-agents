#!/usr/bin/env python3
"""Check the documentation is internally consistent.

Both checks exist because both have already failed silently: an ADR was
written and never listed, and a relative link can rot the moment a file is
renamed. Neither shows up in a Go or npm build.
"""

import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
ADR = ROOT / "docs" / "adr"


def check_adr_index() -> list[str]:
    """Every ADR file must appear in the index, and vice versa."""
    index = (ADR / "README.md").read_text()
    listed = set(re.findall(r"\]\((00\d\d-[a-z0-9-]+\.md)\)", index))
    on_disk = {f.name for f in ADR.glob("0*.md")}

    return [f"ADR not listed in docs/adr/README.md: {name}" for name in sorted(on_disk - listed)] + [
        f"index lists a missing ADR: {name}" for name in sorted(listed - on_disk)
    ]


def check_links() -> list[str]:
    """Every relative markdown link must resolve."""
    problems = []
    for md in ROOT.rglob("*.md"):
        if "node_modules" in md.parts:
            continue
        text = re.sub(r"```.*?```", "", md.read_text(), flags=re.S)
        text = re.sub(r"`[^`]*`", "", text)
        for _, link in re.findall(r"\[([^\]]+)\]\(([^)]+)\)", text):
            if link.startswith(("http://", "https://", "#", "mailto:")):
                continue
            target = (md.parent / link.split("#")[0]).resolve()
            if not target.exists():
                problems.append(f"broken link in {md.relative_to(ROOT)}: {link}")
    return problems


def main() -> int:
    problems = check_adr_index() + check_links()
    for p in problems:
        print(f"FAIL: {p}")
    if problems:
        print(f"{len(problems)} documentation problem(s).")
        return 1
    print("Docs consistent: every ADR is indexed and every link resolves.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
