#!/usr/bin/env python3
"""
Comprehensive scc: link adder for Draw Steel Heroes.md.

Scans every line for game mechanic terms (classes, ancestries) and wraps
them in [Display Text](scc:CODE) markdown links. Handles:
- Case-insensitive matching with original case preserved
- Plurals and possessives
- Multi-word terms (dragon knight, high elf, etc.)
- Already-linked text (skipped)
- Headings (skipped — don't link in ## headings)
- Annotation comments (skipped)
- Ambiguous terms with context checks (null, shadow, talent, human skull, etc.)
"""

import re
import sys
import json
from pathlib import Path


# Terms that are always game references — link unconditionally
CLASSES = {
    "censor": "mcdm.heroes.v1/class/censor",
    "conduit": "mcdm.heroes.v1/class/conduit",
    "elementalist": "mcdm.heroes.v1/class/elementalist",
    "fury": "mcdm.heroes.v1/class/fury",
    "tactician": "mcdm.heroes.v1/class/tactician",
    "troubadour": "mcdm.heroes.v1/class/troubadour",
}

# Classes that need context checks — these words have ordinary English meanings
AMBIGUOUS_CLASSES = {
    "null": "mcdm.heroes.v1/class/null",
    "shadow": "mcdm.heroes.v1/class/shadow",
    "talent": "mcdm.heroes.v1/class/talent",
}

ANCESTRIES = {
    "devil": "mcdm.heroes.v1/ancestry/devil",
    "dragon knight": "mcdm.heroes.v1/ancestry/dragon-knight",
    "dwarf": "mcdm.heroes.v1/ancestry/dwarf",
    "hakaan": "mcdm.heroes.v1/ancestry/hakaan",
    "high elf": "mcdm.heroes.v1/ancestry/high-elf",
    "human": "mcdm.heroes.v1/ancestry/human",
    "memonek": "mcdm.heroes.v1/ancestry/memonek",
    "orc": "mcdm.heroes.v1/ancestry/orc",
    "polder": "mcdm.heroes.v1/ancestry/polder",
    "revenant": "mcdm.heroes.v1/ancestry/revenant",
    "time raider": "mcdm.heroes.v1/ancestry/time-raider",
    "wode elf": "mcdm.heroes.v1/ancestry/wode-elf",
}

# Plural forms mapping to their singular SCC code
PLURALS = {
    "censors": "mcdm.heroes.v1/class/censor",
    "conduits": "mcdm.heroes.v1/class/conduit",
    "elementalists": "mcdm.heroes.v1/class/elementalist",
    "furies": "mcdm.heroes.v1/class/fury",
    "nulls": "mcdm.heroes.v1/class/null",
    "shadows": "mcdm.heroes.v1/class/shadow",
    "tacticians": "mcdm.heroes.v1/class/tactician",
    "talents": "mcdm.heroes.v1/class/talent",
    "troubadours": "mcdm.heroes.v1/class/troubadour",
    "devils": "mcdm.heroes.v1/ancestry/devil",
    "dragon knights": "mcdm.heroes.v1/ancestry/dragon-knight",
    "dwarves": "mcdm.heroes.v1/ancestry/dwarf",
    "dwarfs": "mcdm.heroes.v1/ancestry/dwarf",
    "hakaans": "mcdm.heroes.v1/ancestry/hakaan",
    "high elves": "mcdm.heroes.v1/ancestry/high-elf",
    "humans": "mcdm.heroes.v1/ancestry/human",
    "memoneks": "mcdm.heroes.v1/ancestry/memonek",
    "orcs": "mcdm.heroes.v1/ancestry/orc",
    "polders": "mcdm.heroes.v1/ancestry/polder",
    "revenants": "mcdm.heroes.v1/ancestry/revenant",
    "time raiders": "mcdm.heroes.v1/ancestry/time-raider",
    "wode elves": "mcdm.heroes.v1/ancestry/wode-elf",
}

