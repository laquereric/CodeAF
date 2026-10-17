package main

import (
	"strings"
	"testing"

	"github.com/Agent-Field/codeaf/internal/plan"
	"github.com/Agent-Field/codeaf/internal/revision"
	"github.com/Agent-Field/codeaf/internal/store"
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

// A repair the re-judgment passed leaves the gate whole: the first judgment's coverage finding is replaced by the
// re-judgment's measurement, and a re-judgment that never ran replaces nothing.
func TestIQARepairThatClosedTheCoverageFindingLeavesTheGateWhole(t *testing.T) {
	first := revision.Judgment{Checked: true, Gaps: "2 behaviours ... asserted by no check",
		Unasserted: []string{"SAVE10 coupon gives 10% off — observables never asserted: save10"},
		Exercises:  []store.ExercisedPoint{{Point: "SAVE10 coupon gives 10% off", Check: "test_save10", Unasserted: []string{"save10"}}}}
	evidence := deliveryGateOf(first)
	evidence.PolishClosed = true
	if evidence.Whole() {
		t.Fatal("the stale finding should keep the row short before the fix")
	}
	untaken := evidence
	iqTakeRejudgedCoverage(&untaken, revision.Judgment{})
	if untaken.Whole() {
		t.Fatal("a re-judgment that never ran empties nothing")
	}
	rejudged := revision.Judgment{Checked: true, Pass: true,
		Exercises: []store.ExercisedPoint{{Point: "SAVE10 coupon gives 10% off", Check: "test_save10", Observables: []string{"save10"}}}}
	iqTakeRejudgedCoverage(&evidence, rejudged)
	if !evidence.Whole() || len(evidence.Unasserted) != 0 || evidence.Exercises[0].Unasserted != nil {
		t.Fatalf("the closed repair's measurement is the row's: %+v", evidence)
	}
	still := deliveryGateOf(first)
	still.PolishClosed = true
	iqTakeRejudgedCoverage(&still, revision.Judgment{Checked: true, Pass: true, Unasserted: first.Unasserted})
	if still.Whole() {
		t.Fatal("a re-judgment that still measures the gap keeps it")
	}
}
