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
//
// SC-308 round 3b: Effects is the AUTHORITATIVE, ordered, non-lossy body
// representation — one entry per labeled/cost/bare-prose paragraph, carrying
// the tier list that directly attaches to it (see parseRichFeature). The site
// renders Effects, in order, with no hoisting. PowerRoll/Sections/
// Enhancements/Intro/Body/Trailing below are flat, UNORDERED convenience
// copies of the same data, kept for existing consumers (the SDK featureblock
// schema + its round-trip test); a consumer needing true render order reads
// Effects instead of reconstructing it from them.
type RichFeature struct {
	Icon         string
	Name         string
	Cost         string // "7 Malice", "Signature", "Villain Action 1", …
	Usage        string // "Main action", "Maneuver", … (from the spec table)
	Keywords     []string
	Distance     string
	Target       string
	PowerRoll    *RichPowerRoll    // first tier list only (flat convenience copy)
	Sections     []RichSection     // labeled paragraphs: Effect / Trigger / Special …
	Enhancements []RichEnhancement // cost-labeled paragraphs: "2 Malice:" / "Spend …:"
	Intro        string            // lead-in prose before the power roll/table (e.g. "As a maneuver, … make a Might test.")
	Body         string            // prose of a table-less (passive) feature
	Trailing     string            // prose after the structured parts of an ability
	Level        int               // advancement group level ("Level 5 Fixture Advancement Feature")
	Effects      []RichEffect      // authoritative render order (SC-308 round 3b)
}

// RichPowerRoll is the flat PowerRoll convenience field: Formula "+ 2"
// (labeled form) or "2d10 + R" (dice-in-title form); "" means a bare test
// result. It is always the feature's FIRST tier list only — see RichFeature's
// doc comment.
type RichPowerRoll struct {
	Formula string
	Tiers   map[string]string // keys: low / mid / high
}

type RichSection struct{ Label, Text string }
type RichEnhancement struct{ Cost, Text string }

