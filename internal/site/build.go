package site

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// SCCMapEntry matches the output of SCCMapGenerator.
type SCCMapEntry struct {
	SCC  string `json:"scc"`
	Path string `json:"path"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// BuildResult holds the outcome of a site build.
type BuildResult struct {
	CopiedFiles   int
	Sections      int
	NavFiles      int
	SearchExclude int
	IndexPages    int
	SCCStubs      int
	Errors        []string
}

// Build generates the MkDocs site structure from steel-etl output.
func Build(cfg *Config) (*BuildResult, error) {
	result := &BuildResult{}

	// Clean docs dir (except protected paths)
	if err := cleanDocsDir(cfg.DocsDir); err != nil {
		return nil, fmt.Errorf("clean docs: %w", err)
	}

	// Read scc-to-path.json if available (for metadata), but primarily
	// we walk the source directory to copy files
	entries, err := walkSourceDirs(cfg.SourceDirList())
	if err != nil {
		return nil, fmt.Errorf("walk source: %w", err)
	}

	// Map files to sections
	for _, section := range cfg.Sections {
		count, errs := buildSection(cfg, section, entries)
		result.CopiedFiles += count
		result.Errors = append(result.Errors, errs...)
		result.Sections++
	}

	// Write .nav.yml files
	for _, section := range cfg.Sections {
		if section.GroupByBook {
			continue // ordered per-book nav is written by writeBookNavAndIndexes
		}
		if err := writeNavYaml(cfg.DocsDir, section); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("nav %s: %v", section.Name, err))
		} else {
			result.NavFiles++
		}
	}

	// Per-book ordered nav + indexes for GroupByBook sections.
	for _, section := range cfg.Sections {
		if !section.GroupByBook {
			continue
		}
		n, errs := writeBookNavAndIndexes(cfg, section)
		result.NavFiles += n
		result.Errors = append(result.Errors, errs...)
	}

	// Generate index pages for type directories (skip GroupByBook sections —
	// those get ordered indexes from writeBookNavAndIndexes).
	var genericSections []SectionConfig
	for _, s := range cfg.Sections {
		if !s.GroupByBook {
			genericSections = append(genericSections, s)
		}
	}
	indexCount, indexErrs := generateIndexPages(cfg.DocsDir, genericSections)
	result.IndexPages = indexCount
	result.Errors = append(result.Errors, indexErrs...)

	// Apply search exclusion
	for _, sectionName := range cfg.SearchExclude {
		count, errs := applySearchExclusion(cfg.DocsDir, sectionName)
		result.SearchExclude += count
		result.Errors = append(result.Errors, errs...)
	}

	// Copy static content overrides
	if cfg.StaticContent != "" {
		count, err := copyStaticContent(cfg.StaticContent, cfg.DocsDir)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("static content: %v", err))
		}
		result.CopiedFiles += count
	}

	// Generate SCC permalink stubs. Runs last so it sees the final, post-static
	// frontmatter (static_content overrides may inject scc values too).
	stubCount, stubErrs := generateSCCStubs(cfg.DocsDir)
	result.SCCStubs = stubCount
	result.Errors = append(result.Errors, stubErrs...)

	return result, nil
}

// sourceEntry represents a markdown file found in a source directory.
type sourceEntry struct {
	relPath   string // relative to its source dir (e.g., "class/fury.md")
	absPath   string
	sourceDir string // the source dir this entry came from
}

func walkSourceDir(dir string) ([]sourceEntry, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil // a configured book may not have generated output yet
	}
	var entries []sourceEntry
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		entries = append(entries, sourceEntry{relPath: rel, absPath: path, sourceDir: dir})
		return nil
	})
	return entries, err
}

// walkSourceDirs merges entries from multiple source dirs (later dirs append).
func walkSourceDirs(dirs []string) ([]sourceEntry, error) {
	var all []sourceEntry
	for _, d := range dirs {
		entries, err := walkSourceDir(d)
		if err != nil {
			return nil, err
		}
		all = append(all, entries...)
	}
	return all, nil
}

// buildSection copies matching files from source into the section directory.
func buildSection(cfg *Config, section SectionConfig, entries []sourceEntry) (int, []string) {
	sectionDir := filepath.Join(cfg.DocsDir, section.Name)
	count := 0
	var errs []string

	for _, entry := range entries {
		if !matchesSection(entry.relPath, section) {
			continue
		}

		data, err := os.ReadFile(entry.absPath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("read %s: %v", entry.absPath, err))
			continue
		}

		// The page's book folder (from its scc prefix) — used both for
		// GroupByBook placement and for resolving links that target chapters in
		// a GroupByBook section (cross-references stay within the same book).
		fm, _ := splitFrontmatter(string(data))
		srcBookFolder := ""
		if book, ok := cfg.BookByKey(bookKeyFromSCC(parseFrontmatterField(fm, "scc"))); ok {
			srcBookFolder = book.Folder
		}

		// Determine destination path within the section. GroupByBook sections
		// place pages into a per-book folder; other sections apply SCC-type
		// group remaps.
		var destRel, parentName string
		if dest, ok := groupLandingIndexDest(entry.relPath); ok {
			// Group landing (skill.group/* , monster.group/*) renders AS the
			// <root>/<member>/ index; mergeGroupLanding folds it above the listing.
			destRel = dest
		} else if section.GroupByBook {
			if srcBookFolder == "" {
				key := bookKeyFromSCC(parseFrontmatterField(fm, "scc"))
				errs = append(errs, fmt.Sprintf("no book config for scc prefix %q (%s)", key, entry.relPath))
				continue
			}
			destRel = filepath.ToSlash(filepath.Join(srcBookFolder, filepath.Base(entry.relPath)))
		} else {
			destRel, parentName = applyGroups(entry.relPath, section.Groups, entry.sourceDir)
		}
		destPath := filepath.Join(sectionDir, destRel)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			errs = append(errs, fmt.Sprintf("mkdir %s: %v", destPath, err))
			continue
		}

		data = []byte(rewriteSectionLinks(string(data), entry.relPath, destRel, section.Name, srcBookFolder, cfg.Sections))

		// When a group flattens parent/child into one file, rewrite the
		// frontmatter "name" to combine parent + original name so the H1
		// and mkdocs nav title both show the combined form.
		if parentName != "" {
			data = combineFrontmatterName(data, parentName)
		}

		// Ability/trait pages → high-fantasy steel `.sc-ability` card (site-only;
		// shared data repos untouched). Runs before injectH1 so the card becomes
		// the page body and injectH1 still prepends the "# Name" MkDocs needs.
		if card, ok := buildAbilityCardPage(data); ok {
			data = card
		}

		// Inject h1 header from frontmatter "name" field if the body lacks one
		data = injectH1(data)

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			errs = append(errs, fmt.Sprintf("write %s: %v", destPath, err))
			continue
		}
		count++
	}

	return count, errs
}

// chapterRef is a chapter file with its display name and source order.
type chapterRef struct {
	file  string // basename, e.g. "rewards.md"
	name  string // frontmatter name, e.g. "Rewards"
	order int
	blurb string // first prose paragraph of the chapter body (for the card)
}

// writeBookNavAndIndexes emits, for a GroupByBook section: one ordered .nav.yml
// + index.md per book folder, and a top-level section .nav.yml + index.md that
// lists the books in Book.Order.
func writeBookNavAndIndexes(cfg *Config, section SectionConfig) (int, []string) {
	sectionDir := filepath.Join(cfg.DocsDir, section.Name)
	var errs []string
	navCount := 0

	// Books that actually produced a folder, in Book.Order.
	books := append([]BookConfig(nil), cfg.Books...)
	sort.SliceStable(books, func(i, j int) bool { return books[i].Order < books[j].Order })

	// Every configured book is shown (in Book.Order). A book with no generated
	// chapters still gets a folder + placeholder index so it appears in the tab.
	present := books
	for _, b := range books {
		bookDir := filepath.Join(sectionDir, b.Folder)
		if err := os.MkdirAll(bookDir, 0755); err != nil {
			errs = append(errs, fmt.Sprintf("book dir %s: %v", b.Folder, err))
			continue
		}

		// Collect chapter files (skip index.md) with name + order.
		var chapters []chapterRef
		dirEntries, _ := os.ReadDir(bookDir)
		for _, e := range dirEntries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "index.md" {
				continue
			}
			fm, body := splitFrontmatter(readFile(filepath.Join(bookDir, e.Name())))
			name := parseFrontmatterField(fm, "name")
			if name == "" {
				name = fileToTitle(e.Name())
			}
			chapters = append(chapters, chapterRef{
				file:  e.Name(),
				name:  name,
				order: parseFrontmatterInt(fm, "order", 1<<30),
				blurb: bodyBlurb(body, 200),
			})
		}
		sort.SliceStable(chapters, func(i, j int) bool {
			if chapters[i].order != chapters[j].order {
				return chapters[i].order < chapters[j].order
			}
			return naturalLess(chapters[i].file, chapters[j].file)
		})

		// Per-book .nav.yml: explicit ordered list (index first).
		var nb strings.Builder
		nb.WriteString("title: " + yamlScalar(b.Label) + "\n")
		nb.WriteString("nav:\n")
		nb.WriteString("  - index.md\n")
		for _, c := range chapters {
			nb.WriteString("  - " + c.file + "\n")
		}
		if err := os.WriteFile(filepath.Join(bookDir, ".nav.yml"), []byte(nb.String()), 0644); err != nil {
			errs = append(errs, fmt.Sprintf("book nav %s: %v", b.Folder, err))
		} else {
			navCount++
		}

		// Per-book index.md: ordered chapter cards, or a placeholder when the
		// book has no chapters yet.
		var ib strings.Builder
		ib.WriteString("# " + b.Label + "\n\n---\n\n")
		if len(chapters) == 0 {
			ib.WriteString("*Chapters for this book haven't been added to the compendium yet.*\n")
		} else {
			ib.WriteString("<div class=\"sc-cards\">\n")
			for _, c := range chapters {
				ib.WriteString(chapterCard(c.file, c.name, c.blurb))
			}
			ib.WriteString("</div>\n")
		}
		if err := os.WriteFile(filepath.Join(bookDir, "index.md"), []byte(ib.String()), 0644); err != nil {
			errs = append(errs, fmt.Sprintf("book index %s: %v", b.Folder, err))
		}
	}

	// Section-level .nav.yml: title + ordered book folders (index first).
	title := section.Title
	if title == "" {
		title = section.Name
	}
	var sb strings.Builder
	sb.WriteString("title: " + yamlScalar(title) + "\n")
	sb.WriteString("nav:\n")
	sb.WriteString("  - index.md\n")
	for _, b := range present {
		sb.WriteString("  - " + b.Folder + "\n")
	}
	if err := os.WriteFile(filepath.Join(sectionDir, ".nav.yml"), []byte(sb.String()), 0644); err != nil {
		errs = append(errs, fmt.Sprintf("section nav %s: %v", section.Name, err))
	} else {
		navCount++
	}

	// Section landing index.md: a card per book. (Search exclusion frontmatter
	// is injected later by applySearchExclusion for search-excluded sections.)
	var lb strings.Builder
	lb.WriteString("# " + title + "\n\n---\n\n<div class=\"sc-cards\">\n")
	for _, b := range present {
		lb.WriteString(bookCard(b))
	}
	lb.WriteString("</div>\n")
	if err := os.WriteFile(filepath.Join(sectionDir, "index.md"), []byte(lb.String()), 0644); err != nil {
		errs = append(errs, fmt.Sprintf("section index %s: %v", section.Name, err))
	}

	return navCount, errs
}

// readFile reads a file, returning "" on error (best-effort frontmatter reads).
func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// yamlScalar quotes a YAML scalar if it contains characters that need quoting.
func yamlScalar(s string) string {
	if strings.ContainsAny(s, ":#\"'{}[],&*?|<>=!%@`") {
		return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
	}
	return s
}

