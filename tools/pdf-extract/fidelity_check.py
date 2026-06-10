"""Fidelity gate: prove the annotated markdown contains exactly the publisher's
words. Order-insensitive (multiset) but count-exact, so any added, dropped, or
altered word is reported. Exit 0 = clean, 1 = mismatch.
"""
import argparse
import json
import sys
from collections import Counter
from dataclasses import dataclass

import normalize


@dataclass
class Result:
    ok: bool
    missing: Counter   # in publisher, not in markdown (dropped/changed)
    extra: Counter     # in markdown, not in publisher (added/hallucinated/changed)


def compare(markdown_text: str, publisher_bag: Counter) -> Result:
    md_bag = normalize.wordbag_from_markdown(markdown_text)
    missing = publisher_bag - md_bag
    extra = md_bag - publisher_bag
    return Result(ok=(not missing and not extra), missing=missing, extra=extra)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--markdown", required=True, help="annotated .md file")
    ap.add_argument("--wordbag", required=True, help="publisher wordbag.json")
    args = ap.parse_args()

    with open(args.markdown) as f:
        md = f.read()
    with open(args.wordbag) as f:
        pub = Counter(json.load(f))

    r = compare(md, pub)
    if r.ok:
        print("FIDELITY OK — every publisher word present and unchanged.")
        sys.exit(0)
    if r.missing:
        print(f"MISSING ({sum(r.missing.values())} word-instances dropped or altered):")
        for w, n in r.missing.most_common(200):
            print(f"  -{n:>4}  {w}")
    if r.extra:
        print(f"EXTRA ({sum(r.extra.values())} word-instances added or altered):")
        for w, n in r.extra.most_common(200):
            print(f"  +{n:>4}  {w}")
    sys.exit(1)


if __name__ == "__main__":
    main()
