"""Font-aware, reading-order text extraction for Draw Steel supplement PDFs.

Real-text characters are emitted verbatim. Characters rendered in a custom glyph
font (DrawSteelGlyphs, Wingdings) are emitted as ⟦g:0xNN⟧ / ⟦w:0xNN⟧ placeholders
so they can never masquerade as prose. Reading order sorts text boxes by column
then vertical position to handle the two-column layout.

Two layout-furniture fixes keep the per-page text clean for the fidelity gate:
- **Drop caps:** an oversized decorative initial (e.g. a giant "T" before "he…")
  is joined to the following word so it reads "The", not "T he".
- **Page-number footers:** a digits-only box in the bottom page margin is dropped.

Emits per-page text to pages/NNN.txt and the distinct glyph inventory to
glyphs-found.json. The canonical fidelity wordbag is built separately by
build_wordbag.py (which resolves glyphs via glyphs.json).
"""
import argparse
import json
import os
import statistics
from collections import Counter
from pdfminer.high_level import extract_pages
from pdfminer.layout import LTTextContainer, LTTextLine, LTChar

GLYPH_FONTS = ("DrawSteelGlyphs", "Wingdings")
DROPCAP_RATIO = 1.5   # a leading char this much larger than the line median is a drop cap
FOOTER_Y = 55         # boxes with top edge below this (page bottom margin) are furniture


def _is_glyph(fontname: str):
    for g in GLYPH_FONTS:
        if g in fontname:
            return "w" if "Wingding" in fontname else "g"
    return None


def _line_text(line, glyphs_found):
    """Emit a line's text, tagging glyphs and joining oversized drop-cap initials."""
    chars = list(line)
    body_sizes = [
        c.size for c in chars
        if isinstance(c, LTChar) and c.get_text().strip() and not _is_glyph(c.fontname)
    ]
    median = statistics.median(body_sizes) if body_sizes else 0.0

    s = []
    skip_space = False
    for c in chars:
        text = c.get_text()
        if not isinstance(c, LTChar):
            if skip_space and text == " ":
                skip_space = False
                continue
            s.append(text)
            skip_space = False
            continue
        kind = _is_glyph(c.fontname)
        if kind:
            cp = ord(text) if len(text) == 1 else 0
            s.append(f"⟦{kind}:{hex(cp)}⟧")
            glyphs_found[(kind, hex(cp))] += 1
            skip_space = False
            continue
        # Drop cap: oversized alphabetic initial -> join to the next word.
        if median and text.isalpha() and c.size > DROPCAP_RATIO * median:
            s.append(text)
            skip_space = True
            continue
        if skip_space and text == " ":
            skip_space = False
            continue
        s.append(text)
        skip_space = False
    return "".join(s)


def _is_footer(line) -> bool:
    """A digits-only box sitting in the bottom page margin is a page-number footer."""
    raw = "".join(ch.get_text() for ch in line).strip()
    return raw.isdigit() and line.bbox[3] < FOOTER_Y


def _page_text(layout, page_width, glyphs_found):
    """Reading order: split into left/right column by box midpoint, sort by -y."""
    boxes = [el for el in layout if isinstance(el, LTTextContainer)]
    mid = page_width / 2
    left = sorted([b for b in boxes if (b.x0 + b.x1) / 2 < mid], key=lambda b: -b.y1)
    right = sorted([b for b in boxes if (b.x0 + b.x1) / 2 >= mid], key=lambda b: -b.y1)
    lines = []
    for b in left + right:
        for line in b:
            if not isinstance(line, LTTextLine):
                continue
            if _is_footer(line):
                continue
            t = _line_text(line, glyphs_found).rstrip()
            if t.strip():
                lines.append(t)
    return "\n".join(lines)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("pdf")
    ap.add_argument("--out", required=True)
    args = ap.parse_args()

    pages_dir = os.path.join(args.out, "pages")
    os.makedirs(pages_dir, exist_ok=True)
    glyphs_found = Counter()

    pi = -1
    for pi, layout in enumerate(extract_pages(args.pdf)):
        text = _page_text(layout, layout.width, glyphs_found)
        with open(os.path.join(pages_dir, f"{pi:03d}.txt"), "w") as f:
            f.write(text)

    with open(os.path.join(args.out, "glyphs-found.json"), "w") as f:
        json.dump(
            sorted([{"kind": k, "cp": cp, "count": n} for (k, cp), n in glyphs_found.items()],
                   key=lambda d: -d["count"]),
            f, indent=2,
        )
    print(f"pages: {pi + 1}  distinct glyphs: {len(glyphs_found)}  "
          f"-> {pages_dir}/ + glyphs-found.json (build wordbag with build_wordbag.py)")


if __name__ == "__main__":
    main()
