package main

import (
	"strings"
	"testing"

	"github.com/Agent-Field/codeaf/internal/plan"
)

// The first round is told the behaviours that need a check and where checks go; a read-only request is told nothing.
func TestIQTheFirstRoundIsToldWhichBehavioursNeedACheck(t *testing.T) {
	spec := plan.Spec{Accept: []plan.Point{
		{Behaviour: "SAVE25 coupon gives 25% off", Quote: "SAVE25 is 25% off", Kind: plan.PointBehaviour},
		{Behaviour: "Fix is the smallest correct change", Quote: "smallest correct change", Kind: plan.PointAction},
		{Behaviour: "Report what changed", Quote: "Report what you changed", Kind: plan.PointReply},
	}}
	brief := iqWithTestGuidance("Rules the person set, never broken:\n- Do not change the tests.", spec, "")
	for _, want := range []string{"Do not change the tests.", "- SAVE25 coupon gives 25% off", "tests/test_<name>.py",
		"Adding a new test file is not changing the existing tests"} {
		if !strings.Contains(brief, want) {
			t.Fatalf("the brief never says %q:\n%s", want, brief)
		}
	}
	if strings.Contains(brief, "smallest correct change") || strings.Contains(brief, "Report what changed") {
		t.Fatalf("an action or a reply needs no check:\n%s", brief)
	}
	readOnly := plan.Spec{Accept: spec.Accept[1:]}
	if got := iqWithTestGuidance("method", readOnly, ""); got != "method" {
		t.Fatalf("a request with no behaviour is told nothing: %q", got)
	}
}