// RichEffect is one entry of the ordered Effects list (SC-308 round 3b): the
// SDK feature.schema.json `effect` shape (definitions.effect) — a labeled
// section (Name), a cost enhancement (Cost), or bare prose (neither), plus the
// tier list (Roll/Tier1..3) that directly attaches to it per the attachment
// rule in parseRichFeature. Name and Cost are mutually exclusive; Roll is the
// "**Power Roll + N:**" header text verbatim ("Power Roll + 3") when the
// source had one, "" for a header-less list (the site derives its display
// label from THIS entry's own Effect prose at render time, never stored
// here). An entry with no tier list at all (TierN == "") is bare prose or a
// labeled paragraph with nothing following it.
type RichEffect struct {
	Name                string
	Cost                string
	Effect              string
	Roll                string
	Tier1, Tier2, Tier3 string
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

// DeriveTestLabel finds "**<Characteristic> test**" phrases in text (the
// nearest preceding paragraph to a header-less tier list) and returns the
// head label ("Agility Test") for the LAST one named, or "" if none is known
// (SC-308's label rule; "" means the caller renders the panel bare). The last
// match wins deliberately — it is the one nearest the tier list that follows,
// so "Each target must make either a Might test or an Agility test." heads
// the panel "Agility Test", not "Might Test".
func DeriveTestLabel(text string) string {
	label := ""
	for _, m := range testLabelRe.FindAllStringSubmatch(text, -1) {
		char := m[1]
		if char == "" {
			char = m[2]
		}
		if canon, ok := testCharacteristics[strings.ToLower(char)]; ok {
			label = canon + " Test"
		}
	}
	return label
}

var (
	fbParaSplitRe = regexp.MustCompile(`\n[ \t]*\n`)
	// power-roll header, tolerant of a link-wrapped "Power Roll" label
	// (mirrors internal/site prHeadRe).
	fbPRHeadRe = regexp.MustCompile(`(?s)^\*\*(?:\[Power Roll\]\([^)]*\)|Power Roll)\s*\+\s*(.+?):\*\*\s*$`)
	// a labeled paragraph: "**Effect:** text…" (mirrors internal/site labelRe).
	// The label class tolerates an scc-linked label ("**3 [Malice](scc.v1:…):**",
	// "**[End Effect](scc.v1:…):**") — the book links some Malice costs and rule
	// terms, and an scc.v1: URL carries a colon that a bare [^*:]+ class would
	// stop at, silently falling the whole paragraph through to bare prose (SC-308
	// review r3, I-1). Callers classify on linkDisplay(label), never the raw match.
	fbLabelRe = regexp.MustCompile(`(?s)^\*\*((?:\[[^\]]*\]\([^)]*\)|[^*:])+?):\*\*\s*(.+)$`)
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
//
// SC-308 round 3b: a SINGLE pass builds both the ordered f.Effects list (the
// authoritative render order) and the flat legacy fields (PowerRoll/Sections/
// Enhancements/Intro/Body/Trailing), so the two stay in agreement by
// construction rather than by two separately-maintained algorithms.
//
// Attachment rule: a tier list (headered or not) attaches to the effects
// entry created by the paragraph immediately before it — a labeled section,
// a cost enhancement, or bare prose — as that entry's Roll/Tier1..3. A tier
// list with nothing eligible before it (right after the table, or after an
// entry that already consumed a roll) becomes its own roll-only entry. This
// is tracked with curIdx, an INDEX into f.Effects rather than a pointer,
// because f.Effects keeps growing via append and a raw pointer into it would
// go stale across a reallocation (SC-308 review round 2, finding L-4).
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

	var (
		tableSeen   bool
		pendingRoll string // "+ N" formula from a "**Power Roll + N:**" header, pending until the next tier list consumes it
		firstSeen   bool   // has the flat legacy PowerRoll (the FIRST tier list only) been set
		diceTiers   [3]string
		bareIdx     int
		structured  bool     // a table / power roll header / tiers / labeled paragraph has been seen (Intro/Trailing split)
		introProse  []string // bare prose BEFORE the first structured block (lead-in to a test)
		trailProse  []string // bare prose AFTER a structured block
		curIdx      = -1     // index into f.Effects of the entry eligible to receive the next tier list; -1 = none
	)

	// attach fills the tier list (roll header text + tiers) onto the entry at
	// curIdx if one is pending, else appends a fresh roll-only entry.
	attach := func(roll string, t [3]string) {
		if curIdx >= 0 {
			e := &f.Effects[curIdx]
			e.Roll, e.Tier1, e.Tier2, e.Tier3 = roll, t[0], t[1], t[2]
			curIdx = -1
			return
		}
		f.Effects = append(f.Effects, RichEffect{Roll: roll, Tier1: t[0], Tier2: t[1], Tier3: t[2]})
	}

	for _, para := range paras[1:] {
		tp := strings.TrimSpace(para)
		if tp == "" {
			continue
		}

		// Spec table → keywords / usage (row 1), distance / target (row 2). Not
		// itself an effects entry (matches the SDK's model: table → metadata).
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

		// Power-roll header → formula for the NEXT tier list; not itself a block.
		if m := fbPRHeadRe.FindStringSubmatch(tp); m != nil {
			pendingRoll = "+ " + linkDisplay(strings.TrimSpace(m[1]))
			structured = true
			continue
		}

		// Labeled tier list ("- **≤11:** …") — headered (pendingRoll set above) or bare.
		if fbLooksLikeTiers(tp) {
			var t [3]string
			fbParseTiers(tp, &t)
			structured = true

			head := ""
			if pendingRoll != "" {
				head = "Power Roll " + pendingRoll
			}
			if !firstSeen {
				firstSeen = true
				tm := map[string]string{}
				for i, key := range []string{"low", "mid", "high"} {
					if t[i] != "" {
						tm[key] = t[i]
					}
				}
				f.PowerRoll = &RichPowerRoll{Formula: pendingRoll, Tiers: tm}
			}
			pendingRoll = ""
			attach(head, t)
			continue
		}

		// Dice-in-title abilities: bare digit-led lines are tiers by position (the
		// dice form never co-occurs with a labeled list in the corpus).
		if diceFormula != "" && bareIdx < 3 && sbBareTierRe.MatchString(tp) {
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
				attach(diceFormula, diceTiers)
			}
			continue
		}

		// Labeled paragraph → cost enhancement or titled section. Becomes its own
		// effects entry, eligible for a following tier list to attach to.
		if m := fbLabelRe.FindStringSubmatch(tp); m != nil {
			// Link-free: name/cost are structured fields (SC-308 review r3, I-1 —
			// mirrors the title-parenthetical stripping above). effect/tier VALUES
			// keep their links verbatim.
			label := linkDisplay(strings.TrimSpace(m[1]))
			text := fbCollapse(m[2])
			if fbCostLabelRe.MatchString(label) {
				f.Enhancements = append(f.Enhancements, RichEnhancement{Cost: label, Text: text})
				f.Effects = append(f.Effects, RichEffect{Cost: label, Effect: text})
			} else {
				f.Sections = append(f.Sections, RichSection{Label: label, Text: text})
				f.Effects = append(f.Effects, RichEffect{Name: label, Effect: text})
			}
			curIdx = len(f.Effects) - 1
			structured = true
			continue
		}

		// Bare prose: a lead-in (before any structured block) sets up the roll and
		// renders above it (Intro); prose after a structured block trails it —
		// either way it becomes its own effects entry, eligible for a following
		// tier list to attach to (the attachment rule applies to bare prose too).
		collapsed := fbCollapse(tp)
		if structured {
			trailProse = append(trailProse, collapsed)
		} else {
			introProse = append(introProse, collapsed)
		}
		f.Effects = append(f.Effects, RichEffect{Effect: collapsed})
		curIdx = len(f.Effects) - 1
	}

	// Assign prose. A power roll / spec table is the dividing line: lead-in prose
	// becomes Intro (above it), prose after becomes Trailing. A feature with no
	// power roll and no table is a plain passive — all its prose is the Body.
	switch {
	case !firstSeen && !tableSeen:
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
		m["power_roll"] = pr
	}
	// SC-308 round 3b: the AUTHORITATIVE, ordered, non-lossy render-order copy —
	// see RichFeature's doc comment. Emitted IN ADDITION TO the flat fields
	// above/below (kept for the SDK featureblock schema + its round-trip test).
	if effs := effectMaps(f.Effects); len(effs) > 0 {
		m["effects"] = effs
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

// effectMaps converts the ordered Effects list to the SDK `effects[]` shape
// (feature.schema.json definitions.effect): name/cost/effect/roll/tier1..3,
// empty fields omitted.
func effectMaps(effects []RichEffect) []map[string]any {
	out := make([]map[string]any, 0, len(effects))
	for _, e := range effects {
		em := map[string]any{}
		if e.Name != "" {
			em["name"] = e.Name
		}
		if e.Cost != "" {
			em["cost"] = e.Cost
		}
		if e.Effect != "" {
			em["effect"] = e.Effect
		}
		if e.Roll != "" {
			em["roll"] = e.Roll
		}
		if e.Tier1 != "" {
			em["tier1"] = e.Tier1
		}
		if e.Tier2 != "" {
			em["tier2"] = e.Tier2
		}
		if e.Tier3 != "" {
			em["tier3"] = e.Tier3
		}
		out = append(out, em)
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
