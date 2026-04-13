package scc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFrozen(t *testing.T) {
	r := NewRegistry()
	if r.IsFrozen() {
		t.Error("new registry should not be frozen")
	}

	r.Freeze()
	if !r.IsFrozen() {
		t.Error("registry should be frozen after Freeze()")
	}
}

func TestLoadRegistry_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	if err := os.WriteFile(path, []byte("not valid json{{{"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadRegistry(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadRegistry_NonexistentFile(t *testing.T) {
	_, err := LoadRegistry("/nonexistent/path/classification.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadRegistry_FrozenFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "frozen.json")

	// Save a frozen registry
	r := NewRegistry()
	r.Add("test/code")
	r.Freeze()
	if err := r.Save(path); err != nil {
		t.Fatal(err)
	}

	// Load and verify frozen flag
	loaded, err := LoadRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.IsFrozen() {
		t.Error("loaded registry should be frozen")
	}
}

func TestSaveRegistry_NoAliases(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noalias.json")

	r := NewRegistry()
	r.Add("test/code")
	// No aliases added

	if err := r.Save(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Should not contain aliases key (omitempty)
	content := string(data)
	if contains(content, "aliases") {
		t.Error("expected no aliases field when none are set")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSccToRelPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		scc  string
		ext  string
		want string
	}{
		{"single component", "nosource", ".md", "nosource"},
		{"two components", "source/type", ".md", "type.md"},
		{"normal three", "source/type/item", ".md", "type/item.md"},
		{"dotted type path", "source/a.b.c/item", ".md", "a/b/c/item.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sccToRelPath(tt.scc, tt.ext)
			if got != tt.want {
				t.Errorf("sccToRelPath(%q, %q) = %q, want %q", tt.scc, tt.ext, got, tt.want)
			}
		})
	}
}
