package scc

import "testing"

func TestResolverResolveLinks(t *testing.T) {
	reg := NewRegistry()
	reg.Add("mcdm.heroes.v1/feature.ability.fury.level-1/gouge")
	reg.Add("mcdm.heroes.v1/class/fury")
	reg.Add("mcdm.heroes.v1/condition/dazed")

	resolver := NewResolver(reg, ".md")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "markdown link resolved",
			input: "See [Gouge](scc:mcdm.heroes.v1/feature.ability.fury.level-1/gouge) for details.",
			want:  "See [Gouge](feature/ability/fury/level-1/gouge.md) for details.",
		},
		{
			name:  "markdown link",
			input: "[Gouge](scc:mcdm.heroes.v1/feature.ability.fury.level-1/gouge)",
			want:  "[Gouge](feature/ability/fury/level-1/gouge.md)",
		},
		{
			name:  "multiple markdown links",
			input: "[Fury](scc:mcdm.heroes.v1/class/fury) and [dazed](scc:mcdm.heroes.v1/condition/dazed)",
			want:  "[Fury](class/fury.md) and [dazed](condition/dazed.md)",
		},
		{
			name:  "bare scc reference without link syntax unchanged",
			input: "See scc:mcdm.heroes.v1/class/unknown for details.",
			want:  "See scc:mcdm.heroes.v1/class/unknown for details.",
		},
		{
			name:  "no scc links",
			input: "This is plain text with no links.",
			want:  "This is plain text with no links.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ResolveLinks(tt.input, "", LinkAll)
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
	want := "See [Gouge](feature/ability/fury/level-1/gouge.md)"

	got := resolver.ResolveLinks(input, "", LinkAll)
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
		name  string
		input string
		want  string
	}{
		{
			name:  "first occurrence linked, second stripped",
			input: "[Fury](scc:mcdm.heroes.v1/class/fury) and [Fury](scc:mcdm.heroes.v1/class/fury) again.",
			want:  "[Fury](class/fury.md) and Fury again.",
		},
		{
			name:  "different codes each get one link",
			input: "[Fury](scc:mcdm.heroes.v1/class/fury) [dazed](scc:mcdm.heroes.v1/condition/dazed) [Fury](scc:mcdm.heroes.v1/class/fury).",
			want:  "[Fury](class/fury.md) [dazed](condition/dazed.md) Fury.",
		},
		{
			name:  "single occurrence kept",
			input: "[Fury](scc:mcdm.heroes.v1/class/fury) is great.",
			want:  "[Fury](class/fury.md) is great.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ResolveLinks(tt.input, "", LinkFirst)
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
		name  string
		input string
		want  string
	}{
		{
			name:  "unresolved markdown link stripped to display text",
			input: "See [Unknown](scc:mcdm.heroes.v1/class/unknown) for details.",
			want:  "See Unknown for details.",
		},
		{
			name:  "resolved link still works",
			input: "[Fury](scc:mcdm.heroes.v1/class/fury) is here.",
			want:  "[Fury](class/fury.md) is here.",
		},
		{
			name:  "mix of resolved and unresolved",
			input: "[Fury](scc:mcdm.heroes.v1/class/fury) and [Nope](scc:mcdm.heroes.v1/class/nope).",
			want:  "[Fury](class/fury.md) and Nope.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ResolveLinks(tt.input, "", LinkAll)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
