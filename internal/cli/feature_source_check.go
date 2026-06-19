package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

var (
	// Captures an SCC link target inside (...) — optional scc.vN: prefix, stop at ) or #.
	fsSccLinkRe = regexp.MustCompile(`\(scc(?:\.v\d+)?:([^)#]+)`)
	// Reduces a full code to its "level-N/<id>" tail (the comparison key).
	fsCodeTailRe = regexp.MustCompile(`feature\.summoner\.level-(\d+)/([a-z0-9-]+)$`)
)

// checkSummonerFeatureSource cross-checks each feature listed in the Summoner
// Advancement table against the feature_source the parser would emit. It is a
// no-op for documents without that table. Mismatches are non-fatal warnings.
func checkSummonerFeatureSource(source []byte, doc *parser.Document) []validationIssue {
	expected := parseAdvancementExpectations(string(source)) // key "level-N/<id>" -> "summoner"|"circle"
	if len(expected) == 0 {
		return nil
	}
	actual := map[string]string{} // key "level-N/<id>" -> effective feature_source
	collectFeatureSources(doc.Sections, "", "", actual)

	// Stable output order.
	keys := make([]string, 0, len(expected))
	for k := range expected {
		keys = append(keys, k)
	}
	sortStrings(keys)

	var issues []validationIssue
	for _, key := range keys {
		want := expected[key]
		got, ok := actual[key]
		if !ok {
			issues = append(issues, validationIssue{
				level: "warn",
				msg:   fmt.Sprintf("advancement table lists %q (%s) but no feature with that level/id was emitted", key, want),
			})
			continue
		}
		if got != want {
			issues = append(issues, validationIssue{
				level: "warn",
				msg:   fmt.Sprintf("feature_source mismatch for %q: advancement table column says %q, emitted %q", key, want, got),
			})
		}
	}
	return issues
}

// parseAdvancementExpectations finds the Summoner Advancement table and maps each
// linked feature (by its level-N/<id> tail) to the column it sits in.
func parseAdvancementExpectations(source string) map[string]string {
	lines := strings.Split(source, "\n")
	out := map[string]string{}
	sumCol, circleCol := -1, -1
	inTable := false
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "|") {
			if inTable {
				break // table ended
			}
			continue
		}
		cells := splitPipeRow(t)
		if sumCol < 0 {
			// Header row: locate the two feature columns.
			for i, c := range cells {
				switch strings.TrimSpace(c) {
				case "Summoner Features":
					sumCol = i
				case "Circle Features":
					circleCol = i
				}
			}
			if sumCol >= 0 && circleCol >= 0 {
				inTable = true
			}
			continue
		}
		if strings.Contains(t, "---") {
			continue // separator row
		}
		addColumnCodes(cells, sumCol, "summoner", out)
		addColumnCodes(cells, circleCol, "circle", out)
	}
	return out
}

func addColumnCodes(cells []string, col int, source string, out map[string]string) {
	if col < 0 || col >= len(cells) {
		return
	}
	for _, m := range fsSccLinkRe.FindAllStringSubmatch(cells[col], -1) {
		if tail := fsCodeTailRe.FindStringSubmatch(strings.TrimSpace(m[1])); tail != nil {
			out["level-"+tail[1]+"/"+tail[2]] = source
		}
	}
}

// collectFeatureSources threads level + feature_source down the section tree and
// records the effective feature_source for every feature/ability, keyed by
// "level-N/<id>" — mirroring FeatureParser/AbilityParser (own annotation wins,
// else inherited, else "summoner").
func collectFeatureSources(sections []*parser.Section, inheritedLevel, inheritedSource string, out map[string]string) {
	for _, sec := range sections {
		level, source := inheritedLevel, inheritedSource
		if sec.Annotation != nil {
			if v, ok := sec.Annotation["level"]; ok && v != "" {
				level = v
			}
			if v, ok := sec.Annotation["feature_source"]; ok && v != "" {
				source = v
			}
		}
		if t := sec.Type(); (t == "feature" || t == "ability") && level != "" {
			id := sec.ID()
			if id == "" {
				id = content.Slugify(content.CleanHeading(sec.Heading))
			}
			eff := source
			if eff == "" {
				eff = "summoner"
			}
			out["level-"+level+"/"+id] = eff
		}
		collectFeatureSources(sec.Children, level, source, out)
	}
}

// splitPipeRow splits a markdown table row on pipes, trimming the leading/trailing
// border pipes. (Local helper; the ability-table splitter lives in another package.)
func splitPipeRow(row string) []string {
	row = strings.Trim(strings.TrimSpace(row), "|")
	parts := strings.Split(row, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}
	return cells
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