// applyGroups remaps a file's relative path based on group rules.
// For example, "feature/ability/arcane-archer/exploding-arrow.md" becomes
// "feature/ability/Kits/arcane-archer/exploding-arrow.md" if arcane-archer
// matches a file in the kit/ source directory.
//
// When the matched group has Flatten=true, parent/child paths collapse to
// "parent-child.md" directly under Label/, and the parent component name
// is returned so callers can rewrite the file's frontmatter name accordingly.
// parentName is empty when no flattening occurred.
func applyGroups(relPath string, groups []GroupConfig, sourceDir string) (newPath string, parentName string) {
	if len(groups) == 0 {
		return relPath, ""
	}

	normalized := filepath.ToSlash(relPath)

	for _, g := range groups {
		prefix := g.From + "/"
		if !strings.HasPrefix(normalized, prefix) {
			continue
		}

		// Extract the first path component after the prefix (e.g., "arcane-archer")
		rest := normalized[len(prefix):]
		component := rest
		if idx := strings.Index(rest, "/"); idx >= 0 {
			component = rest[:idx]
		}

		// Cross-reference: does {match_type}/{component}.md exist in source?
		checkPath := filepath.Join(sourceDir, g.MatchType, component+".md")
		if _, err := os.Stat(checkPath); err != nil {
			continue
		}

		if g.Flatten {
			// Collapse parent/child.md → parent-child.md under Label/.
			// If the file IS the parent (no child segment), keep it as parent.md.
			if component == rest || rest == component+".md" {
				return g.From + "/" + g.Label + "/" + component + ".md", ""
			}
			child := strings.TrimSuffix(filepath.Base(rest), ".md")
			return g.From + "/" + g.Label + "/" + component + "-" + child + ".md", component
		}

		// Remap: insert group label between prefix and rest
		return g.From + "/" + g.Label + "/" + rest, ""
	}

	return relPath, ""
}

