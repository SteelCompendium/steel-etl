#!/usr/bin/env python3
"""
Automated annotation of Draw Steel Heroes.md for the steel-etl pipeline.

Reads the unannotated source markdown, analyzes the heading structure
and context, and inserts HTML comment annotations before headings.

Usage:
    python3 annotate_heroes.py [--dry-run]
"""

from __future__ import annotations

import re
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Literal

SRC = Path(__file__).parent.parent / "data-gen" / "input" / "heroes" / "Draw Steel Heroes.md"
DST = Path(__file__).parent / "input" / "heroes" / "Draw Steel Heroes.md"

FRONTMATTER = """\
---
book: mcdm.heroes.v1
source: MCDM
title: Draw Steel Heroes
---

"""

# ---------------------------------------------------------------------------
# Class metadata
# ---------------------------------------------------------------------------

CLASS_NAMES = [
    "Censor", "Conduit", "Elementalist", "Fury",
    "Null", "Shadow", "Tactician", "Talent", "Troubadour",
]

# Map class name -> heroic resource name (for cost annotations)
CLASS_RESOURCES: dict[str, str] = {
    "Censor": "Wrath",
    "Conduit": "Piety",
    "Elementalist": "Essence",
    "Fury": "Ferocity",
    "Null": "Discipline",
    "Shadow": "Insight",
    "Tactician": "Focus",
    "Talent": "Clarity",
    "Troubadour": "Drama",
}

# ---------------------------------------------------------------------------
# Ancestry list (H2 headings inside # Ancestries chapter)
# ---------------------------------------------------------------------------

ANCESTRY_NAMES = [
    "Devil", "Dragon Knight", "Dwarf", "Wode Elf", "High Elf",
    "Hakaan", "Human", "Memonek", "Orc", "Polder", "Revenant", "Time Raider",
]

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def slugify(text: str) -> str:
    """Convert heading text to a URL-safe slug."""
    text = text.lower().strip()
    # Remove trailing ! and ? but keep them for id override detection
    text = re.sub(r"[^a-z0-9\s-]", "", text)
    text = re.sub(r"\s+", "-", text.strip())
    text = re.sub(r"-+", "-", text)
    return text.strip("-")


def needs_id_override(heading_text: str) -> str | None:
    """Return an @id if the heading would produce a problematic slug."""
    # Headings with special chars that affect slug
    if re.search(r"[!?…]", heading_text):
        slug = slugify(heading_text)
        if slug:
            return slug
    return None


def extract_cost_from_heading(heading_text: str, resource: str) -> str | None:
    """Extract cost like '(3 Wrath)' from ability heading text."""
    pattern = rf"\((\d+)\s+{re.escape(resource)}\)"
    m = re.search(pattern, heading_text, re.IGNORECASE)
    if m:
        return f"{m.group(1)} {resource}"
    return None


def detect_triggered(lines: list[str], start_idx: int) -> bool:
    """Check if the ability body contains a Trigger: line."""
    for i in range(start_idx + 1, min(start_idx + 20, len(lines))):
        line = lines[i]
        # Stop at next heading or annotation
        if re.match(r"^#{1,6}\s", line) or line.startswith("> ########") or line.startswith("########"):
            break
        if line.startswith("<!--"):
            break
        if re.match(r"^>\s*\*\*Trigger", line) or re.match(r"^\*\*Trigger", line):
            return True
    return False


# ---------------------------------------------------------------------------
# Section boundary detection
# ---------------------------------------------------------------------------

# H1 chapter boundaries
CHAPTER_STARTS: dict[str, str] = {
    "Introduction": "introduction",
    "The Basics": "the-basics",
    "Making a Hero": "making-a-hero",
    "Ancestries": "ancestries",
    "Background": "background",
    "Classes": "classes",
    "Kits": "kits",
    "Perks": "perks",
    "Complications": "complications",
    "Tests": "tests",
    "Combat": "combat",
    "Negotiation": "negotiation",
    "Downtime Projects": "downtime-projects",
    "Rewards": "rewards",
    "Gods and Religion": "gods-and-religion",
    "For the Director": "for-the-director",
}

# Chapters that get annotated as chapter type
ANNOTATED_CHAPTERS = {
    "Introduction", "The Basics", "Making a Hero", "Ancestries",
    "Background", "Classes", "Kits", "Perks", "Complications",
    "Tests", "Combat", "Negotiation", "Downtime Projects", "Rewards",
    "Gods and Religion", "For the Director",
}


