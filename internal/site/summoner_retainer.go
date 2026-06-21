package site

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
)

// augmentSummonerRetainerPages adds, to each summoner-book retainer page under
// sectionDir/monster/retainer, an "## Advancement Features" preview card (from the
// flattened <id>-advancement-features.md sibling) and a "## Summons" grid of the
// retainer's summoned minions (sectionDir/monster/retainer/summoner/minion), plus a
// "Summoned by" back-link on each minion page. Mirrors augmentRivalSummonerPages;
// runs after pages are written and is idempotent. Scoped to the summoner book
// (scc prefix mcdm.summoner.) and the conjurer (organization != "Minion"), so the
// Monsters-book retainers are untouched. There is exactly one summoner retainer
// today, so every minion under summoner/minion belongs to it. Returns the number of
// pages modified.
func augmentSummonerRetainerPages(sectionDir string) (int, []string) {
	retainerDir := filepath.Join(sectionDir, "monster", "retainer")
	if _, err := os.Stat(retainerDir); err != nil {
		return 0, nil
	}
	minionDir := filepath.Join(retainerDir, "summoner", "minion")
	summonFiles := listSummonFiles(minionDir) // nil when no summons subtree

	ents, err := os.ReadDir(retainerDir)
	if err != nil {
		return 0, []string{fmt.Sprintf("read %s: %v", retainerDir, err)}
	}

	count := 0
	var errs []string
	for _, e := range ents {
		if e.IsDir() || e.Name() == "index.md" || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if strings.HasSuffix(e.Name(), advFeatSuffix+".md") {
			continue // skip advancement-features pages themselves
		}
		path := filepath.Join(retainerDir, e.Name())
		fm, _ := splitFrontmatter(readFile(path))
		scc := strings.TrimSpace(parseFrontmatterField(fm, "scc"))
		org := strings.TrimSpace(parseFrontmatterField(fm, "organization"))
		if !strings.HasPrefix(scc, "mcdm.summoner.") || org == "Minion" {
			continue // only the summoner-book conjurer
		}
		base := strings.TrimSuffix(e.Name(), ".md")
		name := strings.TrimSpace(parseFrontmatterField(fm, "name"))
		if name == "" {
			name = fileToTitle(e.Name())
		}

		page := readFile(path)
		modified := false

		// Advancement Features preview card (if the sibling exists).
		advFile := base + advFeatSuffix + ".md"
		if _, err := os.Stat(filepath.Join(retainerDir, advFile)); err == nil &&
			!strings.Contains(page, "## Advancement Features") {
			inner := advancementCardInner(retainerDir, advFile)
			// href needs a "../" hop: the detective page is served as a dir.
			advCard := card("../"+advFile, "sword-cross", "Advancement Features", name, inner)
			page = strings.TrimRight(page, "\n") +
				"\n\n## Advancement Features\n\n<div class=\"sc-cards\">\n" + advCard + "</div>\n"
			modified = true
		}

		// Summons grid (hrefBase mirrors the rival case: one "../" hop up).
		if len(summonFiles) > 0 && !strings.Contains(page, "## Summons") {
			cards := rivalSummonsCards(minionDir, "../summoner/minion", summonFiles)
			page = strings.TrimRight(page, "\n") + "\n\n## Summons\n\n" + cards + "\n"
			modified = true
		}

		if modified {
			if err := os.WriteFile(path, []byte(page), 0644); err != nil {
				errs = append(errs, fmt.Sprintf("write %s: %v", path, err))
			} else {
				count++
			}
		}

		// Back-link on each minion page (3 hops: summoner/minion/<id>/ → retainer/).
		if len(summonFiles) > 0 {
			backlink := fmt.Sprintf(`<p class="sb-backlink">Summoned by <a href="../../../%s/">%s</a></p>`,
				base, html.EscapeString(name))
			for _, sf := range summonFiles {
				sp := filepath.Join(minionDir, sf)
				spage := readFile(sp)
				if strings.Contains(spage, "sb-backlink") {
					continue
				}
				i := strings.Index(spage, `<div class="sb-wrap"`)
				if i < 0 {
					continue
				}
				spage = spage[:i] + backlink + "\n\n" + spage[i:]
				if err := os.WriteFile(sp, []byte(spage), 0644); err != nil {
					errs = append(errs, fmt.Sprintf("write %s: %v", sp, err))
				} else {
					count++
				}
			}
		}
	}
	return count, errs
}
