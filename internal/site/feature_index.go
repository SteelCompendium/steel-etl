package site

// High-fantasy steel index pages for the NESTED feature & treasure trees — the
// pages that sit BETWEEN the Browse landing and the leaf item cards.
//
// Two node kinds (a node is one or the other, never both):
//
//   - index-of-indexes  (children are directories)  → FOLDER cards
//     e.g. feature/, feature/trait/, feature/trait/censor/, treasure/,
//     treasure/1st-echelon/, the skill/ group landing, and the rule/ glossary
//     landing (its 12 topic groups). Rendered as .sc-folders › .sc-folder
//     anchors. (The skill / rule leaves themselves are flat .sc-card grids —
//     see buildCardsContent.)
//
//   - parent-of-leaves  (children are item pages)   → PREVIEW cards
//     e.g. feature/trait/censor/level-1/ (trait previews) and
//     feature/ability/censor/level-1/ (ability previews). Rendered as
//     .sc-prevs › .sc-prev cards — the exact markup steel-feature-browser.js's
//     SCBrowse.card() emits, so a card looks identical whether you drilled to it
//     or filtered to it.
//
// The feature/ landing additionally carries the Search & Filter data island
// (.sc-browse-mount) that steel-feature-browser.js mounts into a live filter
// over the whole feature tree.
//
// SITE-ONLY, like cards.go / ability_cards.go: it runs in `steel-etl site`
// against the generated md-linked pages — the shared data repos are untouched.
// By the time these index pages are built, each leaf page body has ALREADY been
// rewritten to its .sc-ability / .sc-trait card (buildAbilityCardPage runs in
// buildSection first), so ability data is read from the preserved frontmatter
// and trait flavor / "grants" markers are parsed back out of the rendered HTML.
// Styled by docs/stylesheets/steel-indexes.css.

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// buildFeatureIndexContent renders folder cards (index-of-indexes) or trait/
// ability preview cards (parent-of-leaves) for the nested feature & treasure
// trees. ok=false → the caller falls back to the default browse-index list.
func buildFeatureIndexContent(dir, dirName string, files, subdirs []string) (string, bool) {
	// index-of-indexes: every child is a directory → folder cards. Scoped to the
	// feature, treasure, skill, rule & bestiary (monster / dynamic-terrain /
	// retainer) trees. Monster GROUP dirs are excluded here so they fall through
	// to buildMonsterGroupContent (their featureblock + statblock landing).
	if len(subdirs) > 0 && len(files) == 0 && usesFolderIndex(dir) && !isBestiaryGroupDir(dir) {
		return buildFolderIndex(dir, dirName, subdirs), true
	}

	// parent-of-leaves in the feature subtree → trait / ability preview cards.
	if kind := featureKind(dir); kind != "" && len(files) > 0 && len(subdirs) == 0 {
		return buildPreviewIndex(dir, dirName, files, kind), true
	}

	return "", false
}

// usesFolderIndex reports whether dir is one of the grouped Browse trees
// (feature/, treasure/, skill/, rule/) — the index-of-indexes nodes that render
// as .sc-folder cards. Other sections keep the default browse-index list.
func usesFolderIndex(dir string) bool {
	for _, p := range strings.Split(filepath.ToSlash(dir), "/") {
		switch p {
		case "feature", "treasure", "skill", "rule",
			"monster", "dynamic-terrain", "retainer",
			"minion", "fixture", "champion", "rival",
			"religion":
			return true
		}
	}
	return false
}

// featureKind returns "ability" for feature/ability/** dirs and "trait" for any
// other dir under feature/** — both ancestry/monster traits (feature/trait/**)
// and plain class/domain/college/kit/companion features (feature/<entity>/**),
// which all render as the recessed niche. Returns "" for dirs outside feature/**.
// The segment after `feature` is either the reserved `ability` kind or an entity
// id; everything that is not `ability` is niche-styled (hub-and-spoke paths).
func featureKind(dir string) string {
	parts := strings.Split(filepath.ToSlash(dir), "/")
	for i, p := range parts {
		if p == "feature" && i+1 < len(parts) {
			if parts[i+1] == "ability" {
				return "ability"
			}
			return "trait"
		}
	}
	return ""
}

