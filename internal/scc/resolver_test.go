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
			name:  "inline scc link",
			input: "See scc:mcdm.heroes.v1/feature.ability.fury.level-1/gouge for details.",
			want:  "See feature/ability/fury/level-1/gouge.md for details.",
		},
		{
			name:  "markdown link",
			input: "[Gouge](scc:mcdm.heroes.v1/feature.ability.fury.level-1/gouge)",
			want:  "[Gouge](feature/ability/fury/level-1/gouge.md)",
		},
		{
			name:  "multiple links",
			input: "scc:mcdm.heroes.v1/class/fury and scc:mcdm.heroes.v1/condition/dazed",
			want:  "class/fury.md and condition/dazed.md",
		},
		{
			name:  "unknown scc left as-is",
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
			got := resolver.ResolveLinks(tt.input, "")
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

	input := "See scc:mcdm.heroes.v1/ability/gouge"
	want := "See feature/ability/fury/level-1/gouge.md"

	got := resolver.ResolveLinks(input, "")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
