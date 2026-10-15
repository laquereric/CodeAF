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
