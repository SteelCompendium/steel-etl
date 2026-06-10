# pdf-extract — fidelity-checked supplement conversion

Deterministic, font-aware text extraction for converting Draw Steel supplement
PDFs into annotated-markdown source with **machine-proven word-for-word fidelity**.

The publisher's words come only from a deterministic extractor — no model reads
prose off the page. The AI structures and annotates that extracted text into our
form; it never retypes a word. A word-multiset gate then proves the markdown
contains exactly the publisher's words (none added, dropped, or changed).

**The source PDF is never committed.** Pass its path on the command line. `out/`
is gitignored.

## Pipeline
```bash
# one-time
devbox run -- bash -c "cd steel-etl/tools/pdf-extract && pip install -r requirements.txt"

# 1. Extract (font-aware, reading order). Glyphs -> ⟦g:0xNN⟧ placeholders; drop
#    caps joined; page-number footers stripped. Writes pages/NNN.txt + glyphs-found.json.
devbox run -- bash -c "cd steel-etl/tools/pdf-extract && \
  python3 extract.py '/path/to/Supplement.pdf' --out out/<book>"

# 2. (First time for a new publisher PDF) build/verify glyphs.json from
#    out/<book>/glyphs-found.json. See GLYPHS.md for the annotated map.

# 3. Build the canonical wordbag (resolves glyphs: characteristic icons join
#    words, tiers/potency -> text, decoratives dropped). --range LO-HI optional.
devbox run -- bash -c "cd steel-etl/tools/pdf-extract && \
  python3 build_wordbag.py --pages out/<book>/pages --glyphs glyphs.json \
    --out out/<book>/wordbag.json"

# 4. Convert: structure + annotate out/<book>/pages/*.txt into the book .md
#    (the AI step — words verbatim, render glyphs per glyphs.json).

# 5. Gate: prove no word was added/removed/changed.
devbox run -- bash -c "cd steel-etl/tools/pdf-extract && \
  python3 fidelity_check.py --markdown '../../input/<book>/Draw Steel <Book>.md' \
    --wordbag out/<book>/wordbag.json"
```
`FIDELITY OK` (exit 0) = every publisher word is present and unchanged.
Mismatch prints the MISSING / EXTRA word lists and exits non-zero.

## Files
- `extract.py` — font-aware, reading-order extraction (glyph tagging, drop-cap join, footer strip)
- `glyphs.json` / `GLYPHS.md` — glyph codepoint → text map + annotated reference
- `resolve.py` — applies the glyph map to extracted text
- `build_wordbag.py` — builds the glyph-resolved publisher wordbag (supports `--range`)
- `normalize.py` — text → normalized word-multiset (shared by gate + wordbag)
- `fidelity_check.py` — the gate: count-exact word-multiset diff
- `tests/` — pytest unit tests for normalize, resolve, and the gate
