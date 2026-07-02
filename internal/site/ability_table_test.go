package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAbilityFixture(t *testing.T, dir, rel, fm string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("---\n"+fm+"\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIsAbilityClassDir(t *testing.T) {
	if !isAbilityClassDir("docs/Browse/feature/ability/fury") {
		t.Error("fury dir should match")
	}
	if isAbilityClassDir("docs/Browse/feature/ability") {
		t.Error("ability root should not match")
	}
	if isAbilityClassDir("docs/Browse/feature/ability/fury/level-1") {
		t.Error("level dir should not match")
	}
	if isAbilityClassDir("docs/Browse/feature/trait/censor") {
		t.Error("trait tree should not match")
	}
}

func TestAbilityTable(t *testing.T) {
	dir := t.TempDir()
	writeAbilityFixture(t, dir, "level-1/brutal-slam.md",
		"name: Brutal Slam\nlevel: \"1\"\nsubtype: signature\naction_type: Main action\n"+
			"distance: '[Melee](../../rule/combat/melee.md) 1'\ntarget: One creature or object\ntype: ability")
	writeAbilityFixture(t, dir, "level-5/my-turn.md",
		"name: My Turn!\nlevel: \"5\"\ncost: 9 Ferocity\naction_type: Free triggered\n"+
			"distance: '[Melee](../../rule/combat/melee.md) 1'\ntarget: The triggering creature\ntype: ability")
	writeAbilityFixture(t, dir, "level-1/index.md", "name: Level 1\ntype: index")

	html := abilityTable(dir, []string{"level-1", "level-5"})
	for _, want := range []string{
		`<div class="sc-abtable">`,
		`<a href="level-1/brutal-slam/">Brutal Slam</a>`,
		`<a href="level-5/my-turn/">My Turn!</a>`,
		`<td data-sort="1">1</td>`, // numeric level sort key
		`Signature`,                // signature shown as the cost
		`9 Ferocity`,
		`Melee 1`, // md link stripped
		`Free triggered`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("table missing %q\n%s", want, html)
		}
	}
	if strings.Contains(html, "Level 1</a>") {
		t.Error("index.md must be skipped")
	}
	if abilityTable(t.TempDir(), nil) != "" {
		t.Error("empty dir must yield empty string")
	}
}
