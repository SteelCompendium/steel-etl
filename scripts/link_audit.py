#!/usr/bin/env python3
"""Link audit for the heroes input doc.

Harvests authoritative display names from generated md frontmatter (name+scc),
then scans the input doc for (a) unlinked occurrences of known game entities and
(b) truncated links (a link whose display text is a prefix of a longer adjacent
entity name). Produces a candidate report for per-instance review.
"""
import json, re, os, sys
from collections import defaultdict

ETL = "/home/scott/code/steelCompendium/workspace/steel-etl"
DOC = os.path.join(ETL, "input/heroes/Draw Steel Heroes.md")
MDDIR = "/home/scott/code/steelCompendium/workspace/data/data-rules/en/md"

# ---- 1. harvest code -> display name, and type, from generated md frontmatter
code2name = {}
code2type = {}   # the scc "type" segment (e.g. feature.ability.fury.level-1, kit, condition)
for root, _, files in os.walk(MDDIR):
    for f in files:
        if not f.endswith(".md"):
            continue
        p = os.path.join(root, f)
        txt = open(p, encoding="utf-8").read()
        m = re.match(r"^---\n(.*?)\n---", txt, re.S)
        if not m:
            continue
        fm = m.group(1)
        nm = re.search(r"^name:\s*(.+)$", fm, re.M)
        sc = re.search(r"^scc:\s*(\S+)$", fm, re.M)
        if not (nm and sc):
            continue
        name = nm.group(1).strip().strip('"').strip("'")
        code = sc.group(1).strip()
        if not code.startswith("mcdm.heroes.v1/"):
            continue
        code2name[code] = name
        # type = the part between the version prefix and the final /id
        body = code[len("mcdm.heroes.v1/"):]
        code2type[code] = body.rsplit("/", 1)[0] if "/" in body else body

def top_type(code):
    t = code2type.get(code, "")
    return t.split(".")[0] if t else ""   # feature, kit, condition, ...

def subtype(code):
    t = code2type.get(code, "")
    parts = t.split(".")
    return parts[1] if len(parts) > 1 else ""  # ability/trait for feature.*

# ---- 2. build name -> [codes], with variants
name2codes = defaultdict(set)

def add(name, code):
    name2codes[name].add(code)

for code, name in code2name.items():
    add(name, code)

# variant generation
def variants(name):
    vs = {name}
    # plural / possessive
    vs.add(name + "s")
    vs.add(name + "'s")
    vs.add(name + "’s")
    if name.endswith("y"):
        vs.add(name[:-1] + "ies")
    # hyphen <-> space swap
    if "-" in name:
        vs.add(name.replace("-", " "))
    if " " in name:
        vs.add(name.replace(" ", "-"))
    return vs

allterms = defaultdict(set)  # term -> codes
for code, name in code2name.items():
    for v in variants(name):
        allterms[v].add(code)

# ---- 3. read doc lines, mask links + comments (preserve offsets)
lines = open(DOC, encoding="utf-8").read().split("\n")
LINKSPAN = re.compile(r"\[[^\]]*\]\(scc:[^)]*\)")
COMMENT = re.compile(r"<!--.*?-->")
HEADING = re.compile(r"^#{1,6}\s")

def mask(line):
    line = LINKSPAN.sub(lambda m: " " * len(m.group(0)), line)
    line = COMMENT.sub(lambda m: " " * len(m.group(0)), line)
    return line

# which entity types to treat as high-precision proper nouns
PROPER_SUB = {"ability", "trait"}  # feature.ability / feature.trait
PROPER_TOP = {"kit", "complication", "title", "treasure", "god", "project",
              "career", "ancestry", "class", "perk"}

def is_proper(code):
    tt = top_type(code)
    if tt == "feature":
        return subtype(code) in PROPER_SUB
    return tt in PROPER_TOP

# Single-word common-English entity names to handle with care (report separately)
COMMON_WORDS = {"shadow","fury","null","talent","advance","ride","hide","climb",
                "search","sneak","lift","escape","grab","aid","charge","knockback",
                "free","strike","censor","conduit","might","reason","intuition"}

# ---- 4. scan for unlinked occurrences (combined alternation regex = fast)
missing = defaultdict(list)   # (term,key,flagcommon) -> [(lineno, matched, context)]

# keep only proper-noun terms, length >= 4
lower2 = {}   # lowercased term -> (canonical term, proper codes)
for term, codes in allterms.items():
    if len(term) < 4:
        continue
    pcodes = [c for c in codes if is_proper(c)]
    if not pcodes:
        continue
    lt = term.lower()
    # prefer the longest canonical term for a given lowercase key
    if lt not in lower2 or len(term) > len(lower2[lt][0]):
        prev = lower2.get(lt, (term, set()))
        lower2[lt] = (term, set(pcodes) | prev[1])
    else:
        lower2[lt][1].update(pcodes)

# build one big alternation, longest-first
alts = sorted((re.escape(t) for t in lower2.keys()), key=len, reverse=True)
BIG = re.compile(r"(?<![\w-])(" + "|".join(alts) + r")(?![\w-])", re.I)

for i, raw in enumerate(lines, 1):
    if HEADING.match(raw):
        continue
    masked = mask(raw)
    if not masked.strip():
        continue
    for m in BIG.finditer(masked):
        matched = m.group(0)
        lt = matched.lower()
        if lt not in lower2:
            continue
        term, pcodes = lower2[lt]
        pcodes = sorted(pcodes)
        base = lt.rstrip("s").rstrip("'").rstrip("’")
        flagcommon = base in COMMON_WORDS
        key = pcodes[0] if len(pcodes) == 1 else "AMBIG:" + ",".join(pcodes)
        missing[(term, key, flagcommon)].append((i, matched, raw.strip()[:160]))

# ---- 5. truncated-link detection
trunc = []
LINK = re.compile(r"\[([^\]]+)\]\(scc:([^)]+)\)")
for i, raw in enumerate(lines, 1):
    for m in LINK.finditer(raw):
        disp, code = m.group(1), m.group(2)
        after = raw[m.end():]
        # capitalized continuation right after the link
        cont = re.match(r"\s+((?:[A-Z][\w'’-]*(?:\s+(?:and|of|the|to|for|[A-Z][\w'’-]*))*))", after)
        if not cont:
            continue
        tail = cont.group(1)
        # try disp + tail words against known names (greedy shrink)
        words = tail.split()
        for n in range(min(len(words), 5), 0, -1):
            combo = disp + " " + " ".join(words[:n])
            if combo in allterms and code not in allterms[combo]:
                trunc.append((i, disp, code, combo, sorted(allterms[combo])))
                break

# ---- 6. report
print("="*70)
print("TRUNCATED / WRONG LINK CANDIDATES")
print("="*70)
for i, disp, code, combo, tgt in trunc:
    print(f"L{i}: [{disp}] (-> {code})  SHOULD BE [{combo}] -> {tgt}")

print()
print("="*70)
print("MISSING PROPER-NOUN LINK CANDIDATES (grouped)")
print("="*70)
# group: non-common first
items = sorted(missing.items(), key=lambda kv: (kv[0][2], -len(kv[1])))
for (term, key, flagcommon), occs in items:
    tag = " [COMMON-WORD: verify mundane vs mechanic]" if flagcommon else ""
    print(f"\n### '{term}' -> {key}  ({len(occs)} occ){tag}")
    for (ln, matched, ctx) in occs[:12]:
        print(f"   L{ln}: …{ctx}…")
    if len(occs) > 12:
        print(f"   … +{len(occs)-12} more")