# All terms: merge singulars + plurals, sorted longest-first for greedy matching
ALL_TERMS = {}
ALL_TERMS.update(CLASSES)
ALL_TERMS.update(AMBIGUOUS_CLASSES)
ALL_TERMS.update(ANCESTRIES)
ALL_TERMS.update(PLURALS)

# Sort by length descending so "dragon knight" matches before "knight", etc.
SORTED_TERMS = sorted(ALL_TERMS.keys(), key=len, reverse=True)


def is_heading(line: str) -> bool:
    """Lines starting with # are headings — don't link."""
    return line.lstrip().startswith("#")


def is_annotation(line: str) -> bool:
    """Lines that are annotation comments — don't link."""
    return line.strip().startswith("<!--") and "@type:" in line


def is_table_header(line: str) -> bool:
    """Table separator lines like |---|---| — skip."""
    stripped = line.strip()
    return stripped.startswith("|") and set(stripped.replace("|", "").strip()) <= {"-", " "}


def already_linked(text: str, match_start: int, match_end: int) -> bool:
    """Check if this match position is already inside a markdown link."""
    # Check if we're inside [...](...) — look for [ before and ](scc: after
    before = text[:match_start]
    after = text[match_end:]

    # Inside link display text: [...HERE...](scc:...)
    open_bracket = before.rfind("[")
    close_bracket = before.rfind("]")
    if open_bracket > close_bracket and "](scc:" in text[open_bracket:match_end + 50]:
        return True

    # Inside link URL: [...](scc:...HERE...)
    paren_open = before.rfind("(scc:")
    paren_close = before.rfind(")")
    if paren_open > paren_close:
        return True

    # Check if immediately preceded by [ (start of link display text)
    pre = before.rstrip()
    if pre.endswith("["):
        return True

    return False


def is_ambiguous_context(term_lower: str, line: str, match_start: int) -> bool:
    """
    For ambiguous terms (null, shadow, talent), check if the usage
    is clearly a game mechanic reference vs ordinary English.

    Returns True if the term should NOT be linked (ordinary English usage).
    """
    if term_lower not in AMBIGUOUS_CLASSES:
        return False

    line_lower = line.lower()

    if term_lower in ("shadow", "shadows"):
        before = line_lower[:match_start].rstrip()
        after_text = line_lower[match_start + len(term_lower):].lstrip()
        # Whitelist: definitely the Shadow class in these patterns
        # "a shadow" / "the shadow" as class, "shadow's" possessive
        # "shadows, censors" — class list
        if term_lower == "shadows":
            # "Censors and shadows" / "shadows, censors" — class list
            if re.search(r"^[,\s]+(censors|conduits|elementalists|furies|nulls|tacticians|talents|troubadours|and\s)", after_text):
                return False
            if re.search(r"(censors|conduits|elementalists|furies|nulls|tacticians|talents|troubadours)\s+and\s*$", before):
                return False
            # "Skulk in the Shadows" — test name, not class
            if "skulk" in before[-20:]:
                return True
        # Possessive is almost always the class
        if after_text.startswith("'s"):
            return False
        # "shadow elves" — separate concept
        if after_text.startswith("elves") or after_text.startswith("elf"):
            return True
        # "shadow form" — ability mechanic name, not class
        if after_text.startswith("form"):
            return True
        # For "shadow"/"shadows" as ordinary English (darkness/shade):
        # Skip when preceded by: into, the, a, actual, cast, in, an
        # Unless it's clearly a class reference (e.g. "playing a shadow", "the shadow is")
        skip_prepositions = ("into", "actual", "cast", "with the", "in the")
        for prep in skip_prepositions:
            if before.endswith(prep):
                return True
        # "shadows" plural without possessive is usually darkness
        if term_lower == "shadows":
            # Already handled class list above — skip all other plural uses
            return True
        # "shadow" after articles in non-class contexts
        if before.endswith("the") and after_text.startswith("of"):
            return True
        if before.endswith("a") and not after_text.startswith("'s"):
            # "a shadow" — ambiguous. Check wider context for class indicators
            wider = line_lower
            if any(cls in wider for cls in ("class", "hero", "playing", "censor", "conduit", "fury", "tactician")):
                return False
            return True
        return False

    if term_lower == "talent":
        # "talent" as natural ability — check context
        # Don't link: "artistic talent", "natural talent", "a talent for"
        after_text = line_lower[match_start + len(term_lower):].lstrip()
        before = line_lower[:match_start].rstrip()
        if after_text.startswith("for ") or after_text.startswith("in "):
            return True
        if before.endswith("natural") or before.endswith("artistic") or before.endswith("raw"):
            return True
        return False

    if term_lower == "null":
        # "null" as void/nothing — very rare in game text
        # In Draw Steel, "null" almost always means the Null class
        return False

    return False


