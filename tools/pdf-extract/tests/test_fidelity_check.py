from collections import Counter
import fidelity_check as fc


def _bag(d):
    return Counter(d)


def test_passes_when_words_match():
    md = "**Effect:** the foe is taunted"
    pub = _bag({"effect": 1, "the": 1, "foe": 1, "is": 1, "taunted": 1})
    result = fc.compare(md, pub)
    assert result.ok
    assert result.missing == Counter()
    assert result.extra == Counter()


def test_flags_dropped_word():
    md = "the foe taunted"          # 'is' dropped vs publisher
    pub = _bag({"the": 1, "foe": 1, "is": 1, "taunted": 1})
    result = fc.compare(md, pub)
    assert not result.ok
    assert result.missing == Counter({"is": 1})


def test_flags_added_or_changed_word():
    md = "the brave foe taunted"    # 'brave' hallucinated; 'is' missing
    pub = _bag({"the": 1, "foe": 1, "is": 1, "taunted": 1})
    result = fc.compare(md, pub)
    assert not result.ok
    assert result.extra == Counter({"brave": 1})
    assert result.missing == Counter({"is": 1})


def test_counts_matter_not_just_presence():
    md = "fire fire fire"           # publisher had it twice
    pub = _bag({"fire": 2})
    result = fc.compare(md, pub)
    assert not result.ok
    assert result.extra == Counter({"fire": 1})
