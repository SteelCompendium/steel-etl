package scc

import "testing"

func TestClassifyAbility(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"abilities", "fury"}, "gouge")
	if scc != "mcdm.heroes.v1/abilities.fury/gouge" {
		t.Errorf("expected mcdm.heroes.v1/abilities.fury/gouge, got %s", scc)
	}
}

func TestClassifyClass(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"classes"}, "fury")
	if scc != "mcdm.heroes.v1/classes/fury" {
		t.Errorf("expected mcdm.heroes.v1/classes/fury, got %s", scc)
	}
}

func TestClassifyFeature(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"features", "fury"}, "growing-ferocity")
	if scc != "mcdm.heroes.v1/features.fury/growing-ferocity" {
		t.Errorf("expected mcdm.heroes.v1/features.fury/growing-ferocity, got %s", scc)
	}
}

func TestClassifyChapter(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"chapters"}, "introduction")
	if scc != "mcdm.heroes.v1/chapters/introduction" {
		t.Errorf("expected mcdm.heroes.v1/chapters/introduction, got %s", scc)
	}
}

func TestClassifyCommonAbility(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"abilities", "common"}, "grab")
	if scc != "mcdm.heroes.v1/abilities.common/grab" {
		t.Errorf("expected mcdm.heroes.v1/abilities.common/grab, got %s", scc)
	}
}

func TestClassifyCondition(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", []string{"conditions"}, "dazed")
	if scc != "mcdm.heroes.v1/conditions/dazed" {
		t.Errorf("expected mcdm.heroes.v1/conditions/dazed, got %s", scc)
	}
}

func TestClassifyEmptySource(t *testing.T) {
	scc := Classify("", []string{"classes"}, "fury")
	if scc != "classes/fury" {
		t.Errorf("expected classes/fury, got %s", scc)
	}
}

func TestClassifyEmptyTypePath(t *testing.T) {
	scc := Classify("mcdm.heroes.v1", nil, "fury")
	if scc != "mcdm.heroes.v1/fury" {
		t.Errorf("expected mcdm.heroes.v1/fury, got %s", scc)
	}
}
