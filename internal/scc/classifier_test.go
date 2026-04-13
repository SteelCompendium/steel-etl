package scc

import "testing"

func TestClassifyAbility(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"feature", "ability", "fury", "level-1"}, "gouge")
	if scc != "mcdm.heroes.v1/feature.ability.fury.level-1/gouge" {
		t.Errorf("expected mcdm.heroes.v1/feature.ability.fury.level-1/gouge, got %s", scc)
	}
}

func TestClassifyClass(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"class"}, "fury")
	if scc != "mcdm.heroes.v1/class/fury" {
		t.Errorf("expected mcdm.heroes.v1/class/fury, got %s", scc)
	}
}

func TestClassifyTrait(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"feature", "trait", "fury", "level-1"}, "growing-ferocity")
	if scc != "mcdm.heroes.v1/feature.trait.fury.level-1/growing-ferocity" {
		t.Errorf("expected mcdm.heroes.v1/feature.trait.fury.level-1/growing-ferocity, got %s", scc)
	}
}

func TestClassifyChapter(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"chapter"}, "introduction")
	if scc != "mcdm.heroes.v1/chapter/introduction" {
		t.Errorf("expected mcdm.heroes.v1/chapter/introduction, got %s", scc)
	}
}

func TestClassifyCommonAbility(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"feature", "ability", "common"}, "grab")
	if scc != "mcdm.heroes.v1/feature.ability.common/grab" {
		t.Errorf("expected mcdm.heroes.v1/feature.ability.common/grab, got %s", scc)
	}
}

func TestClassifyKitTrait(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"feature", "trait", "fury", "level-1", "boren"}, "kit-bonuses")
	if scc != "mcdm.heroes.v1/feature.trait.fury.level-1.boren/kit-bonuses" {
		t.Errorf("expected mcdm.heroes.v1/feature.trait.fury.level-1.boren/kit-bonuses, got %s", scc)
	}
}

func TestClassifyCondition(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"condition"}, "dazed")
	if scc != "mcdm.heroes.v1/condition/dazed" {
		t.Errorf("expected mcdm.heroes.v1/condition/dazed, got %s", scc)
	}
}

func TestClassifyEmptySource(t *testing.T) {
	scc := Classify("", []string{"class"}, "fury")
	if scc != "class/fury" {
		t.Errorf("expected class/fury, got %s", scc)
	}
}

func TestClassifyEmptyTypePath(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", nil, "fury")
	if scc != "mcdm.heroes.v1/fury" {
		t.Errorf("expected mcdm.heroes.v1/fury, got %s", scc)
	}
}
