package plan

import (
	"strings"
	"testing"
)

// IQ: a reply point survives normalisation, is an action (no check is sought), and the prompt names it.
func TestIQReplyIsAnActionTheAnswerFulfils(t *testing.T) {
	kept := NormalizeAcceptance("Read calc.py. Say what bug add() has.", []Point{
		{Behaviour: "Read calc.py", Quote: "Read calc.py", Kind: "action"},
		{Behaviour: "State the bug in add()", Quote: "Say what bug add() has", Kind: "Reply"},
	})
	if len(kept) != 2 || kept[1].Kind != PointReply || !kept[1].Action() {
		t.Fatalf("kept %#v", kept)
	}
	if Behaviours(kept) != nil {
		t.Fatalf("no check is sought for an action or a reply: %#v", Behaviours(kept))
	}
	if !strings.Contains(acceptancePrompt, `kind "reply"`) {
		t.Fatal("the prompt never tells the model what a reply is")
	}
}
