package revision

import (
	"strings"

	"github.com/Agent-Field/codeaf/internal/plan"
	"github.com/Agent-Field/codeaf/internal/store"
)

// iqMapNamedChecks maps a point to the check it NAMES (IQ). "tests/test_cart.py::test_coupon passes" is about
// test_coupon by its own words, and asking a model whether test_coupon exercises it -- and getting "none" --
// left a fix whose named test went green unexercised. A check is named when the point (its behaviour or its
// quote) carries the check's whole id, or the check's own last segment as a whole word: test_coupon in
// tests/test_cart.py::test_coupon, TestParse in pkg.TestParse. A segment shorter than six characters, or one with
// a space (a sentence-named JS test), is too common a word to stand for a check and is never matched alone.
func iqMapNamedChecks(mapping []store.ExercisedPoint, points []plan.Point, checks []string) {
	for index := range mapping {
		if index >= len(points) || mapping[index].Check != "" {
			continue
		}
		for _, check := range checks {
			if iqPointNamesCheck(points[index], check) {
				mapping[index].Check = check
				break
			}
		}
	}
}

// iqPointNamesCheck reads a point's behaviour and quote for the check's name. The grounding and assertion doors
// let a named check through on the same reading: the point is about that check by its own words.
func iqPointNamesCheck(point plan.Point, check string) bool {
	return iqNamesCheck(point.Behaviour+" "+point.Quote, check)
}

func iqNamesCheck(text, check string) bool {
	check = strings.TrimSpace(check)
	if check == "" {
		return false
	}
	if strings.Contains(text, check) {
		return true
	}
	segment := check
	if cut := strings.LastIndexAny(segment, ":/."); cut >= 0 {
		segment = segment[cut+1:]
	}
	if len(segment) < 6 || strings.ContainsAny(segment, " \t") {
		return false
	}
	for _, word := range strings.FieldsFunc(text, func(r rune) bool {
		return !(r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	}) {
		if word == segment {
			return true
		}
	}
	return false
}

// iqWhereChecksGo is the coverage brief's sentence on where a check belongs (IQ). The first round's brief
// carries it too (IQWhereChecksGo, cmd/codeaf/iqtests.go), so a fix writes its checks before the gate asks.
const IQWhereChecksGo = iqWhereChecksGo

const iqWhereChecksGo = "Put each check where this project's own verification runs it: a new test file " +
	"beside its existing tests, named the way they are named (for pytest, tests/test_<name>.py), so the " +
	"project's own test command collects it. A check that command does not collect counts for nothing. " +
	"Adding a new test file is not changing the existing tests: a rule against changing the tests forbids " +
	"editing the tests that are there, never adding one beside them."