# ---------------------------------------------------------------------------
# Annotation
# ---------------------------------------------------------------------------

@dataclass
class Annotation:
    """An annotation to insert before a specific line."""
    line_no: int  # 1-based line number in the ORIGINAL file
    text: str     # The annotation comment text


@dataclass
class AnnotationContext:
    """Tracks the current context while scanning."""
    chapter: str = ""
    current_class: str = ""
    current_class_resource: str = ""
    in_feature_group: bool = False
    feature_group_level: int = 0
    in_kits_a_to_z: bool = False
    in_kit: str = ""
    # Track special in-class kit sections (e.g., Stormwight Kits in Fury)
    in_class_kit_section: bool = False
    in_class_kit: str = ""


def parse_level_from_heading(text: str) -> int | None:
    """Extract level number from '1st-Level Features', '2nd-Level Features', etc."""
    m = re.match(r"(\d+)(?:st|nd|rd|th)-Level\s+Features", text)
    if m:
        return int(m.group(1))
    return None


def _is_structural_ability_heading(heading: str) -> bool:
    """Check if an H4 heading is a structural ability-grouping header."""
    patterns = [
        r"^\d+-\w+\s+Abilit",              # '3-Wrath Ability', '11-Ferocity Ability'
        r"^.+\s+Abilities$",                # 'Fury Abilities', 'Heroic Abilities'
        r"^\d+\w+-Level\s+.*\bAbilit",      # '2nd-Level Aspect Ability'
        r"^New\s+\d+-\w+\s+Abilit",         # 'New 5-Essence Ability'
        r"^Signature Abilit",               # 'Signature Ability' (intro text)
    ]
    return any(re.match(p, heading) for p in patterns)


