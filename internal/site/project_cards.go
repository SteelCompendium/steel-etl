package site

// Downtime projects use the approved Steel Plate / ledger treatment (SC-201).
// Enhancement names are bold labels, not document sections. Keep their following
// prose, lists and granted ability together until the next enhancement label.
import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var projectEnhancementRe = regexp.MustCompile(`^(\d+)(?:st|nd|rd|th)-Level (?:Armor|Implement|Weapon) Enhancement$`)
var projectLabelRe = regexp.MustCompile(`^\*\*(.+?):\*\*\s*(.*)$`)
var projectQuoteHeadingRe = regexp.MustCompile(`^>\s*#{6,}\s+(.+)$`)
var projectMarkdown = goldmark.New(goldmark.WithExtensions(extension.Table))

var projectLedgerFields = []struct{ key, label string }{
	{"item_prerequisite", "Item Prerequisite"},
	{"project_source", "Project Source"},
	{"project_roll_characteristic", "Project Roll Characteristic"},
	{"project_goal", "Project Goal"},
}

func buildProjectCardPage(data []byte) ([]byte, bool) {
	fm, body := splitFrontmatter(string(data))
	typ := parseFrontmatterField(fm, "type")
	intro, nodes := parseTraitTree(body)
	if typ != "project" {
		// The Imbue Armor/Implement rule leaves also contain enhancement projects.
		// Other rule pages keep their normal transclusion path.
		if typ != "rule" || !hasEnhancementProjects(nodes) {
			return data, false
		}
		return []byte("---\n" + fm + "\n---\n\n" + projectProse(intro) + renderProjectNodes(nodes)), true
	}
	fields := content.ProjectFields(intro)
	// The frontmatter is the data contract; body values carry links already
	// rewritten relative to this page, so prefer them when available.
	for _, f := range projectLedgerFields {
		if fields[f.key] == "" {
			if v := parseFrontmatterField(fm, f.key); v != "" {
				fields[f.key] = v
			}
		}
	}
	name := parseFrontmatterField(fm, "name")
	prose := stripProjectFields(intro)
	children := renderProjectNodes(nodes)
	var card string
	if len(fields) == 0 && len(nodes) > 0 {
		// Delegating projects introduce a collection. Avoid wrapping all nine
		// enhancement projects in an enormous, visually indistinguishable outer box.
		card = projectPlate(name, fields, projectProse(prose), 0) + children
	} else {
		card = projectPlate(name, fields, projectProse(prose)+children, 0)
	}
	return []byte("---\n" + fm + "\n---\n\n" + card), true
}

func hasEnhancementProjects(nodes []*traitNode) bool {
	for _, n := range nodes {
		if projectEnhancementRe.MatchString(n.name) && len(content.ProjectFields(n.content)) == 4 {
			return true
		}
		if hasEnhancementProjects(n.children) {
			return true
		}
	}
	return false
}

