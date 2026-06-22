// Command subclass-migrate is a one-off migration: it injects @subclass
// annotations into the heroes source doc from the legacy data-gen
// metadata.json. Throwaway — removed after the migration run.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

var (
	annRe  = regexp.MustCompile(`^\s*<!--(.*?)-->\s*$`)
	headRe = regexp.MustCompile(`^\s*>?\s*#{1,6}\s+(.*)$`)
	typeRe = regexp.MustCompile(`@type:\s*(\w+)`)
	idRe   = regexp.MustCompile(`@id:\s*([a-z0-9-]+)`)
	subRe  = regexp.MustCompile(`@subclass:`)
)

type entry struct {
	Subclass *string `json:"subclass"`
}

// key identifies a heading by class context, bucket (ability|feature) and slug.
type key struct{ class, bucket, slug string }

// headingRef points at the annotation comment line for a matched heading.
type headingRef struct {
	commentLine int // 0-based index into lines
}

func bucketOf(t string) string {
	if t == "ability" {
		return "ability"
	}
	return "feature"
}

// indexDoc returns, per (class,bucket,slug), the comment lines of every
// ANNOTATED ability/feature/trait heading. Bare headings are ignored.
func indexDoc(lines []string) map[key][]headingRef {
	idx := map[key][]headingRef{}
	curClass := ""
	for i := 0; i < len(lines); i++ {
		m := annRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		body := m[1]
		tm := typeRe.FindStringSubmatch(body)
		if tm == nil {
			continue
		}
		t := tm[1]
		if t == "class" {
			if id := idRe.FindStringSubmatch(body); id != nil {
				curClass = id[1]
			}
			continue
		}
		if t != "ability" && t != "feature" && t != "trait" {
			continue
		}
		// find the next non-blank line; it must be a heading
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if j >= len(lines) {
			continue
		}
		hm := headRe.FindStringSubmatch(lines[j])
		if hm == nil {
			continue
		}
		slug := ""
		if id := idRe.FindStringSubmatch(body); id != nil {
			slug = id[1]
		} else {
			slug = content.Slugify(content.CleanHeading(hm[1]))
		}
		k := key{curClass, bucketOf(t), slug}
		idx[k] = append(idx[k], headingRef{commentLine: i})
	}
	return idx
}

// parseKey splits a metadata key into (class, bucket, slug).
// e.g. mcdm.heroes.v1:feature.ability.elementalist.1st-level-feature:explosive-assistance
func parseKey(k string) (class, bucket, slug string, ok bool) {
	colon := strings.Split(k, ":")
	if len(colon) < 3 {
		return "", "", "", false
	}
	slug = colon[len(colon)-1]
	path := strings.Split(colon[1], ".")
	if path[0] == "feature" && len(path) >= 3 {
		return path[2], bucketOf(path[1]), slug, true
	}
	return "", "", slug, false // kit-ability.* etc. — all null, skipped anyway
}

// overrides maps a metadata key whose (class,bucket,slug) does not auto-match to
// the resolved (class,bucket,slug) of the correct canonical annotated heading.
// Populated in Task 2.
var overrides = map[string]key{}

// skip lists metadata keys intentionally not injected (e.g. the fact belongs to a
// @type: statblock entity that indexDoc does not index). Reported, not silent.
var skip = map[string]string{}

func main() {
	docPath := flag.String("doc", "input/heroes/Draw Steel Heroes.md", "heroes markdown")
	mdPath := flag.String("metadata", "../data-gen/input/heroes/metadata.json", "legacy metadata.json")
	apply := flag.Bool("apply", false, "rewrite the doc (default: dry-run report)")
	flag.Parse()

	raw, err := os.ReadFile(*docPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	lines := strings.Split(string(raw), "\n")
	idx := indexDoc(lines)

	mdRaw, err := os.ReadFile(*mdPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var md map[string]entry
	if err := json.Unmarshal(mdRaw, &md); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Deterministic order for stable reporting.
	keys := make([]string, 0, len(md))
	for k := range md {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	type hit struct {
		mdKey   string
		ref     headingRef
		slugVal string
	}
	var exact, relaxed, viaOverride []hit
	var residue, already, skipped []string
	nonNull := 0

	for _, mk := range keys {
		e := md[mk]
		if e.Subclass == nil || *e.Subclass == "" {
			continue
		}
		nonNull++
		if reason, has := skip[mk]; has {
			skipped = append(skipped, mk+"  ("+reason+")")
			continue
		}
		class, bucket, slug, ok := parseKey(mk)
		if !ok {
			residue = append(residue, mk+"  (unparseable / kit)")
			continue
		}
		val := content.Slugify(*e.Subclass)

		if k, has := overrides[mk]; has {
			if refs := idx[k]; len(refs) == 1 {
				viaOverride = append(viaOverride, hit{mk, refs[0], val})
				continue
			}
			residue = append(residue, mk+"  (override did not resolve to a unique heading)")
			continue
		}
		// Tier 1: exact (class, bucket, slug)
		if refs := idx[key{class, bucket, slug}]; len(refs) == 1 {
			exact = append(exact, hit{mk, refs[0], val})
			continue
		}
		// Tier 2: type-relaxed (class, slug) — try the other bucket
		other := "feature"
		if bucket == "feature" {
			other = "ability"
		}
		if refs := idx[key{class, other, slug}]; len(refs) == 1 {
			relaxed = append(relaxed, hit{mk, refs[0], val})
			continue
		}
		residue = append(residue, fmt.Sprintf("%s  (class=%s bucket=%s slug=%s)", mk, class, bucket, slug))
	}

	// Mark hits whose comment already carries @subclass (idempotency report).
	allHits := append(append(append([]hit{}, exact...), relaxed...), viaOverride...)
	for _, h := range allHits {
		if subRe.MatchString(lines[h.ref.commentLine]) {
			already = append(already, h.mdKey)
		}
	}

	fmt.Printf("non-null entries: %d\n", nonNull)
	fmt.Printf("exact:      %d\n", len(exact))
	fmt.Printf("relaxed:    %d\n", len(relaxed))
	fmt.Printf("override:   %d\n", len(viaOverride))
	fmt.Printf("skipped:    %d\n", len(skipped))
	fmt.Printf("already set: %d\n", len(already))
	fmt.Printf("RESIDUE:    %d\n", len(residue))
	for _, r := range residue {
		fmt.Println("  -", r)
	}
	for _, s := range skipped {
		fmt.Println("  SKIP", s)
	}

	if !*apply {
		fmt.Println("\n(dry-run; pass -apply to rewrite the doc)")
		return
	}

	// Apply: inject ` | @subclass: <val>` before the closing --> of each comment.
	injected := 0
	for _, h := range allHits {
		ln := lines[h.ref.commentLine]
		if subRe.MatchString(ln) {
			continue // idempotent
		}
		// single-line comment: insert before the final -->
		idxClose := strings.LastIndex(ln, "-->")
		if idxClose < 0 {
			fmt.Fprintf(os.Stderr, "WARN multi-line comment at %d not handled: %q\n", h.ref.commentLine+1, ln)
			continue
		}
		head := strings.TrimRight(ln[:idxClose], " ")
		lines[h.ref.commentLine] = head + " | @subclass: " + h.slugVal + " -->"
		injected++
	}
	if err := os.WriteFile(*docPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("\ninjected: %d\n", injected)
}
