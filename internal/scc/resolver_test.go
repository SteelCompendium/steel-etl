package scc

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestResolverResolveLinks(t *testing.T) {
	reg := NewRegistry()
	reg.Add("mcdm.heroes.v1/feature.ability.fury.level-1/gouge")
	reg.Add("mcdm.heroes.v1/class/fury")
	reg.Add("mcdm.heroes.v1/class/censor")
	reg.Add("mcdm.heroes.v1/condition/dazed")

	resolver := NewResolver(reg, ".md")

	tests := []struct {
		name       string
		input      string
		relativeTo string
		want       string
	}{
		{
			name:       "link from class page to deeply nested ability",
			input:      "See [Gouge](scc:mcdm.heroes.v1/feature.ability.fury.level-1/gouge) for details.",
			relativeTo: "mcdm.heroes.v1/class/fury",
			want:       "See [Gouge](../feature/ability/fury/level-1/gouge.md) for details.",
		},
		{
			name:       "link within same directory",
			input:      "[Censor](scc:mcdm.heroes.v1/class/censor)",
			relativeTo: "mcdm.heroes.v1/class/fury",
			want:       "[Censor](censor.md)",
		},
		{
			name:       "link from class to condition (sibling directories)",
			input:      "[Fury](scc:mcdm.heroes.v1/class/fury) and [dazed](scc:mcdm.heroes.v1/condition/dazed)",
			relativeTo: "mcdm.heroes.v1/class/censor",
			want:       "[Fury](fury.md) and [dazed](../condition/dazed.md)",
		},
		{
			name:       "empty relativeTo falls back to root-relative path",
			input:      "[Fury](scc:mcdm.heroes.v1/class/fury)",
			relativeTo: "",
			want:       "[Fury](class/fury.md)",
		},
		{
			name:       "bare scc reference without link syntax unchanged",
			input:      "See scc:mcdm.heroes.v1/class/unknown for details.",
			relativeTo: "mcdm.heroes.v1/class/fury",
			want:       "See scc:mcdm.heroes.v1/class/unknown for details.",
		},
		{
			name:       "no scc links",
			input:      "This is plain text with no links.",
			relativeTo: "mcdm.heroes.v1/class/fury",
			want:       "This is plain text with no links.",
		},
		{
			name:       "explicit scc.v1 prefix resolves identically to bare",
			input:      "See [Gouge](scc.v1:mcdm.heroes.v1/feature.ability.fury.level-1/gouge).",
			relativeTo: "mcdm.heroes.v1/class/fury",
			want:       "See [Gouge](../feature/ability/fury/level-1/gouge.md).",
		},
		{
			name:       "format qualifier is stripped before lookup",
			input:      "[Censor](scc:mcdm.heroes.v1/class/censor#yaml)",
			relativeTo: "mcdm.heroes.v1/class/fury",
			want:       "[Censor](censor.md)",
		},
		{
			name:       "explicit version and format together",
			input:      "[Censor](scc.v1:mcdm.heroes.v1/class/censor#json)",
			relativeTo: "mcdm.heroes.v1/class/fury",
			want:       "[Censor](censor.md)",
		},
		{
			name:       "non-current scheme version is not resolvable, strips to text",
			input:      "[Censor](scc.v2:mcdm.heroes.v1/class/censor)",
			relativeTo: "mcdm.heroes.v1/class/fury",
			want:       "Censor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ResolveLinks(tt.input, tt.relativeTo, LinkAll)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolverWithAliases(t *testing.T) {
	reg := NewRegistry()
	reg.Add("mcdm.heroes.v1/feature.ability.fury.level-1/gouge")
	reg.AddAlias("mcdm.heroes.v1/ability/gouge", "mcdm.heroes.v1/feature.ability.fury.level-1/gouge")

	resolver := NewResolver(reg, ".md")

	input := "See [Gouge](scc:mcdm.heroes.v1/ability/gouge)"
	want := "See [Gouge](../feature/ability/fury/level-1/gouge.md)"

	got := resolver.ResolveLinks(input, "mcdm.heroes.v1/class/fury", LinkAll)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolverLinkNone(t *testing.T) {
	reg := NewRegistry()
	reg.Add("mcdm.heroes.v1/class/fury")
	reg.Add("mcdm.heroes.v1/condition/dazed")

	resolver := NewResolver(reg, ".md")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strips single link",
			input: "See [Fury](scc:mcdm.heroes.v1/class/fury) for details.",
			want:  "See Fury for details.",
		},
		{
			name:  "strips multiple links",
			input: "[Fury](scc:mcdm.heroes.v1/class/fury) causes [dazed](scc:mcdm.heroes.v1/condition/dazed).",
			want:  "Fury causes dazed.",
		},
		{
			name:  "plain text unchanged",
			input: "No links here.",
			want:  "No links here.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ResolveLinks(tt.input, "", LinkNone)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolverLinkFirst(t *testing.T) {
	reg := NewRegistry()
	reg.Add("mcdm.heroes.v1/class/fury")
	reg.Add("mcdm.heroes.v1/condition/dazed")

	resolver := NewResolver(reg, ".md")

	tests := []struct {
		name       string
		input      string
		relativeTo string
		want       string
	}{
		{
			name:       "first occurrence linked, second stripped",
			input:      "[Fury](scc:mcdm.heroes.v1/class/fury) and [Fury](scc:mcdm.heroes.v1/class/fury) again.",
			relativeTo: "mcdm.heroes.v1/class/censor",
			want:       "[Fury](fury.md) and Fury again.",
		},
		{
			name:       "different codes each get one link",
			input:      "[Fury](scc:mcdm.heroes.v1/class/fury) [dazed](scc:mcdm.heroes.v1/condition/dazed) [Fury](scc:mcdm.heroes.v1/class/fury).",
			relativeTo: "mcdm.heroes.v1/class/censor",
			want:       "[Fury](fury.md) [dazed](../condition/dazed.md) Fury.",
		},
		{
			name:       "single occurrence kept",
			input:      "[Fury](scc:mcdm.heroes.v1/class/fury) is great.",
			relativeTo: "mcdm.heroes.v1/condition/dazed",
			want:       "[Fury](../class/fury.md) is great.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ResolveLinks(tt.input, tt.relativeTo, LinkFirst)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolverUnresolvedLinks(t *testing.T) {
	reg := NewRegistry()
	reg.Add("mcdm.heroes.v1/class/fury")

	resolver := NewResolver(reg, ".md")

	tests := []struct {
		name       string
		input      string
		relativeTo string
		want       string
	}{
		{
			name:       "unresolved markdown link stripped to display text",
			input:      "See [Unknown](scc:mcdm.heroes.v1/class/unknown) for details.",
			relativeTo: "mcdm.heroes.v1/condition/dazed",
			want:       "See Unknown for details.",
		},
		{
			name:       "resolved link from different directory",
			input:      "[Fury](scc:mcdm.heroes.v1/class/fury) is here.",
			relativeTo: "mcdm.heroes.v1/condition/dazed",
			want:       "[Fury](../class/fury.md) is here.",
		},
		{
			name:       "mix of resolved and unresolved",
			input:      "[Fury](scc:mcdm.heroes.v1/class/fury) and [Nope](scc:mcdm.heroes.v1/class/nope).",
			relativeTo: "mcdm.heroes.v1/condition/dazed",
			want:       "[Fury](../class/fury.md) and Nope.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ResolveLinks(tt.input, tt.relativeTo, LinkAll)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolverResolveFrontmatter(t *testing.T) {
	reg := NewRegistry()
	reg.Add("mcdm.heroes.v1/condition/restrained")
	reg.Add("mcdm.heroes.v1/movement/forced-movement")
	resolver := NewResolver(reg, ".md")

	relativeTo := "mcdm.heroes.v1/feature.ability.elementalist.level-2/volcanos-embrace"

	src := map[string]any{
		"name":   "Volcano's Embrace",
		"level":  "2",
		"effect": "[forced movement](scc:mcdm.heroes.v1/movement/forced-movement) increased by 2.",
		"tier1":  "5 + R fire damage; [restrained](scc:mcdm.heroes.v1/condition/restrained) (save ends)",
		"keywords": []any{
			"Fire",
			"hits a [restrained](scc:mcdm.heroes.v1/condition/restrained) target",
		},
		"nested": map[string]any{
			"detail": "see [restrained](scc:mcdm.heroes.v1/condition/restrained)",
		},
		"count": 3,
	}

	got := resolver.ResolveFrontmatter(src, relativeTo, LinkAll)

	// String values get scc links rewritten to relative paths.
	if want := "[forced movement](../../../../movement/forced-movement.md) increased by 2."; got["effect"] != want {
		t.Errorf("effect: got %q, want %q", got["effect"], want)
	}
	if want := "5 + R fire damage; [restrained](../../../../condition/restrained.md) (save ends)"; got["tier1"] != want {
		t.Errorf("tier1: got %q, want %q", got["tier1"], want)
	}
	// Slice elements are resolved.
	kw, ok := got["keywords"].([]any)
	if !ok || len(kw) != 2 {
		t.Fatalf("keywords: got %#v", got["keywords"])
	}
	if want := "hits a [restrained](../../../../condition/restrained.md) target"; kw[1] != want {
		t.Errorf("keywords[1]: got %q, want %q", kw[1], want)
	}
	// Nested maps are resolved recursively.
	nested, ok := got["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested: got %#v", got["nested"])
	}
	if want := "see [restrained](../../../../condition/restrained.md)"; nested["detail"] != want {
		t.Errorf("nested.detail: got %q, want %q", nested["detail"], want)
	}
	// Non-string scalars are untouched.
	if got["count"] != 3 {
		t.Errorf("count: got %v, want 3", got["count"])
	}

	// The original input map must NOT be mutated (shared with other generators).
	if src["effect"] != "[forced movement](scc:mcdm.heroes.v1/movement/forced-movement) increased by 2." {
		t.Errorf("input map was mutated: effect=%q", src["effect"])
	}
	if srcNested := src["nested"].(map[string]any); srcNested["detail"] != "see [restrained](scc:mcdm.heroes.v1/condition/restrained)" {
		t.Errorf("input nested map was mutated: detail=%q", srcNested["detail"])
	}
}

func TestResolverNonCurrentSchemeVersionWarns(t *testing.T) {
	reg := NewRegistry()
	reg.Add("mcdm.heroes.v1/class/censor")
	resolver := NewResolver(reg, ".md")

	// Capture stderr for the duration of the resolve.
	old := os.Stderr
	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	os.Stderr = wPipe

	got := resolver.ResolveLinks(
		"[Censor](scc.v10:mcdm.heroes.v1/class/censor)",
		"mcdm.heroes.v1/class/fury",
		LinkAll,
	)

	wPipe.Close()
	os.Stderr = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rPipe); err != nil {
		t.Fatalf("read captured stderr: %v", err)
	}
	stderr := buf.String()

	// A non-current scheme version is not resolvable: strips to display text.
	if got != "Censor" {
		t.Errorf("got %q, want %q", got, "Censor")
	}
	// The WARN must name both the link's version (v10) and the build's (v1)
	// and explain the link was not resolved.
	if !strings.Contains(stderr, "v10") {
		t.Errorf("WARN missing link version v10; stderr=%q", stderr)
	}
	if !strings.Contains(stderr, "not resolvable") {
		t.Errorf("WARN missing explanation; stderr=%q", stderr)
	}
}