// combineFrontmatterName rewrites the "name" field in frontmatter so that
// the value becomes "ParentTitle (OriginalName)". Used by the flatten group
// mode where parent/child pages collapse into one page. The parent slug
// (e.g. "arcane-archer") is title-cased to "Arcane Archer".
func combineFrontmatterName(data []byte, parentSlug string) []byte {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return data
	}

	fm, body := splitFrontmatter(content)
	original := parseFrontmatterField(fm, "name")
	if original == "" {
		return data
	}

	parentTitle := titleCase(strings.ReplaceAll(parentSlug, "-", " "))
	combined := parentTitle + " (" + original + ")"

	newFM := replaceFrontmatterField(fm, "name", combined)

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(newFM)
	sb.WriteString("\n---")
	sb.WriteString(body)
	return []byte(sb.String())
}

// replaceFrontmatterField replaces the value of a simple top-level scalar
// field in YAML frontmatter. Indented (nested) keys are not matched.
func replaceFrontmatterField(fm, key, value string) string {
	lines := strings.Split(fm, "\n")
	prefix := key + ":"
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed != line {
			continue // skip indented (nested) keys
		}
		if strings.HasPrefix(trimmed, prefix) {
			lines[i] = key + ": " + value
			return strings.Join(lines, "\n")
		}
	}
	return fm
}

