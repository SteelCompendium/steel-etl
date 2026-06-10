from collections import Counter
import normalize


def test_lowercases_and_splits_on_whitespace():
    assert normalize.wordbag("The Summoner Calls") == Counter(["the", "summoner", "calls"])


def test_strips_surrounding_punctuation_but_keeps_internal():
    # quotes/commas/periods dropped; intra-word hyphen/apostrophe kept
    assert normalize.wordbag('"rock-solid," she said.') == Counter(
        ["rock-solid", "she", "said"]
    )


def test_dehyphenates_line_break_splits():
    # a word split across a column line-break must rejoin to one token
    assert normalize.wordbag("rock-\nsolid take") == Counter(["rocksolid", "take"])


def test_normalizes_smart_quotes_and_dashes():
    # curly quotes -> straight, em/en dash -> space-separated, so tokens are stable
    assert normalize.wordbag("don't — stop") == Counter(["don't", "stop"])


def test_drops_glyph_placeholders():
    assert normalize.wordbag("inflict ⟦g:0x69⟧ taunted") == Counter(
        ["inflict", "taunted"]
    )


def test_keeps_numbers_as_tokens():
    assert normalize.wordbag("5 Essence, 12-16 damage") == Counter(
        ["5", "essence", "12-16", "damage"]
    )


def test_strips_markdown_and_scc_links():
    md = "**Effect:** the [taunted](scc:mcdm.x/condition/taunted) foe"
    assert normalize.wordbag_from_markdown(md) == Counter(
        ["effect", "the", "taunted", "foe"]
    )
