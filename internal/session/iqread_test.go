package session

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Agent-Field/codeaf/internal/exec/bare"
)

// read_document on a plain-text file answers as read, with the same paging.
func TestIQPlainReadAliasPagesLikeRead(t *testing.T) {
	agent, workspace := newDocumentAgent(t, &scriptedParser{}, nil, nil)
	name := dropFile(t, workspace, "lines.txt", []byte("one\ntwo\nthree\nfour\n"))

	text, isError := documentTool(t, agent, map[string]any{"path": name, "offset": 3, "limit": 1})
	if isError {
		t.Fatalf("aliased read failed: %s", text)
	}
	if !strings.Contains(text, "three") || strings.Contains(text, "one") || strings.Contains(text, "four\n") {
		t.Fatalf("offset and limit should reach read: %q", text)
	}
}

// Every other answer of read_document passes through, and a belt missing either
// hand comes back as it was.
func TestIQPlainReadAliasLeavesTheRestAlone(t *testing.T) {
	calls := 0
	read := bare.Tool{Name: "read", Execute: func(context.Context, json.RawMessage) (string, bool, error) {
		calls++
		return "read", false, nil
	}}
	answer := "read_document reads PDFs, images and office documents; x.bin is none of those"
	document := bare.Tool{Name: "read_document", Execute: func(context.Context, json.RawMessage) (string, bool, error) {
		return answer, true, nil
	}}
	belt := iqPlainReadAlias([]bare.Tool{read, document}, nil)
	text, isError, err := belt[1].Execute(context.Background(), json.RawMessage(`{"path":"x.bin"}`))
	if err != nil || !isError || text != answer || calls != 0 {
		t.Fatalf("a non-plain refusal must pass through: %q %v %v (read ran %d)", text, isError, err, calls)
	}

	answer = plainReadRedirect + "x.txt is plain text"
	alone := iqPlainReadAlias([]bare.Tool{document}, nil)
	if text, _, _ := alone[0].Execute(context.Background(), json.RawMessage(`{"path":"x.txt"}`)); text != answer {
		t.Fatalf("without read and without a fallback the refusal stands: %q", text)
	}
}

// The bash belt has no read: the fallback answers the plain-text call, and is
// never added to the belt.
func TestIQPlainReadAliasFallsBackWhereTheBeltHasNoRead(t *testing.T) {
	agent, workspace := newDocumentAgent(t, &scriptedParser{}, nil, nil)
	name := dropFile(t, workspace, "calc.py", []byte("def add(a, b):\n    return a - b\n"))
	var document bare.Tool
	for _, tool := range agent.belt() {
		if tool.Name == "read_document" {
			document = tool
		}
	}
	document.Execute = func(context.Context, json.RawMessage) (string, bool, error) {
		return plainReadRedirect + name + " is plain text", true, nil
	}
	belt := iqPlainReadAlias([]bare.Tool{{Name: "bash"}, document}, agent.iqWorkspaceRead)
	if len(belt) != 2 {
		t.Fatalf("the fallback must not join the belt: %d tools", len(belt))
	}
	text, isError, err := belt[1].Execute(context.Background(), json.RawMessage(`{"path":"`+name+`"}`))
	if err != nil || isError || !strings.Contains(text, "return a - b") {
		t.Fatalf("the fallback read should answer with the file: %q %v %v", text, isError, err)
	}
}
