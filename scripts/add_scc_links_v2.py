#!/usr/bin/env python3
"""
Comprehensive scc: link adder v2 — adds all remaining term types:
kits, careers, perks, complications, titles, treasures.

Strategy:
- UNAMBIGUOUS terms (unique game names): link everywhere
- AMBIGUOUS terms (common English words): link only within their own chapter

Chapter line ranges (1-indexed):
  Careers: 3494-4065
  Kits: 17607-18580
  Perks: 18581-18946
  Complications: 18947-20167
  Titles: 25259-26339
  Treasures: 23221-25258
"""

import re
import sys
from pathlib import Path

CHAPTER_RANGES = {
    "career": (3494, 4065),
    "kit": (17607, 18580),
    "perk": (18581, 18946),
    "complication": (18947, 20167),
    "title": (25259, 26339),
    "treasure": (23221, 25258),
}

# ── KITS ──────────────────────────────────────────────────────
# Most kit names are unique game terms → link everywhere
# Exceptions: "mountain", "raider", "ranger", "whirlwind", "sniper" are ambiguous
KITS = {
    # Unambiguous — link everywhere
    "arcane archer": ("mcdm.heroes.v1/kit/arcane-archer", False),
    "battlemind": ("mcdm.heroes.v1/kit/battlemind", False),
    "boren": ("mcdm.heroes.v1/kit/boren", False),
    "cloak and dagger": ("mcdm.heroes.v1/kit/cloak-and-dagger", False),
    "corven": ("mcdm.heroes.v1/kit/corven", False),
    "dual wielder": ("mcdm.heroes.v1/kit/dual-wielder", False),
    "guisarmier": ("mcdm.heroes.v1/kit/guisarmier", False),
    "martial artist": ("mcdm.heroes.v1/kit/martial-artist", False),
    "panther": ("mcdm.heroes.v1/kit/panther", False),
    "pugilist": ("mcdm.heroes.v1/kit/pugilist", False),
    "raden": ("mcdm.heroes.v1/kit/raden", False),
    "retiarius": ("mcdm.heroes.v1/kit/retiarius", False),
    "shining armor": ("mcdm.heroes.v1/kit/shining-armor", False),
    "spellsword": ("mcdm.heroes.v1/kit/spellsword", False),
    "stick and robe": ("mcdm.heroes.v1/kit/stick-and-robe", False),
    "swashbuckler": ("mcdm.heroes.v1/kit/swashbuckler", False),
    "sword and board": ("mcdm.heroes.v1/kit/sword-and-board", False),
    "vuken": ("mcdm.heroes.v1/kit/vuken", False),
    "warrior priest": ("mcdm.heroes.v1/kit/warrior-priest", False),
    "rapid fire": ("mcdm.heroes.v1/kit/rapid-fire", False),
    # Ambiguous — own chapter only
    "mountain": ("mcdm.heroes.v1/kit/mountain", True),
    "raider": ("mcdm.heroes.v1/kit/raider", True),
    "ranger": ("mcdm.heroes.v1/kit/ranger", True),
    "sniper": ("mcdm.heroes.v1/kit/sniper", True),
    "whirlwind": ("mcdm.heroes.v1/kit/whirlwind", True),
}

# ── CAREERS ───────────────────────────────────────────────────
# Most career names are common English → ambiguous
CAREERS = {
    # Unambiguous
    "mage's apprentice": ("mcdm.heroes.v1/career/mages-apprentice", False),
    "watch officer": ("mcdm.heroes.v1/career/watch-officer", False),
    # Ambiguous — own chapter only
    "agent": ("mcdm.heroes.v1/career/agent", True),
    "aristocrat": ("mcdm.heroes.v1/career/aristocrat", True),
    "artisan": ("mcdm.heroes.v1/career/artisan", True),
    "beggar": ("mcdm.heroes.v1/career/beggar", True),
    "criminal": ("mcdm.heroes.v1/career/criminal", True),
    "disciple": ("mcdm.heroes.v1/career/disciple", True),
    "explorer": ("mcdm.heroes.v1/career/explorer", True),
    "farmer": ("mcdm.heroes.v1/career/farmer", True),
    "gladiator": ("mcdm.heroes.v1/career/gladiator", True),
    "laborer": ("mcdm.heroes.v1/career/laborer", True),
    "performer": ("mcdm.heroes.v1/career/performer", True),
    "politician": ("mcdm.heroes.v1/career/politician", True),
    "sage": ("mcdm.heroes.v1/career/sage", True),
    "sailor": ("mcdm.heroes.v1/career/sailor", True),
    "soldier": ("mcdm.heroes.v1/career/soldier", True),
    "warden": ("mcdm.heroes.v1/career/warden", True),
}