// groupLandingIndexDest maps a unified group-landing source path to the group's
// index page:
//
//	<root>/group/<member>.md   ->   <root>/<member>/index.md
//
// So a skill.group/crafting page (file skill/group/crafting.md) renders AS the
// /Browse/skill/crafting/ index — carrying its scc to the permalink stub — and no
// phantom <root>/group/ subtree is ever created. ok=false for anything else.
func groupLandingIndexDest(relPath string) (string, bool) {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) == 3 && parts[1] == "group" && strings.HasSuffix(parts[2], ".md") {
		member := strings.TrimSuffix(parts[2], ".md")
		return parts[0] + "/" + member + "/index.md", true
	}
	return "", false
}

// mergeGroupLanding folds a relocated group-landing page (placed at dir/index.md
// by buildSection, carrying scc frontmatter + lore) into the generated index
// `generated` (card grid for skills, browse list for monsters). It preserves the
// landing's frontmatter — so the scc permalink stub targets THIS dir — and its
// lore, drops the generated listing's duplicate leading "# Title\n\n---\n\n", and
// strips any trailing table in the lore that the listing below supersedes. If
// dir/index.md is absent or has no scc, `generated` is returned unchanged.
func mergeGroupLanding(dir, generated string) string {
	data, err := os.ReadFile(filepath.Join(dir, "index.md"))
	if err != nil {
		return generated
	}
	fm, body := splitFrontmatter(string(data))
	if parseFrontmatterField(fm, "scc") == "" {
		return generated
	}
	lore := stripTrailingTable(strings.TrimRight(body, "\n"))
	listing := stripLeadingHeading(generated)

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fm)
	sb.WriteString("\n---\n")
	sb.WriteString(strings.TrimLeft(lore, "\n"))
	sb.WriteString("\n\n---\n\n")
	sb.WriteString(listing)
	return sb.String()
}

// stripLeadingHeading drops the "# Title\n\n---\n\n" head that generated index
// content begins with, so a merged landing keeps only ITS own H1.
func stripLeadingHeading(s string) string {
	const sep = "\n---\n\n"
	if strings.HasPrefix(s, "# ") {
		if i := strings.Index(s, sep); i >= 0 {
			return s[i+len(sep):]
		}
	}
	return s
}

