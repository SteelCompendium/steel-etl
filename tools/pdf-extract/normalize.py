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
# Leading YAML frontmatter block (book/source/title metadata, not publisher prose)
_FRONTMATTER = re.compile(r"\A---\r?\n.*?\r?\n---\r?\n", re.DOTALL)
# HTML annotation comments <!-- @type: ... -->
_COMMENT = re.compile(r"<!--.*?-->", re.DOTALL)
# Real HTML tags (e.g. <br>, <br/>, <sup>) used as markup in tables — NOT publisher
# words. Deliberately requires a letter or "/" right after "<" so the potency
# notation "M < WEAK" (space after "<") is never matched.
_HTML_TAG = re.compile(r"</?[a-zA-Z][a-zA-Z0-9]*\s*/?>")
# markdown emphasis/heading/table/pipe punctuation to strip to spaces
_MD_PUNCT = re.compile(r"[#>*_`|]+")
_LINE_HYPHEN = re.compile(r"-\n")           # hyphenated line-break -> join
# A hyphen between two letters is ambiguous across a line break: the publisher's
# "tear-stained" can wrap as "tear-\nstained" (keep hyphen) while a syllable break
# like "rock-\nsolid" drops it. Text alone can't disambiguate, so we make
# letter-letter hyphens irrelevant to the comparison by removing them on BOTH
# sides ("tear-stained" == "tearstained"). Numeric ranges like "12-16" keep their
# hyphen (digits, not letters), so power-roll tiers stay intact.
_LETTER_HYPHEN = re.compile(r"(?<=[a-z])-(?=[a-z])")
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
    text = text.lower()
    text = _LETTER_HYPHEN.sub("", text)     # tear-stained -> tearstained (both sides)
    return text


def wordbag(text: str) -> Counter:
    text = _GLYPH.sub(" ", text)
    text = _canon(text)
    return Counter(_TOKEN.findall(text))


def wordbag_from_markdown(md: str) -> Counter:
    md = _FRONTMATTER.sub(" ", md)
    md = _COMMENT.sub(" ", md)
    md = _HTML_TAG.sub(" ", md)
    md = _SCC_LINK.sub(r"\1", md)
    md = _MD_LINK.sub(r"\1", md)
    md = _MD_PUNCT.sub(" ", md)
    return wordbag(md)