# ── PERKS ─────────────────────────────────────────────────────
# Multi-word perk names are unambiguous; single common words are ambiguous
PERKS = {
    # Unambiguous (multi-word or unique)
    "arcane trick": ("mcdm.heroes.v1/perk/arcane-trick", False),
    "area of expertise": ("mcdm.heroes.v1/perk/area-of-expertise", False),
    "camouflage hunter": ("mcdm.heroes.v1/perk/camouflage-hunter", False),
    "charming liar": ("mcdm.heroes.v1/perk/charming-liar", False),
    "creature sense": ("mcdm.heroes.v1/perk/creature-sense", False),
    "criminal contacts": ("mcdm.heroes.v1/perk/criminal-contacts", False),
    "danger sense": ("mcdm.heroes.v1/perk/danger-sense", False),
    "eidetic memory": ("mcdm.heroes.v1/perk/eidetic-memory", False),
    "engrossing monologue": ("mcdm.heroes.v1/perk/engrossing-monologue", False),
    "expert artisan": ("mcdm.heroes.v1/perk/expert-artisan", False),
    "expert sage": ("mcdm.heroes.v1/perk/expert-sage", False),
    "forgettable face": ("mcdm.heroes.v1/perk/forgettable-face", False),
    "friend catapult": ("mcdm.heroes.v1/perk/friend-catapult", False),
    "gum up the works": ("mcdm.heroes.v1/perk/gum-up-the-works", False),
    "improvisation creation": ("mcdm.heroes.v1/perk/improvisation-creation", False),
    "inspired artisan": ("mcdm.heroes.v1/perk/inspired-artisan", False),
    "invisible force": ("mcdm.heroes.v1/perk/invisible-force", True),
    "lie detector": ("mcdm.heroes.v1/perk/lie-detector", False),
    "lucky dog": ("mcdm.heroes.v1/perk/lucky-dog", False),
    "master of disguise": ("mcdm.heroes.v1/perk/master-of-disguise", False),
    "monster whisperer": ("mcdm.heroes.v1/perk/monster-whisperer", False),
    "open book": ("mcdm.heroes.v1/perk/open-book", True),
    "pardon my friend": ("mcdm.heroes.v1/perk/pardon-my-friend", True),
    "power player": ("mcdm.heroes.v1/perk/power-player", False),
    "psychic whisper": ("mcdm.heroes.v1/perk/psychic-whisper", False),
    "put your back into it": ("mcdm.heroes.v1/perk/put-your-back-into-it", True),
    "slipped lead": ("mcdm.heroes.v1/perk/slipped-lead", False),
    "so tell me": ("mcdm.heroes.v1/perk/so-tell-me", True),
    "spot the tell": ("mcdm.heroes.v1/perk/spot-the-tell", False),
    "team leader": ("mcdm.heroes.v1/perk/team-leader", False),
    "traveling artisan": ("mcdm.heroes.v1/perk/traveling-artisan", False),
    "traveling sage": ("mcdm.heroes.v1/perk/traveling-sage", False),
    "wood wise": ("mcdm.heroes.v1/perk/wood-wise", False),
    "thingspeaker": ("mcdm.heroes.v1/perk/thingspeaker", False),
    "harmonizer": ("mcdm.heroes.v1/perk/harmonizer", False),
    "dazzler": ("mcdm.heroes.v1/perk/dazzler", False),
    "ritualist": ("mcdm.heroes.v1/perk/ritualist", False),
    # Ambiguous — own chapter only
    "brawny": ("mcdm.heroes.v1/perk/brawny", True),
    "familiar": ("mcdm.heroes.v1/perk/familiar", True),
    "handy": ("mcdm.heroes.v1/perk/handy", True),
    "linguist": ("mcdm.heroes.v1/perk/linguist", True),
    "polymath": ("mcdm.heroes.v1/perk/polymath", True),
    "specialist": ("mcdm.heroes.v1/perk/specialist", True),
    "teamwork": ("mcdm.heroes.v1/perk/teamwork", True),
}

