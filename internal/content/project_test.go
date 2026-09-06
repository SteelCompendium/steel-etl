package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestProjectFieldsAndDelegation(t *testing.T) {
	body := "**[Item Prerequisite](scc.v1:example/prerequisite):** None\n\n**Project Source:** A book\n\n**[Project Roll](scc.v1:example/roll) Characteristic:** [Reason](scc.v1:example/reason)\n\n**Project Goal:** 45 (per mile)"
	section := &parser.Section{Heading: "Build Road", BodySource: body}
	parsed, err := (&ProjectParser{}).Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"item_prerequisite": "None", "project_source": "A book", "project_roll_characteristic": "[Reason](scc.v1:example/reason)", "project_goal": "45 (per mile)"} {
		if parsed.Frontmatter[key] != want {
			t.Errorf("%s = %v; want %s", key, parsed.Frontmatter[key], want)
		}
	}
	parent := &parser.Section{Heading: "Delegate", BodySource: "See the projects below.", Children: []*parser.Section{section}}
	parsed, err = (&ProjectParser{}).Parse(context.NewContextStack(nil), parent)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := parsed.Frontmatter["project_goal"]; ok {
		t.Fatal("parent inherited a child goal")
	}
}
