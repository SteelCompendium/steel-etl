package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// freeStrikeHostPage is the Free Strike common-action rule page as buildSection
// leaves it: preserved frontmatter + the rendered .sc-trait card.
const freeStrikeHostPage = `---
name: Free Strike
scc: mcdm.heroes.v1/feature.common.main-actions/free-strike
type: feature
---

# Free Strike

---

<section class="sc-trait sc-trait--crest sc-trait--lead" data-action="trait">
<div class="sc-trait__body">
<p>A creature can use this main action to make a free strike.</p>
</div>
</section>
`

// abilityLeafPage builds a rendered common-ability leaf page carrying subtype.
func abilityLeafPage(name, id, subtype, distance string) string {
	return "---\n" +
		"action_type: Main action\n" +
		"distance: '" + distance + "'\n" +
		"name: " + name + "\n" +
		"scc: mcdm.heroes.v1/feature.ability.common/" + id + "\n" +
		"subtype: " + subtype + "\n" +
		"target: One creature or object\n" +
		"type: ability\n" +
		"---\n\n# " + name + "\n\n---\n\n" +
		`<article class="sc-ability sc-fil" data-action="main"></article>` + "\n"
}

// writeSitePage writes p (relative to root) with its parent directories.
func writeSitePage(t *testing.T, root, p, content string) string {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", full, err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
	return full
}

// newFreeStrikeSection lays out a minimal Browse section: the Free Strike rule
// page plus the two standard free strike ability leaves and one unrelated common
// ability.
func newFreeStrikeSection(t *testing.T) (sectionDir, hostPath string) {
	t.Helper()
	sectionDir = t.TempDir()
	hostPath = writeSitePage(t, sectionDir, "feature/common/main-actions/free-strike.md", freeStrikeHostPage)
	writeSitePage(t, sectionDir, "feature/ability/common/melee-weapon-free-strike.md",
		abilityLeafPage("Melee Weapon Free Strike", "melee-weapon-free-strike", "free-strike", "Melee 1"))
	writeSitePage(t, sectionDir, "feature/ability/common/ranged-weapon-free-strike.md",
		abilityLeafPage("Ranged Weapon Free Strike", "ranged-weapon-free-strike", "free-strike", "Ranged 5"))
	writeSitePage(t, sectionDir, "feature/ability/common/grab.md",
		abilityLeafPage("Grab", "grab", "", "Melee 1"))
	return sectionDir, hostPath
}

