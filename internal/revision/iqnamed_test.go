package revision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Agent-Field/codeaf/internal/plan"
	"github.com/Agent-Field/codeaf/internal/store"
)

// A point that names a test is exercised by it, through the grounding and assertion doors; a point that only
// shares a short word with a check is not, and an unnamed point is left for the model.
func TestIQAPointThatNamesACheckIsExercisedByIt(t *testing.T) {
	space := t.TempDir()
	if err := os.MkdirAll(filepath.Join(space, "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "def test_coupon():\n    cart = Cart()\n    cart.add(1000, 2)\n    cart.use_coupon(\"SAVE10\")\n    assert cart.subtotal() == 1800\n"
	if err := os.WriteFile(filepath.Join(space, "tests", "test_cart.py"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	points := []plan.Point{
		{Behaviour: "tests/test_cart.py::test_coupon passes instead of failing with 'assert 1998.0 == 1800'",
			Quote: "tests/test_cart.py::test_coupon fails with 'assert 1998.0 == 1800'", Kind: plan.PointBehaviour},
		{Behaviour: "SAVE25 coupon gives 25% off", Quote: "SAVE25 is 25% off", Kind: plan.PointBehaviour},
		{Behaviour: "The cart total is correct", Quote: "the cart total", Kind: plan.PointBehaviour},
	}
	checks := []string{"tests/test_cart.py::test_coupon", "tests/test_cart.py::test_tax", "tests/test_cart.py::cart"}
	mapping := make([]store.ExercisedPoint, len(points))
	for index, point := range points {
		mapping[index].Point = point.Behaviour
	}
	iqMapNamedChecks(mapping, points, checks)
	if mapping[0].Check != "tests/test_cart.py::test_coupon" {
		t.Fatalf("the named test exercises its point: %+v", mapping[0])
	}
	if mapping[1].Check != "" || mapping[2].Check != "" {
		t.Fatalf("unnamed points, and a five-letter word, are left for the model: %+v", mapping[1:])
	}
	grounded := GroundMapping(space, nil, points, mapping)
	weighed := WeighAssertions(space, "job", nil, points, grounded)
	if weighed[0].Check != "tests/test_cart.py::test_coupon" || len(weighed[0].Unasserted) != 0 {
		t.Fatalf("a named check passes both doors: %+v", weighed[0])
	}
	if !iqNamesCheck("TestParse handles empty input", "pkg.TestParse") || iqNamesCheck("adds two numbers", "math adds two numbers") {
		t.Fatal("a Go-style name is matched by its last segment; a sentence-named test only whole")
	}
}

// The coverage brief says where a check goes, and that adding a test file is not changing the tests.
func TestIQTheCoverageBriefSaysWhereAChecksGoes(t *testing.T) {
	judgment := unexercisedFinding(coverage{missing: []string{"SAVE25 gives 25% off"},
		missingLines: []string{"SAVE25 gives 25% off"}}, 3)
	brief := judgment.Gaps
	for _, want := range []string{"Write the check for each, and make it pass.", "where this project's own verification runs it",
		"tests/test_<name>.py", "Adding a new test file is not changing the existing tests"} {
		if !strings.Contains(brief, want) {
			t.Fatalf("the brief never says %q:\n%s", want, brief)
		}
	}
}
