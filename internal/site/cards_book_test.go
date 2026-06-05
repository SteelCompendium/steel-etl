package site

import (
	"strings"
	"testing"
)

func TestBookCard_IconLabelDescriptionLink(t *testing.T) {
	b := BookConfig{
		Folder:      "heroes",
		Label:       "Heroes",
		Order:       1,
		Icon:        "sword-cross",
		Description: "The core rulebook for building heroes.",
	}
	got := bookCard(b)
	for _, want := range []string{
		`class="sc-card`,          // it is a stat-card
		`href="heroes/"`,          // stretched link to the book folder
		`>Book<`,                  // type label
		`>Heroes<`,                // name
		`class="sc-card__flavor"`, // description rendered as flavor
		"The core rulebook for building heroes.",
		iconPaths["sword-cross"], // the per-book crest glyph
	} {
		if !strings.Contains(got, want) {
			t.Errorf("bookCard missing %q in:\n%s", want, got)
		}
	}
}

func TestBookCard_DefaultsToBookGlyphWhenNoIcon(t *testing.T) {
	got := bookCard(BookConfig{Folder: "bestiary", Label: "Bestiary"})
	if !strings.Contains(got, iconPaths["book"]) {
		t.Errorf("bookCard without Icon should use the 'book' glyph:\n%s", got)
	}
	if strings.Contains(got, `class="sc-card__flavor"`) {
		t.Errorf("bookCard without Description should emit no flavor div:\n%s", got)
	}
}

func TestChapterCard_NameBlurbAndLink(t *testing.T) {
	body := "# Ancestries\n\n---\n\nFantastic peoples inhabit the worlds of [Draw Steel](x.md).\n"
	got := chapterCard("ancestries.md", "Ancestries", body)
	for _, want := range []string{
		`href="ancestries/"`,      // link to the sibling chapter
		`>Chapter<`,               // type label
		`>Ancestries<`,            // name
		`class="sc-card__flavor"`, // blurb
		"Fantastic peoples inhabit the worlds of Draw Steel.", // links flattened to text
		iconPaths["chapter"], // shared chapter glyph
	} {
		if !strings.Contains(got, want) {
			t.Errorf("chapterCard missing %q in:\n%s", want, got)
		}
	}
}