// ════════════════════════════════════════════════════════════════════════════
//  1 · FOLDER CARDS — index-of-indexes
// ════════════════════════════════════════════════════════════════════════════

var folderChevron = `<svg viewBox="0 0 24 24"><path d="m9 6 6 6-6 6"/></svg>`

// buildFolderIndex renders a directory of subdirectories as navigational folder
// cards. The feature/ landing also gets the Search & Filter island below.
func buildFolderIndex(dir, dirName string, subdirs []string) string {
	sort.Slice(subdirs, func(i, j int) bool { return naturalLess(subdirs[i], subdirs[j]) })

	wrapper := "sc-folders"
	if len(subdirs) <= 3 { // 2–3 big nodes (e.g. feature → Trait | Ability)
		wrapper = "sc-folders sc-folders--lg"
	}

	var sb strings.Builder
	sb.WriteString("# " + dirToTitle(dirName) + "\n\n---\n\n")
	if intro := folderIntro(dir, dirName); intro != "" {
		sb.WriteString(intro + "\n\n")
	}
	sb.WriteString("<div class=\"" + wrapper + "\">\n")
	for _, d := range subdirs {
		count := countLeafFiles(filepath.Join(dir, d))
		sb.WriteString(folderCard(d+"/", folderCrestIcon(dir, d), dirToTitle(d), count))
	}
	sb.WriteString("</div>\n")

	// The feature/ landing is the recommended home of the cross-tree filter.
	if dirName == "feature" {
		sb.WriteString(buildFeatureBrowseSection(dir))
	}
	return sb.String()
}

// folderCard renders one .sc-folder anchor: crest · name · count · chevron. The
// editorial one-line __sub is intentionally omitted (the card reads without it).
func folderCard(href, icon, name string, count int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<a class=\"sc-folder\" href=\"%s\">\n", html.EscapeString(href))
	fmt.Fprintf(&sb, "  <span class=\"sc-crest sc-folder__crest\"><span>%s</span></span>\n", crestSVG(icon))
	fmt.Fprintf(&sb, "  <div class=\"sc-folder__main\"><h3 class=\"sc-folder__name\">%s</h3></div>\n", html.EscapeString(name))
	sb.WriteString("  <div class=\"sc-folder__meta\">")
	if count > 0 {
		fmt.Fprintf(&sb, "<span class=\"sc-folder__count\">%d</span>", count)
	}
	fmt.Fprintf(&sb, "<span class=\"sc-folder__chev\">%s</span>", folderChevron)
	sb.WriteString("</div>\n")
	sb.WriteString("</a>\n")
	return sb.String()
}

// folderCrestIcon picks a crest from the existing filled-MDI icon vocabulary
// (crestSVG renders fill, so stroke glyphs would not match) keyed by the node's
// place in the tree, so every folder carries its category's icon.
func folderCrestIcon(parentDir, childDir string) string {
	slash := filepath.ToSlash(parentDir)
	switch strings.ToLower(childDir) {
	case "trait":
		return "scroll"
	case "ability":
		return "perk"
	case "artifact":
		return "title"
	case "god":
		return "god" // hands-pray crest, matching the god leaf cards
	case "saint":
		return "title" // laurel — venerated heroes, distinct from gods
	}
	switch {
	case strings.Contains(slash, "/feature/ability"):
		return "perk"
	case strings.Contains(slash, "/feature/trait"):
		return "scroll"
	case strings.Contains(slash, "/feature"): // direct children handled above
		return "scroll"
	case strings.Contains(slash, "/treasure"):
		return "treasure"
	case strings.Contains(slash, "/skill"):
		return "skill"
	case strings.Contains(slash, "/rule"):
		return "rule"
	case strings.Contains(slash, "/monster"),
		strings.Contains(slash, "/dynamic-terrain"),
		strings.Contains(slash, "/retainer"):
		return "skull"
	default:
		return "scroll"
	}
}

