"""Reduce text to a normalized word-multiset for the fidelity gate.

The multiset is order-insensitive (so column de-interleaving by the AI cannot
cause false mismatches) but counts every word, so a dropped/added/changed word
always changes the multiset.
"""
import re
from collections import Counter

# ⟦g:0xNN⟧ glyph placeholders emitted by extract.py
_GLYPH = re.compile(r"⟦[gw]:0x[0-9a-fA-F]+⟧")
# scc: link targets inside markdown: [text](scc:...) -> keep text, drop target
_SCC_LINK = re.compile(r"\[([^\]]*)\]\(scc:[^)]*\)")
_MD_LINK = re.compile(r"\[([^\]]*)\]\([^)]*\)")
# HTML annotation comments <!-- @type: ... -->
_COMMENT = re.compile(r"<!--.*?-->", re.DOTALL)
# markdown emphasis/heading/table/pipe punctuation to strip to spaces
_MD_PUNCT = re.compile(r"[#>*_`|]+")
_LINE_HYPHEN = re.compile(r"-\n")           # hyphenated line-break -> join
_SMART = {
    "‘": "'", "’": "'", "“": '"', "”": '"',
    "—": " ", "–": " ", "…": " ",
    "®": " ", "™": " ",            # ® ™ are layout marks, not words
}
# keep letters/digits plus internal ' and - ; drop everything else
_TOKEN = re.compile(r"[0-9a-z]+(?:['-][0-9a-z]+)*")


def _canon(text: str) -> str:
    for k, v in _SMART.items():
        text = text.replace(k, v)
    text = _LINE_HYPHEN.sub("", text)       # rock-\nsolid -> rocksolid
    return text.lower()


def wordbag(text: str) -> Counter:
    text = _GLYPH.sub(" ", text)
    text = _canon(text)
    return Counter(_TOKEN.findall(text))


def wordbag_from_markdown(md: str) -> Counter:
    md = _COMMENT.sub(" ", md)
    md = _SCC_LINK.sub(r"\1", md)
    md = _MD_LINK.sub(r"\1", md)
    md = _MD_PUNCT.sub(" ", md)
    return wordbag(md)