# ── COMPLICATIONS ─────────────────────────────────────────────
# Almost all are ambiguous common English — link within own chapter only
# Multi-word unique ones can link everywhere
COMPLICATIONS = {}
_complication_terms = [
    ("advanced studies", False), ("amnesia", True), ("animal form", False),
    ("antihero", False), ("artifact bonded", False), ("bereaved", True),
    ("betrothed", True), ("chaos touched", False), ("chosen one", True),
    ("consuming interest", False), ("corrupted mentor", False), ("coward", True),
    ("crash landed", False), ("cult victim", False), ("curse of caution", False),
    ("curse of immortality", False), ("curse of misfortune", False),
    ("curse of poverty", False), ("curse of punishment", False),
    ("curse of stone", False), ("cursed weapon", False), ("disgraced", True),
    ("dragon dreams", False), ("elemental inside", False),
    ("evanesceria", False), ("exile", True), ("fallen immortal", False),
    ("famous relative", False), ("feytouched", False), ("fiery ideal", False),
    ("fire and chaos", False), ("following in the footsteps", False),
    ("forbidden romance", False), ("frostheart", False),
    ("getting too old for this", False), ("gnoll mauled", False),
    ("greening", True), ("grifter", True), ("grounded", True),
    ("guilty conscience", False), ("hawk rider", False), ("host body", False),
    ("hunted", True), ("hunter", True), ("indebted", True),
    ("infernal contract", False), ("infernal contract but like bad", False),
    ("ivory tower", True), ("lifebonded", False), ("lightning soul", False),
    ("loner", True), ("lost in time", False), ("lost your head", False),
    ("lucky", True), ("master chef", False), ("meddling butler", False),
    ("medium", True), ("medusa blood", False), ("misunderstood", True),
    ("mundane", True), ("outlaw", True), ("pirate", True),
    ("preacher", True), ("primordial sickness", False),
    ("prisoner of the synlirii", False), ("promising apprentice", False),
    ("psychic eruption", False), ("raised by beasts", False),
    ("refugee", True), ("rival", True), ("rogue talent", False),
    ("runaway", True), ("searching for a cure", False),
    ("secret identity", False), ("secret twin", False), ("self taught", False),
    ("sewer folk", False), ("shadow born", False), ("shared spirit", False),
    ("shattered legacy", False), ("shipwrecked", True),
    ("siblings shield", False), ("silent sentinel", False),
    ("slight case of lycanthropy", False), ("stolen face", False),
    ("strange inheritance", False), ("stripped of rank", False),
    ("thrill seeker", False), ("vampire scion", False),
    ("voice in your head", False), ("vow of duty", False),
    ("vow of honesty", False), ("waking dreams", False),
    ("war dog collar", False), ("war of assassins", False),
    ("ward", True), ("waterborn", False), ("wodewalker", False),
    ("wrathful spirit", False), ("wrongly imprisoned", False),
]
for term, ambig in _complication_terms:
    scc_id = term.replace(" ", "-")
    COMPLICATIONS[term] = (f"mcdm.heroes.v1/complication/{scc_id}", ambig)

# ── TITLES ────────────────────────────────────────────────────
# Most multi-word titles are unambiguous; single words like "knight", "noble" are very ambiguous
TITLES = {}
_title_terms = [
    ("ancient loremaster", False), ("arena fighter", False),
    ("armed and dangerous", False), ("awakened", True),
    ("back from the grave", False), ("battleaxe diplomat", False),
    ("battlefield commander", False), ("blood magic", False),
    ("brawler", True), ("champion competitor", False), ("city rat", False),
    ("corsair", False), ("demigod", False), ("demon slayer", False),
    ("diabolist", False), ("doomed", True), ("dragon blooded", False),
    ("dwarven legionnaire", False), ("elemental dabbler", False),
    ("enlightened", True), ("faction member", False), ("faction officer", False),
    ("fey friend", False), ("fleet admiral", False), ("follower types", False),
    ("forsaken", True), ("giant slayer", False), ("godsworn", False),
    ("heist hero", False), ("knight", True), ("local hero", False),
    ("maestro", True), ("mage hunter", False), ("marshal", True),
    ("master crafter", False), ("master librarian", False),
    ("monarch", True), ("monster bane", False), ("noble", True),
    ("owed a favor", False), ("peace bringer", False),
    ("planar voyager", False), ("presumed dead", False),
    ("ratcatcher", False), ("reborn", True), ("saved for a worse fate", False),
    ("scarred", True), ("ship captain", False), ("siege breaker", False),
    ("special agent", False), ("stronghold", True), ("sworn hunter", False),
    ("teacher", True), ("theoretical warrior", False), ("tireless", True),
    ("troupe leading player", False), ("unchained", True),
    ("undead slain", False), ("unstoppable", True),
    ("wanted dead or alive", False), ("zombie slayer", False),
]
for term, ambig in _title_terms:
    scc_id = term.replace(" ", "-")
    TITLES[term] = (f"mcdm.heroes.v1/title/{scc_id}", ambig)

