package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/pipeline"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

func writeTestRegistry(t *testing.T, path string, frozen bool) {
	t.Helper()
	r := scc.NewRegistry()
	r.Add("mcdm.heroes.v1/skill.group/crafting")
	if frozen {
		r.Freeze()
	}
	if err := r.Save(path); err != nil {
		t.Fatalf("save registry: %v", err)
	}
}

func TestResetRegistryForRebuild_UnfrozenRemoved(t *testing.T) {
	dir := t.TempDir()
	regPath := filepath.Join(dir, "classification.json")
	writeTestRegistry(t, regPath, false)

	cfg := &pipeline.Config{ConfigDir: dir}
	cfg.Classification.Registry = "classification.json"

	if err := resetRegistryForRebuild(cfg); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := os.Stat(regPath); !os.IsNotExist(err) {
		t.Errorf("expected unfrozen registry to be removed, stat err=%v", err)
	}
}

func TestResetRegistryForRebuild_FrozenPreserved(t *testing.T) {
	dir := t.TempDir()
	regPath := filepath.Join(dir, "classification.json")
	writeTestRegistry(t, regPath, true)

	cfg := &pipeline.Config{ConfigDir: dir}
	cfg.Classification.Registry = "classification.json"

	if err := resetRegistryForRebuild(cfg); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := os.Stat(regPath); err != nil {
		t.Errorf("expected frozen registry to be preserved, stat err=%v", err)
	}
}

func TestResetRegistryForRebuild_MissingNoError(t *testing.T) {
	dir := t.TempDir()
	cfg := &pipeline.Config{ConfigDir: dir}
	cfg.Classification.Registry = "classification.json"
	if err := resetRegistryForRebuild(cfg); err != nil {
		t.Errorf("expected no error for missing registry, got %v", err)
	}
}

func TestResetRegistryForRebuild_NoRegistryConfigured(t *testing.T) {
	cfg := &pipeline.Config{}
	if err := resetRegistryForRebuild(cfg); err != nil {
		t.Errorf("expected no error when no registry configured, got %v", err)
	}
}

func multiBookCfg() *pipeline.Config {
	return &pipeline.Config{
		Book:  "mcdm.heroes.v1",
		Input: "./heroes.md",
		Books: []pipeline.BookConfig{
			{Book: "mcdm.monsters.v1", Input: "./monsters.md"},
			{Book: "mcdm.beastheart.v1", Input: "./beastheart.md"},
		},
	}
}

func books(cfgs []*pipeline.Config) []string {
	out := make([]string, len(cfgs))
	for i, c := range cfgs {
		out[i] = c.Book
	}
	return out
}

func TestSelectBookConfigs_DefaultPrimaryOnly(t *testing.T) {
	cfg := multiBookCfg()
	got, err := selectBookConfigs(cfg, "", false)
	if err != nil {
		t.Fatalf("selectBookConfigs: %v", err)
	}
	// A bare gen processes only the primary book — the documented gotcha that
	// keeps secondary books' data/ output stale unless --all/--book is passed.
	if want := []string{"mcdm.heroes.v1"}; !equalStr(books(got), want) {
		t.Errorf("default selection = %v, want %v", books(got), want)
	}
	if got[0] != cfg {
		t.Error("default selection should return the primary cfg pointer unchanged")
	}
}

func TestSelectBookConfigs_AllReturnsPrimaryPlusSecondaries(t *testing.T) {
	cfg := multiBookCfg()
	got, err := selectBookConfigs(cfg, "", true)
	if err != nil {
		t.Fatalf("selectBookConfigs: %v", err)
	}
	want := []string{"mcdm.heroes.v1", "mcdm.monsters.v1", "mcdm.beastheart.v1"}
	if !equalStr(books(got), want) {
		t.Errorf("--all selection = %v, want %v", books(got), want)
	}
}

func TestSelectBookConfigs_AllTakesPrecedenceOverBookFilter(t *testing.T) {
	cfg := multiBookCfg()
	// Both --all and --book are passed; --all wins (full rebuild).
	got, err := selectBookConfigs(cfg, "mcdm.monsters.v1", true)
	if err != nil {
		t.Fatalf("selectBookConfigs: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("--all should win over --book; got %v", books(got))
	}
}

func TestSelectBookConfigs_BookFilterSelectsSecondary(t *testing.T) {
	cfg := multiBookCfg()
	got, err := selectBookConfigs(cfg, "mcdm.beastheart.v1", false)
	if err != nil {
		t.Fatalf("selectBookConfigs: %v", err)
	}
	if want := []string{"mcdm.beastheart.v1"}; !equalStr(books(got), want) {
		t.Errorf("--book selection = %v, want %v", books(got), want)
	}
	if got[0].Input != "./beastheart.md" {
		t.Errorf("derived input = %q, want ./beastheart.md", got[0].Input)
	}
}

func TestSelectBookConfigs_BookFilterMatchingPrimary(t *testing.T) {
	cfg := multiBookCfg()
	got, err := selectBookConfigs(cfg, "mcdm.heroes.v1", false)
	if err != nil {
		t.Fatalf("selectBookConfigs: %v", err)
	}
	if want := []string{"mcdm.heroes.v1"}; !equalStr(books(got), want) {
		t.Errorf("--book primary selection = %v, want %v", books(got), want)
	}
	if got[0] != cfg {
		t.Error("--book matching the primary should return the primary cfg unchanged")
	}
}

func TestSelectBookConfigs_UnknownBookErrors(t *testing.T) {
	cfg := multiBookCfg()
	_, err := selectBookConfigs(cfg, "mcdm.nope.v1", false)
	if err == nil {
		t.Fatal("expected an error for an unknown book filter")
	}
}

func equalStr(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
