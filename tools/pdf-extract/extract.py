"""Font-aware, reading-order text extraction for Draw Steel supplement PDFs.

Real-text characters are emitted verbatim. Characters rendered in a custom glyph
font (DrawSteelGlyphs, Wingdings) are emitted as ⟦g:0xNN⟧ / ⟦w:0xNN⟧ placeholders
so they can never masquerade as prose. Reading order sorts text boxes by column
then vertical position to handle the two-column layout.
"""
import argparse
import json
import os
from collections import Counter
from pdfminer.high_level import extract_pages
from pdfminer.layout import LTTextContainer, LTTextLine, LTChar

GLYPH_FONTS = ("DrawSteelGlyphs", "Wingdings")


def _is_glyph(fontname: str):
    for g in GLYPH_FONTS:
        if g in fontname:
            return "w" if "Wingding" in fontname else "g"
    return None


def _line_text(line, glyphs_found):
    s = []
    for c in line:
        if not isinstance(c, LTChar):
            s.append(c.get_text())
            continue
        kind = _is_glyph(c.fontname)
        if kind:
            cp = ord(c.get_text()) if len(c.get_text()) == 1 else 0
            tag = f"⟦{kind}:{hex(cp)}⟧"
            s.append(tag)
            glyphs_found[(kind, hex(cp))] += 1
        else:
            s.append(c.get_text())
    return "".join(s)


def _page_text(layout, page_width, glyphs_found):
    """Reading order: split into left/right column by box midpoint, sort by -y."""
    boxes = [el for el in layout if isinstance(el, LTTextContainer)]
    mid = page_width / 2
    left = sorted([b for b in boxes if (b.x0 + b.x1) / 2 < mid], key=lambda b: -b.y1)
    right = sorted([b for b in boxes if (b.x0 + b.x1) / 2 >= mid], key=lambda b: -b.y1)
    lines = []
    for b in left + right:
        for line in b:
            if isinstance(line, LTTextLine):
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

    import normalize
    full_bag = Counter()
    for pi, layout in enumerate(extract_pages(args.pdf)):
        text = _page_text(layout, layout.width, glyphs_found)
        with open(os.path.join(pages_dir, f"{pi:03d}.txt"), "w") as f:
            f.write(text)
        full_bag += normalize.wordbag(text)

    with open(os.path.join(args.out, "glyphs-found.json"), "w") as f:
        json.dump(
            sorted([{"kind": k, "cp": cp, "count": n} for (k, cp), n in glyphs_found.items()],
                   key=lambda d: -d["count"]),
            f, indent=2,
        )
    with open(os.path.join(args.out, "wordbag.json"), "w") as f:
        json.dump(dict(full_bag), f, indent=2, ensure_ascii=False)
    print(f"pages: {pi + 1}  distinct glyphs: {len(glyphs_found)}  words: {sum(full_bag.values())}")


if __name__ == "__main__":
    main()
