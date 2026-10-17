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
