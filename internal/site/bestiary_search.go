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
	"strconv"
	"strings"
)

// bestiaryItem is one searchable record. JSON keys are consumed by
// steel-bestiary-browser.js — keep them in sync with that file.
type bestiaryItem struct {
	Type         string   `json:"type"` // statblock | terrain | retainer
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
	case fmType == "statblock" && strings.HasPrefix(relSlash, "monster/"):
		return "statblock"
	case fmType == "dynamic-terrain":
		return "terrain"
	default: // featureblock, monster (group lore), anything else
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
			Name:         stripMD(parseFrontmatterField(fm, "name")),
			Level:        lvl,
			EV:           unquote(parseFrontmatterField(fm, "ev")),
			Role:         stripMD(parseFrontmatterField(fm, "role")),
			Organization: stripMD(parseFrontmatterField(fm, "organization")),
			Keywords:     kw,
			Size:         stripMD(parseFrontmatterField(fm, "size")),
			Href:         "../Browse/" + dirURL(relSlash),
		})
		return nil
	})
	return items
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

// (json import is used by buildBestiarySearchPage in Task 2.)
var _ = json.Marshal
