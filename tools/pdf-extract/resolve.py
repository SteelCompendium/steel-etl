"""Resolve glyph placeholders (⟦g:0xNN⟧ / ⟦w:0xNN⟧) to their semantic text.

Used to build the canonical fidelity wordbag: characteristic icons join their
adjacent word ("Might"), potency strength/value and power-roll tier markers
become their text, and decorative glyphs are dropped. The markdown converter
renders glyphs by the same map (glyphs.json), so the gate stays exact.

Unknown glyphs (not in the map) are dropped — treated as decorative. Inspect
out/<book>/glyphs-found.json when onboarding a new PDF to catch new codepoints.
"""
import json
import re
from functools import lru_cache

_TAG = re.compile(r"⟦([gw]:0x[0-9a-fA-F]+)⟧")


@lru_cache(maxsize=None)
def load_map(path: str) -> tuple:
    """Load glyphs.json as a hashable tuple of (key, value) pairs (skips _doc keys).

    Returned as a tuple so the lru_cache key is hashable; callers pass it to
    resolve() which rebuilds a dict. Small map, so the rebuild cost is trivial.
    """
    with open(path) as f:
        raw = json.load(f)
    return tuple((k, v) for k, v in raw.items() if not k.startswith("_"))


def resolve(text: str, glyph_map) -> str:
    """Replace every ⟦k:0xNN⟧ placeholder with its mapped text (or '' if unmapped)."""
    m = dict(glyph_map)
    return _TAG.sub(lambda mo: m.get(mo.group(1), ""), text)