// stripTrailingTable removes a trailing GFM table (and its blank separator) from
// a group landing's lore — the index listing below already enumerates those rows.
func stripTrailingTable(body string) string {
	lines := strings.Split(body, "\n")
	end := len(lines)
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	start := end
	for start > 0 && strings.HasPrefix(strings.TrimSpace(lines[start-1]), "|") {
		start--
	}
	if start == end { // no trailing table
		return body
	}
	for start > 0 && strings.TrimSpace(lines[start-1]) == "" {
		start-- // drop the blank line before the table
	}
	return strings.TrimRight(strings.Join(lines[:start], "\n"), "\n")
}

// injectH1 adds a "# Name" header after frontmatter if the body doesn't already
// have one. Reads the name from the frontmatter "name" field.
func injectH1(data []byte) []byte {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return data
	}

	fm, body := splitFrontmatter(content)
	name := parseFrontmatterField(fm, "name")
	if name == "" {
		return data
	}

	// If body already starts with an h1, inject hr after it if missing
	trimmed := strings.TrimLeft(body, "\n")
	if strings.HasPrefix(trimmed, "# ") {
		return injectHRAfterH1(data)
	}

	// Rebuild: frontmatter + h1 + hr + body
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fm)
	sb.WriteString("\n---\n\n# ")
	sb.WriteString(name)
	sb.WriteString("\n\n---\n\n")
	sb.WriteString(strings.TrimLeft(body, "\n"))
	return []byte(sb.String())
}

// injectHRAfterH1 finds the first "# ..." line in the content and inserts
// a markdown hr (---) on the line after it, unless one is already there.
func injectHRAfterH1(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "# ") {
			continue
		}
		// Check if an hr already follows (skip blank lines)
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "" {
				continue
			}
			if strings.TrimSpace(lines[j]) == "---" {
				return data
			}
			break
		}
		result := make([]string, 0, len(lines)+2)
		result = append(result, lines[:i+1]...)
		result = append(result, "", "---")
		result = append(result, lines[i+1:]...)
		return []byte(strings.Join(result, "\n"))
	}
	return data
}

// matchesSection checks if a file's relative path matches the section's include/exclude rules.
func matchesSection(relPath string, section SectionConfig) bool {
	// Normalize path separators
	normalized := filepath.ToSlash(relPath)

	// Check excludes first
	for _, excl := range section.Exclude {
		if strings.HasPrefix(normalized, excl) {
			return false
		}
	}

	// If no includes, match everything
	if len(section.Include) == 0 {
		return true
	}

	// Check includes
	for _, incl := range section.Include {
		if strings.HasPrefix(normalized, incl) {
			return true
		}
	}

	return false
}

// writeNavYaml creates a .nav.yml file for the section.
func writeNavYaml(docsDir string, section SectionConfig) error {
	sectionDir := filepath.Join(docsDir, section.Name)
	if _, err := os.Stat(sectionDir); os.IsNotExist(err) {
		return nil // section dir doesn't exist, skip
	}

	nav := map[string]any{}
	if section.Title != "" {
		nav["title"] = section.Title
	}
	if section.Sort != "" {
		nav["sort"] = map[string]string{
			"type": section.Sort,
			"by":   "title",
		}
	}

	// If the only content is title or sort, use simpler YAML format
	data, err := yaml.Marshal(nav)
	if err != nil {
		return fmt.Errorf("marshal nav: %w", err)
	}

	return os.WriteFile(filepath.Join(sectionDir, ".nav.yml"), data, 0644)
}

// applySearchExclusion adds search: exclude: true frontmatter to all .md files in a section.
func applySearchExclusion(docsDir, sectionName string) (int, []string) {
	sectionDir := filepath.Join(docsDir, sectionName)
	if _, err := os.Stat(sectionDir); os.IsNotExist(err) {
		return 0, nil
	}

	count := 0
	var errs []string

	filepath.Walk(sectionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("read %s: %v", path, err))
			return nil
		}

		content := string(data)
		if strings.HasPrefix(content, "---\n") {
			// Has frontmatter — inject search exclude after opening ---
			rest := content[4:]
			content = "---\nsearch:\n  exclude: true\n" + rest
		} else {
			// No frontmatter — prepend it
			content = "---\nsearch:\n  exclude: true\n---\n\n" + content
		}

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			errs = append(errs, fmt.Sprintf("write %s: %v", path, err))
			return nil
		}
		count++
		return nil
	})

	return count, errs
}

