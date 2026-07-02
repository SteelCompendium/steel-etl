package site

// Per-class "all abilities" table for the feature/ability/<class>/ index
// pages: one sortable row per ability leaf across every level bucket, so a
// player can survey a class's abilities without clicking into each level
// folder. Sorting is client-side via the site-wide tablesort.js. SITE-ONLY.
// See workspace docs/superpowers/plans/2026-07-01-p5-class-ability-table.md.

import (
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// isAbilityClassDir reports whether dir is a per-class ability bucket dir:
// …/feature/ability/<class> (exactly one path segment after "ability").
func isAbilityClassDir(dir string) bool {
	parts := strings.Split(filepath.ToSlash(dir), "/")
	for i, p := range parts {
		if p == "feature" && i+1 < len(parts) && parts[i+1] == "ability" && i+2 == len(parts)-1 {
			return true
		}
	}
	return false
}

type abilityRow struct {
	name, href, level, cost, action, distance, target string
}

// abilityTable renders the sortable ability table for a class dir, reading
// each leaf's frontmatter. Returns "" when the dir holds no ability leaves.
// Row hrefs are directory URLs relative to the index page (raw-HTML hrefs are
// not rewritten by mkdocs, so "<subdir>/<slug>/" is the correct form).
func abilityTable(dir string, subdirs []string) string {
	var rows []abilityRow
	for _, sub := range subdirs {
		entries, err := os.ReadDir(filepath.Join(dir, sub))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") ||
				e.Name() == "index.md" || e.Name() == "_Index.md" {
				continue
			}
			fm, _ := splitFrontmatter(readFile(filepath.Join(dir, sub, e.Name())))
			if strings.TrimSpace(parseFrontmatterField(fm, "type")) != "ability" {
				continue
			}
			cost := stripMD(unquote(strings.TrimSpace(parseFrontmatterField(fm, "cost"))))
			if cost == "" && strings.TrimSpace(parseFrontmatterField(fm, "subtype")) == "signature" {
				cost = "Signature"
			}
			rows = append(rows, abilityRow{
				name:     stripMD(parseFrontmatterField(fm, "name")),
				href:     sub + "/" + strings.TrimSuffix(e.Name(), ".md") + "/",
				level:    unquote(strings.TrimSpace(parseFrontmatterField(fm, "level"))),
				cost:     cost,
				action:   stripMD(unquote(strings.TrimSpace(parseFrontmatterField(fm, "action_type")))),
				distance: stripMD(unquote(strings.TrimSpace(parseFrontmatterField(fm, "distance")))),
				target:   stripMD(unquote(strings.TrimSpace(parseFrontmatterField(fm, "target")))),
			})
		}
	}
	if len(rows) == 0 {
		return ""
	}
	sort.Slice(rows, func(i, j int) bool {
		li, lj := rows[i].level, rows[j].level
		if (li == "") != (lj == "") {
			return lj == "" // unleveled abilities (e.g. stormwight kits) sort last
		}
		if li != lj {
			return naturalLess(li, lj)
		}
		return naturalLess(rows[i].name, rows[j].name)
	})

	dash := func(s string) string {
		if s == "" {
			return "—"
		}
		return html.EscapeString(s)
	}
	var sb strings.Builder
	sb.WriteString(`<div class="sc-abtable">` + "\n")
	sb.WriteString(`<table><thead><tr><th>Ability</th><th>Lv</th><th>Cost</th>` +
		`<th>Action</th><th>Distance</th><th>Target</th></tr></thead><tbody>` + "\n")
	for _, r := range rows {
		lvlSort := r.level
		if lvlSort == "" {
			lvlSort = "0" // deterministic tablesort key for unleveled rows
		}
		sb.WriteString(`<tr><td><a href="` + html.EscapeString(r.href) + `">` +
			html.EscapeString(r.name) + `</a></td>` +
			`<td data-sort="` + html.EscapeString(lvlSort) + `">` + dash(r.level) + `</td>` +
			`<td>` + dash(r.cost) + `</td>` +
			`<td>` + dash(r.action) + `</td>` +
			`<td>` + dash(r.distance) + `</td>` +
			`<td>` + dash(r.target) + `</td></tr>` + "\n")
	}
	sb.WriteString("</tbody></table>\n</div>\n")
	return sb.String()
}
