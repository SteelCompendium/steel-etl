package content

import (
	"regexp"
	"strconv"
	"strings"
)

// RichFeature is the non-lossy feature shape shared by featureblocks, dynamic
// terrain, and fixture statblocks (spec:
// docs/superpowers/specs/2026-06-12-featureblock-cards-design.md §2). Unlike
// the SDK statblock feature shape (ParseStatblockFeatures), it keeps labeled
// Effect/Trigger sections, cost enhancements, trailing notes, and the source
// emoji icon. Link markdown in text fields is kept verbatim (the data-field
// convention); only the power-roll formula is link-stripped (cosmetic).
type RichFeature struct {
	Icon         string
	Name         string
	Cost         string // "7 Malice", "Signature", "Villain Action 1", …
	Usage        string // "Main action", "Maneuver", … (from the spec table)
	Keywords     []string
	Distance     string
	Target       string
	PowerRoll    *RichPowerRoll
	Sections     []RichSection     // labeled paragraphs: Effect / Trigger / Special …
	Enhancements []RichEnhancement // cost-labeled paragraphs: "2 Malice:" / "Spend …:"
	Intro        string            // lead-in prose before the power roll/table (e.g. "As a maneuver, … make a Might test.")
	Body         string            // prose of a table-less (passive) feature
	Trailing     string            // prose after the structured parts of an ability
	Level        int               // advancement group level ("Level 5 Fixture Advancement Feature")
	// Post holds every block that follows the FIRST power roll's own tier list, in
	// document order, for a feature with MORE THAN ONE tier list (SC-308): later
	// sections/enhancements/prose paragraphs and any later power rolls, interleaved
	// exactly as they sit in the source. Sections/Enhancements above still carry
	// every entry regardless of position (ToMap/data-fidelity stays non-lossy);
	// Post is the render-order guide for anything past the first roll and is nil
	// for the overwhelming majority of features that have zero or one tier list —
	// existing single-roll rendering is untouched (see parseRichFeature).
	Post []RichPostBlock
}

// RichPowerRoll is a power roll: Formula "+ 2" (labeled form) or "2d10 + R"
// (dice-in-title form); "" means a bare test result (renderer omits the head)
// UNLESS Label is set (SC-308): a header-less tier list that is NOT the
// feature's only one derives a head from the bold "**<Characteristic> test**"
// phrase in the nearest preceding paragraph (e.g. "Agility Test") instead of
// rendering fully bare, so a reader can tell two later tier tables apart.
type RichPowerRoll struct {
	Formula string
	Label   string
	Tiers   map[string]string // keys: low / mid / high
}

type RichSection struct{ Label, Text string }
type RichEnhancement struct{ Cost, Text string }

// RichPostBlock is one block of a multi-roll feature's Post sequence: exactly
// one of the four fields is set. Section/Enhancement point at the SAME entry
// already appended to RichFeature.Sections/Enhancements (render-order pointer,
// not a data copy); Prose is a single collapsed paragraph; Roll is a tier list
// after the feature's first (rendered as its own panel).
type RichPostBlock struct {
	Section     *RichSection
	Enhancement *RichEnhancement
	Prose       string
	Roll        *RichPowerRoll
}

// testCharacteristics are the five characteristics a "**<Char> test**" phrase
// can name (SC-308 label rule), keyed lower-case → canonical display form.
var testCharacteristics = map[string]string{
	"might": "Might", "agility": "Agility", "reason": "Reason",
	"intuition": "Intuition", "presence": "Presence",
}

// testLabelRe matches a bold "**<Characteristic> test**" phrase, tolerant of a
// link-wrapped characteristic ("**[Agility](scc:…) test**" — the Monsters book
// links some but not all occurrences). Matched case-insensitively; the result
// is canonicalized against testCharacteristics.
var testLabelRe = regexp.MustCompile(`(?i)\*\*(?:\[([A-Za-z]+)\]\([^)]*\)|([A-Za-z]+))\s+test\*\*`)

// DeriveTestLabel finds the first "**<Characteristic> test**" phrase in text
// (the nearest preceding paragraph to a header-less tier list) and returns its
// head label ("Agility Test"), or "" if no known characteristic is named
// (SC-308's label rule; "" means the caller renders the panel bare).
func DeriveTestLabel(text string) string {
	for _, m := range testLabelRe.FindAllStringSubmatch(text, -1) {
		char := m[1]
		if char == "" {
			char = m[2]
		}
		if canon, ok := testCharacteristics[strings.ToLower(char)]; ok {
			return canon + " Test"
		}
	}
	return ""
}