def should_skip_human(line: str, match_start: int, match_end: int) -> bool:
    """Special handling for 'human' — skip in some narrative contexts."""
    before = line[:match_start]
    after = line[match_end:]
    # "*human* skull" in italic context — referring to a physical object
    if before.rstrip().endswith("*") and after.lstrip().startswith("*"):
        if "skull" in after[:20].lower():
            return True
    # "human-like" — compound adjective, not a game reference
    if after.startswith("-like") or after.startswith("-centric"):
        return True
    return False


def process_line(line: str, line_num: int) -> str:
    """Process a single line, adding scc: links for unlinked game terms."""
    if is_heading(line):
        return line
    if is_annotation(line):
        return line
    if is_table_header(line):
        return line

    result = line
    # Process terms longest-first to handle multi-word terms before their components
    for term in SORTED_TERMS:
        scc_code = ALL_TERMS[term]
        # Build regex: case-insensitive, word-boundary aware
        # For multi-word terms, just match the phrase
        # Handle possessives: term + 's
        pattern = re.compile(
            r"(?<!\[)"           # not preceded by [ (already in link)
            r"\b(" + re.escape(term) + r"(?:'s)?)\b"
            r"(?!\]\(scc:)",     # not followed by ](scc: (already linked)
            re.IGNORECASE
        )

        offset = 0
        new_result = []
        last_end = 0

        for match in pattern.finditer(result):
            matched_text = match.group(1)
            start = match.start(1)
            end = match.end(1)

            # Skip if already inside a markdown link
            if already_linked(result, start, end):
                continue

            # Skip ambiguous terms in non-game contexts
            if is_ambiguous_context(term, result, start):
                continue

            # Special skip for "human skull" etc.
            if term == "human" and should_skip_human(result, start, end):
                continue

            # Build the link
            link = f"[{matched_text}](scc:{scc_code})"
            new_result.append(result[last_end:start])
            new_result.append(link)
            last_end = end

        if new_result:
            new_result.append(result[last_end:])
            result = "".join(new_result)

    return result


def main():
    input_file = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("input/heroes/Draw Steel Heroes.md")

    if not input_file.exists():
        print(f"ERROR: {input_file} not found", file=sys.stderr)
        sys.exit(1)

    lines = input_file.read_text().splitlines(keepends=True)

    changed = 0
    output_lines = []
    for i, line in enumerate(lines, 1):
        new_line = process_line(line, i)
        if new_line != line:
            changed += 1
        output_lines.append(new_line)

    if "--dry-run" in sys.argv:
        # Show diff
        for i, (old, new) in enumerate(zip(lines, output_lines), 1):
            if old != new:
                print(f"L{i}:")
                print(f"  - {old.rstrip()}")
                print(f"  + {new.rstrip()}")
        print(f"\n{changed} lines would change")
    else:
        input_file.write_text("".join(output_lines))
        print(f"{changed} lines changed")

    # Count total links
    total = sum(line.count("scc:") for line in output_lines)
    print(f"Total scc: links: {total}")


if __name__ == "__main__":
    main()
