package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestChapterOrderIsDocumentOrder verifies chapters receive a per-book `order`
// frontmatter field reflecting their position in the source document.
func TestChapterOrderIsDocumentOrder(t *testing.T) {
	src := strings.Join([]string{
		"---", "book: mcdm.test.v1", "---", "",
		"<!-- @type: chapter | @id: alpha -->", "# Alpha", "", "Intro alpha.", "",
		"<!-- @type: chapter | @id: bravo -->", "# Bravo", "", "Intro bravo.", "",
		"<!-- @type: chapter | @id: charlie -->", "# Charlie", "", "Intro charlie.", "",
	}, "\n")

	dir := t.TempDir()
	in := filepath.Join(dir, "in.md")
	if err := os.WriteFile(in, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	reg := filepath.Join(dir, "registry.json")

	if _, err := Run(in, out, reg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	for name, want := range map[string]string{"alpha": "order: 0", "bravo": "order: 1", "charlie": "order: 2"} {
		path := filepath.Join(out, "chapter", name+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("chapter %s.md not generated: %v", name, err)
		}
		if !strings.Contains(string(data), want) {
			t.Errorf("%s: expected %q in frontmatter, got:\n%s", name, want, string(data))
		}
	}
}