var (
	fbParaSplitRe = regexp.MustCompile(`\n[ \t]*\n`)
	// power-roll header, tolerant of a link-wrapped "Power Roll" label
	// (mirrors internal/site prHeadRe).
	fbPRHeadRe = regexp.MustCompile(`(?s)^\*\*(?:\[Power Roll\]\([^)]*\)|Power Roll)\s*\+\s*(.+?):\*\*\s*$`)
	// a labeled paragraph: "**Effect:** text…" (mirrors internal/site labelRe).
	fbLabelRe = regexp.MustCompile(`(?s)^\*\*([^*:]+):\*\*\s*(.+)$`)
	// a label that is a cost ("2 Malice", "5+ Malice", "Spend …").
	fbCostLabelRe = regexp.MustCompile(`(?i)^(?:\d+\+?\s+\S+.*|spend\b.*)$`)
	// a standalone bold level-group label inside a blockquote:
	// "**Level 5 Fixture Advancement Feature**".
	fbLevelLabelRe = regexp.MustCompile(`^\*\*Level\s+(\d+)\b[^*]*\*\*$`)
	fbCollapseRe   = regexp.MustCompile(`\s*\n\s*`)
)

// fbCollapse joins a multi-line paragraph into one line.
func fbCollapse(s string) string {
	return strings.TrimSpace(fbCollapseRe.ReplaceAllString(s, " "))
}

// parenToCost maps a title parenthetical to its cost label: the canonical
// "Signature" for "Signature Ability", everything else verbatim ("7 Malice").
func parenToCost(paren string) string {
	if strings.EqualFold(paren, "Signature Ability") {
		return "Signature"
	}
	return paren
}

// ParseRichFeatures parses a body's feature blockquotes into RichFeatures.
// A standalone bold "Level N …" block sets the Level carried by all features
// that follow it (the fixture-advancement form).
func ParseRichFeatures(body string) []RichFeature {
	var out []RichFeature
	level := 0
	for _, block := range splitBlockquoteBlocks(body) {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if m := fbLevelLabelRe.FindStringSubmatch(block); m != nil {
			level, _ = strconv.Atoi(m[1])
			continue
		}
		if f, ok := parseRichFeature(block); ok {
			f.Level = level
			out = append(out, f)
		}
	}
	return out
}

