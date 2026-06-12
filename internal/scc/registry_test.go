package scc

import (
	"os"
	"path/filepath"
	"strings"
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

	if got := r2.SchemeVersion(); got != 1 {
		t.Errorf("reloaded SchemeVersion = %d, want 1", got)
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

func TestRegistryBookPrintingsRoundTrip(t *testing.T) {
	r := NewRegistry()
	r.Add("mcdm.heroes.v1/class/fury")
	r.SetBookPrinting("mcdm.heroes.v1", "1.01b")
	r.SetBookPrinting("mcdm.heroes.v1", "1.02")
	r.SetBookPrinting("", "9.99")      // ignored: empty book
	r.SetBookPrinting("mcdm.x.v1", "") // ignored: empty printing

	path := filepath.Join(t.TempDir(), "classification.json")
	if err := r.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got := loaded.BookPrintings()
	if len(got) != 1 || got["mcdm.heroes.v1"] != "1.02" {
		t.Errorf("BookPrintings = %v, want map[mcdm.heroes.v1:1.02]", got)
	}
}

func TestRegistryBookPrintingsAbsent(t *testing.T) {
	// A registry without printings must round-trip cleanly and omit the books key.
	r := NewRegistry()
	r.Add("mcdm.heroes.v1/class/fury")
	path := filepath.Join(t.TempDir(), "classification.json")
	if err := r.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := loaded.BookPrintings(); len(got) != 0 {
		t.Errorf("BookPrintings = %v, want empty", got)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), `"books"`) {
		t.Error("books key written for registry with no printings")
	}
}

func TestRegistrySchemeVersion(t *testing.T) {
	// New registry defaults to scheme version 1.
	r := NewRegistry()
	if got := r.SchemeVersion(); got != 1 {
		t.Fatalf("NewRegistry SchemeVersion = %d, want 1", got)
	}

	// Save writes scheme_version, and a round-trip preserves it.
	dir := t.TempDir()
	path := filepath.Join(dir, "classification.json")
	r.Add("mcdm.heroes.v1/class/fury")
	if err := r.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(raw), `"scheme_version": 1`) {
		t.Errorf("saved registry missing scheme_version; got:\n%s", raw)
	}

	// Loading a file WITHOUT scheme_version defaults to 1 (backward compat).
	legacy := filepath.Join(dir, "legacy.json")
	if err := os.WriteFile(legacy, []byte(`{"version":1,"frozen":false,"codes":[]}`), 0644); err != nil {
		t.Fatalf("write legacy failed: %v", err)
	}
	lr, err := LoadRegistry(legacy)
	if err != nil {
		t.Fatalf("LoadRegistry(legacy) failed: %v", err)
	}
	if got := lr.SchemeVersion(); got != 1 {
		t.Errorf("legacy SchemeVersion = %d, want 1", got)
	}

	// Loading a file WITH scheme_version honors it.
	v2 := filepath.Join(dir, "v2.json")
	if err := os.WriteFile(v2, []byte(`{"version":1,"scheme_version":2,"frozen":false,"codes":[]}`), 0644); err != nil {
		t.Fatalf("write v2 failed: %v", err)
	}
	v2r, err := LoadRegistry(v2)
	if err != nil {
		t.Fatalf("LoadRegistry(v2) failed: %v", err)
	}
	if got := v2r.SchemeVersion(); got != 2 {
		t.Errorf("v2 SchemeVersion = %d, want 2", got)
	}
}