func TestAugmentActionAbilityPages(t *testing.T) {
	sectionDir, hostPath := newFreeStrikeSection(t)

	n, errs := augmentActionAbilityPages(sectionDir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if n != 1 {
		t.Fatalf("modified pages = %d, want 1", n)
	}

	page := readFile(hostPath)
	if !strings.Contains(page, "## Free Strike Abilities") {
		t.Errorf("host page missing the abilities heading:\n%s", page)
	}
	if !strings.Contains(page, `<div class="sc-prevs">`) {
		t.Errorf("host page missing the .sc-prevs grid:\n%s", page)
	}
	for _, want := range []string{"Melee Weapon Free Strike", "Ranged Weapon Free Strike"} {
		if !strings.Contains(page, want) {
			t.Errorf("host page missing ability %q:\n%s", want, page)
		}
	}
	// An ability without the registered subtype must not be listed.
	if strings.Contains(page, ">Grab<") {
		t.Errorf("unrelated common ability leaked onto the host page:\n%s", page)
	}
	// Hrefs are directory URLs relative to the host page's own URL directory.
	for _, want := range []string{
		`href="../../../ability/common/melee-weapon-free-strike/"`,
		`href="../../../ability/common/ranged-weapon-free-strike/"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("host page missing href %q:\n%s", want, page)
		}
	}
	// Melee sorts before Ranged.
	if strings.Index(page, "Melee Weapon Free Strike") > strings.Index(page, "Ranged Weapon Free Strike") {
		t.Errorf("abilities not in natural order:\n%s", page)
	}
	// The original card body survives.
	if !strings.Contains(page, "A creature can use this main action to make a free strike.") {
		t.Errorf("host page body lost:\n%s", page)
	}
	// The ability leaves themselves are untouched (one-way reference).
	leaf := readFile(filepath.Join(sectionDir, "feature", "ability", "common", "melee-weapon-free-strike.md"))
	if strings.Contains(leaf, "## Free Strike Abilities") {
		t.Errorf("ability leaf was modified:\n%s", leaf)
	}
}

// TestAugmentActionAbilityPagesSubheadingDoesNotSuppress covers LOW-1: the
// idempotency guard must be line-anchored to "## <heading>", not a bare
// substring match — a "### Free Strike Abilities" sub-heading (exactly what
// SC-303's feature-group work would add under the Chapter 10 "Free Strikes"
// section) contains "## Free Strike Abilities" as a substring and must NOT be
// mistaken for this pass's own already-appended block.
func TestAugmentActionAbilityPagesSubheadingDoesNotSuppress(t *testing.T) {
	sectionDir, hostPath := newFreeStrikeSection(t)
	withSubheading := freeStrikeHostPage + "\n### Free Strike Abilities\n\nSome unrelated prose.\n"
	if err := os.WriteFile(hostPath, []byte(withSubheading), 0644); err != nil {
		t.Fatalf("write %s: %v", hostPath, err)
	}

	n, errs := augmentActionAbilityPages(sectionDir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if n != 1 {
		t.Fatalf("modified pages = %d, want 1 (guard must not be fooled by the ### sub-heading)", n)
	}
	page := readFile(hostPath)
	if strings.Count(page, "## Free Strike Abilities") != 2 {
		// one from the pre-existing "### Free Strike Abilities" line, one
		// from the pass's own "## Free Strike Abilities" — both must be
		// present; the guard must not have skipped the append.
		t.Errorf("expected the pre-existing sub-heading AND the pass's own heading, got:\n%s", page)
	}
	if !strings.Contains(page, "\n## Free Strike Abilities\n") {
		t.Errorf("host page missing the pass's own line-anchored heading:\n%s", page)
	}
}

func TestAugmentActionAbilityPagesIdempotent(t *testing.T) {
	sectionDir, hostPath := newFreeStrikeSection(t)

	if _, errs := augmentActionAbilityPages(sectionDir); len(errs) > 0 {
		t.Fatalf("first pass errors: %v", errs)
	}
	first := readFile(hostPath)

	n, errs := augmentActionAbilityPages(sectionDir)
	if len(errs) > 0 {
		t.Fatalf("second pass errors: %v", errs)
	}
	if n != 0 {
		t.Errorf("second pass modified %d pages, want 0", n)
	}
	if got := readFile(hostPath); got != first {
		t.Errorf("second pass changed the page:\n%s", got)
	}
	if c := strings.Count(first, "## Free Strike Abilities"); c != 1 {
		t.Errorf("heading appears %d times, want 1", c)
	}
}

func TestAugmentActionAbilityPagesNoOps(t *testing.T) {
	t.Run("no feature tree", func(t *testing.T) {
		dir := t.TempDir()
		if n, errs := augmentActionAbilityPages(dir); n != 0 || len(errs) > 0 {
			t.Errorf("empty section: n=%d errs=%v", n, errs)
		}
	})

	t.Run("abilities present but host page absent", func(t *testing.T) {
		// A half-match (LOW-2): the book shipped the subtyped ability but not
		// its host rule page. This must surface, not stay silent — a silent
		// half-match is exactly how the original SC-179 bug (rule and ability
		// pages that never met) would have gone undetected here too.
		dir := t.TempDir()
		writeSitePage(t, dir, "feature/ability/common/melee-weapon-free-strike.md",
			abilityLeafPage("Melee Weapon Free Strike", "melee-weapon-free-strike", "free-strike", "Melee 1"))
		n, errs := augmentActionAbilityPages(dir)
		if n != 0 {
			t.Errorf("host-less section: n=%d, want 0", n)
		}
		if len(errs) == 0 {
			t.Error("host-less section: want a reported error for the half-match, got none")
		}
	})

	t.Run("host page present but no subtyped abilities", func(t *testing.T) {
		// The other half-match direction (LOW-2 residual): the rule page is
		// right here — so this book IS in the build — but nothing carries
		// the registered subtype. That is exactly the "annotation got
		// renamed/typo'd" scenario the cross-reference could silently vanish
		// under; it must surface, not stay silent (round 5 re-review).
		dir := t.TempDir()
		host := writeSitePage(t, dir, "feature/common/main-actions/free-strike.md", freeStrikeHostPage)
		writeSitePage(t, dir, "feature/ability/common/grab.md",
			abilityLeafPage("Grab", "grab", "", "Melee 1"))
		n, errs := augmentActionAbilityPages(dir)
		if n != 0 {
			t.Errorf("ability-less section: n=%d, want 0", n)
		}
		if len(errs) != 1 {
			t.Errorf("ability-less section: errs=%v, want exactly 1", errs)
		}
		if strings.Contains(readFile(host), "## Free Strike Abilities") {
			t.Error("host page must be untouched with no matching abilities")
		}
	})
}

func TestFindPageBySCC(t *testing.T) {
	sectionDir, hostPath := newFreeStrikeSection(t)

	if got, dupErr := findPageBySCC(sectionDir, "mcdm.heroes.v1/feature.common.main-actions/free-strike"); got != hostPath || dupErr != "" {
		t.Errorf("findPageBySCC = (%q, %q), want (%q, \"\")", got, dupErr, hostPath)
	}
	if got, dupErr := findPageBySCC(sectionDir, "mcdm.heroes.v1/feature.common.main-actions/nope"); got != "" || dupErr != "" {
		t.Errorf("unknown code should not match, got (%q, %q)", got, dupErr)
	}
}

// TestFindPageBySCCDuplicate covers LOW-3: the SCC registry is supposed to
// guarantee unique codes, but this pass must not silently trust that and pick
// whichever duplicate the filesystem walk visits first — it should refuse to
// guess and report the collision instead.
func TestFindPageBySCCDuplicate(t *testing.T) {
	sectionDir, hostPath := newFreeStrikeSection(t)
	// A second page, lexically earlier than the real host, carrying the same
	// scc code (the decoy that beat the canonical page pre-fix).
	decoy := writeSitePage(t, sectionDir, "feature/common/aaa-earlier/free-strike.md", freeStrikeHostPage)

	got, dupErr := findPageBySCC(sectionDir, "mcdm.heroes.v1/feature.common.main-actions/free-strike")
	if got != "" {
		t.Errorf("duplicate scc: got a path (%q), want \"\" (must not guess)", got)
	}
	if dupErr == "" {
		t.Fatal("duplicate scc: want a reported error, got none")
	}
	for _, want := range []string{decoy, hostPath} {
		if !strings.Contains(dupErr, want) {
			t.Errorf("duplicate error %q missing path %q", dupErr, want)
		}
	}
}

func TestRelHref(t *testing.T) {
	cases := []struct{ from, to, want string }{
		{
			from: "feature/common/main-actions/free-strike",
			to:   "feature/ability/common/melee-weapon-free-strike",
			want: "../../../ability/common/melee-weapon-free-strike/",
		},
		{
			from: "feature/common/main-actions/charge",
			to:   "feature/common/main-actions/free-strike",
			want: "../free-strike/",
		},
	}
	for _, tc := range cases {
		if got := relHref(tc.from, tc.to); got != tc.want {
			t.Errorf("relHref(%q, %q) = %q, want %q", tc.from, tc.to, got, tc.want)
		}
	}
}

// TestAugmentActionAbilityPagesCrossBookExcluded covers MED-2 and its round-5
// residual close: a page carrying the registered subtype but from a
// DIFFERENT book than the relation's host (e.g. a future monster/summoner
// ability that happens to reuse "free-strike") must not join the Heroes host
// page's grid — it would falsify the page's own lead sentence ("Every hero
// has these standard free strike abilities") and could sort in between the
// two canonical cards. Nor must a SAME-book but CLASS-specific ability (a
// future class's signature option annotated with the same subtype) — the
// grid is for the STANDARD, class-agnostic free strikes only
// (`feature.ability.common`), not every ability that happens to grant one.
func TestAugmentActionAbilityPagesCrossBookExcluded(t *testing.T) {
	sectionDir, hostPath := newFreeStrikeSection(t)
	writeSitePage(t, sectionDir, "feature/ability/monster/goblin-shank.md",
		"---\n"+
			"action_type: Main action\n"+
			"distance: 'Melee 1'\n"+
			"name: Goblin Shank\n"+
			"scc: mcdm.monsters.v1/feature.ability.monster/goblin-shank\n"+
			"subtype: free-strike\n"+
			"target: One creature\n"+
			"type: ability\n"+
			"---\n\n# Goblin Shank\n\n---\n\n"+
			`<article class="sc-ability sc-fil" data-action="main"></article>`+"\n")
	writeSitePage(t, sectionDir, "feature/ability/troubadour/level-1/encore-strike.md",
		"---\n"+
			"action_type: Main action\n"+
			"distance: 'Melee 1'\n"+
			"name: Encore Strike\n"+
			"scc: mcdm.heroes.v1/feature.ability.troubadour.level-1/encore-strike\n"+
			"subtype: free-strike\n"+
			"target: One creature\n"+
			"type: ability\n"+
			"---\n\n# Encore Strike\n\n---\n\n"+
			`<article class="sc-ability sc-fil" data-action="main"></article>`+"\n")

	n, errs := augmentActionAbilityPages(sectionDir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if n != 1 {
		t.Fatalf("modified pages = %d, want 1", n)
	}
	page := readFile(hostPath)
	if strings.Contains(page, "Goblin Shank") {
		t.Errorf("cross-book ability leaked onto the Heroes host page:\n%s", page)
	}
	if strings.Contains(page, "Encore Strike") {
		t.Errorf("same-book class-specific ability leaked onto the host page:\n%s", page)
	}
	if c := strings.Count(page, `class="sc-prev sc-prev--ability`); c != 2 {
		t.Errorf("grid cards = %d, want exactly 2 (the two standard free strikes):\n%s", c, page)
	}
	for _, want := range []string{"Melee Weapon Free Strike", "Ranged Weapon Free Strike"} {
		if !strings.Contains(page, want) {
			t.Errorf("host page missing in-book ability %q:\n%s", want, page)
		}
	}
}

func TestCollectSubtypeAbilities(t *testing.T) {
	sectionDir, _ := newFreeStrikeSection(t)

	got := collectSubtypeAbilities(sectionDir)
	if len(got) != 1 {
		t.Fatalf("subtypes found = %d, want 1 (%v)", len(got), got)
	}
	pages := got["free-strike"]
	if len(pages) != 2 {
		t.Fatalf("free-strike abilities = %d, want 2", len(pages))
	}
	for _, p := range pages {
		if strings.HasSuffix(p.urlDir, ".md") {
			t.Errorf("urlDir must have .md stripped, got %q", p.urlDir)
		}
		if !strings.HasPrefix(p.urlDir, "feature/ability/common/") {
			t.Errorf("urlDir must be docs-relative with forward slashes, got %q", p.urlDir)
		}
	}
}
