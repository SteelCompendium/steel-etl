#!/usr/bin/env python3
"""Apply a single (pattern -> scc code) link rule to the heroes doc.

Skips headings, and never links inside existing [..](scc:..) links, inside
<!-- comments -->, or inside markdown table separator rows. Prints a unified-ish
diff of changed lines. Writes only when --apply is passed.
"""
import re, sys

DOC = "/home/scott/code/steelCompendium/workspace/steel-etl/input/heroes/Draw Steel Heroes.md"

# rule: PATTERN (regex, must capture group 1 = display text), CODE
PATTERN = sys.argv[1]
CODE = sys.argv[2]
APPLY = "--apply" in sys.argv
# args like "100-200" = skip those line ranges (1-based, inclusive) — own-section excl.
EXCL = []
for a in sys.argv[3:]:
    m = re.fullmatch(r"(\d+)-(\d+)", a)
    if m:
        EXCL.append((int(m.group(1)), int(m.group(2))))
def excluded(ln):
    return any(s <= ln <= e for s, e in EXCL)

rx = re.compile(PATTERN)
LINKSPAN = re.compile(r"\[[^\]]*\]\(scc:[^)]*\)")
COMMENT = re.compile(r"<!--.*?-->")
HEADING = re.compile(r"^\s*(>\s*)*#{1,6}\s")

lines = open(DOC, encoding="utf-8").read().split("\n")
changed = 0
out = []
for i, raw in enumerate(lines, 1):
    if HEADING.match(raw) or not raw.strip() or excluded(i):
        out.append(raw); continue
    # mask protected spans (preserve offsets)
    masked = LINKSPAN.sub(lambda m: "\x00"*len(m.group(0)), raw)
    masked = COMMENT.sub(lambda m: "\x00"*len(m.group(0)), masked)
    # find matches in masked, splice into raw right-to-left
    spans = []
    for m in rx.finditer(masked):
        if "\x00" in masked[m.start():m.end()]:
            continue
        spans.append((m.start(1), m.end(1), m.group(1)))
    if not spans:
        out.append(raw); continue
    new = raw
    for (s, e, disp) in reversed(spans):
        new = new[:s] + f"[{disp}](scc:{CODE})" + new[e:]
    if new != raw:
        changed += len(spans)
        print(f"L{i}:\n  - {raw.strip()[:170]}\n  + {new.strip()[:200]}")
    out.append(new)

print(f"\n== {changed} replacement(s) on rule {PATTERN!r} -> {CODE} ==")
if APPLY and changed:
    open(DOC, "w", encoding="utf-8").write("\n".join(out))
    print("APPLIED.")
elif changed:
    print("(dry run — pass --apply to write)")
