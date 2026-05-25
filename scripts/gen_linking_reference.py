#!/usr/bin/env python3
"""Generate linking-reference.md from classification.json.

Reads the frozen SCC registry and produces a markdown reference table
organized by type, with display names, plural variants, and SCC codes.
"""

import json
import os
import re
import sys
from collections import defaultdict
from pathlib import Path

# Types to include, in display order
INCLUDED_TYPES = [
    "class",
    "ancestry",
    "career",
    "kit",
    "perk",
    "complication",
    "title",
    "treasure",
    "chapter",
]

# Section display names (type -> plural heading)
TYPE_HEADINGS = {
    "class": "Classes",
    "ancestry": "Ancestries",
    "career": "Careers",
    "kit": "Kits",
    "perk": "Perks",
    "complication": "Complications",
    "title": "Titles",
    "treasure": "Treasures",
    "chapter": "Chapters",
}

# Special-case display name overrides (slug -> display name)
DISPLAY_NAME_OVERRIDES = {
    "mages-apprentice": "Mage's Apprentice",
}

# Irregular plural overrides (display name -> plural)
IRREGULAR_PLURALS = {
    "Dwarf": "Dwarves",
    "Elf": "Elves",
    "High Elf": "High Elves",
    "Wode Elf": "Wode Elves",
    "Fury": "Furies",
    "Null": "Nulls",
}


def slug_to_display_name(slug: str) -> str:
    """Convert a hyphenated slug to a title-cased display name.

    Handles special cases via DISPLAY_NAME_OVERRIDES.
    """
    if slug in DISPLAY_NAME_OVERRIDES:
        return DISPLAY_NAME_OVERRIDES[slug]
    return slug.replace("-", " ").title()


def pluralize(name: str) -> str:
    """Generate the plural form of a display name.

    Uses irregular plural table first, then standard English rules.
    """
    if name in IRREGULAR_PLURALS:
        return IRREGULAR_PLURALS[name]

    # Standard pluralization rules applied to the last word
    lower = name.lower()
    if lower.endswith(("s", "x", "sh", "ch")):
        return name + "es"
    if lower.endswith("y") and len(lower) > 1 and lower[-2] not in "aeiou":
        return name[:-1] + "ies"
    return name + "s"


def parse_code(code: str) -> tuple[str, str] | None:
    """Parse an SCC code into (type, slug).

    Returns None for codes that should be skipped (e.g., feature.* codes).
    """
    parts = code.split("/")
    if len(parts) < 3:
        return None
    # parts[0] = source (e.g., mcdm.heroes.v1)
    # parts[1] = type
    # parts[2] = item slug
    code_type = parts[1]
    slug = parts[2]

    # Skip feature.* types
    if code_type.startswith("feature."):
        return None

    if code_type not in INCLUDED_TYPES:
        return None

    return (code_type, slug)


def main() -> None:
    script_dir = Path(__file__).resolve().parent
    classification_path = script_dir / ".." / "classification.json"
    output_path = script_dir / ".." / "docs" / "linking-reference.md"

    # Read classification.json
    try:
        with open(classification_path) as f:
            data = json.load(f)
    except FileNotFoundError:
        print(f"Error: {classification_path} not found", file=sys.stderr)
        sys.exit(1)

    codes = data.get("codes", [])

    # Group codes by type
    by_type: dict[str, list[tuple[str, str, str]]] = defaultdict(list)
    for code in codes:
        parsed = parse_code(code)
        if parsed is None:
            continue
        code_type, slug = parsed
        display_name = slug_to_display_name(slug)
        plural = pluralize(display_name)
        by_type[code_type].append((display_name, plural, code))

    # Sort entries within each type alphabetically by display name
    for code_type in by_type:
        by_type[code_type].sort(key=lambda entry: entry[0].lower())

    # Count total terms
    total = sum(len(entries) for entries in by_type.values())

    # Ensure output directory exists
    output_path.parent.mkdir(parents=True, exist_ok=True)

    # Write markdown
    lines: list[str] = []
    lines.append("# Linking Reference Table")
    lines.append("")
    lines.append("Generated from `classification.json`. See `linking-guide.md` for rules.")
    lines.append("")
    lines.append(f"**Total linkable terms:** {total}")
    lines.append("")

    for code_type in INCLUDED_TYPES:
        entries = by_type.get(code_type, [])
        if not entries:
            continue

        heading = TYPE_HEADINGS[code_type]
        lines.append(f"## {heading} ({len(entries)} terms)")
        lines.append("")
        lines.append("| Display Name | Variants | SCC Code |")
        lines.append("|-------------|----------|----------|")

        for display_name, plural, code in entries:
            lines.append(f"| {display_name} | {plural.lower()} | `{code}` |")

        lines.append("")

    with open(output_path, "w") as f:
        f.write("\n".join(lines))

    print(f"Wrote {output_path} ({total} terms across {len(by_type)} types)")


if __name__ == "__main__":
    main()
