package site

// Book/chapter index cards for the "Books" tab (the GroupByBook "Read" section).
//
// The Read-section landing lists books as bookCard()s; each per-book index lists
// its chapters as chapterCard()s. Both reuse card() (cards.go) and the shared
// .sc-card CSS, matching the Browse tab's type-index cards. Emitted by
// writeBookNavAndIndexes() in build.go.

// bookCard renders one book's card for the Read-section landing index: a per-book
// crest (BookConfig.Icon, falling back to the generic "book" glyph), the type
// label "Book", the book label, and the hand-authored description (site.yaml) as
// flavor prose. The stretched link points at the book folder.
func bookCard(b BookConfig) string {
	icon := b.Icon
	if icon == "" {
		icon = "book"
	}
	inner := flavorDiv(b.Description, 0) // "" when no description → empty string
	return card(b.Folder+"/index.md", icon, "Book", b.Label, inner)
}

// chapterCard renders one chapter's card for a per-book index: the shared
// "chapter" crest, the type label "Chapter", the chapter name, and a blurb taken
// from the chapter body's first prose paragraph (truncated). file is the chapter
// basename (e.g. "ancestries.md"); the stretched link resolves to its directory
// URL (e.g. "ancestries/").
func chapterCard(file, name, body string) string {
	inner := flavorDiv(bodyBlurb(body, 200), 0)
	return card(file, "chapter", "Chapter", name, inner)
}
