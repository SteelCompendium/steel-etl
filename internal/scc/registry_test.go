package scc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryAddAndContains(t *testing.T) {
	r := NewRegistry()
	r.Add("mcdm.heroes.v1/classes/fury")
	r.Add("mcdm.heroes.v1/abilities.fury/gouge")

	if !r.Contains("mcdm.heroes.v1/classes/fury") {
		t.Error("expected registry to contain fury")
	}
	if !r.Contains("mcdm.heroes.v1/abilities.fury/gouge") {
		t.Error("expected registry to contain gouge")
	}
	if r.Contains("mcdm.heroes.v1/classes/shadow") {
		t.Error("expected registry to NOT contain shadow")
	}
}

func TestRegistryAddAlias(t *testing.T) {
	r := NewRegistry()
	r.AddAlias("mcdm.heroes.v1/abilities.common/reactive-strike", "mcdm.heroes.v1/abilities.fury/reactive-strike")

	target, ok := r.ResolveAlias("mcdm.heroes.v1/abilities.common/reactive-strike")
	if !ok || target != "mcdm.heroes.v1/abilities.fury/reactive-strike" {
		t.Errorf("expected alias to resolve, got %s (ok=%v)", target, ok)
	}
}

func TestRegistryFreezeEnforcement(t *testing.T) {
	r := NewRegistry()
	r.Add("mcdm.heroes.v1/classes/fury")
	r.Add("mcdm.heroes.v1/classes/shadow")
	r.Freeze()

	// New codes can still be added
	r.Add("mcdm.heroes.v1/classes/tactician")
	if !r.Contains("mcdm.heroes.v1/classes/tactician") {
		t.Error("should allow new codes after freeze")
	}

	// But check validates no removals
	subset := NewRegistry()
	subset.Add("mcdm.heroes.v1/classes/fury")
	// shadow is missing

	err := subset.ValidateAgainstFrozen(r)
	if err == nil {
		t.Error("expected validation error for missing code")
	}
}

func TestRegistryFreezeAllPresent(t *testing.T) {
	r := NewRegistry()
	r.Add("mcdm.heroes.v1/classes/fury")
	r.Freeze()

	newR := NewRegistry()
	newR.Add("mcdm.heroes.v1/classes/fury")
	newR.Add("mcdm.heroes.v1/classes/shadow") // new code OK

	err := newR.ValidateAgainstFrozen(r)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRegistrySaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "classification.json")

	// Save
	r := NewRegistry()
	r.Add("mcdm.heroes.v1/classes/fury")
	r.Add("mcdm.heroes.v1/abilities.fury/gouge")
	r.AddAlias("alias1", "target1")

	err := r.Save(path)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	r2, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !r2.Contains("mcdm.heroes.v1/classes/fury") {
		t.Error("expected loaded registry to contain fury")
	}
	if !r2.Contains("mcdm.heroes.v1/abilities.fury/gouge") {
		t.Error("expected loaded registry to contain gouge")
	}

	target, ok := r2.ResolveAlias("alias1")
	if !ok || target != "target1" {
		t.Errorf("expected alias to resolve, got %s", target)
	}

	// Verify file format
	data, _ := os.ReadFile(path)
	t.Logf("File contents:\n%s", string(data))
}

func TestRegistryCodes(t *testing.T) {
	r := NewRegistry()
	r.Add("b/c")
	r.Add("a/b")
	r.Add("c/d")

	codes := r.Codes()
	if len(codes) != 3 {
		t.Fatalf("expected 3 codes, got %d", len(codes))
	}
	// Should be sorted
	if codes[0] != "a/b" || codes[1] != "b/c" || codes[2] != "c/d" {
		t.Errorf("expected sorted codes, got %v", codes)
	}
}
