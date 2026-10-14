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
	belt := iqPlainReadAlias([]bare.Tool{read, document})
	text, isError, err := belt[1].Execute(context.Background(), json.RawMessage(`{"path":"x.bin"}`))
	if err != nil || !isError || text != answer || calls != 0 {
		t.Fatalf("a non-plain refusal must pass through: %q %v %v (read ran %d)", text, isError, err, calls)
	}

	answer = plainReadRedirect + "x.txt is plain text"
	alone := iqPlainReadAlias([]bare.Tool{document})
	if text, _, _ := alone[0].Execute(context.Background(), json.RawMessage(`{"path":"x.txt"}`)); text != answer {
		t.Fatalf("without read on the belt the refusal stands: %q", text)
	}
}