def annotate_heroes(lines: list[str]) -> list[Annotation]:
    """Generate all annotations for the Heroes book."""
    annotations: list[Annotation] = []
    ctx = AnnotationContext()
    n = len(lines)

    i = 0
    while i < n:
        line = lines[i]
        lineno = i + 1  # 1-based

        # --- H1: Chapter headings ---
        m_h1 = re.match(r"^# (.+)$", line)
        if m_h1:
            chapter_name = m_h1.group(1).strip()
            ctx.chapter = chapter_name
            ctx.current_class = ""
            ctx.current_class_resource = ""
            ctx.in_feature_group = False
            ctx.in_kits_a_to_z = False
            ctx.in_kit = ""

            if chapter_name in ANNOTATED_CHAPTERS:
                slug = CHAPTER_STARTS.get(chapter_name, slugify(chapter_name))
                annotations.append(Annotation(lineno, f"<!-- @type: chapter | @id: {slug} -->"))
            i += 1
            continue

        # --- H2: Major sections ---
        m_h2 = re.match(r"^## (.+)$", line)
        if m_h2:
            heading = m_h2.group(1).strip()
            ctx.in_feature_group = False
            ctx.in_kit = ""

            # Class sections
            if heading in CLASS_NAMES and ctx.chapter == "Classes":
                ctx.current_class = heading
                ctx.current_class_resource = CLASS_RESOURCES.get(heading, "")
                slug = slugify(heading)
                annotations.append(Annotation(lineno, f"<!-- @type: class | @id: {slug} -->"))
                i += 1
                continue

            # Ancestry sections
            if heading in ANCESTRY_NAMES and ctx.chapter == "Ancestries":
                slug = slugify(heading)
                annotations.append(Annotation(lineno, f"<!-- @type: ancestry | @id: {slug} -->"))
                i += 1
                continue

            # Culture section
            if heading == "Culture" and ctx.chapter == "Background":
                annotations.append(Annotation(lineno, "<!-- @type: chapter | @id: cultures -->"))
                i += 1
                continue

            # Careers section
            if heading == "Careers" and ctx.chapter == "Background":
                annotations.append(Annotation(lineno, "<!-- @type: chapter | @id: careers -->"))
                i += 1
                continue

            # Treasures / Titles / Renown / Wealth in Rewards chapter
            if ctx.chapter == "Rewards":
                if heading == "Treasures":
                    annotations.append(Annotation(lineno, "<!-- @type: chapter | @id: treasures -->"))
                elif heading == "Titles":
                    annotations.append(Annotation(lineno, "<!-- @type: chapter | @id: titles -->"))
                i += 1
                continue

            i += 1
            continue

        # --- H3: Sub-sections ---
        m_h3 = re.match(r"^### (.+)$", line)
        if m_h3:
            heading = m_h3.group(1).strip()

            # Feature-group in a class
            if ctx.current_class:
                level = parse_level_from_heading(heading)
                if level is not None:
                    ctx.in_feature_group = True
                    ctx.feature_group_level = level
                    ctx.in_class_kit_section = False
                    ctx.in_class_kit = ""
                    annotations.append(Annotation(
                        lineno,
                        f"<!-- @type: feature-group | @level: {level} -->"
                    ))
                    i += 1
                    continue

                # In-class kit sections (e.g., Stormwight Kits in Fury)
                # These are H3 headings that contain H4 kit entries
                if "Kit" in heading and ctx.current_class:
                    ctx.in_class_kit_section = True
                    ctx.in_feature_group = False
                    annotations.append(Annotation(lineno, "<!-- @type: feature -->"))
                    i += 1
                    continue

                # "Basics" is just structural, no annotation needed
                if heading == "Basics":
                    i += 1
                    continue

            # Ancestry sub-sections (### On Devils, ### Devil Traits, etc.)
            if ctx.chapter == "Ancestries":
                # Trait sections get annotated
                if heading.endswith(" Traits"):
                    ancestry_slug = slugify(heading.replace(" Traits", ""))
                    annotations.append(Annotation(
                        lineno,
                        f"<!-- @type: feature | @id: {ancestry_slug}-traits -->"
                    ))
                    i += 1
                    continue
                # "On X" narrative sections - no annotation needed
                if heading.startswith("On "):
                    i += 1
                    continue

            # Culture Benefits
            if heading == "Culture Benefits" and ctx.chapter == "Background":
                annotations.append(Annotation(lineno, "<!-- @type: feature-group | @id: culture-benefits -->"))
                i += 1
                continue

            # Careers A to Z
            if heading == "Careers A to Z" and ctx.chapter == "Background":
                annotations.append(Annotation(lineno, "<!-- @type: feature-group | @id: careers-list -->"))
                i += 1
                continue

            # Kits A to Z
            if heading == "Kits A to Z" and ctx.chapter == "Kits":
                ctx.in_kits_a_to_z = True
                i += 1
                continue

            # Perk type groups
            if ctx.chapter == "Perks" and heading.endswith(" Perks"):
                slug = slugify(heading)
                annotations.append(Annotation(
                    lineno,
                    f"<!-- @type: feature-group | @id: {slug} -->"
                ))
                i += 1
                continue

            # Skill groups
            if ctx.chapter == "Tests" and heading == "Skills":
                annotations.append(Annotation(lineno, "<!-- @type: chapter | @id: skills -->"))
                i += 1
                continue

            # Movement rules (inside Combat chapter)
            if ctx.chapter == "Combat" and heading == "Movement":
                annotations.append(Annotation(lineno, "<!-- @type: movement | @id: movement -->"))
                i += 1
                continue

            # Free Strikes, Maneuvers sections in Combat
            if ctx.chapter == "Combat" and heading in ("Free Strikes", "Maneuvers"):
                slug = slugify(heading)
                annotations.append(Annotation(lineno, f"<!-- @type: feature-group | @id: {slug} -->"))
                i += 1
                continue

            # Title echelon groups in Rewards
            if ctx.chapter == "Rewards" and re.match(r"\d+(?:st|nd|rd|th)-Echelon Titles", heading):
                m_ech = re.match(r"(\d+)", heading)
                if m_ech:
                    echelon = m_ech.group(1)
                    annotations.append(Annotation(
                        lineno,
                        f"<!-- @type: feature-group | @id: {echelon}-echelon-titles -->"
                    ))
                i += 1
                continue

            i += 1
            continue

        # --- H4: Individual items ---
        m_h4 = re.match(r"^#### (.+)$", line)
        if m_h4:
            heading = m_h4.group(1).strip()

            # Inside a class kit section: H4 entries are kits (or structural)
            if ctx.current_class and ctx.in_class_kit_section:
                # Structural headings within kit sections — not kits themselves
                if heading in ("Kit Features",):
                    annotations.append(Annotation(lineno, "<!-- @type: feature -->"))
                    i += 1
                    continue
                slug = slugify(heading)
                ctx.in_class_kit = slug
                annotations.append(Annotation(lineno, f"<!-- @type: kit | @id: {slug} -->"))
                i += 1
                continue

            # Inside a class feature-group: features (skip structural grouping headings)
            if ctx.current_class and ctx.in_feature_group:
                # Structural headings that group abilities — not features themselves
                if _is_structural_ability_heading(heading):
                    i += 1
                    continue
                ann_parts = ["@type: feature"]
                id_override = needs_id_override(heading)
                if id_override:
                    ann_parts.append(f"@id: {id_override}")
                annotations.append(Annotation(lineno, f"<!-- {' | '.join(ann_parts)} -->"))
                i += 1
                continue

            # Kits A to Z: individual kit entries
            if ctx.in_kits_a_to_z and ctx.chapter == "Kits":
                slug = slugify(heading)
                ctx.in_kit = slug
                annotations.append(Annotation(lineno, f"<!-- @type: kit | @id: {slug} -->"))
                i += 1
                continue

            # Perks chapter: individual perks
            if ctx.chapter == "Perks":
                slug = slugify(heading)
                id_override = needs_id_override(heading)
                ann_parts = ["@type: perk"]
                if id_override:
                    ann_parts.append(f"@id: {id_override}")
                annotations.append(Annotation(lineno, f"<!-- {' | '.join(ann_parts)} -->"))
                i += 1
                continue

            # Complications: individual complications
            if ctx.chapter == "Complications":
                # Skip structural headings
                if heading in ("Benefit and Drawback", "Modifying the Story",
                               "Choosing a Complication"):
                    i += 1
                    continue
                ann_parts = ["@type: complication"]
                id_override = needs_id_override(heading)
                if id_override:
                    ann_parts.append(f"@id: {id_override}")
                annotations.append(Annotation(lineno, f"<!-- {' | '.join(ann_parts)} -->"))
                i += 1
                continue

            # Background chapter: career entries under Careers A to Z
            if ctx.chapter == "Background":
                # Check if previous H3 was "Careers A to Z" (we're in the career list)
                # Career entries are #### headings after ### Careers A to Z
                slug = slugify(heading)
                ann_parts = ["@type: career"]
                id_override = needs_id_override(heading)
                if id_override:
                    ann_parts.append(f"@id: {id_override}")
                annotations.append(Annotation(lineno, f"<!-- {' | '.join(ann_parts)} -->"))
                i += 1
                continue

            # Rewards: title entries, treasure entries
            if ctx.chapter == "Rewards":
                ann_parts = ["@type: title"]
                id_override = needs_id_override(heading)
                if id_override:
                    ann_parts.append(f"@id: {id_override}")
                annotations.append(Annotation(lineno, f"<!-- {' | '.join(ann_parts)} -->"))
                i += 1
                continue

            i += 1
            continue

        # --- Blockquoted H8: Abilities (in classes) ---
        m_bq_h8 = re.match(r"^> ######## (.+)$", line)
        if m_bq_h8 and ctx.current_class:
            heading = m_bq_h8.group(1).strip()
            resource = ctx.current_class_resource

            ann_parts = ["@type: ability"]

            # Detect cost from heading: "Ability Name (3 Wrath)"
            cost = extract_cost_from_heading(heading, resource)
            if cost:
                ann_parts.append(f"@cost: {cost}")

            # Detect triggered from body
            if detect_triggered(lines, i):
                ann_parts.append("@subtype: triggered")

            # ID override for problematic slugs
            # Strip cost suffix for slug: "Ability Name (3 Wrath)" -> "Ability Name"
            clean_heading = re.sub(rf"\s*\(\d+\s+{re.escape(resource)}\)\s*$", "", heading)
            id_override = needs_id_override(clean_heading)
            if id_override:
                ann_parts.append(f"@id: {id_override}")

            annotations.append(Annotation(lineno, f"<!-- {' | '.join(ann_parts)} -->"))
            i += 1
            continue

        # --- Bare H8: Kit signature abilities (Kits chapter or in-class kits) ---
        m_h8 = re.match(r"^######## (.+)$", line)
        if m_h8:
            heading = m_h8.group(1).strip()
            in_kits_chapter = ctx.chapter == "Kits" and ctx.in_kits_a_to_z
            in_class_kit = ctx.current_class and ctx.in_class_kit_section

            if in_kits_chapter or in_class_kit:
                ann_parts = ["@type: ability", "@subtype: signature"]
                id_override = needs_id_override(heading)
                if id_override:
                    ann_parts.append(f"@id: {id_override}")
                annotations.append(Annotation(lineno, f"<!-- {' | '.join(ann_parts)} -->"))
                i += 1
                continue

        # --- H5: Sub-features (in class feature-groups, class kits, etc.) ---
        m_h5 = re.match(r"^##### (.+)$", line)
        if m_h5:
            heading = m_h5.group(1).strip()

            # Inside a class (feature-group or kit section): H5 = sub-features
            if ctx.current_class and (ctx.in_feature_group or ctx.in_class_kit_section):
                # Skip structural sub-headings (e.g., "Heroic Abilities", "Wrath in Combat")
                if _is_structural_ability_heading(heading):
                    i += 1
                    continue
                # Skip "Wrath/Ferocity/etc. in Combat" and "... Outside of Combat"
                if re.match(r".+\s+(?:in|Outside of)\s+Combat$", heading):
                    i += 1
                    continue
                # Skip "Judgment Order Benefit" style sub-sections (narrative, not features)
                ann_parts = ["@type: feature"]
                id_override = needs_id_override(heading)
                if id_override:
                    ann_parts.append(f"@id: {id_override}")
                annotations.append(Annotation(lineno, f"<!-- {' | '.join(ann_parts)} -->"))
                i += 1
                continue

            # Inside Kits A to Z: H5 entries are kit sub-sections (Equipment, Kit Bonuses, etc.)
            # These are structural — don't annotate them
            if ctx.chapter == "Kits" and ctx.in_kits_a_to_z:
                i += 1
                continue

        i += 1

    return annotations