// folderIntro returns a short lead paragraph for the top-level hub pages. Leaf
// folder pages (class / level / echelon) read fine without one.
func folderIntro(dir, dirName string) string {
	switch {
	case dirName == "feature":
		return "Class and ancestry features — split into **traits** (what your hero *is*) and " +
			"**abilities** (what your hero *does*). Pick a branch, or use Search & Filter below to " +
			"cut straight to a single feature."
	case dirName == "treasure":
		return "Rewards earned through adventure — organized by echelon, plus artifacts and leveled gear. " +
			"Pick a branch to keep browsing."
	case dirName == "rule":
		return "Every rules term and glossary entry, grouped by topic. Pick a category to browse its " +
			"definitions, or use **search** to jump straight to a term."
	case dirName == "monster":
		return "Adversaries from the Monsters book, grouped by kind. Pick a group to see its " +
			"lore, malice, and statblocks — or use the **Bestiary** tab to search and filter every creature."
	case dirName == "dynamic-terrain":
		return "Hazards, fieldworks, mechanisms, and other interactive terrain. Pick a category to browse."
	case dirName == "retainer":
		return "Retainers your heroes can recruit to fight alongside them."
	case dirName == "religion":
		return "The deities of Orden and the legendary heroes who carry out their will. Pick the " +
			"**gods** or their **saints** to browse."
	}
	return ""
}

// countLeafFiles counts the leaf .md pages beneath dir (recursively, skipping
// index pages) — the same total the old .browse-index count showed.
func countLeafFiles(dir string) int {
	n := 0
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		base := info.Name()
		if strings.HasSuffix(base, ".md") && base != "index.md" && base != "_Index.md" &&
			strings.TrimSuffix(base, ".md") != filepath.Base(filepath.Dir(path)) {
			n++
		}
		return nil
	})
	return n
}

// ════════════════════════════════════════════════════════════════════════════
//  2 · PREVIEW CARDS — parent-of-leaves
// ════════════════════════════════════════════════════════════════════════════

var levelDirRe = regexp.MustCompile(`^level-\d+$`)

// buildPreviewIndex renders a parent-of-leaves feature directory as .sc-prev
// preview cards — trait niches or ability plates, matching SCBrowse.card().
func buildPreviewIndex(dir, dirName string, files []string, kind string) string {
	sort.Slice(files, func(i, j int) bool { return naturalLess(files[i], files[j]) })

	klass := klassFromDir(dir, kind)

	var sb strings.Builder
	sb.WriteString("# " + previewTitle(dir, dirName, klass) + "\n\n---\n\n<div class=\"sc-prevs\">\n")
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			continue
		}
		fm, body := splitFrontmatter(string(data))
		it := extractPreviewItem(fm, body, kind, klass)
		it.Href = dirURL(f) // raw-HTML index links aren't rewritten by MkDocs.
		sb.WriteString(renderPrevCard(it, false))
	}
	sb.WriteString("</div>\n")
	return sb.String()
}

// previewTitle composes the page H1. Level pages get a "<Class> — Level N" title;
// flat ancestry/kit pages use their own name. Hub-and-spoke paths with no literal
// kind segment (e.g. feature/companion/beastheart/wolf/level-6) yield an empty
// klass — drop the dangling em-dash and use just "Level N".
func previewTitle(dir, dirName, klass string) string {
	if levelDirRe.MatchString(dirName) {
		if klass == "" {
			return dirToTitle(dirName)
		}
		return klass + " — " + dirToTitle(dirName)
	}
	return dirToTitle(dirName)
}

// klassFromDir derives the owning class/ancestry/kit title for a preview page
// from the directory component immediately after trait/ or ability/.
func klassFromDir(dir, kind string) string {
	parts := strings.Split(filepath.ToSlash(dir), "/")
	for i, p := range parts {
		if p == kind && i+1 < len(parts) {
			return dirToTitle(parts[i+1])
		}
	}
	return ""
}