// parseRichFeature parses one feature blockquote. Ported from the statblock
// island parser (internal/site/statblock_page.go parseStatblockIslandFeature);
// Plan 2 of the featureblock effort swaps the island onto this shared copy.
func parseRichFeature(block string) (RichFeature, bool) {
	paras := fbParaSplitRe.Split(block, -1)
	if len(paras) == 0 {
		return RichFeature{}, false
	}
	tm := sbTitleRe.FindStringSubmatch(strings.TrimSpace(paras[0]))
	if tm == nil {
		return RichFeature{}, false
	}
	// Strip scc links from the title: Name/Cost are structured fields stored link-free
	// (a markdown link's own ")" otherwise breaks the cost-paren split, sbParenRe), matching
	// the statblock title fix. Effect/tier VALUES keep their links (handled below).
	f := RichFeature{Icon: strings.TrimSpace(tm[1]), Name: linkDisplay(strings.TrimSpace(tm[2]))}

	// Dice-in-title power roll (summoner signatures) is checked BEFORE the
	// parenthetical-cost strip: a link-wrapped characteristic ([R](scc:…)) holds
	// "(...)" that sbParenRe would misread as a cost. linkDisplay collapses that
	// link to bare text first; any parenthetical THEN remaining on the formula is
	// a genuine cost — real titles read "Name 2d10 + R (Signature Ability)".
	diceFormula := ""
	if dm := sbDiceRe.FindStringSubmatch(f.Name); dm != nil {
		f.Name = strings.TrimSpace(dm[1])
		diceFormula = linkDisplay(strings.TrimSpace(dm[2]))
		if pm := sbParenRe.FindStringSubmatch(diceFormula); pm != nil {
			diceFormula = strings.TrimSpace(pm[1])
			f.Cost = parenToCost(strings.TrimSpace(pm[2]))
		}
	}

	// Parenthetical → Signature / cost / Villain Action N (non-dice titles).
	if diceFormula == "" {
		if pm := sbParenRe.FindStringSubmatch(f.Name); pm != nil {
			f.Name = strings.TrimSpace(pm[1])
			f.Cost = parenToCost(strings.TrimSpace(pm[2]))
		}
	}

	// SC-308: a feature with more than one tier list (e.g. Snackies for Sweeties'
	// poison-damage roll AND its header-less Agility-test outcomes) needs every
	// list kept, in document order, instead of the single [3]string below letting
	// each new list silently overwrite the last. Pre-count them so the ORIGINAL
	// single-list algorithm runs completely unchanged (byte-identical output) for
	// the overwhelming majority of features that have zero or one list; only a
	// genuinely multi-list feature takes the ordered parseRichFeatureMulti path.
	tierListCount := 0
	for _, para := range paras[1:] {
		if fbLooksLikeTiers(strings.TrimSpace(para)) {
			tierListCount++
		}
	}
	if tierListCount > 1 {
		parseRichFeatureMulti(&f, paras[1:], diceFormula)
		return f, true
	}

	var (
		tableSeen  bool
		formula    = diceFormula
		tiers      [3]string
		tiersSeen  bool
		bareIdx    int
		structured bool     // a table / power roll / tiers / section / enhancement has been seen
		introProse []string // bare prose BEFORE the first structured block (lead-in to a test)
		trailProse []string // bare prose AFTER a structured block
	)

	for _, para := range paras[1:] {
		tp := strings.TrimSpace(para)
		if tp == "" {
			continue
		}

		// Spec table → keywords / usage (row 1), distance / target (row 2).
		if strings.HasPrefix(tp, "|") {
			rows := featureTableRows(strings.Split(para, "\n"))
			if len(rows) >= 1 {
				f.Keywords = splitCommaList(stripBold(rows[0][0]))
				f.Usage = stripBold(rows[0][1])
			}
			if len(rows) >= 2 {
				f.Distance = cleanIconCell(rows[1][0])
				f.Target = cleanIconCell(rows[1][1])
			}
			tableSeen = true
			structured = true
			continue
		}

		// Power-roll header → formula ("+ 2"); the next list holds the tiers.
		if m := fbPRHeadRe.FindStringSubmatch(tp); m != nil {
			formula = "+ " + linkDisplay(strings.TrimSpace(m[1]))
			structured = true
			continue
		}

		// Labeled tier list ("- **≤11:** …").
		if fbLooksLikeTiers(tp) {
			fbParseTiers(tp, &tiers)
			tiersSeen = true
			structured = true
			continue
		}

		// Dice-in-title abilities: bare digit-led lines are tiers by position.
		if diceFormula != "" && bareIdx < 3 && sbBareTierRe.MatchString(tp) {
			tiers[bareIdx] = fbCollapse(tp)
			bareIdx++
			tiersSeen = true
			structured = true
			continue
		}

		// Labeled paragraph → cost enhancement or titled section.
		if m := fbLabelRe.FindStringSubmatch(tp); m != nil {
			label := strings.TrimSpace(m[1])
			text := fbCollapse(m[2])
			if fbCostLabelRe.MatchString(label) {
				f.Enhancements = append(f.Enhancements, RichEnhancement{Cost: label, Text: text})
			} else {
				f.Sections = append(f.Sections, RichSection{Label: label, Text: text})
			}
			structured = true
			continue
		}

		// Bare prose: a lead-in (before any structured block) sets up the roll and
		// must render above it (Intro); prose after a structured block trails it.
		if structured {
			trailProse = append(trailProse, fbCollapse(tp))
		} else {
			introProse = append(introProse, fbCollapse(tp))
		}
	}

	if tiersSeen {
		t := map[string]string{}
		for i, key := range []string{"low", "mid", "high"} {
			if tiers[i] != "" {
				t[key] = tiers[i]
			}
		}
		f.PowerRoll = &RichPowerRoll{Formula: formula, Tiers: t}
	}

	// Assign prose. A power roll / spec table is the dividing line: lead-in prose
	// becomes Intro (above it), prose after becomes Trailing. A feature with no
	// power roll and no table is a plain passive — all its prose is the Body.
	switch {
	case !tiersSeen && !tableSeen:
		if all := append(introProse, trailProse...); len(all) > 0 {
			f.Body = strings.Join(all, "\n\n")
		}
	case tableSeen:
		f.Intro = strings.Join(introProse, "\n\n")
		f.Trailing = strings.Join(trailProse, " ")
	default: // tiers, no table
		f.Intro = strings.Join(introProse, "\n\n")
		f.Trailing = strings.Join(trailProse, "\n\n")
	}
	return f, true
}

