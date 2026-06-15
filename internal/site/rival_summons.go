package site

import (
	"path/filepath"
	"sort"
	"strings"
)

// rivalSummonsCards renders the statblock .md files in readDir as a .sc-cards
// grid. Unlike statblockCards, the file-read directory (readDir) is separate from
// the href base (hrefBase, relative to the page embedding the cards) so the block
// can be placed on a page that is not the files' parent index — e.g. the Rival
// Summoner page, one level above its summons' echelon index.
func rivalSummonsCards(readDir, hrefBase string, files []string) string {
	if len(files) == 0 {
		return ""
	}
	sort.Slice(files, func(i, j int) bool { return naturalLess(files[i], files[j]) })
	var sb strings.Builder
	sb.WriteString("<div class=\"sc-cards\">\n")
	for _, f := range files {
		fm, body := splitFrontmatter(readFile(filepath.Join(readDir, f)))
		name := parseFrontmatterField(fm, "name")
		if name == "" {
			name = fileToTitle(f)
		}
		href := filepath.ToSlash(filepath.Join(hrefBase, f))
		sb.WriteString(bestiaryLeafCard(readDir, fm, body, href, name))
	}
	sb.WriteString("</div>\n")
	return sb.String()
}