// browseItem is one preview card's data — shared by the build-time preview cards
// and the Search & Filter JSON island. JSON keys match steel-feature-browser.js.
type browseItem struct {
	Kind     string   `json:"kind"`
	Name     string   `json:"name"`
	Klass         string `json:"klass,omitempty"`
	Subclass      string `json:"subclass,omitempty"`
	FeatureSource string `json:"feature_source,omitempty"` // summoner | circle (Summoner book)
	Source        string `json:"source,omitempty"`         // class | ancestry | kit | other
	Level    int      `json:"level"`
	Action   string   `json:"action,omitempty"`
	Cost     string   `json:"cost,omitempty"`
	Keywords []string `json:"keywords,omitempty"`
	Distance string   `json:"distance,omitempty"`
	Targets  string   `json:"targets,omitempty"`
	Flavor   string   `json:"flavor,omitempty"`
	Grants   string   `json:"grants,omitempty"`  // single-ability grant phrase
	Options  int      `json:"options,omitempty"` // count of sub-feature options
	Tag      string   `json:"tag,omitempty"`
	Href     string   `json:"href"`

	levelStr string // unexported: original frontmatter level, for the level pill
}

// extractPreviewItem reads one leaf page (preserved frontmatter + already-
// rendered HTML body) into a browseItem.
func extractPreviewItem(fm, body, kind, klassFallback string) browseItem {
	it := browseItem{
		Klass:         klassFromMeta(fm, klassFallback),
		Source:        sourceFromMeta(fm),
		Subclass:      titleCase(strings.ReplaceAll(strings.TrimSpace(parseFrontmatterField(fm, "subclass")), "-", " ")),
		FeatureSource: strings.TrimSpace(parseFrontmatterField(fm, "feature_source")),
	}

	it.Kind = strings.TrimSpace(parseFrontmatterField(fm, "type"))
	if it.Kind != "ability" && it.Kind != "trait" && it.Kind != "feature" {
		it.Kind = kind
	}

	it.Name = strings.TrimSpace(parseFrontmatterField(fm, "name"))
	it.levelStr = strings.TrimSpace(parseFrontmatterField(fm, "level"))
	if it.levelStr == "" {
		if m := sccLevelRe.FindStringSubmatch(parseFrontmatterField(fm, "scc")); m != nil {
			it.levelStr = m[1]
		}
	}
	it.Level, _ = strconv.Atoi(it.levelStr)

	if it.Kind == "ability" {
		it.Action = actionInfo(parseFrontmatterField(fm, "action_type"), "ability").key
		it.Cost = strings.TrimSpace(parseFrontmatterField(fm, "cost"))
		if it.Cost == "" && parseFrontmatterField(fm, "subtype") == "signature" {
			it.Cost = "Signature"
		}
		it.Keywords = parseFrontmatterList(fm, "keywords")
		for i, k := range it.Keywords {
			it.Keywords[i] = plainInline(k)
		}
		it.Distance = plainInline(strings.TrimSpace(parseFrontmatterField(fm, "distance")))
		it.Targets = plainInline(strings.TrimSpace(firstField(fm, "target", "targets")))
		it.Flavor = plainInline(strings.TrimSpace(parseFrontmatterField(fm, "flavor")))
	} else {
		// Traits use the trait accent consistently. Flavor + the sub-feature
		// markers come from the rendered .sc-trait HTML / its data-* attributes
		// (frontmatter is too sparse).
		it.Action = "trait"
		it.Flavor = traitFlavorFromHTML(body)
		if m := reDataGrant.FindStringSubmatch(body); m != nil {
			it.Grants = html.UnescapeString(m[1])
		} else if m := reDataSub.FindStringSubmatch(body); m != nil {
			it.Options, _ = strconv.Atoi(m[1])
		}
	}
	return it
}

