# pdf-extract — fidelity-checked supplement conversion

Deterministic, font-aware text extraction for converting Draw Steel supplement
PDFs into annotated-markdown source with machine-proven word-for-word fidelity.

**The source PDF is never committed.** Pass its path on the command line.

## Pipeline
```bash
cd steel-etl/tools/pdf-extract
devbox run -- pip install -r requirements.txt          # one-time
# 1. Extract (font-aware, reading order). Glyphs become ⟦g:0xNN⟧ placeholders.
devbox run -- python3 extract.py "/path/to/Supplement.pdf" --out out/<book>
# 2. (First time for a new publisher PDF) build/verify glyphs.json from glyphs-found.json
# 3. Convert: structure + annotate out/<book>/pages/*.txt into the book .md (AI step)
# 4. Gate: prove no word was added/removed/changed
devbox run -- python3 fidelity_check.py \
  --markdown ../../input/<book>/Draw\ Steel\ <Book>.md \
  --wordbag out/<book>/wordbag.json
```
Exit 0 = every publisher word is present and unchanged. Non-zero = mismatch (printed).