// parseRichFeatureMulti fills f for a feature with MORE THAN ONE tier list
// (SC-308): the first roll still lands in f.PowerRoll (same card slot as ever),
// and every later block — sections, enhancements, bare prose, and every later
// roll — is additionally recorded in f.Post, in document order, so the
// renderer can place a later tier table immediately after whatever it follows
// in the source instead of the fixed Sections-then-Trailing-then-Enhancements
// layout the single-roll path uses. A header-less roll (no preceding
// "**Power Roll + N:**") derives its head from the nearest preceding
// paragraph's bold "**<Characteristic> test**" phrase (DeriveTestLabel);
// otherwise it renders bare, exactly like the single-list convention.
func parseRichFeatureMulti(f *RichFeature, paras []string, diceFormula string) {
	var (
		formula       = diceFormula
		diceTiers     [3]string
		bareIdx       int
		structured    bool
		haveFirstRoll bool
		introProse    []string
		lastParaText  string
	)

	for _, para := range paras {
		tp := strings.TrimSpace(para)
		if tp == "" {
			continue
		}

		// Spec table → keywords / usage (row 1), distance / target (row 2).
		if strings.HasPrefix(tp, "|") {
			rows := featureTableRows(strings.Split(para, "\n"))
			if len(rows) >= 1 {
				f.Keywords = splitCommaList(stripBold(rows[0][0]))
				f.Usage = stripBold(rows[0][1])
			}
			if len(rows) >= 2 {
				f.Distance = cleanIconCell(rows[1][0])
				f.Target = cleanIconCell(rows[1][1])
			}
			structured = true
			continue
		}

		// Power-roll header → formula for the NEXT tier list; not itself a block.
		if m := fbPRHeadRe.FindStringSubmatch(tp); m != nil {
			formula = "+ " + linkDisplay(strings.TrimSpace(m[1]))
			structured = true
			continue
		}

		// Labeled tier list ("- **≤11:** …") — headered (formula set above) or bare.
		if fbLooksLikeTiers(tp) {
			var t [3]string
			fbParseTiers(tp, &t)
			tm := map[string]string{}
			for i, key := range []string{"low", "mid", "high"} {
				if t[i] != "" {
					tm[key] = t[i]
				}
			}
			label := ""
			if formula == "" {
				label = DeriveTestLabel(lastParaText)
			}
			roll := RichPowerRoll{Formula: formula, Label: label, Tiers: tm}
			formula = ""
			structured = true
			if !haveFirstRoll {
				f.PowerRoll = &roll
				haveFirstRoll = true
			} else {
				r := roll
				f.Post = append(f.Post, RichPostBlock{Roll: &r})
			}
			continue
		}

		// Dice-in-title abilities: bare digit-led lines are tiers by position (the
		// dice form never co-occurs with a second list in the corpus today; this
		// mirrors the single-list handling for the FIRST roll only).
		if diceFormula != "" && !haveFirstRoll && bareIdx < 3 && sbBareTierRe.MatchString(tp) {
			diceTiers[bareIdx] = fbCollapse(tp)
			bareIdx++
			structured = true
			if bareIdx == 3 {
				tm := map[string]string{}
				for i, key := range []string{"low", "mid", "high"} {
					if diceTiers[i] != "" {
						tm[key] = diceTiers[i]
					}
				}
				f.PowerRoll = &RichPowerRoll{Formula: diceFormula, Tiers: tm}
				haveFirstRoll = true
			}
			continue
		}

		// Labeled paragraph → cost enhancement or titled section. Recorded on the
		// feature (full data fidelity, any position) AND on Post (render order).
		if m := fbLabelRe.FindStringSubmatch(tp); m != nil {
			label := strings.TrimSpace(m[1])
			text := fbCollapse(m[2])
			if fbCostLabelRe.MatchString(label) {
				f.Enhancements = append(f.Enhancements, RichEnhancement{Cost: label, Text: text})
				e := &f.Enhancements[len(f.Enhancements)-1]
				f.Post = append(f.Post, RichPostBlock{Enhancement: e})
			} else {
				f.Sections = append(f.Sections, RichSection{Label: label, Text: text})
				s := &f.Sections[len(f.Sections)-1]
				f.Post = append(f.Post, RichPostBlock{Section: s})
			}
			structured = true
			lastParaText = text
			continue
		}

		// Bare prose: a lead-in (before any structured block) sets up the first
		// roll and renders above it (Intro, unchanged); everything else joins Post
		// as its own paragraph so a later roll can slot in between two of them.
		collapsed := fbCollapse(tp)
		lastParaText = collapsed
		if !structured {
			introProse = append(introProse, collapsed)
		} else {
			f.Post = append(f.Post, RichPostBlock{Prose: collapsed})
		}
	}

	if len(introProse) > 0 {
		f.Intro = strings.Join(introProse, "\n\n")
	}
}

