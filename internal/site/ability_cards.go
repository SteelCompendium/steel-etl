package site

// High-fantasy steel ABILITY / TRAIT cards for the Steel Compendium MkDocs site.
//
// Where cards.go rewrites the category *index* pages, this rewrites each
// standalone ability/trait *page body* into the `.sc-ability` card (heraldic
// crest keyed to action type, Distance/Targets rail, a cohesive Power-Roll
// panel, and titled Effect/Trigger panels). Styled by
// docs/stylesheets/steel-ability-cards.css.
//
// SITE-ONLY: this runs inside `steel-etl site` against the generated md-linked
// pages — the shared data repos (Obsidian / JSON / YAML / plain md) are never
// touched. Frontmatter is lossy for power rolls (the parser's regex misses
// multi-characteristic rolls like "Might or Presence"), so the card is built
// from the page BODY (reliable + complete), reading frontmatter only for the
// name, action type, and cost.
//
// Scope: standalone `feature/ability/**` and `feature/trait/**` pages. Abilities
// rendered INLINE inside a class/kit page body keep their book-faithful markdown
// (the runtime ability-cards.js badges those) — converting those is a separate,
// larger pass (see v2-handoff ABILITY-CARDS.md "Not yet covered").

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// buildAbilityCardPage replaces an ability/trait/feature page's body with its
// steel card: abilities become the raised `.sc-ability` plate (renderAbilityCard);
// traits AND plain features (type: feature) become the recessed `.sc-trait` codex
// niche (renderTraitCard, see trait_cards.go). Returns
// (newData, true) when the page is an ability, trait, or feature; (data, false) otherwise so
// the caller writes it unchanged. The frontmatter is preserved verbatim; injectH1
// (called next in buildSection) prepends the "# Name" MkDocs needs for the
// title/nav/TOC, which the CSS hides.
func buildAbilityCardPage(data []byte, standalone map[string]bool) ([]byte, bool) {
	fm, body := splitFrontmatter(string(data))
	var card string
	switch parseFrontmatterField(fm, "type") {
	case "ability":
		card = renderAbilityCard(fm, body, "")
	case "trait", "feature":
		// A feature/trait page whose subtree contains a STANDALONE item
		// (statblock/featureblock/dynamic-terrain/feature-group) is left uncarded:
		// renderTraitCard would render those items as generic .sc-trait niches —
		// mangling their `>`-blockquote stat/feature content into escaped text —
		// AND consume the {data-scc} markers the embedItemCards post-pass needs.
		// Leaving the raw markdown lets embed render the subtree: it splices the
		// proper statblock/featureblock cards and re-cards the non-standalone
		// sub-features, exactly as it already does on type: class pages. See
		// docs/superpowers/specs/2026-06-16-inline-item-cards-design.md.
		if bodyHasStandaloneDescendant(body, standalone) {
			return data, false
		}
		card = renderTraitCard(fm, body)
	default:
		return data, false
	}
	return []byte("---\n" + fm + "\n---\n\n" + card), true
}

// bodyHasStandaloneDescendant reports whether a feature/trait page body carries a
// {data-scc} descendant heading whose code names a standalone leaf
// (statblock/featureblock/dynamic-terrain/feature-group, per standaloneType). The
// page's own code is never standalone, so it never matches itself.
func bodyHasStandaloneDescendant(body string, standalone map[string]bool) bool {
	if len(standalone) == 0 {
		return false
	}
	for _, line := range strings.Split(body, "\n") {
		if m := dataSCCHeadingRe.FindStringSubmatch(line); m != nil && standalone[m[2]] {
			return true
		}
	}
	return false
}

// ── action type → eyebrow label + crest glyph ───────────────────────────────
// glyph values are DrawSteelGlyphs codepoints — PLACEHOLDERS until the official
// action glyphs land (mirror the ACTIONS map in steel-ability-cards.js). Swap
// them here in one place. The accent COLOR for each key is final, set in CSS.
type actionMeta struct{ key, label, glyph string }

