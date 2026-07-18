package site

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// rivalSummonsCards renders the statblock .md files in readDir as a .sb-cards
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
	sb.WriteString(sbCardsOpen())
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

// findRivalSummonerPage returns the conjurer page in an echelon dir: the statblock
// .md file (not index.md) from the summoner book (scc prefix mcdm.summoner.) whose
// organization is not "Minion". This selects the Rival Summoner NPC and ignores the
// co-located Monsters-book rivals (rival-fury, …) and the minion summons.
func findRivalSummonerPage(echelonDir string) (file, name string, ok bool) {
	ents, err := os.ReadDir(echelonDir)
	if err != nil {
		return "", "", false
	}
	for _, e := range ents {
		if e.IsDir() || e.Name() == "index.md" || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fm, _ := splitFrontmatter(readFile(filepath.Join(echelonDir, e.Name())))
		scc := strings.TrimSpace(parseFrontmatterField(fm, "scc"))
		org := strings.TrimSpace(parseFrontmatterField(fm, "organization"))
		if strings.HasPrefix(scc, "mcdm.summoner.") && org != "Minion" {
			return e.Name(), strings.TrimSpace(parseFrontmatterField(fm, "name")), true
		}
	}
	return "", "", false
}

// listSummonFiles returns the statblock .md files (not index.md) in a summon dir.
func listSummonFiles(summonDir string) []string {
	ents, err := os.ReadDir(summonDir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range ents {
		if e.IsDir() || e.Name() == "index.md" || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		files = append(files, e.Name())
	}
	return files
}

// augmentRivalSummonerPages adds Rival Summoner ⇄ summons cross-references under
// sectionDir/monster/rival: a "## Summons" card block on each Rival Summoner page
// and a "Summoned by" back-link on each summon page. Derives the relationship from
// the tree (a Rival Summoner's summons are its sibling summoner/minion/* set), runs
// after pages are written, and is idempotent. Returns the number of pages modified.
func augmentRivalSummonerPages(sectionDir string) (int, []string) {
	rivalsDir := filepath.Join(sectionDir, "monster", "rival")
	if _, err := os.Stat(rivalsDir); err != nil {
		return 0, nil
	}
	ents, err := os.ReadDir(rivalsDir)
	if err != nil {
		return 0, []string{fmt.Sprintf("read %s: %v", rivalsDir, err)}
	}

	count := 0
	var errs []string
	for _, e := range ents {
		if !e.IsDir() || !echelonDirRe.MatchString(e.Name()) {
			continue
		}
		echelonDir := filepath.Join(rivalsDir, e.Name())
		summonDir := filepath.Join(echelonDir, "summoner", "minion")
		if _, err := os.Stat(summonDir); err != nil {
			continue // echelon with no summoner/minion subtree
		}
		rivalFile, rivalName, ok := findRivalSummonerPage(echelonDir)
		if !ok {
			continue
		}
		summonFiles := listSummonFiles(summonDir)
		if len(summonFiles) == 0 {
			continue
		}
		rivalBase := strings.TrimSuffix(rivalFile, ".md")

		// Forward: append "## Summons" + cards to the Rival Summoner page.
		rivalPath := filepath.Join(echelonDir, rivalFile)
		page := readFile(rivalPath)
		if !strings.Contains(page, "## Summons") {
			cards := rivalSummonsCards(summonDir, "../summoner/minion", summonFiles)
			page = strings.TrimRight(page, "\n") + "\n\n## Summons\n\n" + cards + "\n"
			if err := os.WriteFile(rivalPath, []byte(page), 0644); err != nil {
				errs = append(errs, fmt.Sprintf("write %s: %v", rivalPath, err))
			} else {
				count++
			}
		}

		// Back: insert a back-link as the first child of each summon page's
		// .sb-wrap card. Must be a first child, not a preceding page-level
		// sibling — see firstCardOpenEnd (class_backlinks.go) for why: a
		// preceding sibling breaks the h1+hr+card adjacency v2's CSS depends
		// on to hide the duplicate MkDocs title chrome.
		backlink := fmt.Sprintf(`<p class="sb-backlink">Summoned by <a href="../../../%s/">%s</a></p>`,
			rivalBase, html.EscapeString(rivalName))
		for _, sf := range summonFiles {
			sp := filepath.Join(summonDir, sf)
			spage := readFile(sp)
			if strings.Contains(spage, "sb-backlink") {
				continue
			}
			i := firstCardOpenEnd(spage)
			if i < 0 {
				continue // not a rendered statblock page; nothing to anchor to
			}
			spage = spage[:i] + backlink + spage[i:]
			if err := os.WriteFile(sp, []byte(spage), 0644); err != nil {
				errs = append(errs, fmt.Sprintf("write %s: %v", sp, err))
			} else {
				count++
			}
		}
	}
	return count, errs
}