# ---------------------------------------------------------------------------
# Signature ability detection
# ---------------------------------------------------------------------------

def mark_signature_abilities(
    annotations: list[Annotation],
    lines: list[str],
) -> None:
    """
    Mark abilities that appear to be signature abilities.

    Signature abilities appear immediately after a "##### Signature Ability"
    heading in the source, within a class's 1st-level feature-group.
    """
    # Build a set of line ranges where signature abilities appear.
    # A "##### Signature Ability" heading starts a range; it ends at
    # the next ##### or ###### heading that isn't a > ######## ability.
    sig_ranges: list[tuple[int, int]] = []  # (start_line, end_line) inclusive

    for i, line in enumerate(lines):
        if re.match(r"^#{4,5}\s+(?:Kit\s+)?Signature Abilit", line):
            heading_level = len(line) - len(line.lstrip("#"))
            start = i + 1  # 0-based, line after the heading
            end = start
            # Scan forward to find the end of the signature section
            for j in range(start + 1, min(start + 200, len(lines))):
                src = lines[j]
                # Skip ability lines within the section
                if src.startswith("> ########") or src.startswith("########"):
                    continue
                m_heading = re.match(r"^(#{1,6})\s+(.+)", src)
                if m_heading:
                    h_level = len(m_heading.group(1))
                    h_text = m_heading.group(2).strip()
                    # Same or higher level heading = end of sig section
                    if h_level <= heading_level:
                        end = j - 1
                        break
                    # Lower-level heading that starts a cost section = end
                    if (re.match(r"Heroic Abilit", h_text)
                            or re.match(r"\d+-\w+\s+Abilit", h_text)):
                        end = j - 1
                        break
            else:
                end = min(start + 200, len(lines) - 1)
            sig_ranges.append((start + 1, end + 1))  # convert to 1-based

    # Mark ability annotations that fall within a signature range
    for ann in annotations:
        if "@type: ability" in ann.text and "@subtype:" not in ann.text:
            for (start, end) in sig_ranges:
                if start <= ann.line_no <= end:
                    ann.text = ann.text.replace(
                        "@type: ability",
                        "@type: ability | @subtype: signature"
                    )
                    break


