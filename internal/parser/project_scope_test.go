package parser

import (
	"strings"
	"testing"
)

func TestUnannotatedQuotedAbilityStaysInOwner(t *testing.T) {
	source := "# Book\n\n<!-- @type: project -->\n##### Imbue Weapon\n\n###### 5th-Level Weapon Enhancement\n\n**Chargebreaker:** You have the following ability.\n\n> ###### Stop Right There\n>\n> **Effect:** Damage.\n\n**Chilling II:** Cold damage.\n\n###### Enhancement Table\n\nTable body.\n"
	doc, err := ParseDocument([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var owner *Section
	for _, root := range doc.Sections {
		for _, s := range root.AllSections() {
			if s.Heading == "Stop Right There" {
				t.Fatal("unannotated quote opened a section")
			}
			if s.Heading == "5th-Level Weapon Enhancement" {
				owner = s
			}
		}
	}
	if owner == nil || !strings.Contains(owner.BodySource, "> ###### Stop Right There") || !strings.Contains(owner.BodySource, "**Chilling II:**") {
		t.Fatalf("owner lost content: %#v", owner)
	}
	if strings.Contains(owner.BodySource, "Table body") {
		t.Fatal("next regular section was swallowed")
	}
}

func TestAnnotatedQuotedAbilityRemainsAnEntity(t *testing.T) {
	source := "# Book\n\n##### Owner\n\n<!-- @type: ability -->\n> ###### Granted\n>\n> **Effect:** Damage.\n"
	doc, err := ParseDocument([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range doc.Sections {
		for _, s := range root.AllSections() {
			if s.Heading == "Granted" && s.Type() == "ability" && s.Parent.Heading == "Owner" {
				return
			}
		}
	}
	t.Fatal("typed quote no longer has its own section")
}
