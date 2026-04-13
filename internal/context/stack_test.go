package context

import (
	"testing"
)

func TestNewContextStack(t *testing.T) {
	doc := Metadata{"book": "mcdm.heroes.v1", "source": "MCDM"}
	s := NewContextStack(doc)

	if s.Document()["book"] != "mcdm.heroes.v1" {
		t.Errorf("expected book=mcdm.heroes.v1, got %s", s.Document()["book"])
	}
}

func TestPushAndCurrent(t *testing.T) {
	s := NewContextStack(Metadata{})
	s.Push(2, Metadata{"type": "class", "id": "fury"})

	cur := s.Current(2)
	if cur == nil {
		t.Fatal("expected metadata at level 2, got nil")
	}
	if cur["id"] != "fury" {
		t.Errorf("expected id=fury, got %s", cur["id"])
	}
}

func TestPushClearsDeeperLevels(t *testing.T) {
	s := NewContextStack(Metadata{})
	s.Push(2, Metadata{"type": "class", "id": "fury"})
	s.Push(3, Metadata{"type": "feature-group", "level": "1"})
	s.Push(4, Metadata{"type": "ability", "id": "gouge"})

	// Push at level 3 should clear level 3+ and set new value
	s.Push(3, Metadata{"type": "feature-group", "level": "2"})

	if s.Current(4) != nil {
		t.Error("expected level 4 to be cleared after push at level 3")
	}
	if s.Current(3)["level"] != "2" {
		t.Errorf("expected level=2 at H3, got %s", s.Current(3)["level"])
	}
	if s.Current(2)["id"] != "fury" {
		t.Error("expected level 2 to be preserved")
	}
}

func TestPushAtSameLevelReplaces(t *testing.T) {
	s := NewContextStack(Metadata{})
	s.Push(2, Metadata{"type": "class", "id": "fury"})
	s.Push(2, Metadata{"type": "class", "id": "shadow"})

	if s.Current(2)["id"] != "shadow" {
		t.Errorf("expected id=shadow, got %s", s.Current(2)["id"])
	}
}

func TestLookupWalksUp(t *testing.T) {
	doc := Metadata{"book": "mcdm.heroes.v1"}
	s := NewContextStack(doc)
	s.Push(1, Metadata{"type": "chapter", "id": "classes"})
	s.Push(2, Metadata{"type": "class", "id": "fury"})
	s.Push(3, Metadata{"type": "feature-group", "level": "1"})
	s.Push(4, Metadata{"type": "ability", "id": "gouge"})

	// Lookup "id" from level 4 should find "gouge" at level 4
	val, ok := s.Lookup(4, "id")
	if !ok || val != "gouge" {
		t.Errorf("expected id=gouge from level 4, got %s (ok=%v)", val, ok)
	}

	// Lookup "level" from level 4 should find "1" at level 3
	val, ok = s.Lookup(4, "level")
	if !ok || val != "1" {
		t.Errorf("expected level=1, got %s (ok=%v)", val, ok)
	}

	// Lookup "book" from level 4 should find it at document level
	val, ok = s.Lookup(4, "book")
	if !ok || val != "mcdm.heroes.v1" {
		t.Errorf("expected book=mcdm.heroes.v1, got %s (ok=%v)", val, ok)
	}

	// Lookup nonexistent key
	_, ok = s.Lookup(4, "nonexistent")
	if ok {
		t.Error("expected ok=false for nonexistent key")
	}
}

func TestLookupSkipsNilLevels(t *testing.T) {
	doc := Metadata{"book": "mcdm.heroes.v1"}
	s := NewContextStack(doc)
	s.Push(1, Metadata{"type": "chapter"})
	// Level 2 is nil (no annotation at H2)
	s.Push(3, Metadata{"type": "feature-group", "level": "1"})

	val, ok := s.Lookup(3, "type")
	if !ok || val != "feature-group" {
		t.Errorf("expected type=feature-group, got %s", val)
	}
}