// fbLooksLikeTiers reports whether a paragraph is a labeled tier list.
func fbLooksLikeTiers(para string) bool {
	return sbTierRe.MatchString(strings.TrimSpace(strings.Split(para, "\n")[0]))
}

// fbParseTiers fills tiers[0..2] (low/mid/high) from "- **≤11:** …" lines.
func fbParseTiers(para string, tiers *[3]string) {
	for _, line := range strings.Split(para, "\n") {
		m := sbTierRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		switch {
		case strings.HasPrefix(m[1], "≤"):
			tiers[0] = strings.TrimSpace(m[2])
		case strings.Contains(m[1], "-"):
			tiers[1] = strings.TrimSpace(m[2])
		case strings.HasSuffix(m[1], "+"):
			tiers[2] = strings.TrimSpace(m[2])
		}
	}
}

// ToMap converts a RichFeature to the featureblock.schema.json features[]
// shape (snake_case keys, empty fields omitted).
func (f RichFeature) ToMap() map[string]any {
	m := map[string]any{"name": f.Name}
	if f.Icon != "" {
		m["icon"] = f.Icon
	}
	if f.Cost != "" {
		m["cost"] = f.Cost
	}
	if f.Usage != "" {
		m["usage"] = f.Usage
	}
	if len(f.Keywords) > 0 {
		m["keywords"] = f.Keywords
	}
	if f.Distance != "" {
		m["distance"] = f.Distance
	}
	if f.Target != "" {
		m["target"] = f.Target
	}
	if f.PowerRoll != nil {
		pr := map[string]any{"tiers": f.PowerRoll.Tiers}
		if f.PowerRoll.Formula != "" {
			pr["formula"] = f.PowerRoll.Formula
		}
		if f.PowerRoll.Label != "" {
			pr["label"] = f.PowerRoll.Label
		}
		m["power_roll"] = pr
	}
	// SC-308: everything after the first tier list, in document order — sections/
	// enhancements/prose/later rolls interleaved exactly as in the source, so a
	// consumer that needs true render order (the site card) doesn't have to
	// reconstruct it from the flattened sections/enhancements/trailing arrays
	// below (which stay unordered relative to each other, as always). Present
	// only for a feature with more than one tier list; the pre-existing
	// single-effect data shape (SC-309) is otherwise unchanged.
	if post := postMaps(f.Post); len(post) > 0 {
		m["post"] = post
	}
	if len(f.Sections) > 0 {
		ss := make([]map[string]any, 0, len(f.Sections))
		for _, s := range f.Sections {
			ss = append(ss, map[string]any{"label": s.Label, "text": s.Text})
		}
		m["sections"] = ss
	}
	if len(f.Enhancements) > 0 {
		es := make([]map[string]any, 0, len(f.Enhancements))
		for _, e := range f.Enhancements {
			es = append(es, map[string]any{"cost": e.Cost, "text": e.Text})
		}
		m["enhancements"] = es
	}
	if f.Intro != "" {
		m["intro"] = f.Intro
	}
	if f.Body != "" {
		m["body"] = f.Body
	}
	if f.Trailing != "" {
		m["trailing"] = f.Trailing
	}
	if f.Level > 0 {
		m["level"] = f.Level
	}
	return m
}

// postMaps converts a multi-roll feature's ordered Post blocks to the
// featureblock.schema.json `post` shape: one map per block, holding exactly
// one of "section" / "enhancement" / "prose" / "roll".
func postMaps(post []RichPostBlock) []map[string]any {
	var out []map[string]any
	for _, b := range post {
		switch {
		case b.Section != nil:
			out = append(out, map[string]any{"section": map[string]any{"label": b.Section.Label, "text": b.Section.Text}})
		case b.Enhancement != nil:
			out = append(out, map[string]any{"enhancement": map[string]any{"cost": b.Enhancement.Cost, "text": b.Enhancement.Text}})
		case b.Roll != nil:
			pr := map[string]any{"tiers": b.Roll.Tiers}
			if b.Roll.Formula != "" {
				pr["formula"] = b.Roll.Formula
			}
			if b.Roll.Label != "" {
				pr["label"] = b.Roll.Label
			}
			out = append(out, map[string]any{"roll": pr})
		case b.Prose != "":
			out = append(out, map[string]any{"prose": b.Prose})
		}
	}
	return out
}

// RichFeatureMaps converts a parsed feature list to schema-shaped maps.
func RichFeatureMaps(fs []RichFeature) []map[string]any {
	out := make([]map[string]any, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.ToMap())
	}
	return out
}
