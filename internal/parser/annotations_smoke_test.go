package parser

import (
	"os"
	"testing"
)

func TestSmokeRealHeroesFile(t *testing.T) {
	path := "../../input/heroes/Draw Steel Heroes.md"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("skipping smoke test: %v", err)
	}

	anns := ExtractAnnotations(string(data))

	// README says 1,523 annotations
	if len(anns) < 1400 {
		t.Errorf("expected ~1523 annotations, got %d (too few)", len(anns))
	}

	// Count by type
	typeCounts := make(map[string]int)
	for _, a := range anns {
		typeCounts[a.Fields["type"]]++
	}

	t.Logf("Total annotations: %d", len(anns))
	for typ, count := range typeCounts {
		t.Logf("  %s: %d", typ, count)
	}

	// Spot-check known counts from README
	if typeCounts["ability"] < 400 {
		t.Errorf("expected ~507 ability annotations, got %d", typeCounts["ability"])
	}
	if typeCounts["class"] != 9 {
		t.Errorf("expected 9 class annotations, got %d", typeCounts["class"])
	}
}
