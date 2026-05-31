package site

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
)

// generateSCCStubs walks the built docs directory, finds every .md page with an
// scc value in its frontmatter, and emits a static HTML stub at
// docs/scc/<scc>/index.html that redirects to that page. Stubs use relative
// URLs so they work under any deploy prefix (localhost root, /v2/ on prod).
//
// Stubs use noindex + canonical pointing at the friendly path so search engines
// don't double-index. The SCC URL is a stable, shareable redirect entry point;
// the friendly page is the canonical, indexable content.
func generateSCCStubs(docsDir string) (int, []string) {
	count := 0
	var errs []string

	// Remove any prior scc/ dir to avoid stale stubs surviving across renames.
	stubRoot := filepath.Join(docsDir, "scc")
	if err := os.RemoveAll(stubRoot); err != nil {
		errs = append(errs, fmt.Sprintf("clean scc dir: %v", err))
	}

	err := filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Don't descend into the stub dir we're populating.
			if path == stubRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			errs = append(errs, fmt.Sprintf("read %s: %v", path, readErr))
			return nil
		}
		content := string(data)
		if !strings.HasPrefix(content, "---\n") {
			return nil
		}
		fm, _ := splitFrontmatter(content)
		scc := parseFrontmatterField(fm, "scc")
		if scc == "" {
			return nil
		}

		// Compute the friendly URL path the .md page resolves to under MkDocs:
		//   docs/Browse/foo/bar.md      -> Browse/foo/bar/
		//   docs/Browse/foo/index.md    -> Browse/foo/
		rel, relErr := filepath.Rel(docsDir, path)
		if relErr != nil {
			errs = append(errs, fmt.Sprintf("rel %s: %v", path, relErr))
			return nil
		}
		friendly := mdPathToURLPath(filepath.ToSlash(rel))

		// Stub lives at docs/scc/<scc>/index.html.
		stubDir := filepath.Join(stubRoot, filepath.FromSlash(scc))
		if mkErr := os.MkdirAll(stubDir, 0755); mkErr != nil {
			errs = append(errs, fmt.Sprintf("mkdir %s: %v", stubDir, mkErr))
			return nil
		}

		// Relative path from the stub's directory to docs root, then to friendly.
		relTarget := relativeFromStub(scc, friendly)

		stubHTML := renderStub(relTarget)
		stubPath := filepath.Join(stubDir, "index.html")
		if writeErr := os.WriteFile(stubPath, []byte(stubHTML), 0644); writeErr != nil {
			errs = append(errs, fmt.Sprintf("write %s: %v", stubPath, writeErr))
			return nil
		}

		count++
		return nil
	})
	if err != nil {
		errs = append(errs, fmt.Sprintf("walk docs: %v", err))
	}

	return count, errs
}

// mdPathToURLPath converts a docs-relative markdown path into the URL path
// MkDocs serves it at (forward-slash separated, no leading slash, trailing slash).
//
//	Browse/foo/bar.md      -> Browse/foo/bar/
//	Browse/foo/index.md    -> Browse/foo/
//	index.md               -> ""  (site root)
func mdPathToURLPath(rel string) string {
	rel = strings.TrimSuffix(rel, ".md")
	if rel == "index" {
		return ""
	}
	if strings.HasSuffix(rel, "/index") {
		return strings.TrimSuffix(rel, "/index") + "/"
	}
	return rel + "/"
}

// relativeFromStub returns a relative URL from the stub directory to the
// friendly URL path. The stub lives at scc/<scc>/index.html, so depth is
// 1 (for "scc/") + number of "/" in scc + 1 (the SCC's own final segment).
//
//	scc=foo, friendly="Browse/x/"     -> "../../Browse/x/"
//	scc=a/b, friendly="Browse/x/"     -> "../../../Browse/x/"
//	scc=a/b/c, friendly="Browse/x/"   -> "../../../../Browse/x/"
func relativeFromStub(scc, friendly string) string {
	depth := strings.Count(scc, "/") + 2 // 1 for the scc/ prefix + (slash count + 1) for scc segments
	if friendly == "" {
		// site root: trim trailing slash on the .. chain
		return strings.Repeat("../", depth-1) + ".."
	}
	return strings.Repeat("../", depth) + friendly
}

// renderStub returns the HTML body of a permalink stub. The stub:
//   - meta-refresh redirects with no JS (covers crawlers, JS-off browsers)
//   - sets canonical to the friendly URL so search engines don't index the stub
//   - noindex so the stub itself stays out of search results
//   - JS uses location.replace() preserving hash + query for faster, hash-safe redirect
func renderStub(target string) string {
	escapedAttr := html.EscapeString(target)
	// JS string literal needs different escaping; keep target ASCII-safe (URLs are).
	jsLiteral := strings.ReplaceAll(target, `\`, `\\`)
	jsLiteral = strings.ReplaceAll(jsLiteral, `"`, `\"`)

	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Redirecting…</title>
<meta name="robots" content="noindex">
<link rel="canonical" href="` + escapedAttr + `">
<meta http-equiv="refresh" content="0; url=` + escapedAttr + `">
<script>location.replace("` + jsLiteral + `" + location.search + location.hash);</script>
</head>
<body>
<p>Redirecting to <a href="` + escapedAttr + `">` + escapedAttr + `</a>…</p>
</body>
</html>
`
}
