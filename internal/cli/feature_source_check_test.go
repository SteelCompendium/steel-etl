package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// fsDoc builds a minimal Summoner-shaped document: an advancement table that puts
// `perk` in the Summoner column and `summoners-dominion` in the Circle column at
// L2, plus the two feature sections under a level-2 feature-group. perkExtra /
// domExtra are the optional " | @feature_source: circle" suffixes on each
// feature's annotation line.
func fsDoc(perkExtra, domExtra string) []byte {
	const tmpl = "---\nbook: mcdm.summoner.v1\n---\n\n" +
		"###### Summoner Advancement\n\n" +
		"| Level | Summoner Features | Circle Features | Minions | Abilities |\n" +
		"|---|---|---|---|---|\n" +
		"| 2nd | [Perk](scc.v1:mcdm.summoner.v1/feature.summoner.level-2/perk) | [Summoner's Dominion](scc.v1:mcdm.summoner.v1/feature.summoner.level-2/summoners-dominion) | 1 | 5 |\n\n" +
		"## Summoner\n<!-- @type: class | @id: summoner -->\n\n" +
		"<!-- @type: feature-group | @level: 2 -->\n### 2nd-Level Features\n\n" +
		"<!-- @type: feature | @id: perk | @level: 2%s -->\n#### Perk\n\nText.\n\n" +
		"<!-- @type: feature | @id: summoners-dominion | @level: 2%s -->\n#### Summoner's Dominion\n\nText.\n"
	return []byte(fmt.Sprintf(tmpl, perkExtra, domExtra))
}

func parseFS(t *testing.T, perkExtra, domExtra string) ([]byte, *parser.Document) {
	t.Helper()
	src := fsDoc(perkExtra, domExtra)
	doc, err := parser.ParseDocument(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return src, doc
}

func TestFeatureSourceCheck_Aligned(t *testing.T) {
	// perk → summoner (unmarked, default), dominion → circle (marked): no warnings.
	src, doc := parseFS(t, "", " | @feature_source: circle")
	if issues := checkSummonerFeatureSource(src, doc); len(issues) != 0 {
		t.Errorf("aligned table should yield no warnings, got %d: %+v", len(issues), issues)
	}
}

func TestFeatureSourceCheck_CircleColumnNotMarked(t *testing.T) {
	// dominion is in the Circle column but left unmarked (defaults summoner): WARN.
	src, doc := parseFS(t, "", "")
	issues := checkSummonerFeatureSource(src, doc)
	if len(issues) != 1 {
		t.Fatalf("want 1 warning, got %d: %+v", len(issues), issues)
	}
	if !strings.Contains(issues[0].msg, "summoners-dominion") {
		t.Errorf("warning should name summoners-dominion, got %q", issues[0].msg)
	}
}

func TestFeatureSourceCheck_SummonerColumnMarkedCircle(t *testing.T) {
	// perk is in the Summoner column but wrongly marked circle: WARN.
	src, doc := parseFS(t, " | @feature_source: circle", " | @feature_source: circle")
	issues := checkSummonerFeatureSource(src, doc)
	if len(issues) != 1 {
		t.Fatalf("want 1 warning, got %d: %+v", len(issues), issues)
	}
	if !strings.Contains(issues[0].msg, "perk") {
		t.Errorf("warning should name perk, got %q", issues[0].msg)
	}
}

func TestFeatureSourceCheck_NoTableNoOp(t *testing.T) {
	doc, _ := parser.ParseDocument([]byte("## Heroes\n<!-- @type: class | @id: fury -->\n\nNo table here.\n"))
	if issues := checkSummonerFeatureSource([]byte("no table"), doc); len(issues) != 0 {
		t.Errorf("no advancement table should be a no-op, got %+v", issues)
	}
}
