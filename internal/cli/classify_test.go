package cli

import "testing"

func TestGroupByType(t *testing.T) {
	codes := []string{
		"mcdm.heroes.v1/chapter/introduction",
		"mcdm.heroes.v1/chapter/classes",
		"mcdm.heroes.v1/class/fury",
		"mcdm.heroes.v1/class/shadow",
		"mcdm.heroes.v1/feature.ability.fury.level-1/gouge",
		"mcdm.heroes.v1/feature.ability.fury.level-1/brutal-slam",
		"mcdm.heroes.v1/feature.trait.fury.level-1/growing-ferocity",
		"mcdm.heroes.v1/ancestry/dwarf",
		"mcdm.heroes.v1/kit/panther",
	}

	grouped := groupByType(codes)

	tests := []struct {
		typeName string
		count    int
	}{
		{"chapter", 2},
		{"class", 2},
		{"feature.ability", 2},
		{"feature.trait", 1},
		{"ancestry", 1},
		{"kit", 1},
	}

	for _, tt := range tests {
		items, ok := grouped[tt.typeName]
		if !ok {
			t.Errorf("expected type group %q", tt.typeName)
			continue
		}
		if len(items) != tt.count {
			t.Errorf("type %q: expected %d items, got %d", tt.typeName, tt.count, len(items))
		}
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string][]string{
		"z": {"1"},
		"a": {"2"},
		"m": {"3"},
	}
	keys := sortedKeys(m)
	expected := []string{"a", "m", "z"}
	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("index %d: expected %q, got %q", i, expected[i], k)
		}
	}
}

func TestToSet(t *testing.T) {
	slice := []string{"a", "b", "c", "a"}
	s := toSet(slice)

	if len(s) != 3 {
		t.Errorf("expected 3 unique items, got %d", len(s))
	}
	for _, v := range []string{"a", "b", "c"} {
		if !s[v] {
			t.Errorf("expected %q in set", v)
		}
	}
}

func TestFilterIssues(t *testing.T) {
	issues := []validationIssue{
		{level: "error", msg: "e1"},
		{level: "warn", msg: "w1"},
		{level: "error", msg: "e2"},
		{level: "info", msg: "i1"},
	}

	errors := filterIssues(issues, "error")
	if len(errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errors))
	}

	warns := filterIssues(issues, "warn")
	if len(warns) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warns))
	}

	infos := filterIssues(issues, "info")
	if len(infos) != 1 {
		t.Errorf("expected 1 info, got %d", len(infos))
	}
}

func TestValidationIssueString(t *testing.T) {
	tests := []struct {
		issue    validationIssue
		expected string
	}{
		{
			issue:    validationIssue{level: "error", heading: "Fury", hlevel: 2, msg: "unknown type"},
			expected: `[H2 "Fury"] unknown type`,
		},
		{
			issue:    validationIssue{level: "error", msg: "global error"},
			expected: "global error",
		},
	}

	for _, tt := range tests {
		got := tt.issue.String()
		if got != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, got)
		}
	}
}

