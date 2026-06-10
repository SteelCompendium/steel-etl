from collections import Counter
import normalize
import resolve

# A minimal in-test glyph map mirroring the real glyphs.json semantics.
_MAP = (
    ("g:0x4d", "M"),
    ("g:0x52", "R"),
    ("g:0x3c", "<"),
    ("g:0x76", "average"),
    ("g:0xe1", "≤11"),
    ("g:0x32", "2"),
    ("g:0xa2", ""),
)


def test_characteristic_glyph_joins_following_word():
    # ⟦M⟧ight -> Might (no space inserted)
    assert resolve.resolve("⟦g:0x4d⟧ight", _MAP) == "Might"


def test_characteristic_glyph_standalone_in_formula():
    assert resolve.resolve("2 + ⟦g:0x52⟧ damage", _MAP) == "2 + R damage"


def test_potency_strength_resolves_to_word():
    assert resolve.resolve("⟦g:0x4d⟧⟦g:0x3c⟧⟦g:0x76⟧ slowed", _MAP) == "M<average slowed"


def test_numeric_potency_resolves_to_digit():
    assert resolve.resolve("⟦g:0x52⟧⟦g:0x3c⟧⟦g:0x32⟧ bleeding", _MAP) == "R<2 bleeding"


def test_tier_marker_resolves():
    assert resolve.resolve("⟦g:0xe1⟧ Three creatures", _MAP) == "≤11 Three creatures"


def test_decorative_glyph_dropped():
    assert resolve.resolve("⟦g:0xa2⟧Distraction", _MAP) == "Distraction"


def test_unknown_glyph_dropped():
    assert resolve.resolve("a ⟦g:0xff⟧ b", _MAP) == "a  b"


def test_resolved_text_feeds_wordbag_correctly():
    # The end-to-end point: "Might" becomes a real word token, not "ight".
    resolved = resolve.resolve("⟦g:0x4d⟧ight ⟦g:0x4d⟧⟦g:0x3c⟧⟦g:0x76⟧ slowed", _MAP)
    assert normalize.wordbag(resolved) == Counter(["might", "m", "average", "slowed"])