# ---------------------------------------------------------------------------
# Career context tracking
# ---------------------------------------------------------------------------

def is_in_careers_section(lines: list[str], line_no: int) -> bool:
    """Check if a line number is within the Careers A to Z section."""
    # Look backwards for ### Careers A to Z
    for j in range(line_no - 2, max(0, line_no - 500), -1):
        if lines[j].startswith("### Careers A to Z"):
            return True
        if lines[j].startswith("### ") and "Careers A to Z" not in lines[j]:
            return False
        if lines[j].startswith("## "):
            return False
        if lines[j].startswith("# "):
            return False
    return False


def refine_background_annotations(
    annotations: list[Annotation],
    lines: list[str],
) -> None:
    """Refine Background chapter H4 annotations to only tag careers in the right section."""
    to_remove = []
    for idx, ann in enumerate(annotations):
        if "@type: career" in ann.text:
            if not is_in_careers_section(lines, ann.line_no):
                to_remove.append(idx)
    # Remove non-career H4s from Background (they're structural)
    for idx in reversed(to_remove):
        annotations.pop(idx)


# ---------------------------------------------------------------------------
# Rewards refinement
# ---------------------------------------------------------------------------

def refine_rewards_annotations(
    annotations: list[Annotation],
    lines: list[str],
) -> None:
    """
    In the Rewards chapter, H4 entries could be treasures OR titles.
    Determine which based on whether they're in the Treasures or Titles section.
    """
    # Find the line numbers for ## Treasures, ## Titles
    treasures_start = None
    titles_start = None
    rewards_end = None

    for i, line in enumerate(lines):
        if line.startswith("## Treasures"):
            treasures_start = i + 1
        elif line.startswith("## Titles"):
            titles_start = i + 1
        elif line.startswith("# Gods and Religion"):
            rewards_end = i + 1
            break

    if not treasures_start or not titles_start:
        return

    for ann in annotations:
        if "@type: title" in ann.text and ann.line_no < titles_start:
            # This is actually a treasure, not a title
            ann.text = ann.text.replace("@type: title", "@type: treasure")