func actionInfo(actionType, contentType string) actionMeta {
	a := strings.ToLower(strings.TrimSpace(actionType))
	switch {
	case strings.Contains(a, "trigger"):
		return actionMeta{"triggered", "Triggered Action", ")"}
	case strings.Contains(a, "maneuver"):
		return actionMeta{"maneuver", "Maneuver", "f"}
	case strings.Contains(a, "move"):
		return actionMeta{"move", "Move Action", "o"}
	case strings.Contains(a, "main"):
		return actionMeta{"main", "Main Action", "l"}
	case strings.Contains(a, "free") || strings.Contains(a, "no action"):
		return actionMeta{"none", "No Action", "*"}
	case contentType == "trait":
		return actionMeta{"trait", "Trait", "*"}
	case a != "":
		return actionMeta{"none", titleCase(a), "*"}
	default:
		return actionMeta{"main", "Main Action", "l"}
	}
}

var tierGlyph = [3]string{"!", "@", "#"} // DrawSteelGlyphs ≤11 / 12–16 / 17+
var tierKey = [3]string{"low", "mid", "high"}

var (
	// The SCC linking sweep wraps the header as "**[Power Roll](…) + [Might](…)…:**",
	// so accept either the bare "Power Roll" or a link-wrapped "[Power Roll](…)"
	// before the "+". The captured characteristics may themselves carry links
	// (rendered via richInline, not escaped).
	prHeadRe    = regexp.MustCompile(`(?s)^\*\*(?:\[Power Roll\]\([^)]*\)|Power Roll)\s*\+\s*(.+?):\*\*\s*$`)
	labelRe     = regexp.MustCompile(`(?s)^\*\*([^*:]+):\*\*\s*(.+)$`)
	tierLineRe  = regexp.MustCompile(`^\s*[-*]?\s*\*\*([^*]+?):\*\*\s*(.+?)\s*$`)
	mdLinkRe    = regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`)
	mdBoldRe    = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	railEmojiRe = regexp.MustCompile(`[\x{1F300}-\x{1FAFF}\x{2600}-\x{27BF}\x{2B00}-\x{2BFF}]\s*`)
	paraSplitRe = regexp.MustCompile(`\n[ \t]*\n`)
	costNumRe   = regexp.MustCompile(`^(\d+)\s+(.*)$`)
)

// abilityOrigin builds the left-deck provenance "Class · Subclass" from
// frontmatter (slug → Title Case, "-" → space). Either part may be absent.
func abilityOrigin(fm string) string {
	var parts []string
	for _, key := range []string{"class", "subclass"} {
		if v := strings.TrimSpace(parseFrontmatterField(fm, key)); v != "" {
			parts = append(parts, titleCase(strings.ReplaceAll(v, "-", " ")))
		}
	}
	return strings.Join(parts, " · ")
}

// kitAbilityNameRe splits a combined kit signature-ability name "Kit (Ability)".
var kitAbilityNameRe = regexp.MustCompile(`^(.+) \((.+)\)$`)

// splitKitAbilityName splits "Kit (Ability)" into (kitName, abilityName). ok is
// false when the name is not in that combined form.
func splitKitAbilityName(name string) (kit, ability string, ok bool) {
	m := kitAbilityNameRe.FindStringSubmatch(strings.TrimSpace(name))
	if m == nil {
		return "", "", false
	}
	return strings.TrimSpace(m[1]), strings.TrimSpace(m[2]), true
}

// renderAbilityCard builds the contiguous (no blank-line) raw-HTML card so
// md_in_html passes it through verbatim. originOverride sets the left-deck
// provenance for a nested ability (whose synthesized frontmatter has no
// class/subclass); when empty, the deck is derived from frontmatter via
// abilityOrigin (the standalone-page path).
func renderAbilityCard(fm, body, originOverride string) string {
	ctype := parseFrontmatterField(fm, "type")
	name := strings.TrimSpace(parseFrontmatterField(fm, "name"))
	if name == "" {
		name = "Ability"
	}

	flavor := strings.TrimSpace(parseFrontmatterField(fm, "flavor"))
	actionType := parseFrontmatterField(fm, "action_type")
	cost := strings.TrimSpace(parseFrontmatterField(fm, "cost"))
	if cost == "" && parseFrontmatterField(fm, "subtype") == "signature" {
		cost = "Signature"
	}

	var keywords []string
	var distance, target string
	var prChars string
	var tiers [3]string
	hasPR := false

	// A section holds one or more raw paragraph/list blocks rendered inside a
	// single panel. Consecutive unlabeled paragraphs (and lists) following a
	// labeled paragraph fold into that section's blocks — so e.g. a multi-
	// paragraph Effect renders as one container, not several.
	type section struct {
		label  string
		blocks []string
	}
	var sections []section
	type enhancement struct{ cost, text string }
	var enhancements []enhancement
	cur := -1 // index of the open section unlabeled prose appends to (-1 = none)

	expectTiers := false
	for _, p := range paraSplitRe.Split(body, -1) {
		tp := strings.TrimSpace(p)
		if tp == "" {
			continue
		}

		// 2×2 spec table → keywords / action type / distance / target
		if strings.HasPrefix(tp, "|") {
			kws, act, dist, tgt := parseAbilityTable(p)
			if len(kws) > 0 && keywords == nil {
				keywords = kws
			}
			if act != "" && strings.TrimSpace(actionType) == "" {
				actionType = act
			}
			if dist != "" {
				distance = dist
			}
			if tgt != "" {
				target = tgt
			}
			expectTiers, cur = false, -1
			continue
		}

		// flavor: a single-star italic paragraph
		if isItalicPara(tp) {
			if flavor == "" {
				flavor = strings.TrimSpace(strings.Trim(tp, "*"))
			}
			expectTiers, cur = false, -1
			continue
		}

		// power-roll header → the next list paragraph holds the tiers
		if m := prHeadRe.FindStringSubmatch(tp); m != nil {
			prChars = strings.TrimSpace(m[1])
			hasPR = true
			expectTiers, cur = true, -1
			continue
		}

		// power-roll tier list
		if expectTiers && looksLikeTiers(tp) {
			parseTiers(tp, &tiers)
			expectTiers = false
			continue
		}
		expectTiers = false

		// header-less tier triple: a "test" whose outcomes reuse the
		// ≤11/12-16/17+ tiers but with no "**Power Roll +**" header line. Render
		// the same panel (prChars stays empty, so no synthesized header).
		if !hasPR && isTierListBlock(tp) {
			parseTiers(tp, &tiers)
			hasPR = true
			continue
		}

		// labeled paragraph → Effect / Trigger / Special… or a Spend enhancement
		if m := labelRe.FindStringSubmatch(tp); m != nil {
			label := strings.TrimSpace(m[1])
			if strings.HasPrefix(strings.ToLower(label), "spend") {
				enhancements = append(enhancements, enhancement{cost: label, text: collapseLines(m[2])})
				cur = -1
			} else {
				sections = append(sections, section{label: label, blocks: []string{m[2]}})
				cur = len(sections) - 1
			}
			continue
		}

		// unlabeled prose / list → fold into the open section, or start an
		// untitled one (common for traits).
		if cur >= 0 {
			sections[cur].blocks = append(sections[cur].blocks, tp)
		} else {
			sections = append(sections, section{blocks: []string{tp}})
			cur = len(sections) - 1
		}
	}

	act := actionInfo(actionType, ctype)
	dia := `<span class="sc-ability__dia"></span>`

	var b strings.Builder
	fmt.Fprintf(&b, "<article class=\"sc-ability sc-fil\" data-action=\"%s\">\n", act.key)

	// head: shared 6-slot header. usage → right-deck chip, cost → right-primary
	// mini-title, level → right-eyebrow chip; the crest carries the usage accent.
	// left-deck = class · subclass (abilityOrigin) surfaces the subclass that the
	// prior eyebrow-suffix fix (b3a55fc) added — more fully, and consistently.
	level := ""
	if lv := strings.TrimSpace(parseFrontmatterField(fm, "level")); lv != "" {
		level = "Level " + html.EscapeString(lv)
	}
	origin := originOverride
	if origin == "" {
		origin = abilityOrigin(fm)
	}
	// Kit signature abilities carry a combined "Kit (Ability)" name
	// (combineFrontmatterName) and a `kit` frontmatter field. Surface the ability
	// name as the primary and the kit name as the deck (e.g. "Unmooring" over
	// "Battlemind") instead of the combined string.
	primaryName := name
	if parseFrontmatterField(fm, "kit") != "" {
		if kitName, abilityName, ok := splitKitAbilityName(name); ok {
			primaryName, origin = abilityName, kitName
		}
	}
	b.WriteString(renderCardHead(cardHeadSlots{
		Crest:        fmt.Sprintf(`<span class="sc-crest sc-ability__crest"><span class="sc-ability__glyph">%s</span></span>`, html.EscapeString(act.glyph)),
		LeftEyebrow:  hLine("Ability"),
		LeftPrimary:  hLine(html.EscapeString(primaryName)),
		LeftDeck:     hLine(html.EscapeString(origin)),
		RightEyebrow: hChip(level),
		RightPrimary: hMini(html.EscapeString(cost)),
		RightDeck:    hChip(html.EscapeString(act.label)),
	}))
	b.WriteString("\n")

	if flavor != "" {
		fmt.Fprintf(&b, "<p class=\"sc-ability__flavor\">%s</p>\n", richInline(flavor))
	}

	if len(keywords) > 0 {
		b.WriteString("<div class=\"sc-ability__kw\">")
		for _, k := range keywords {
			fmt.Fprintf(&b, "<span class=\"sc-ability__chip\">%s</span>", richInline(k))
		}
		b.WriteString("</div>\n")
	}

	if distance != "" || target != "" {
		b.WriteString("<div class=\"sc-ability__rail\">\n")
		fmt.Fprintf(&b, "<div class=\"sc-ability__cell\"><div class=\"l\">Distance</div><div class=\"v\">%s</div></div>\n", railValue(distance))
		fmt.Fprintf(&b, "<div class=\"sc-ability__cell\"><div class=\"l\">Targets</div><div class=\"v\">%s</div></div>\n", railValue(target))
		b.WriteString("</div>\n")
	}

	if hasPR {
		b.WriteString(tierPanelHTML(dia, prChars, tiers, richInline))
	}

	for _, s := range sections {
		b.WriteString("<div class=\"sc-ability__section\">\n")
		if s.label != "" {
			fmt.Fprintf(&b, "<div class=\"sc-ability__section-head\">%s<span class=\"tag\">%s</span></div>\n", dia, html.EscapeString(s.label))
		}
		b.WriteString("<div class=\"sc-ability__section-body\">")
		for _, blk := range s.blocks {
			b.WriteString(renderSectionBlock(blk))
		}
		b.WriteString("</div>\n")
		b.WriteString("</div>\n")
	}

	for _, e := range enhancements {
		fmt.Fprintf(&b, "<div class=\"sc-ability__enh\"><span class=\"cost\">%s</span><span class=\"txt\">%s</span></div>\n",
			html.EscapeString(e.cost), richInline(e.text))
	}

	b.WriteString("</article>\n")
	return b.String()
}

// costBadge renders the persistent top-right cost. "" → no badge. A leading
// integer ("3 Piety") is split so the number renders in mono.
func costBadge(cost string) string {
	if cost == "" {
		return ""
	}
	if m := costNumRe.FindStringSubmatch(cost); m != nil {
		return fmt.Sprintf("<div class=\"sc-ability__cost\"><span class=\"num\">%s</span> %s</div>",
			html.EscapeString(m[1]), html.EscapeString(m[2]))
	}
	return fmt.Sprintf("<div class=\"sc-ability__cost\">%s</div>", html.EscapeString(cost))
}

// parseAbilityTable reads the 2×2 keyword/action/distance/target spec table.
func parseAbilityTable(para string) (keywords []string, action, distance, target string) {
	var rows [][]string
	for _, line := range strings.Split(para, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || strings.Contains(line, "---") {
			continue
		}
		rows = append(rows, splitRow(line))
	}
	if len(rows) >= 1 && len(rows[0]) >= 2 {
		keywords = splitKeywords(cellText(rows[0][0]))
		action = cellText(rows[0][1])
	}
	if len(rows) >= 2 && len(rows[1]) >= 2 {
		distance = stripRailEmoji(cellText(rows[1][0]))
		target = stripRailEmoji(cellText(rows[1][1]))
	}
	return
}

func splitRow(row string) []string {
	row = strings.Trim(strings.TrimSpace(row), "|")
	parts := strings.Split(row, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// cellText strips the **bold** wrapper a spec cell always carries.
func cellText(s string) string {
	if m := mdBoldRe.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(strings.Trim(s, "*"))
}

func splitKeywords(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func stripRailEmoji(s string) string {
	return strings.TrimSpace(railEmojiRe.ReplaceAllString(s, ""))
}

// isItalicPara reports whether a paragraph is wholly *italic* (flavor), not
// **bold**.
func isItalicPara(p string) bool {
	return strings.HasPrefix(p, "*") && strings.HasSuffix(p, "*") &&
		!strings.HasPrefix(p, "**") && len(p) >= 2
}

func looksLikeTiers(p string) bool {
	return strings.Contains(p, "≤11") || strings.Contains(p, "11 or lower") ||
		strings.Contains(p, "12-16") || strings.Contains(p, "12–16") ||
		strings.Contains(p, "17+") || strings.Contains(p, "17 or higher")
}

// parseTiers fills tiers[low,mid,high] from a "- **≤11:** …" list paragraph.
func parseTiers(para string, tiers *[3]string) {
	for _, line := range strings.Split(para, "\n") {
		m := tierLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key, val := m[1], strings.TrimSpace(m[2])
		switch {
		case strings.Contains(key, "≤11") || strings.Contains(key, "11 or lower"):
			tiers[0] = val
		case strings.Contains(key, "12-16") || strings.Contains(key, "12–16"):
			tiers[1] = val
		case strings.Contains(key, "17+") || strings.Contains(key, "17 or higher"):
			tiers[2] = val
		}
	}
}

// isTierListBlock reports whether a paragraph is a bullet list whose items are
// the ≤11/12-16/17+ power-roll tier outcomes — the distinctive signature a
// "test" shares with a power roll, but WITHOUT any "**Power Roll +**" header.
// Airtight by design (no over-reach): it must be a list block AND parse into at
// least two recognized tiers, so an ordinary bullet list never qualifies.
func isTierListBlock(tp string) bool {
	if !isListBlock(tp) {
		return false
	}
	var t [3]string
	parseTiers(tp, &t)
	n := 0
	for _, v := range t {
		if v != "" {
			n++
		}
	}
	return n >= 2
}

// tierPanelHTML renders the steel power-roll tier panel: an optional
// "Power Roll + <chars>" head (only when chars is non-empty — a bare test's tier
// triple carries no characteristic) followed by the glyph-badged tier rows. The
// inline renderer is supplied by the caller (richInline for abilities,
// traitInline for traits) so links/emphasis in the outcomes resolve in-context.
func tierPanelHTML(dia, chars string, tiers [3]string, inline func(string) string) string {
	var b strings.Builder
	b.WriteString("<div class=\"sc-ability__pr\">\n")
	if strings.TrimSpace(chars) != "" {
		fmt.Fprintf(&b, "<div class=\"sc-ability__pr-head\">%s<span class=\"pre\">Power Roll +</span><span class=\"chars\">%s</span></div>\n", dia, inline(chars))
	}
	b.WriteString("<div class=\"sc-ability__pr-rows\">\n")
	for i := 0; i < 3; i++ {
		if tiers[i] == "" {
			continue
		}
		fmt.Fprintf(&b, "<div class=\"sc-ability__tier\" data-tier=\"%s\"><span class=\"badge\">%s</span><span class=\"res\">%s</span></div>\n",
			tierKey[i], tierGlyph[i], inline(tiers[i]))
	}
	b.WriteString("</div>\n")
	b.WriteString("</div>\n")
	return b.String()
}

// collapseLines joins a multi-line paragraph into a single line.
func collapseLines(s string) string {
	fields := strings.Fields(strings.ReplaceAll(s, "\n", " "))
	return strings.Join(fields, " ")
}

// renderSectionBlock renders one paragraph/list block of a section body: a
// bullet list becomes a <ul>, anything else a <p>. Keeps a multi-paragraph
// Effect (with optional bullet list) inside a single section container.
func renderSectionBlock(block string) string {
	if isListBlock(block) {
		var sb strings.Builder
		sb.WriteString("<ul>")
		for _, ln := range strings.Split(block, "\n") {
			t := strings.TrimSpace(ln)
			if t == "" {
				continue
			}
			sb.WriteString("<li>" + richInline(strings.TrimSpace(t[1:])) + "</li>")
		}
		sb.WriteString("</ul>")
		return sb.String()
	}
	return "<p>" + richInline(collapseLines(block)) + "</p>"
}

// isListBlock reports whether every non-empty line of a block is a "- "/"* "
// bullet item.
func isListBlock(block string) bool {
	any := false
	for _, ln := range strings.Split(block, "\n") {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		if !strings.HasPrefix(t, "- ") && !strings.HasPrefix(t, "* ") {
			return false
		}
		any = true
	}
	return any
}

// railValue is a rail cell value: strip emoji, then render inline emphasis.
func railValue(s string) string {
	s = stripRailEmoji(strings.TrimSpace(s))
	if s == "" {
		return "—"
	}
	return richInline(s)
}

// richInline escapes text for HTML and renders the small bit of inline markdown
// rules text carries: **bold** → <b>, and [text](url) → a real <a> link. The
// card is raw HTML MkDocs never post-processes, so we resolve the link target
// ourselves (cardHref) instead of leaving a dead ".md" href.
func richInline(s string) string {
	s = html.EscapeString(s)
	s = mdBoldRe.ReplaceAllString(s, "<b>$1</b>")
	s = mdLinkRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := mdLinkRe.FindStringSubmatch(m)
		return fmt.Sprintf(`<a href="%s">%s</a>`, cardHref(sub[2]), sub[1])
	})
	return s
}

// cardHref resolves a markdown link target for a standalone ability/trait card
// page. Body links are file-relative ".md" links already remapped for the
// destination by rewriteSectionLinks; dirURL converts ".md" → the directory-URL
// form MkDocs serves. Because use_directory_urls serves every (non-index) page
// one directory deeper than its source file, a relative link needs one extra
// "../" — the adjustment MkDocs makes for markdown pages but can't make here.
// External, anchor, and mailto targets pass through untouched.
func cardHref(target string) string {
	if target == "" || strings.HasPrefix(target, "#") ||
		strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") ||
		strings.HasPrefix(target, "mailto:") {
		return target
	}
	return "../" + dirURL(target)
}
