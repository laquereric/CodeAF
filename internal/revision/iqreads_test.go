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
	outcomes := AnswerChecklistReads(points, store.DeliveryGate{}, false, nil, read)
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
