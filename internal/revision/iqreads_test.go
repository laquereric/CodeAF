package revision

import (
	"testing"

	"github.com/Agent-Field/codeaf/internal/store"
)

// An action point naming a file the run read is answered by the read; a behaviour point is not, and a run
// that read nothing is still told nothing it wrote is that file.
func TestIQChecklistCreditsAnActionPointsRead(t *testing.T) {
	points := []store.AcceptancePoint{
		{Behaviour: "Read calc.py using the file-reading tool", Quote: "Read calc.py", Kind: "action"},
		{Behaviour: "calc.py's add returns the sum", Quote: "fix calc.py", Kind: "behaviour"},
	}
	read := []string{"/w/calc.py"}
	outcomes := AnswerChecklistReads(points, store.DeliveryGate{}, false, nil, read, false)
	if outcomes[0].State != PointAnswered || outcomes[0].Why != "the run read calc.py" {
		t.Fatalf("the read answers the action point: %+v", outcomes[0])
	}
	if outcomes[1].State != PointNotAnswered {
		t.Fatalf("a behaviour is not settled by looking at it: %+v", outcomes[1])
	}
	if unread := AnswerChecklist(points, store.DeliveryGate{}, false, nil); unread[0].State != PointNotAnswered {
		t.Fatalf("no reads, upstream's answer: %+v", unread[0])
	}
}

// A reply point is answered by a clean run's answer, and only by one.
func TestIQChecklistCreditsAReplyPointToTheAnswer(t *testing.T) {
	points := []store.AcceptancePoint{
		{Behaviour: "State in one sentence what bug add() has", Quote: "say in one sentence what bug add() has", Kind: "reply"},
		{Behaviour: "Run the tests", Quote: "run the tests", Kind: "action"},
	}
	outcomes := AnswerChecklistReads(points, store.DeliveryGate{}, false, nil, nil, true)
	if outcomes[0].State != PointAnswered || outcomes[0].Why != "the run's answer gives it" {
		t.Fatalf("the answer gives the reply point: %+v", outcomes[0])
	}
	if outcomes[1].State == PointAnswered {
		t.Fatalf("an answer does not run the tests: %+v", outcomes[1])
	}
	if silent := AnswerChecklistReads(points, store.DeliveryGate{}, false, nil, nil, false); silent[0].State == PointAnswered {
		t.Fatalf("no clean answer, nothing credited: %+v", silent[0])
	}
}

// An action point naming folders is answered when every folder has a read under it, and only then.
func TestIQChecklistCreditsAFolderPointsReads(t *testing.T) {
	points := []store.AcceptancePoint{
		{Behaviour: "Read the code under shop/ and tests/ with the file-reading tool", Quote: "Read the code under shop/ and tests/", Kind: "action"},
		{Behaviour: "The code under shop/ computes the coupon total", Quote: "fix shop/", Kind: "behaviour"},
		{Behaviour: "Look through src/api/ for the handler", Quote: "look through `src/api/`", Kind: "action"},
	}
	read := []string{"/w/shop/cart.py", "/w/shop/discounts.py", "/w/tests/test_cart.py", "/w/myshop/x.py"}
	outcomes := AnswerChecklistReads(points, store.DeliveryGate{}, false, nil, read, false)
	if outcomes[0].State != PointAnswered || outcomes[0].Why != "the run read 2 files under shop/, 1 under tests/" {
		t.Fatalf("the reads answer the folder point: %+v", outcomes[0])
	}
	if outcomes[1].State == PointAnswered {
		t.Fatalf("a behaviour is not settled by looking at it: %+v", outcomes[1])
	}
	if outcomes[2].State == PointAnswered {
		t.Fatalf("nothing was read under src/api/: %+v", outcomes[2])
	}
	partial := AnswerChecklistReads(points[:1], store.DeliveryGate{}, false, nil, read[:2], false)
	if partial[0].State != PointNotReached {
		t.Fatalf("tests/ unread leaves the point as upstream leaves it: %+v", partial[0])
	}
	nested := AnswerChecklistReads(points[2:], store.DeliveryGate{}, false, nil, []string{"/w/src/api/routes.go"}, false)
	if nested[0].State != PointAnswered || nested[0].Why != "the run read 1 file under src/api/" {
		t.Fatalf("a nested folder is matched whole: %+v", nested[0])
	}
}
