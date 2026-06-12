package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"github.com/SteelCompendium/steel-etl/internal/site"
)

// SCCAPIGenerator builds a static JSON API for SCC code resolution.
// It produces:
//   - index.json: API metadata and discovery
//   - scc.json: full registry with all entries and aliases
//   - types.json: entries grouped by type
//   - resolve/{source}/{type}/{item}.json: per-entry lookup files
type SCCAPIGenerator struct {
	OutputDir     string               // e.g., "output/api"
	BaseURL       string               // e.g., "https://steelcompendium.io/v2"
	SchemeVersion int                  // SCC scheme (grammar) version; 0 ⇒ 1
	Sections      []site.SectionConfig // site sections for URL mapping
	Aliases       map[string]string    // alias → canonical SCC
	Printings     map[string]string    // book source → printing (non-identity provenance)
	entries       map[string]apiEntry
}

type apiEntry struct {
	SCC      string `json:"scc"`
	URL      string `json:"url"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Source   string `json:"source"`
	Printing string `json:"printing,omitempty"`
}

// apiBook is per-book non-identity provenance metadata surfaced by the API.
type apiBook struct {
	Printing string `json:"printing,omitempty"`
}

type apiIndex struct {
	Version       int                `json:"version"`
	SchemeVersion int                `json:"scheme_version"`
	Generated     string             `json:"generated"`
	TotalCodes    int                `json:"total_codes"`
	TotalAliases  int                `json:"total_aliases"`
	BaseURL       string             `json:"base_url"`
	Books         map[string]apiBook `json:"books,omitempty"`
	Endpoints     apiEndpoints       `json:"endpoints"`
}

type apiEndpoints struct {
	Registry string `json:"registry"`
	Resolve  string `json:"resolve"`
	Types    string `json:"types"`
}

type apiRegistry struct {
	Version       int                `json:"version"`
	SchemeVersion int                `json:"scheme_version"`
	Generated     string             `json:"generated"`
	BaseURL       string             `json:"base_url"`
	Books         map[string]apiBook `json:"books,omitempty"`
	Entries       []apiEntry         `json:"entries"`
	Aliases       map[string]string  `json:"aliases,omitempty"`
}

type apiTypes struct {
	Version       int                   `json:"version"`
	SchemeVersion int                   `json:"scheme_version"`
	Types         map[string][]apiEntry `json:"types"`
}

// apiResolveEntry is the body of a per-entry resolve/*.json file: a single
// entry made self-describing with the scheme version it was minted under.
type apiResolveEntry struct {
	apiEntry
	SchemeVersion int `json:"scheme_version"`
}

func (g *SCCAPIGenerator) Format() string { return "scc-api" }

func (g *SCCAPIGenerator) WriteSection(sccCode string, parsed *content.ParsedContent) error {
	if sccCode == "" || parsed == nil {
		return nil
	}

	if g.entries == nil {
		g.entries = make(map[string]apiEntry)
	}

	name, _ := parsed.Frontmatter["name"].(string)
	typeName, _ := parsed.Frontmatter["type"].(string)

	source := extractSource(sccCode)
	url := g.resolveURL(sccCode)

	g.entries[sccCode] = apiEntry{
		SCC:      sccCode,
		URL:      url,
		Name:     name,
		Type:     typeName,
		Source:   source,
		Printing: g.Printings[source],
	}

	return nil
}

// Finalize writes all API JSON files.
func (g *SCCAPIGenerator) Finalize() error {
	if len(g.entries) == 0 {
		return nil
	}

	apiDir := filepath.Join(g.OutputDir, "v1")
	now := time.Now().UTC().Format(time.RFC3339)

	schemeVer := g.SchemeVersion
	if schemeVer == 0 {
		schemeVer = 1
	}

	sorted := g.sortedEntries()

	aliases := g.Aliases
	if aliases == nil {
		aliases = make(map[string]string)
	}

	var books map[string]apiBook
	if len(g.Printings) > 0 {
		books = make(map[string]apiBook, len(g.Printings))
		for b, p := range g.Printings {
			books[b] = apiBook{Printing: p}
		}
	}

	// 1. index.json
	if err := g.writeJSON(filepath.Join(apiDir, "index.json"), apiIndex{
		Version:       1,
		SchemeVersion: schemeVer,
		Generated:     now,
		TotalCodes:    len(sorted),
		TotalAliases:  len(aliases),
		BaseURL:       g.BaseURL,
		Books:         books,
		Endpoints: apiEndpoints{
			Registry: "/api/v1/scc.json",
			Resolve:  "/api/v1/resolve/{source}/{type}/{item}.json",
			Types:    "/api/v1/types.json",
		},
	}); err != nil {
		return fmt.Errorf("write index.json: %w", err)
	}

	// 2. scc.json (full registry)
	if err := g.writeJSON(filepath.Join(apiDir, "scc.json"), apiRegistry{
		Version:       1,
		SchemeVersion: schemeVer,
		Generated:     now,
		BaseURL:       g.BaseURL,
		Books:         books,
		Entries:       sorted,
		Aliases:       aliases,
	}); err != nil {
		return fmt.Errorf("write scc.json: %w", err)
	}

	// 3. types.json (grouped by type)
	grouped := make(map[string][]apiEntry)
	for _, e := range sorted {
		grouped[e.Type] = append(grouped[e.Type], e)
	}
	if err := g.writeJSON(filepath.Join(apiDir, "types.json"), apiTypes{
		Version:       1,
		SchemeVersion: schemeVer,
		Types:         grouped,
	}); err != nil {
		return fmt.Errorf("write types.json: %w", err)
	}

	// 4. resolve/{scc}.json (per-entry files)
	// Wipe the resolve tree first so codes that were renamed or removed since the
	// last run don't linger as stale dead-link files (the index/scc/types files
	// above are single files and self-correct on overwrite; resolve/ accumulates).
	// See FOLLOWUPS #7.
	resolveDir := filepath.Join(apiDir, "resolve")
	if err := os.RemoveAll(resolveDir); err != nil {
		return fmt.Errorf("clean resolve dir: %w", err)
	}
	for _, e := range sorted {
		relPath := e.SCC + ".json"
		if err := g.writeJSON(filepath.Join(resolveDir, relPath), apiResolveEntry{apiEntry: e, SchemeVersion: schemeVer}); err != nil {
			return fmt.Errorf("write resolve %s: %w", e.SCC, err)
		}
	}

	// 5. Alias resolve files (point to canonical entry)
	for alias, canonical := range aliases {
		if canonicalEntry, ok := g.entries[canonical]; ok {
			aliasEntry := canonicalEntry
			aliasEntry.SCC = canonical // ensure canonical SCC in response
			relPath := alias + ".json"
			if err := g.writeJSON(filepath.Join(resolveDir, relPath), apiResolveEntry{apiEntry: aliasEntry, SchemeVersion: schemeVer}); err != nil {
				return fmt.Errorf("write alias resolve %s: %w", alias, err)
			}
		}
	}

	return nil
}

// resolveURL builds the full website URL for an SCC code.
func (g *SCCAPIGenerator) resolveURL(sccCode string) string {
	filePath := SCCToFilePath(sccCode, "")

	// Try to match against site sections for the correct section prefix
	if len(g.Sections) > 0 {
		normalized := filepath.ToSlash(filePath)
		for _, section := range g.Sections {
			if matchesSectionIncludes(normalized, section) {
				return g.BaseURL + "/" + section.Name + "/" + normalized + "/"
			}
		}
	}

	// Fallback: no section prefix
	return g.BaseURL + "/" + filePath + "/"
}

// matchesSectionIncludes checks if a file path matches a section's include rules.
// This mirrors site.matchesSection but operates on SCC-derived file paths.
func matchesSectionIncludes(filePath string, section site.SectionConfig) bool {
	for _, excl := range section.Exclude {
		if strings.HasPrefix(filePath, excl) {
			return false
		}
	}
	if len(section.Include) == 0 {
		return true
	}
	for _, incl := range section.Include {
		if strings.HasPrefix(filePath, incl) {
			return true
		}
	}
	return false
}

// extractSource returns the source component from an SCC code.
func extractSource(sccCode string) string {
	if idx := strings.Index(sccCode, "/"); idx > 0 {
		return sccCode[:idx]
	}
	return sccCode
}

func (g *SCCAPIGenerator) sortedEntries() []apiEntry {
	keys := make([]string, 0, len(g.entries))
	for k := range g.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	entries := make([]apiEntry, 0, len(keys))
	for _, k := range keys {
		entries = append(entries, g.entries[k])
	}
	return entries
}

func (g *SCCAPIGenerator) writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
