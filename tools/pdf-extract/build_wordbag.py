"""Build the canonical fidelity wordbag from extracted pages, resolving glyphs.

The wordbag is the publisher's word-multiset that the fidelity gate checks the
converted markdown against. Glyphs are resolved (via glyphs.json) BEFORE
tokenizing so characteristic icons join their words ("Might", not "ight").

Usage:
  python3 build_wordbag.py --pages out/summoner/pages --glyphs glyphs.json \
      --out out/summoner/wordbag.json [--range 4-60]

--range restricts to inclusive page-index bounds (e.g. 4-8 for a slice). Page
files are named NNN.txt by zero-padded page index.
"""
import argparse
import glob
import json
import os
from collections import Counter

import normalize
import resolve


def build(pages_dir: str, glyphs_path: str, page_range: str | None) -> Counter:
    glyph_map = resolve.load_map(glyphs_path)
    files = sorted(glob.glob(os.path.join(pages_dir, "*.txt")))
    if page_range:
        lo, hi = (int(x) for x in page_range.split("-"))
        files = [f for f in files if lo <= int(os.path.basename(f)[:-4]) <= hi]
    bag = Counter()
    for f in files:
        with open(f) as fh:
            bag += normalize.wordbag(resolve.resolve(fh.read(), glyph_map))
    return bag, len(files)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--pages", required=True, help="dir of NNN.txt page files")
    ap.add_argument("--glyphs", required=True, help="glyphs.json")
    ap.add_argument("--out", required=True, help="wordbag.json output path")
    ap.add_argument("--range", dest="page_range", help="inclusive page range, e.g. 4-60")
    args = ap.parse_args()

    bag, n = build(args.pages, args.glyphs, args.page_range)
    with open(args.out, "w") as fh:
        json.dump(dict(bag), fh, indent=2, ensure_ascii=False)
    print(f"pages: {n}  words: {sum(bag.values())}  -> {args.out}")


if __name__ == "__main__":
    main()