// cleanDocsDir removes generated content from the docs directory,
// preserving protected paths (stylesheets, javascripts, Media, index.md, etc.)
func cleanDocsDir(docsDir string) error {
	protected := map[string]bool{
		"javascripts":    true,
		"stylesheets":    true,
		"Media":          true,
		"index.md":       true,
		"preferences.md": true,
		".nav.yml":       true,
	}

	entries, err := os.ReadDir(docsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(docsDir, 0755)
		}
		return err
	}

	for _, entry := range entries {
		if protected[entry.Name()] {
			continue
		}
		path := filepath.Join(docsDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}

	return nil
}

// copyStaticContent copies static content overrides into the docs directory.
func copyStaticContent(srcDir, docsDir string) (int, error) {
	count := 0
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		rel, _ := filepath.Rel(srcDir, path)
		dest := filepath.Join(docsDir, rel)

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.WriteFile(dest, data, 0644); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

// mdRelLinkRe matches markdown links with relative paths (not http(s), not anchors, not absolute).
var mdRelLinkRe = regexp.MustCompile(`(\[[^\]]*\])\(([^):#][^):]*\.md)\)`)

// rewriteSectionLinks adjusts relative markdown links so they resolve correctly
// after files are placed under section directories (e.g., Browse/, Read/).
// Links in the source files were computed relative to the flat ETL output;
// cross-section links need new relative paths that traverse section boundaries.
func rewriteSectionLinks(content, srcRelPath, destRelPath, sectionName, srcBookFolder string, allSections []SectionConfig) string {
	srcDir := filepath.ToSlash(filepath.Dir(srcRelPath))
	destDir := filepath.ToSlash(filepath.Dir(filepath.Join(sectionName, destRelPath)))

	return mdRelLinkRe.ReplaceAllStringFunc(content, func(match string) string {
		sub := mdRelLinkRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		linkText := sub[1]
		linkPath := sub[2]

		rootRel := filepath.ToSlash(filepath.Clean(filepath.Join(srcDir, linkPath)))

		targetSection := ""
		targetGroupByBook := false
		for _, section := range allSections {
			if matchesSection(rootRel, section) {
				targetSection = section.Name
				targetGroupByBook = section.GroupByBook
				break
			}
		}

		if targetSection == "" {
			return match
		}

		// GroupByBook sections flatten SCC type paths (e.g. "chapter/x.md") into
		// per-book folders ("<book>/x.md"). Cross-references to a chapter stay
		// within the source page's book, so resolve the target under that folder.
		var targetFull string
		if targetGroupByBook && srcBookFolder != "" {
			targetFull = filepath.ToSlash(filepath.Join(targetSection, srcBookFolder, filepath.Base(rootRel)))
		} else {
			targetFull = filepath.ToSlash(filepath.Join(targetSection, rootRel))
		}
		newRel, err := filepath.Rel(destDir, targetFull)
		if err != nil {
			return match
		}

		return linkText + "(" + filepath.ToSlash(newRel) + ")"
	})
}

// splitFrontmatter separates YAML frontmatter from body content.
func splitFrontmatter(content string) (frontmatter, body string) {
	if !strings.HasPrefix(content, "---\n") {
		return "", content
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return "", content
	}
	return content[4 : 4+end], content[4+end+4:]
}

// parseFrontmatterField extracts a simple string value from YAML frontmatter.
func parseFrontmatterField(fm, key string) string {
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		prefix := key + ":"
		if strings.HasPrefix(line, prefix) {
			val := strings.TrimSpace(line[len(prefix):])
			val = strings.Trim(val, "\"'")
			return val
		}
	}
	return ""
}

// bookKeyFromSCC returns the book prefix of an SCC code (substring before the
// first '/'); returns the input unchanged when there is no '/'.
func bookKeyFromSCC(scc string) string {
	if i := strings.Index(scc, "/"); i >= 0 {
		return scc[:i]
	}
	return scc
}