// klassFromMeta returns the owning source's display name, preferring the
// frontmatter class/ancestry/kit field, falling back to the directory-derived
// name. (Beastheart companion features carry class: beastheart → "Beastheart".)
func klassFromMeta(fm, fallback string) string {
	for _, key := range []string{"class", "ancestry", "kit"} {
		if v := strings.TrimSpace(parseFrontmatterField(fm, key)); v != "" {
			return titleCase(strings.ReplaceAll(v, "-", " "))
		}
	}
	return fallback
}

// sourceFromMeta classifies a feature's origin for the Source facet's
// colour-coding. class wins over ancestry/kit (companion features carry
// class: beastheart).
func sourceFromMeta(fm string) string {
	switch {
	case strings.TrimSpace(parseFrontmatterField(fm, "class")) != "":
		return "class"
	case strings.TrimSpace(parseFrontmatterField(fm, "ancestry")) != "":
		return "ancestry"
	case strings.TrimSpace(parseFrontmatterField(fm, "kit")) != "":
		return "kit"
	default:
		return "other"
	}
}

// ── build-time preview card markup (mirrors SCBrowse.card, context=false) ─────

var prevDigitRe = regexp.MustCompile(`\d+`)

// plainInline reduces inline markdown links to their display text:
// "[Melee](scc:…/melee)" → "Melee". Keyword/distance/flavor data carries SCC
// cross-reference links, but a preview card is itself an <a> (the whole card
// links to the ability), so nesting keyword <a> links inside it is invalid HTML.
// The clickable cross-ref lives on the full ability card; here we show plain
// names — which also keeps the JSON data island, its facet filters, and search
// text clean (they consume the same struct fields).
func plainInline(s string) string {
	return mdLinkRe.ReplaceAllString(s, "$1")
}

// actionByKey maps an action key → {eyebrow label, crest glyph}, mirroring the
// ACTIONS map in steel-feature-browser.js / actionInfo() here.
var actionByKey = map[string][2]string{
	"main":      {"Main Action", "l"},
	"maneuver":  {"Maneuver", "f"},
	"triggered": {"Triggered Action", ")"},
	"move":      {"Move Action", "o"},
	"none":      {"No Action", "*"},
	"trait":     {"Trait", "*"},
}

func renderPrevCard(it browseItem, ctx bool) string {
	if it.Kind == "ability" {
		return renderAbilityPrev(it, ctx)
	}
	return renderTraitPrev(it, ctx)
}

func renderTraitPrev(it browseItem, ctx bool) string {
	eyebrow := strings.TrimSpace(html.EscapeString(it.Klass) + " " + featureNoun(it.Kind))
	if it.Subclass != "" {
		eyebrow += " · " + html.EscapeString(it.Subclass)
	}
	tag := ""
	switch {
	case it.Tag != "":
		tag = "<div class=\"sc-prev__tag\">" + html.EscapeString(it.Tag) + "</div>"
	case it.levelStr != "":
		tag = "<div class=\"sc-prev__tag\">Level <span class=\"num\">" + html.EscapeString(it.levelStr) + "</span></div>"
	}
	flavor := ""
	if it.Flavor != "" {
		flavor = "<div class=\"sc-prev__flavor\">" + html.EscapeString(it.Flavor) + "</div>"
	}
	return "<a class=\"sc-prev sc-prev--trait sc-fil\" data-action=\"trait\" href=\"" + html.EscapeString(it.Href) + "\">" +
		"<div class=\"sc-prev__head\">" +
		"<span class=\"sc-crest sc-prev__crest\"><span class=\"sc-prev__glyph\">" + traitGlyph + "</span></span>" +
		"<div class=\"sc-prev__titles\">" +
		"<div class=\"sc-prev__eyebrow\"><span class=\"sc-prev__dia\"></span>" + eyebrow + "</div>" +
		"<h3 class=\"sc-prev__name\">" + html.EscapeString(it.Name) + "</h3></div>" + tag + "</div>" +
		flavor + traitFootMarker(it) + "</a>\n"
}

