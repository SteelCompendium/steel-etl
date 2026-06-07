#!/usr/bin/env python3
"""Targeted unlinked-occurrence report for specific SCC code categories."""
import json, re, os, sys
from collections import defaultdict

ETL = "/home/scott/code/steelCompendium/workspace/steel-etl"
DOC = os.path.join(ETL, "input/heroes/Draw Steel Heroes.md")
MDDIR = "/home/scott/code/steelCompendium/workspace/data/data-rules/en/md"

code2name = {}
for root, _, files in os.walk(MDDIR):
    for f in files:
        if not f.endswith(".md"): continue
        txt = open(os.path.join(root, f), encoding="utf-8").read()
        m = re.match(r"^---\n(.*?)\n---", txt, re.S)
        if not m: continue
        fm = m.group(1)
        nm = re.search(r"^name:\s*(.+)$", fm, re.M)
        sc = re.search(r"^scc:\s*(\S+)$", fm, re.M)
        if not (nm and sc): continue
        name = nm.group(1).strip().strip('"').strip("'")
        code = sc.group(1).strip()
        if code.startswith("mcdm.heroes.v1/"):
            code2name[code] = name

# category filter from argv: substring(s) the code body must match
PATS = sys.argv[1:]
def codebody(code): return code[len("mcdm.heroes.v1/"):]
def match_cat(code):
    b = codebody(code)
    return any(p in b for p in PATS)

# term -> codes (only selected category), with hyphen/space + plural variants
def variants(name):
    vs = {name, name+"s", name+"'s", name+"’s"}
    if name.endswith("y"): vs.add(name[:-1]+"ies")
    if "-" in name: vs.add(name.replace("-"," "))
    if " " in name: vs.add(name.replace(" ","-"))
    return vs
terms = defaultdict(set)
for code,name in code2name.items():
    if not match_cat(code): continue
    if len(name) < 4: continue
    for v in variants(name):
        terms[v.lower()] = terms[v.lower()] | {(name,code)}

lines = open(DOC, encoding="utf-8").read().split("\n")
LINKSPAN = re.compile(r"\[[^\]]*\]\(scc:[^)]*\)")
COMMENT = re.compile(r"<!--.*?-->")
HEADING = re.compile(r"^#{1,6}\s")
def mask(line):
    line = LINKSPAN.sub(lambda m:" "*len(m.group(0)), line)
    line = COMMENT.sub(lambda m:" "*len(m.group(0)), line)
    return line

alts = sorted((re.escape(t) for t in terms), key=len, reverse=True)
BIG = re.compile(r"(?<![\w-])("+"|".join(alts)+r")(?![\w-])", re.I)

occ = defaultdict(list)
for i,raw in enumerate(lines,1):
    if HEADING.match(raw): continue
    masked = mask(raw)
    for m in BIG.finditer(masked):
        lt = m.group(0).lower()
        if lt not in terms: continue
        for (name,code) in sorted(terms[lt]):
            occ[(name,code)].append((i, m.group(0), raw.strip()[:150]))

for (name,code),os_ in sorted(occ.items(), key=lambda kv:-len(kv[1])):
    print(f"\n### {name}  ->  {code}   ({len(os_)})")
    for (ln,mt,ctx) in os_:
        print(f"   L{ln}: …{ctx}…")
