package site

// Bestiary Search & Filter data island (Plan B). Walks the Browse monster /
// dynamic-terrain / retainer pages and emits one JSON record per searchable
// entity into a .sc-bestiary-mount island on the Bestiary landing, mounted
// client-side by v2/docs/javascripts/steel-bestiary-browser.js (window.SCBestiary).
// SITE-ONLY: all data is read from existing frontmatter — no data-repo change.
// See docs/superpowers/specs/2026-06-10-bestiary-restructure-and-search-design.md.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// bestiaryItem is one searchable record. JSON keys are consumed by
// steel-bestiary-browser.js — keep them in sync with that file.
type bestiaryItem struct {
	Type         string   `json:"type"`             // statblock | terrain | retainer | fixture
	Source       string   `json:"source,omitempty"` // e.g. "Summoner" (scc-derived); "" = Monsters book
	Name         string   `json:"name"`
	Level        int      `json:"level"`
	EV           string   `json:"ev"` // string: may be "-" (no EV); JS parses for range
	Role         string   `json:"role,omitempty"`
	Organization string   `json:"organization,omitempty"`
	Keywords     []string `json:"keywords,omitempty"`
	Size         string   `json:"size,omitempty"`
	Href         string   `json:"href"`
}

// bestiaryItemType classifies a Browse page by its frontmatter `type` + its tree
// (the statblock/ folder was hoisted away, so the path no longer carries a
// statblock segment). Returns "" for non-searchable pages (group lore, Malice
// featureblocks, indexes).
func bestiaryItemType(relSlash, fmType string) string {
	base := relSlash[strings.LastIndexByte(relSlash, '/')+1:]
	if base == "index.md" || base == "_Index.md" {
		return ""
	}
	switch {
	case fmType == "statblock" && strings.HasPrefix(relSlash, "retainer/"):
		return "retainer"
	case fmType == "statblock" && strings.HasPrefix(relSlash, "monster/retainer/"):
		// Monsters-book retainers joined the monster.* family (Plan 6) but keep
		// their own "retainer" bestiary facet. Their sibling advancement-features
		// and role-advancement pages are type:featureblock → excluded by default.
		return "retainer"
	case fmType == "statblock" && (strings.HasPrefix(relSlash, "monster/") ||
		strings.HasPrefix(relSlash, "minion/") || strings.HasPrefix(relSlash, "fixture/") ||
		strings.HasPrefix(relSlash, "champion/") || strings.HasPrefix(relSlash, "rival/")):
		// Monsters-book creatures + the summoner book's portfolio minions/
		// champions and the rival summoner all index as statblocks.
		return "statblock"
	case fmType == "featureblock" && strings.HasPrefix(relSlash, "monster/fixture/") &&
		!strings.Contains(relSlash, "/advancement-features/"):
		// Summoner fixtures became monster.fixture.<element>.featureblock entities
		// (Plan 5c); their base page stays searchable as its own "fixture" facet.
		// The sibling advancement-features page is internal — excluded.
		return "fixture"
	case fmType == "dynamic-terrain":
		return "terrain"
	default: // malice/feature featureblocks, monster (group lore), anything else
		return ""
	}
}

// collectBestiaryItems walks browseDir (docs/Browse) and returns one record per
// searchable monster-statblock / terrain / retainer leaf, name-sorted by the
// caller's marshal order (stable: file walk is lexical).
func collectBestiaryItems(browseDir string) []bestiaryItem {
	var items []bestiaryItem
	_ = filepath.Walk(browseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(browseDir, path)
		relSlash := filepath.ToSlash(rel)
		fm, _ := splitFrontmatter(readFile(path))
		kind := bestiaryItemType(relSlash, strings.TrimSpace(parseFrontmatterField(fm, "type")))
		if kind == "" {
			return nil
		}
		lvl, _ := strconv.Atoi(unquote(parseFrontmatterField(fm, "level")))
		var kw []string
		for _, k := range parseFrontmatterList(fm, "keywords") {
			kw = append(kw, stripMD(k))
		}
		items = append(items, bestiaryItem{
			Type:         kind,
			Source:       bestiarySource(fm),
			Name:         stripMD(parseFrontmatterField(fm, "name")),
			Level:        lvl,
			EV:           unquote(statField(fm, "ev", "EV")),
			Role:         stripMD(parseFrontmatterField(fm, "role")),
			Organization: stripMD(parseFrontmatterField(fm, "organization")),
			Keywords:     kw,
			Size:         sizeFacet(kind, stripMD(statField(fm, "size", "Size"))),
			Href:         "../Browse/" + dirURL(relSlash),
		})
		return nil
	})
	return items
}

// canonicalSizeRe matches real creature sizes: "1T"/"1S"/"1M"/"1L", bare
// numbers, and the variable forms "1S-2" and "2 or 3".
var canonicalSizeRe = regexp.MustCompile(`^\d+[TSML]?(-\d+)?( or \d+)?$`)

// sizeFacet normalizes the Size filter value. Statblock sizes pass through;
// dynamic-terrain / fixture pages carry free-text area descriptions in their
// `size` frontmatter ("any area; the area can't be moved through") which would
// each become their own filter chip on the Bestiary page — bucket those under
// "Area". Anything else non-canonical becomes "Special" so the chip vocabulary
// stays closed.
func sizeFacet(kind, size string) string {
	if size == "" || canonicalSizeRe.MatchString(size) {
		return size
	}
	if kind == "terrain" || kind == "fixture" {
		return "Area"
	}
	return "Special"
}

// unquote strips a single layer of surrounding double/single quotes and trims
// (frontmatter scalars like `ev: "3"` or `ev: '-'` keep their quotes through
// parseFrontmatterField).
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// buildBestiarySearchPage writes docs/Bestiary/index.md: the Search & Filter
// landing carrying a .sc-bestiary-mount JSON data island over every Browse
// statblock / terrain / retainer. Returns false (no write) when there are no
// items, so a build without the Monsters book leaves no empty tab.
func buildBestiarySearchPage(docsDir string) (bool, error) {
	items := collectBestiaryItems(filepath.Join(docsDir, "Browse"))
	if len(items) == 0 {
		return false, nil
	}
	data, err := json.Marshal(items) // default escapes <,>,& → safe inside <script>
	if err != nil {
		return false, err
	}
	var sb strings.Builder
	sb.WriteString("---\nsearch:\n  exclude: true\n---\n\n")
	sb.WriteString("# Bestiary — Search & Filter\n\n")
	sb.WriteString("Find statblocks, dynamic terrain, and retainers across every sourcebook. " +
		"Search by name, filter by type, role, organization, size, or keyword, and narrow by " +
		"**Level** and **EV** range — then jump straight to the page you need.\n\n")
	sb.WriteString("<div class=\"sc-bestiary-mount\">\n")
	sb.WriteString("<script type=\"application/json\" class=\"sc-browse-data\">\n")
	sb.Write(data)
	sb.WriteString("\n</script>\n</div>\n")

	dir := filepath.Join(docsDir, "Bestiary")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(sb.String()), 0o644); err != nil {
		return false, err
	}
	return true, nil
}