# ---------------------------------------------------------------------------
# Condition annotations (inside Combat chapter, conditions subsection)
# ---------------------------------------------------------------------------

def add_condition_annotations(
    annotations: list[Annotation],
    lines: list[str],
) -> None:
    """
    Conditions are in the Combat chapter under specific subsections.
    They use ##### or #### headings. This needs special handling since
    the main loop may not catch them.
    """
    # Find the Conditions section - it's actually listed inline in combat rules
    # Let's check if there's a dedicated conditions section
    pass  # Conditions may not be separate H4s; need to check structure


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    dry_run = "--dry-run" in sys.argv

    if not SRC.exists():
        print(f"Source file not found: {SRC}", file=sys.stderr)
        sys.exit(1)

    lines = SRC.read_text(encoding="utf-8").splitlines(keepends=True)
    raw_lines = [l.rstrip("\n") for l in lines]  # For analysis without newlines

    print(f"Source: {SRC}")
    print(f"Lines: {len(lines)}")

    # Generate annotations
    annotations = annotate_heroes(raw_lines)

    # Post-processing passes
    mark_signature_abilities(annotations, raw_lines)
    refine_background_annotations(annotations, raw_lines)
    refine_rewards_annotations(annotations, raw_lines)

    # Sort by line number
    annotations.sort(key=lambda a: a.line_no)

    # Report
    type_counts: dict[str, int] = {}
    for ann in annotations:
        m = re.search(r"@type: (\S+)", ann.text)
        if m:
            t = m.group(1)
            type_counts[t] = type_counts.get(t, 0) + 1

    print(f"\nTotal annotations: {len(annotations)}")
    print("By type:")
    for t, count in sorted(type_counts.items()):
        print(f"  {t}: {count}")

    if dry_run:
        print("\n[DRY RUN] Not writing output.")
        # Print first few annotations
        print("\nFirst 20 annotations:")
        for ann in annotations[:20]:
            print(f"  L{ann.line_no}: {ann.text}")
        print(f"\nLast 10 annotations:")
        for ann in annotations[-10:]:
            print(f"  L{ann.line_no}: {ann.text}")
        return

    # Build annotation map
    ann_map: dict[int, str] = {}
    for ann in annotations:
        ann_map[ann.line_no] = ann.text

    # Build output
    output_lines: list[str] = [FRONTMATTER]
    for i, line in enumerate(lines):
        lineno = i + 1
        if lineno in ann_map:
            output_lines.append(ann_map[lineno] + "\n")
        output_lines.append(line)

    # Write
    DST.parent.mkdir(parents=True, exist_ok=True)
    DST.write_text("".join(output_lines), encoding="utf-8")

    print(f"\nOutput: {DST}")
    out_line_count = sum(l.count("\n") for l in output_lines) + len(output_lines)
    print(f"Output lines: {out_line_count}")


if __name__ == "__main__":
    main()