func stripProjectFields(body string) string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		if len(content.ProjectFields(line)) > 0 {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func projectProse(body string) string {
	var b bytes.Buffer
	if err := projectMarkdown.Convert([]byte(body), &b); err != nil {
		return html.EscapeString(body)
	}
	return rewriteHrefs(b.String())
}

func renderProjectNodes(nodes []*traitNode) string {
	var out strings.Builder
	for i := 0; i < len(nodes); i++ {
		n := nodes[i]
		fields := content.ProjectFields(n.content)
		if projectEnhancementRe.MatchString(n.name) && len(fields) == 4 {
			body := stripProjectFields(n.content)
			// A typed ability at the H6 cap is a sibling in the source tree. Its
			// immediately preceding enhancement owns the grant (Dragon Soul II).
			var grant *traitNode
			if i+1 < len(nodes) && nodes[i+1].isAbility && strings.Contains(body, "following ability") {
				grant = nodes[i+1]
				i++
			}
			items, count := renderProjectEnhancements(body, grant)
			out.WriteString(projectPlate(n.name, fields, items, count))
			out.WriteString(renderProjectNodes(n.children))
			continue
		}
		level := n.level
		if level < 2 {
			level = 2
		}
		if level > 6 {
			level = 6
		}
		attrs := ""
		if n.scc != "" {
			attrs = ` data-scc="` + html.EscapeString(n.scc) + `"`
		}
		fmt.Fprintf(&out, "<h%d id=\"%s\"%s>%s</h%d>\n", level, content.Slugify(n.name), attrs, traitInline(n.name), level)
		if n.isAbility {
			out.WriteString(renderAbilityCard(synthAbilityFM(n, false), n.content, ""))
		} else {
			out.WriteString(projectProse(n.content))
			out.WriteString(renderProjectNodes(n.children))
		}
	}
	return out.String()
}

func projectPlate(name string, fields map[string]string, body string, count int) string {
	head := cardHeadSlots{Crest: projectCrest, Class: "pj__head", NameTag: "h3", LeftEyebrow: hLine("Downtime Project"), LeftPrimary: hLine(traitInline(name))}
	if m := projectEnhancementRe.FindStringSubmatch(name); m != nil {
		head.RightEyebrow = hChip("Item Level " + m[1])
	}
	if goal := fields["project_goal"]; goal != "" {
		head.RightPrimary = hMini(`Goal <span class="num">` + traitInline(goal) + `</span>`)
	}
	if count > 0 {
		head.RightDeck = hChip(fmt.Sprintf("%d enhancements", count))
	}
	var b strings.Builder
	if count > 0 {
		fmt.Fprintf(&b, "<article class=\"pj\" id=\"%s\">\n", content.Slugify(name))
	} else {
		b.WriteString("<article class=\"pj\">\n")
	}
	b.WriteString(renderCardHead(head))
	if len(fields) > 0 {
		b.WriteString("<dl class=\"pj__ledger\">\n")
		for _, f := range projectLedgerFields[:3] {
			if value := fields[f.key]; value != "" {
				fmt.Fprintf(&b, "<div class=\"row\"><dt>%s</dt><dd>%s</dd></div>\n", f.label, traitInline(value))
			}
		}
		b.WriteString("</dl>\n")
	}
	b.WriteString("<div class=\"pj__prose\">\n" + body + "</div>\n</article>\n")
	// md_in_html must treat the whole card as one raw HTML block.
	return strings.ReplaceAll(b.String(), "\n\n", "\n")
}

func renderProjectEnhancements(body string, trailingGrant *traitNode) (string, int) {
	type item struct {
		name  string
		lines []string
	}
	var items []item
	var intro []string
	for _, line := range strings.Split(body, "\n") {
		m := projectLabelRe.FindStringSubmatch(strings.TrimSpace(line))
		// Power-roll headings are part of the current enhancement, never an item.
		if m != nil && !strings.HasPrefix(content.CleanHeading(m[1]), "Power Roll") && !strings.Contains(m[1], "Power Roll") {
			items = append(items, item{name: m[1], lines: []string{m[2]}})
		} else if len(items) > 0 {
			items[len(items)-1].lines = append(items[len(items)-1].lines, line)
		} else {
			intro = append(intro, line)
		}
	}
	var b strings.Builder
	b.WriteString(projectProse(strings.Join(intro, "\n")))
	b.WriteString("<div class=\"pj__items\">\n")
	for i, it := range items {
		fmt.Fprintf(&b, "<section class=\"pj__item\"><div class=\"pj__item-head\"><span class=\"dia\" aria-hidden=\"true\"></span><span class=\"pj__item-name\">%s</span></div>\n<div class=\"pj__item-body\">\n", traitInline(it.name))
		b.WriteString(renderProjectGrantBody(strings.Join(it.lines, "\n")))
		if i == len(items)-1 && trailingGrant != nil {
			b.WriteString(projectGrant(trailingGrant))
		}
		b.WriteString("</div></section>\n")
	}
	b.WriteString("</div>\n")
	return b.String(), len(items)
}

func projectGrant(n *traitNode) string {
	anchor := ` id="` + html.EscapeString(content.Slugify(n.name)) + `"`
	if n.scc != "" {
		anchor += ` data-scc="` + html.EscapeString(n.scc) + `"`
	}
	return `<div class="pj__grant"` + anchor + `><div class="pj__grant-cap">Grants ability</div>` + renderAbilityCard(synthAbilityFM(n, false), n.content, "") + "</div>\n"
}

func renderProjectGrantBody(body string) string {
	lines := strings.Split(body, "\n")
	var out strings.Builder
	start := 0
	for i := 0; i < len(lines); i++ {
		m := projectQuoteHeadingRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		out.WriteString(projectProse(strings.Join(lines[start:i], "\n")))
		n := &traitNode{name: m[1], isAbility: true}
		var quote []string
		j := i + 1
		for ; j < len(lines); j++ {
			if !strings.HasPrefix(lines[j], ">") {
				break
			}
			quote = append(quote, strings.TrimPrefix(strings.TrimPrefix(lines[j], ">"), " "))
		}
		n.content = strings.Join(quote, "\n")
		out.WriteString(projectGrant(n))
		start = j
		i = j - 1
	}
	out.WriteString(projectProse(strings.Join(lines[start:], "\n")))
	return out.String()
}

// Material Design hammer/wrench crest used in the approved SC-201 prototype.
const projectCrest = `<span class="sc-crest" aria-hidden="true"><span><svg viewBox="0 0 24 24" width="19" height="19" fill="currentColor"><path d="m13.78 15.3 6 6 2.11-2.16-6-6zm3.72-5.2c-.39 0-.81-.05-1.14-.19L4.97 21.25l-2.11-2.11 7.41-7.4L8.5 9.96l-.72.7-1.45-1.41v2.86l-.7.7-3.52-3.56.7-.7h2.81l-1.4-1.41 3.56-3.56a2.976 2.976 0 0 1 4.22 0L9.89 5.74l1.41 1.4-.71.71 1.79 1.78 1.82-1.88c-.14-.33-.2-.75-.2-1.12a3.49 3.49 0 0 1 3.5-3.52c.59 0 1.11.14 1.58.42L16.41 6.2l1.5 1.5 2.67-2.67c.28.47.42.97.42 1.6 0 1.92-1.55 3.47-3.5 3.47"/></svg></span></span>`
