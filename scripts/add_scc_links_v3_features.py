#!/usr/bin/env python3
"""
Feature/ability scc: link adder — adds the 1,123 feature codes.

Strategy:
- Multi-word names (3+ words or unique 2-word): link everywhere
- Single-word and common 2-word names: link only within their owner's class section
- Names with multiple owners: link only within the matching class section
- Skip headings (lines starting with #)
- Skip annotation comments
- Skip already-linked text
"""

import json
import re
import sys
from pathlib import Path

# Class section line ranges (1-indexed)
CLASS_SECTIONS = {
    "censor": (4553, 6005),
    "conduit": (6006, 7793),
    "elementalist": (7794, 9343),
    "fury": (9344, 10926),
    "null": (10927, 12210),
    "shadow": (12211, 13487),
    "tactician": (13488, 14641),
    "talent": (14642, 16179),
    "troubadour": (16180, 17606),
    # Kits chapter for kit abilities
    "arcane-archer": (17607, 18580),
    "battlemind": (17607, 18580),
    "boren": (9344, 10926),  # Boren is a fury stormwight kit
    "cloak-and-dagger": (17607, 18580),
    "corven": (9344, 10926),
    "dual-wielder": (17607, 18580),
    "guisarmier": (17607, 18580),
    "martial-artist": (17607, 18580),
    "mountain": (17607, 18580),
    "panther": (17607, 18580),
    "pugilist": (17607, 18580),
    "raden": (9344, 10926),
    "raider": (17607, 18580),
    "ranger": (17607, 18580),
    "rapid-fire": (17607, 18580),
    "retiarius": (17607, 18580),
    "shining-armor": (17607, 18580),
    "sniper": (17607, 18580),
    "spellsword": (17607, 18580),
    "stick-and-robe": (17607, 18580),
    "swashbuckler": (17607, 18580),
    "sword-and-board": (17607, 18580),
    "vuken": (9344, 10926),
    "warrior-priest": (17607, 18580),
    "whirlwind": (17607, 18580),
    # Ancestries chapter for ancestry traits
    "devil": (1527, 1673),
    "dragon-knight": (1675, 1832),
    "dwarf": (1833, 1978),
    "wode-elf": (1979, 2115),
    "high-elf": (2116, 2225),
    "hakaan": (2226, 2361),
    "human": (2362, 2437),
    "memonek": (2438, 2573),
    "orc": (2574, 2731),
    "polder": (2732, 2902),
    "revenant": (2903, 3027),
    "time-raider": (3028, 3199),
    # Generic/common features — link within Classes chapter intro
    "common": (4066, 4552),
    "generic": (4066, 4552),
}

# Extremely common English words that should NEVER be linked outside their section
# Terms to NEVER link, not even within their own section — too common as English
NEVER_LINK_AT_ALL = {
    "again", "back", "now", "look", "one", "friend", "iron", "steel",
    "mark", "order", "slow", "fade", "drain", "kit", "perk", "skill",
    "breath", "prayer", "focus", "insight", "drama", "essence",
    "ferocity", "discipline", "piety", "wrath", "vow",
    "equipment", "masterwork", "vision", "epic", "spotlight",
    "doubt", "fate", "command", "blur", "setup", "scan",
    "encore", "foil", "hustle", "reap", "rout",
    "arise", "begone", "goaded", "blocking",
}

# Terms to link only within their own section (not globally)
NEVER_LINK_GLOBALLY = {
    "parry", "arrest", "judgment", "repent", "banish",
    "choke", "overwhelm", "overwatch", "pounce", "riposte",
    "impaled", "sentenced", "dancer", "feedback", "instigator",
    "avatar", "psion", "templar", "ordained", "sanctuary",
    "prophecy", "enchantment", "invocation", "deluge", "meteor",
    "accelerate", "conflagration", "eviscerate", "incinerate",
    "assassinate", "shadowfall", "shadowstrike", "shadowmeld", "shadowgrasp",
    "triangulate", "precognition", "mindlink", "mindwipe",
    "harmonize", "melodrama", "upstage", "flashback", "choreography",
    "acrobatics", "dramaturgy", "ventriloquist",
    "stormborn", "thunderstruck", "windwalk", "hoarfrost",
    "overkill", "steelbreaker", "hypersonic",
    "excommunication", "congregation", "demonologist", "revelator",
    "apostate", "censored", "intercede", "penance",
    "soulbound", "rejuvenate", "godstorm", "lightfall",
    "realitas", "zeitgeist", "wyrding", "prism",
    "distracted", "anticipation", "compulsion", "awe",
    "applause", "subterfuge", "smolder", "cinderstorm",
    "warmaster", "routines", "counterstrategy",
    "unfettered", "materialize", "omnisensory", "interphase",
    "menagerie", "seance", "bounder", "parkour",
    "wither", "repel", "reorder",
}


COMMON_PHRASE_WORDS = {
    "triggered action", "free strike", "hit and", "out of",
    "in all", "fog of", "shake it", "stay with", "to the",
    "on my", "coat the", "call the", "face the", "keep it",
    "pain for", "get in", "go now", "hear ye", "wall of",
    "net and", "try me", "win this",
}

def _is_common_phrase(display: str) -> bool:
    """Check if a multi-word display name is a common English phrase."""
    dl = display.lower()
    # Any 2-word name containing very common verbs/prepositions is likely ambiguous
    words = dl.split()
    if len(words) == 2:
        common_first = {"hit", "get", "go", "out", "in", "on", "to", "no", "my", "do"}
        common_second = {"me", "it", "up", "down", "out", "now", "here", "there", "back"}
        if words[0] in common_first or words[1] in common_second:
            return True
    # Check against known common phrase starts
    for phrase in COMMON_PHRASE_WORDS:
        if dl.startswith(phrase):
            return True
    return False