// parseFrontmatterInt extracts an integer scalar from frontmatter, or def if
// absent/unparseable.
func parseFrontmatterInt(fm, key string, def int) int {
	v := parseFrontmatterField(fm, key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return n
}

// naturalLess compares two strings with numeric-aware ordering,
// so "level-2" sorts before "level-10".
func naturalLess(a, b string) bool {
	ia, ib := 0, 0
	for ia < len(a) && ib < len(b) {
		ca, cb := a[ia], b[ib]
		aDigit := ca >= '0' && ca <= '9'
		bDigit := cb >= '0' && cb <= '9'

		if aDigit && bDigit {
			// Compare numeric spans as integers
			na, nb := 0, 0
			for ia < len(a) && a[ia] >= '0' && a[ia] <= '9' {
				na = na*10 + int(a[ia]-'0')
				ia++
			}
			for ib < len(b) && b[ib] >= '0' && b[ib] <= '9' {
				nb = nb*10 + int(b[ib]-'0')
				ib++
			}
			if na != nb {
				return na < nb
			}
		} else {
			// Compare single characters (case-insensitive)
			la, lb := ca, cb
			if la >= 'A' && la <= 'Z' {
				la += 'a' - 'A'
			}
			if lb >= 'A' && lb <= 'Z' {
				lb += 'a' - 'A'
			}
			if la != lb {
				return la < lb
			}
			ia++
			ib++
		}
	}
	return len(a) < len(b)
}

// typeTitles maps lowercase type directory names to display titles.
var typeTitles = map[string]string{
	"ancestry":     "Ancestries",
	"career":       "Careers",
	"chapter":      "Chapters",
	"class":        "Classes",
	"complication": "Complications",
	"condition":    "Conditions",
	"culture":      "Cultures",
	"feature":      "Features",
	"kit":          "Kits",
	"perk":         "Perks",
	"skill":        "Skills",
	"title":        "Titles",
	"treasure":     "Treasures",
	"ability":      "Abilities",
	"trait":        "Traits",
}

// generateIndexPages creates index.md files for type directories within sections.
func generateIndexPages(docsDir string, sections []SectionConfig) (int, []string) {
	count := 0
	var errs []string

	for _, section := range sections {
		sectionDir := filepath.Join(docsDir, section.Name)
		if _, err := os.Stat(sectionDir); os.IsNotExist(err) {
			continue
		}
		n, e := generateIndexesRecursive(sectionDir, sectionDir)
		count += n
		errs = append(errs, e...)
	}

	return count, errs
}

// generateIndexesRecursive creates index.md files for directories that contain
// .md files or subdirectories with content.
func generateIndexesRecursive(dir, sectionRoot string) (int, []string) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return 0, []string{fmt.Sprintf("read dir %s: %v", dir, err)}
	}

	var files []string
	var subdirs []string

	for _, e := range dirEntries {
		name := e.Name()
		if e.IsDir() {
			subdirs = append(subdirs, name)
		} else if strings.HasSuffix(name, ".md") && name != "index.md" && name != "_Index.md" {
			files = append(files, name)
		}
	}

	count := 0
	var errs []string

	// Recurse into subdirectories
	for _, d := range subdirs {
		n, e := generateIndexesRecursive(filepath.Join(dir, d), sectionRoot)
		count += n
		errs = append(errs, e...)
	}

	// Don't generate index for the section root — that's provided by static content
	if dir == sectionRoot {
		return count, errs
	}

	// Skip if nothing to list
	if len(files) == 0 && len(subdirs) == 0 {
		return count, errs
	}

	content := buildIndexContent(dir, filepath.Base(dir), files, subdirs)
	content = mergeGroupLanding(dir, content)
	indexPath := filepath.Join(dir, "index.md")
	if err := os.WriteFile(indexPath, []byte(content), 0644); err != nil {
		errs = append(errs, fmt.Sprintf("write index %s: %v", indexPath, err))
	} else {
		count++
	}

	return count, errs
}

