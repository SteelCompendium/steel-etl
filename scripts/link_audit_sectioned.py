#!/usr/bin/env python3
"""Section-aware unlinked-occurrence report for the heroes doc.

For the ambiguity-heavy distinctive-ability / heroic-resource tail (FOLLOWUPS #6
part C), a flat term count is misleading: most occurrences sit *inside* the term's
own class section (the definition site, which must not self-link). This report
buckets each unlinked occurrence by the class section it falls in, so we can see
the genuinely-linkable cross-section yield.

Usage: link_audit_sectioned.py '<term>' [<term> ...]
  Each <term> is matched case-insensitively as a whole word (with plural/possessive
  variants). Prints, per term, occurrences grouped by enclosing class (or "—no class—").
"""
import re, sys

DOC = "/home/scott/code/steelCompendium/workspace/steel-etl/input/heroes/Draw Steel Heroes.md"

# class section boundaries (start line of each @type: class annotation), document order.
# The classes chapter ends at the Kits chapter (~17607).
CLASS_BOUNDS = [
    ("censor", 4655), ("conduit", 6115), ("elementalist", 7910), ("fury", 9467),
    ("null", 11057), ("shadow", 12348), ("tactician", 13632), ("talent", 14794),
    ("troubadour", 16339), ("__end__", 17607),
]
def class_of(ln):
    cur = None
    for name, start in CLASS_BOUNDS:
        if ln >= start:
            cur = name
        else:
            break
    if cur == "__end__":
        return None
    # before the first class = not in a class section
    if ln < CLASS_BOUNDS[0][1]:
        return None
    return cur

lines = open(DOC, encoding="utf-8").read().split("\n")
LINKSPAN = re.compile(r"\[[^\]]*\]\(scc:[^)]*\)")
COMMENT = re.compile(r"<!--.*?-->")
HEADING = re.compile(r"^\s*(>\s*)*#{1,6}\s")
def mask(line):
    line = LINKSPAN.sub(lambda m: " " * len(m.group(0)), line)
    line = COMMENT.sub(lambda m: " " * len(m.group(0)), line)
    return line

def variants(name):
    vs = {name, name + "s", name + "'s", name + "’s"}
    if name.endswith("y"):
        vs.add(name[:-1] + "ies")
    return vs

for term in sys.argv[1:]:
    alts = sorted((re.escape(v) for v in variants(term)), key=len, reverse=True)
    rx = re.compile(r"(?<![\w-])(" + "|".join(alts) + r")(?![\w-])", re.I)
    buckets = {}
    for i, raw in enumerate(lines, 1):
        if HEADING.match(raw):
            continue
        masked = mask(raw)
        for m in rx.finditer(masked):
            cls = class_of(i) or "—no class—"
            buckets.setdefault(cls, []).append((i, m.group(0), raw.strip()[:150]))
    total = sum(len(v) for v in buckets.values())
    print(f"\n{'='*70}\nTERM '{term}'  ({total} unlinked occ)\n{'='*70}")
    for cls in sorted(buckets, key=lambda c: -len(buckets[c])):
        print(f"  [{cls}] {len(buckets[cls])}")
    # detail mode: env DETAIL=<classname-to-treat-as-own> prints everything else
    own = sys.argv[0]  # placeholder
    import os
    detail_own = os.environ.get("OWN")
    if detail_own is not None:
        print(f"  --- out-of-class detail (own='{detail_own}') ---")
        for cls in sorted(buckets):
            if cls == detail_own:
                continue
            for (ln, mt, ctx) in buckets[cls]:
                print(f"    [{cls}] L{ln}: …{ctx}…")