// traitFootMarker renders the foot marker: "Grants the X maneuver" for a single
// granted ability, else "N option(s)" for a choice/option list, else nothing.
func traitFootMarker(it browseItem) string {
	text := ""
	switch {
	case it.Grants != "":
		text = "Grants " + html.EscapeString(it.Grants)
	case it.Options == 1:
		text = "1 option"
	case it.Options > 1:
		text = strconv.Itoa(it.Options) + " options"
	}
	if text == "" {
		return ""
	}
	return "<div class=\"sc-prev__foot\"><span class=\"sc-prev__grant\"><span class=\"dot\"></span>" + text + "</span></div>"
}

func renderAbilityPrev(it browseItem, ctx bool) string {
	act := it.Action
	if act == "" {
		act = "main"
	}
	meta, ok := actionByKey[act]
	if !ok {
		meta = actionByKey["main"]
	}
	tag := ""
	if it.Cost != "" {
		tag = "<div class=\"sc-prev__tag\">" + costPrevHTML(it.Cost) + "</div>"
	}
	kw := ""
	if len(it.Keywords) > 0 {
		var b strings.Builder
		b.WriteString("<div class=\"sc-prev__kw\">")
		for _, k := range it.Keywords {
			b.WriteString("<span class=\"sc-prev__chip\">" + html.EscapeString(k) + "</span>")
		}
		b.WriteString("</div>")
		kw = b.String()
	}
	var feet []string
	if ctx && it.Klass != "" {
		feet = append(feet, prevMeta("from", html.EscapeString(it.Klass)+" · Lv "+html.EscapeString(it.levelStr)))
	}
	if it.Distance != "" {
		feet = append(feet, prevMeta("distance", boldNums(it.Distance)))
	}
	if it.Targets != "" {
		feet = append(feet, prevMeta("targets", html.EscapeString(it.Targets)))
	}
	foot := ""
	if len(feet) > 0 {
		foot = "<div class=\"sc-prev__foot\">" + strings.Join(feet, "") + "</div>"
	}
	flavor := ""
	if it.Flavor != "" {
		flavor = "<div class=\"sc-prev__flavor\">" + html.EscapeString(it.Flavor) + "</div>"
	}
	return "<a class=\"sc-prev sc-prev--ability sc-fil\" data-action=\"" + html.EscapeString(act) +
		"\" href=\"" + html.EscapeString(it.Href) + "\">" +
		"<div class=\"sc-prev__head\">" +
		"<span class=\"sc-crest sc-prev__crest\"><span class=\"sc-prev__glyph\">" + html.EscapeString(meta[1]) + "</span></span>" +
		"<div class=\"sc-prev__titles\">" +
		"<div class=\"sc-prev__eyebrow\"><span class=\"sc-prev__dia\"></span>" + html.EscapeString(meta[0]) + "</div>" +
		"<h3 class=\"sc-prev__name\">" + html.EscapeString(it.Name) + "</h3></div>" + tag + "</div>" +
		flavor + kw + foot + "</a>\n"
}

func prevMeta(label, valueHTML string) string {
	return "<span class=\"sc-prev__meta\"><span class=\"l\">" + html.EscapeString(label) +
		"</span><span class=\"v\">" + valueHTML + "</span></span>"
}

// costPrevHTML renders a cost tag, splitting a leading integer into a mono span
// (mirrors abilityCard()'s costHTML in steel-feature-browser.js).
func costPrevHTML(cost string) string {
	if m := costNumRe.FindStringSubmatch(cost); m != nil {
		return "<span class=\"num\">" + html.EscapeString(m[1]) + "</span> " + html.EscapeString(m[2])
	}
	return html.EscapeString(cost)
}

// boldNums escapes text and wraps bare integers in <b> — matching the JS md()
// helper's treatment of "Ranged **10**" without needing the markers in data.
func boldNums(s string) string {
	return prevDigitRe.ReplaceAllString(html.EscapeString(s), "<b>$0</b>")
}