// buildIndexContent generates the markdown content for a directory index page.
// dir is the absolute directory; used to read frontmatter "name" fields from
// the listed files so the index labels match the pages' actual titles.
func buildIndexContent(dir, dirName string, files, subdirs []string) string {
	// Rich stat-cards for supported index types (kit, …); falls back below.
	if cards, ok := buildCardsContent(dir, dirName, files, subdirs); ok {
		return cards
	}
	// Folder cards (index-of-indexes) + trait/ability preview cards
	// (parent-of-leaves) for the nested feature & treasure trees.
	if idx, ok := buildFeatureIndexContent(dir, dirName, files, subdirs); ok {
		return idx
	}
	title := dirToTitle(dirName)

	sort.Slice(files, func(i, j int) bool { return naturalLess(files[i], files[j]) })
	sort.Slice(subdirs, func(i, j int) bool { return naturalLess(subdirs[i], subdirs[j]) })

	var sb strings.Builder
	sb.WriteString("# ")
	sb.WriteString(title)
	sb.WriteString("\n\n---\n\n")

	var plainSubdirs []string
	for _, d := range subdirs {
		name := dirToTitle(d)
		childFiles, childSubdirs := listDirChildren(filepath.Join(dir, d))
		if len(childFiles) == 0 && len(childSubdirs) == 0 {
			plainSubdirs = append(plainSubdirs, d)
			continue
		}
		sb.WriteString("<details class=\"browse-expand\" markdown>\n")
		sb.WriteString("<summary><a href=\"")
		sb.WriteString(d)
		sb.WriteString("/\">")
		sb.WriteString(name)
		sb.WriteString("</a></summary>\n\n")
		sb.WriteString("<div class=\"browse-index\" markdown>\n\n")
		for _, sd := range childSubdirs {
			sdName := dirToTitle(sd)
			sb.WriteString("- [")
			sb.WriteString(sdName)
			sb.WriteString("](")
			sb.WriteString(d)
			sb.WriteString("/")
			sb.WriteString(sd)
			sb.WriteString("/)\n")
		}
		for _, cf := range childFiles {
			cfName := readFrontmatterName(filepath.Join(dir, d, cf))
			if cfName == "" {
				cfName = fileToTitle(cf)
			}
			sb.WriteString("- [")
			sb.WriteString(cfName)
			sb.WriteString("](")
			sb.WriteString(d)
			sb.WriteString("/")
			sb.WriteString(cf)
			sb.WriteString(")\n")
		}
		sb.WriteString("\n</div>\n\n")
		sb.WriteString("</details>\n\n")
	}

	if len(plainSubdirs) > 0 || len(files) > 0 {
		sb.WriteString("<div class=\"browse-index\" markdown>\n\n")
		for _, d := range plainSubdirs {
			name := dirToTitle(d)
			sb.WriteString("- [")
			sb.WriteString(name)
			sb.WriteString("](")
			sb.WriteString(d)
			sb.WriteString("/)\n")
		}
		for _, f := range files {
			name := readFrontmatterName(filepath.Join(dir, f))
			if name == "" {
				name = fileToTitle(f)
			}
			sb.WriteString("- [")
			sb.WriteString(name)
			sb.WriteString("](")
			sb.WriteString(f)
			sb.WriteString(")\n")
		}
		sb.WriteString("\n</div>\n")
	}

	return sb.String()
}

// readFrontmatterName returns the "name" field from a markdown file's
// frontmatter, or "" if the file lacks frontmatter or a name field.
func readFrontmatterName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	fm, _ := splitFrontmatter(string(data))
	if fm == "" {
		return ""
	}
	return parseFrontmatterField(fm, "name")
}

// listDirChildren returns the sorted .md files and subdirectories
// immediately inside dir (one level only, excluding index.md).
func listDirChildren(dir string) (files, subdirs []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			subdirs = append(subdirs, name)
		} else if strings.HasSuffix(name, ".md") && name != "index.md" && name != "_Index.md" {
			files = append(files, name)
		}
	}
	sort.Slice(files, func(i, j int) bool { return naturalLess(files[i], files[j]) })
	sort.Slice(subdirs, func(i, j int) bool { return naturalLess(subdirs[i], subdirs[j]) })
	return files, subdirs
}

// dirToTitle converts a directory name to a display title.
func dirToTitle(name string) string {
	if t, ok := typeTitles[name]; ok {
		return t
	}
	return titleCase(strings.ReplaceAll(name, "-", " "))
}

// fileToTitle converts a filename (without path) to a display title.
func fileToTitle(name string) string {
	name = strings.TrimSuffix(name, ".md")
	return titleCase(strings.ReplaceAll(name, "-", " "))
}

// titleCase capitalizes the first letter of each word.
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// LoadSCCMap reads scc-to-path.json and returns the entries sorted by SCC code.
func LoadSCCMap(path string) ([]SCCMapEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read scc map %s: %w", path, err)
	}

	var entries []SCCMapEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse scc map: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SCC < entries[j].SCC
	})

	return entries, nil
}