def load_features():
    """Load feature terms from classification.json."""
    with open("classification.json", "r") as f:
        data = json.load(f)

    # Build: display_name -> [(scc_code, owner)]
    raw = {}
    for code in data["codes"]:
        parts = code.split("/")
        if len(parts) < 3 or "feature." not in parts[1]:
            continue
        type_path = parts[1]
        item_id = parts[2]
        display = item_id.replace("-", " ")

        tp_parts = type_path.split(".")
        owner = tp_parts[2] if len(tp_parts) >= 3 else "generic"

        if display not in raw:
            raw[display] = []
        raw[display].append((code, owner))

    # Build final term list
    # For each name: determine if it's ambiguous and which owners it has
    terms = {}
    for display, entries in raw.items():
        owners = list(set(e[1] for e in entries))
        # Use the first code (ability preferred over trait)
        code = entries[0][0]
        for c, o in entries:
            if ".ability." in c:
                code = c
                break

        word_count = len(display.split())
        # Conservative: only globally link 2-3 word names that are clearly
        # unique game terms (not common English phrases)
        # Everything else links only within its own class section
        # Skip terms that are too common to ever link
        if display in NEVER_LINK_AT_ALL:
            continue

        is_ambiguous = (
            word_count == 1 or
            word_count >= 4 or  # Long phrases are too sentence-like
            display in NEVER_LINK_GLOBALLY or
            len(owners) > 1 or
            _is_common_phrase(display)
        )

        terms[display] = {
            "code": code,
            "owners": owners,
            "ambiguous": is_ambiguous,
        }

    return terms


def is_heading(line: str) -> bool:
    stripped = line.lstrip()
    if stripped.startswith("#"):
        return True
    # Blockquote headings: "> ###### Name"
    if stripped.startswith(">"):
        inner = stripped[1:].lstrip()
        if inner.startswith("#"):
            return True
    return False

def is_annotation(line: str) -> bool:
    return line.strip().startswith("<!--") and "@type:" in line

def already_linked(text: str, match_start: int, match_end: int) -> bool:
    before = text[:match_start]
    open_bracket = before.rfind("[")
    close_bracket = before.rfind("]")
    if open_bracket > close_bracket and "](scc:" in text[open_bracket:match_end + 80]:
        return True
    paren_open = before.rfind("(scc:")
    paren_close = before.rfind(")")
    if paren_open > paren_close:
        return True
    if before.rstrip().endswith("["):
        return True
    return False

def in_owner_section(line_num: int, owners: list) -> bool:
    """Check if line_num is within any of the owner's sections."""
    for owner in owners:
        if owner in CLASS_SECTIONS:
            start, end = CLASS_SECTIONS[owner]
            if start <= line_num <= end:
                return True
    return False


def precompile_patterns(terms, sorted_terms):
    """Pre-compile regex patterns for all terms."""
    patterns = {}
    for term in sorted_terms:
        patterns[term] = re.compile(
            r"(?<!\[)"
            r"\b(" + re.escape(term) + r")\b"
            r"(?!\]\(scc:)",
            re.IGNORECASE
        )
    return patterns


def process_line(line: str, line_num: int, terms: dict, sorted_terms: list, patterns: dict) -> str:
    if is_heading(line):
        return line
    if is_annotation(line):
        return line

    line_lower = line.lower()
    result = line
    for term in sorted_terms:
        # Quick check: skip if term not even present in line (case-insensitive)
        if term not in line_lower:
            continue

        info = terms[term]
        if info["ambiguous"] and not in_owner_section(line_num, info["owners"]):
            continue

        pattern = patterns[term]
        new_result = []
        last_end = 0
        for match in pattern.finditer(result):
            matched_text = match.group(1)
            start = match.start(1)
            end = match.end(1)
            if already_linked(result, start, end):
                continue
            link = f"[{matched_text}](scc:{info['code']})"
            new_result.append(result[last_end:start])
            new_result.append(link)
            last_end = end

        if new_result:
            new_result.append(result[last_end:])
            result = "".join(new_result)
            line_lower = result.lower()  # update for subsequent terms

    return result


def main():
    input_file = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("input/heroes/Draw Steel Heroes.md")
    lines = input_file.read_text().splitlines(keepends=True)

    terms = load_features()
    sorted_terms = sorted(terms.keys(), key=len, reverse=True)
    patterns = precompile_patterns(terms, sorted_terms)

    print(f"Loaded {len(terms)} feature terms", file=sys.stderr)
    global_terms = sum(1 for t, i in terms.items() if not i["ambiguous"])
    local_terms = sum(1 for t, i in terms.items() if i["ambiguous"])
    print(f"  Global (unambiguous): {global_terms}", file=sys.stderr)
    print(f"  Local (ambiguous, own section only): {local_terms}", file=sys.stderr)

    changed = 0
    output_lines = []
    for i, line in enumerate(lines, 1):
        new_line = process_line(line, i, terms, sorted_terms, patterns)
        if new_line != line:
            changed += 1
        output_lines.append(new_line)

    if "--dry-run" in sys.argv:
        for i, (old, new) in enumerate(zip(lines, output_lines), 1):
            if old != new:
                print(f"L{i}:")
                print(f"  - {old.rstrip()}")
                print(f"  + {new.rstrip()}")
        print(f"\n{changed} lines would change")
    else:
        input_file.write_text("".join(output_lines))
        print(f"{changed} lines changed")

    total = sum(line.count("scc:") for line in output_lines)
    print(f"Total scc: links: {total}")


if __name__ == "__main__":
    main()