// ── trait flavor / grants, parsed from the rendered .sc-trait HTML body ───────

var (
	reTraitFlavorP = regexp.MustCompile(`(?s)<p class="sc-trait__flavor">(.*?)</p>`)
	reTraitLeadinP = regexp.MustCompile(`(?s)<p class="sc-trait__leadin">(.*?)</p>`)
	reTraitFirstP  = regexp.MustCompile(`(?s)<p[^>]*>(.*?)</p>`)
	reHTMLTag      = regexp.MustCompile(`<[^>]+>`)
	// data-* markers stamped onto the top-level <section class="sc-trait"> by
	// trait_cards.go (the first match is the root trait).
	reDataSub   = regexp.MustCompile(`data-sub="(\d+)"`)
	reDataGrant = regexp.MustCompile(`data-grant="([^"]*)"`)
)

// traitFlavorFromHTML returns the trait's one-line summary: the italic flavor if
// present, else the lead-in run-in, else the first paragraph. Plain text (tags
// stripped, entities decoded) — the caller re-escapes for output.
func traitFlavorFromHTML(body string) string {
	for _, re := range []*regexp.Regexp{reTraitFlavorP, reTraitLeadinP, reTraitFirstP} {
		if m := re.FindStringSubmatch(body); m != nil {
			if t := plainText(m[1]); t != "" {
				return t
			}
		}
	}
	return ""
}

// plainText strips HTML tags, decodes entities, and collapses whitespace.
func plainText(s string) string {
	s = reHTMLTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

// ════════════════════════════════════════════════════════════════════════════
//  3 · SEARCH & FILTER — the cross-tree data island
// ════════════════════════════════════════════════════════════════════════════

// buildFeatureBrowseSection emits the Search & Filter heading + the
// .sc-browse-mount data island steel-feature-browser.js auto-mounts. Walks the
// whole feature subtree, one JSON object per leaf.
func buildFeatureBrowseSection(featureDir string) string {
	items := collectBrowseItems(featureDir)
	if len(items) == 0 {
		return ""
	}
	data, err := json.Marshal(items) // default escapes <, >, & → safe inside <script>
	if err != nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n## Search & Filter\n\n")
	sb.WriteString("The whole feature tree on one page — search by name, filter by type, class, level, " +
		"action or keyword, then jump straight to the card you need.\n\n")
	sb.WriteString("<div class=\"sc-browse-mount\">\n")
	sb.WriteString("<script type=\"application/json\" class=\"sc-browse-data\">\n")
	sb.Write(data)
	sb.WriteString("\n</script>\n</div>\n")
	return sb.String()
}

// collectBrowseItems walks the feature tree under featureDir and returns one
// browseItem per leaf page, with hrefs as directory URLs relative to the
// feature/ landing.
func collectBrowseItems(featureDir string) []browseItem {
	var items []browseItem
	_ = filepath.Walk(featureDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		base := info.Name()
		if !strings.HasSuffix(base, ".md") || base == "index.md" || base == "_Index.md" {
			return nil
		}
		kind := featureKind(filepath.Dir(path))
		if kind == "" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		fm, body := splitFrontmatter(string(data))
		it := extractPreviewItem(fm, body, kind, klassFromDir(filepath.Dir(path), kind))

		rel, _ := filepath.Rel(featureDir, path)
		it.Href = strings.TrimSuffix(filepath.ToSlash(rel), ".md") + "/"
		it.Distance = starBoldNums(it.Distance) // JS md() bolds **n**
		items = append(items, it)
		return nil
	})
	sort.SliceStable(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items
}

// starBoldNums wraps bare integers in ** so steel-feature-browser.js's md()
// renders them bold (the frontmatter distance has no markers).
func starBoldNums(s string) string {
	if s == "" {
		return ""
	}
	return prevDigitRe.ReplaceAllString(s, "**$0**")
}
