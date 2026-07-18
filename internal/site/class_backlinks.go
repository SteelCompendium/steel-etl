package site

import (
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// owningClassNoun maps an owning-class slug to the noun used in its back-link
// sentence ("A Beastheart companion" / "A Summoner fixture").
var owningClassNoun = map[string]string{
	"beastheart": "companion",
	"summoner":   "fixture",
}

// owningClass returns the (slug, displayName) of the hero class that owns a
// bestiary entity, derived purely from its SCC type-path — the class-owned
// analog of summonerProvenanceEyebrow. Beastheart companions
// (`monster.companion.beastheart.*`) belong to Beastheart; summoner fixtures
// (`monster.fixture.*`, summoner-book only) belong to Summoner. Returns
// ("", "") for anything else — notably `monster.retainer.*` (the Devil
// Detective summoner retainer is a different type-path shape and is not
// covered here; FOLLOWUPS #15 scope decision defers it).
func owningClass(scc string) (slug, name string) {
	scc = strings.TrimSpace(scc)
	src, rest, ok := strings.Cut(scc, "/")
	if !ok {
		return "", ""
	}
	typePath, _, ok := strings.Cut(rest, "/")
	if !ok {
		return "", ""
	}
	seg := strings.Split(typePath, ".")
	switch {
	case len(seg) >= 3 && seg[0] == "monster" && seg[1] == "companion" && seg[2] == "beastheart" &&
		strings.HasPrefix(src, "mcdm.beastheart."):
		return "beastheart", "Beastheart"
	case len(seg) >= 2 && seg[0] == "monster" && seg[1] == "fixture" &&
		strings.HasPrefix(src, "mcdm.summoner."):
		return "summoner", "Summoner"
	}
	return "", ""
}

// classBackLinkHref computes the relative href from a Browse leaf page to its
// owning class landing page (sectionDir/class/<slug>/), derived from the page's
// own path depth rather than a hard-coded hop count — MkDocs directory URLs mean
// each path segment (including the leaf file itself) is one "../" hop back to
// the section root.
func classBackLinkHref(sectionDir, pagePath, slug string) string {
	rel, err := filepath.Rel(sectionDir, pagePath)
	if err != nil {
		rel = pagePath
	}
	ups := len(strings.Split(filepath.ToSlash(rel), "/"))
	return strings.Repeat("../", ups) + "class/" + slug + "/"
}

// firstCardMarker returns the index of the earliest `.sb-wrap` or `.fb-wrap`
// card opening tag in page, or -1 if neither is present (e.g. an index page).
func firstCardMarker(page string) int {
	i := strings.Index(page, `<div class="sb-wrap"`)
	j := strings.Index(page, `<div class="fb-wrap"`)
	switch {
	case i < 0:
		return j
	case j < 0:
		return i
	case i < j:
		return i
	default:
		return j
	}
}

// firstCardOpenEnd returns the index immediately after the closing `>` of the
// earliest `.sb-wrap`/`.fb-wrap` card's opening tag in page, or -1 if neither
// is present (e.g. an index page). Content inserted at this index becomes the
// card div's first child rather than a preceding page-level sibling — the
// distinction matters because v2's CSS only hides the MkDocs-injected H1+hr
// chrome when h1 + hr + the card div are immediately adjacent siblings; a
// preceding sibling paragraph breaks that adjacency and un-hides the chrome.
func firstCardOpenEnd(page string) int {
	i := firstCardMarker(page)
	if i < 0 {
		return -1
	}
	end := strings.IndexByte(page[i:], '>')
	if end < 0 {
		return -1
	}
	return i + end + 1
}

// augmentClassOwnedBackLinks adds a "Companion/fixture of <class>" back-link to
// every bestiary page owned by a hero class — beastheart companions
// (sectionDir/monster/companion/beastheart/*) and summoner fixtures
// (sectionDir/monster/fixture/*/*), on both the statblock/featureblock base page
// and its `-advancement-features` sibling. The relationship is derived from each
// page's `scc` frontmatter via owningClass; no per-entity data edits. Mirrors
// augmentRivalSummonerPages/augmentSummonerRetainerPages: runs after pages are
// written, is idempotent (guards on an existing `sb-backlink`), and is a no-op
// when the owning subtree is absent. The back-link is inserted as the card
// div's first child (via firstCardOpenEnd), not a preceding page-level
// sibling — see firstCardOpenEnd for why that distinction matters. Returns
// the number of pages modified.
func augmentClassOwnedBackLinks(sectionDir string) (int, []string) {
	count := 0
	var errs []string
	for _, sub := range []string{filepath.Join("monster", "companion"), filepath.Join("monster", "fixture")} {
		dir := filepath.Join(sectionDir, sub)
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || d.Name() == "index.md" || !strings.HasSuffix(d.Name(), ".md") {
				return nil
			}
			page := readFile(path)
			fm, _ := splitFrontmatter(page)
			scc := strings.TrimSpace(parseFrontmatterField(fm, "scc"))
			slug, className := owningClass(scc)
			if slug == "" {
				return nil
			}
			if strings.Contains(page, "sb-backlink") {
				return nil // idempotent
			}
			i := firstCardOpenEnd(page)
			if i < 0 {
				return nil // not a rendered card page
			}
			noun := owningClassNoun[slug]
			href := classBackLinkHref(sectionDir, path, slug)
			backlink := fmt.Sprintf(`<p class="sb-backlink">A <a href="%s">%s</a> %s</p>`,
				html.EscapeString(href), html.EscapeString(className), noun)
			page = page[:i] + backlink + page[i:]
			if err := os.WriteFile(path, []byte(page), 0644); err != nil {
				errs = append(errs, fmt.Sprintf("write %s: %v", path, err))
			} else {
				count++
			}
			return nil
		})
		if walkErr != nil {
			errs = append(errs, fmt.Sprintf("walk %s: %v", dir, walkErr))
		}
	}
	return count, errs
}