# ── TREASURES ─────────────────────────────────────────────────
# All treasure terms are multi-word section headers — link within own chapter only
TREASURES = {
    "1st echelon consumables": ("mcdm.heroes.v1/treasure/1st-echelon-consumables", True),
    "1st echelon trinkets": ("mcdm.heroes.v1/treasure/1st-echelon-trinkets", True),
    "2nd echelon consumables": ("mcdm.heroes.v1/treasure/2nd-echelon-consumables", True),
    "2nd echelon trinkets": ("mcdm.heroes.v1/treasure/2nd-echelon-trinkets", True),
    "3rd echelon consumables": ("mcdm.heroes.v1/treasure/3rd-echelon-consumables", True),
    "3rd echelon trinkets": ("mcdm.heroes.v1/treasure/3rd-echelon-trinkets", True),
    "4th echelon consumables": ("mcdm.heroes.v1/treasure/4th-echelon-consumables", True),
    "4th echelon trinkets": ("mcdm.heroes.v1/treasure/4th-echelon-trinkets", True),
    "leveled armor treasures": ("mcdm.heroes.v1/treasure/leveled-armor-treasures", True),
    "leveled benefits": ("mcdm.heroes.v1/treasure/leveled-benefits", True),
    "leveled implement treasures": ("mcdm.heroes.v1/treasure/leveled-implement-treasures", True),
    "leveled weapon treasures": ("mcdm.heroes.v1/treasure/leveled-weapon-treasures", True),
    "magic and psionic treasures": ("mcdm.heroes.v1/treasure/magic-and-psionic-treasures", True),
    "other leveled treasures": ("mcdm.heroes.v1/treasure/other-leveled-treasures", True),
}

# ── Merge all terms ───────────────────────────────────────────
ALL_TERMS = {}
for terms_dict, scc_type in [
    (KITS, "kit"), (CAREERS, "career"), (PERKS, "perk"),
    (COMPLICATIONS, "complication"), (TITLES, "title"), (TREASURES, "treasure"),
]:
    for term, (code, ambiguous) in terms_dict.items():
        ALL_TERMS[term] = (code, ambiguous, scc_type)

SORTED_TERMS = sorted(ALL_TERMS.keys(), key=len, reverse=True)


def is_heading(line: str) -> bool:
    return line.lstrip().startswith("#")

def is_annotation(line: str) -> bool:
    return line.strip().startswith("<!--") and "@type:" in line

def already_linked(text: str, match_start: int, match_end: int) -> bool:
    before = text[:match_start]
    after = text[match_end:]
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

def in_chapter(line_num: int, scc_type: str) -> bool:
    if scc_type in CHAPTER_RANGES:
        start, end = CHAPTER_RANGES[scc_type]
        return start <= line_num <= end
    return False

def process_line(line: str, line_num: int) -> str:
    if is_heading(line):
        return line
    if is_annotation(line):
        return line

    result = line
    for term in SORTED_TERMS:
        scc_code, ambiguous, scc_type = ALL_TERMS[term]

        # Skip ambiguous terms outside their own chapter
        if ambiguous and not in_chapter(line_num, scc_type):
            continue

        pattern = re.compile(
            r"(?<!\[)"
            r"\b(" + re.escape(term) + r"(?:'s)?)\b"
            r"(?!\]\(scc:)",
            re.IGNORECASE
        )

        new_result = []
        last_end = 0
        for match in pattern.finditer(result):
            matched_text = match.group(1)
            start = match.start(1)
            end = match.end(1)
            if already_linked(result, start, end):
                continue
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
    lines = input_file.read_text().splitlines(keepends=True)

    changed = 0
    output_lines = []
    for i, line in enumerate(lines, 1):
        new_line = process_line(line, i)
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
