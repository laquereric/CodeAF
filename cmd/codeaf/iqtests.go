// @use_case Development/UseCase/UC_MetaHarness
// @facet Agent
//
// iqtests.go -- the first round is told which behaviours need a check (IQ). The delivery gate measures whether a
// check the project's own command collects exercises each behaviour the request states; a first round that fixed
// the code and wrote no checks was failed for it every time, and the coverage round that followed cost as much
// as the fix. So the brief names those behaviours up front, with where their checks go. A request that states no
// behaviour of finished work (read, explain, report) is told nothing.
package main

import (
	"strings"

	"github.com/Agent-Field/codeaf/internal/plan"
	"github.com/Agent-Field/codeaf/internal/revision"
	"github.com/Agent-Field/codeaf/internal/store"
	"github.com/Agent-Field/codeaf/internal/verify"
)

func iqWithTestGuidance(contract string, spec plan.Spec, intent string) string {
	points := spec.Accept
	if len(points) == 0 {
		points = revision.ChecklistFor(verify.JobKey(intent))
	}
	behaviours := plan.Behaviours(points)
	if len(behaviours) == 0 {
		return contract
	}
	var guidance strings.Builder
	guidance.WriteString("What the work is checked on:\nThe finished work is accepted only where a check in the " +
		"project's own tests exercises each behaviour the request states, and would fail if it were absent or wrong:\n")
	for _, point := range behaviours {
		guidance.WriteString("- " + strings.TrimSpace(point.Behaviour) + "\n")
	}
	guidance.WriteString("For each one no existing test already exercises, write that test as part of the work. " +
		revision.IQWhereChecksGo)
	if strings.TrimSpace(contract) == "" {
		return guidance.String()
	}
	return strings.TrimRight(contract, "\n") + "\n\n" + guidance.String()
}

// iqTakeRejudgedCoverage puts the re-judgment's coverage measurement on the gate row when a repair closed the gate
// (IQ). The row is built from the FIRST judgment, and a repair that passed its re-judgment only ever set
// PolishClosed: the first judgment's Unexercised and Unasserted stayed on the row, store.DeliveryGate.Whole refused
// it, and the run left "incomplete" over a finding the re-judgment had measured closed (a test rewritten in the
// repair to assert `save10`, still recorded as asserting nothing). A coverage finding empties the one way it does
// -- a later mapping measures a check that covers it -- and the re-judgment's mapping is that later mapping. A
// re-judgment that took no measurement carries the standing finding (revision.UnexercisedFor), so nothing is
// emptied that was not measured.
func iqTakeRejudgedCoverage(evidence *store.DeliveryGate, rejudged revision.Judgment) {
	if !rejudged.Checked {
		return
	}
	row := deliveryGateOf(rejudged)
	evidence.Exercises, evidence.Unexercised, evidence.Unasserted = row.Exercises, row.Unexercised, row.Unasserted
}