func TestAncestorsOfType(t *testing.T) {
	s := NewContextStack(Metadata{"book": "mcdm.heroes.v1"})
	s.Push(1, Metadata{"type": "chapter", "id": "classes"})
	s.Push(2, Metadata{"type": "class", "id": "fury"})
	s.Push(3, Metadata{"type": "feature-group", "level": "1"})
	s.Push(4, Metadata{"type": "ability", "id": "gouge"})

	ancestors := s.AncestorsOfType(4)
	if len(ancestors) != 3 {
		t.Fatalf("expected 3 ancestors, got %d", len(ancestors))
	}
	if ancestors[0]["type"] != "chapter" {
		t.Errorf("expected first ancestor type=chapter, got %s", ancestors[0]["type"])
	}
	if ancestors[1]["type"] != "class" {
		t.Errorf("expected second ancestor type=class, got %s", ancestors[1]["type"])
	}
	if ancestors[2]["type"] != "feature-group" {
		t.Errorf("expected third ancestor type=feature-group, got %s", ancestors[2]["type"])
	}
}

func TestAncestorsOfTypeSkipsNilLevels(t *testing.T) {
	s := NewContextStack(Metadata{})
	s.Push(1, Metadata{"type": "chapter"})
	// Level 2 is nil
	s.Push(3, Metadata{"type": "feature-group"})

	ancestors := s.AncestorsOfType(3)
	if len(ancestors) != 1 {
		t.Fatalf("expected 1 ancestor, got %d", len(ancestors))
	}
	if ancestors[0]["type"] != "chapter" {
		t.Errorf("expected ancestor type=chapter, got %s", ancestors[0]["type"])
	}
}

func TestCurrentOutOfRange(t *testing.T) {
	s := NewContextStack(Metadata{})
	if s.Current(0) != nil {
		t.Error("expected nil for level 0")
	}
	if s.Current(7) != nil {
		t.Error("expected nil for level 7")
	}
}

func TestPushOutOfRange(t *testing.T) {
	s := NewContextStack(Metadata{})
	// Should not panic
	s.Push(0, Metadata{"type": "bad"})
	s.Push(7, Metadata{"type": "bad"})
}

func TestFullWalkthrough(t *testing.T) {
	// Simulates the example from context-stack.md
	doc := Metadata{"book": "mcdm.heroes.v1"}
	s := NewContextStack(doc)

	// # Classes (H1)
	s.Push(1, Metadata{"type": "chapter", "id": "classes"})

	// ## Fury (H2)
	s.Push(2, Metadata{"type": "class", "id": "fury"})

	// ### 1st-Level Features (H3)
	s.Push(3, Metadata{"type": "feature-group", "level": "1"})

	// #### Gouge (H4)
	s.Push(4, Metadata{"type": "ability", "id": "gouge", "cost": "3 Ferocity"})

	// Verify Gouge context
	val, _ := s.Lookup(4, "id")
	if val != "gouge" {
		t.Errorf("at Gouge: expected id=gouge, got %s", val)
	}
	val, _ = s.Lookup(4, "level")
	if val != "1" {
		t.Errorf("at Gouge: expected level=1, got %s", val)
	}

	// #### Blood for Blood! (H4) - replaces Gouge at same level
	s.Push(4, Metadata{"type": "ability", "id": "blood-for-blood", "cost": "5 Ferocity"})
	val, _ = s.Lookup(4, "id")
	if val != "blood-for-blood" {
		t.Errorf("at Blood for Blood: expected id=blood-for-blood, got %s", val)
	}
	// Level should still be 1
	val, _ = s.Lookup(4, "level")
	if val != "1" {
		t.Errorf("at Blood for Blood: expected level=1, got %s", val)
	}

	// ### 2nd-Level Features (H3) - clears H4+
	s.Push(3, Metadata{"type": "feature-group", "level": "2"})
	if s.Current(4) != nil {
		t.Error("expected H4 cleared after H3 push")
	}

	// #### Savage Rush (H4)
	s.Push(4, Metadata{"type": "ability", "id": "savage-rush", "cost": "5 Ferocity"})
	val, _ = s.Lookup(4, "level")
	if val != "2" {
		t.Errorf("at Savage Rush: expected level=2, got %s", val)
	}

	// ## Shadow (H2) - clears H3-H6
	s.Push(2, Metadata{"type": "class", "id": "shadow"})
	if s.Current(3) != nil {
		t.Error("expected H3 cleared after H2 push")
	}
	if s.Current(4) != nil {
		t.Error("expected H4 cleared after H2 push")
	}
	val, _ = s.Lookup(2, "id")
	if val != "shadow" {
		t.Errorf("at Shadow: expected id=shadow, got %s", val)
	}
}
