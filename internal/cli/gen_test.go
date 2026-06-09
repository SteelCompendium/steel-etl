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
