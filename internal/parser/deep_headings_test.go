package parser

import (
	"strings"
	"testing"
)

// The Monsters book uses H7 for statblocks and H9 for malice/terrain blocks.
// goldmark caps ATX headings at H6, so these must be collected separately.
func TestParseDeepHeadings(t *testing.T) {
	src := []byte("# Monsters\n\n" +
		"<!-- @type: monster | @category: goblins -->\n" +
		"## Goblins\n\n" +
		"Goblins are crafty.\n\n" +
		"<!-- @type: statblock -->\n" +
		"####### Goblin Warrior\n\n" +
		"| Goblin | - | Level 1 | Minion Brute | EV 1 |\n" +
		"|:--:|:--:|:--:|:--:|:--:|\n\n" +
		"> ⭐️ **Tough**\n>\n> Hardy.\n\n" +
		"<!-- @type: featureblock -->\n" +
		"######### Goblin Malice (Malice Features)\n\n" +
		"Spend malice to do things.\n")

	doc, err := ParseDocument(src)
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}

	var statblock, malice *Section
	for _, s := range doc.Sections[0].AllSections() {
		switch s.Heading {
		case "Goblin Warrior":
			statblock = s
		case "Goblin Malice (Malice Features)":
			malice = s
		}
	}

	if statblock == nil {
		t.Fatal("H7 statblock heading not collected")
	}
	if statblock.Type() != "statblock" {
		t.Errorf("statblock @type: got %q, want statblock", statblock.Type())
	}
	if !strings.Contains(statblock.BodySource, "Minion Brute") {
		t.Errorf("statblock body missing grid: %q", statblock.BodySource)
	}

	if malice == nil {
		t.Fatal("H9 malice heading not collected")
	}
	if malice.Type() != "featureblock" {
		t.Errorf("malice @type: got %q, want featureblock", malice.Type())
	}
}
