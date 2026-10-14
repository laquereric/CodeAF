package session

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Agent-Field/codeaf/internal/exec/bare"
)

// plainReadRedirect opens read_document's answer for a plain-text file: the
// file is read's, and the document road refuses to spend a rung on it.
const plainReadRedirect = "the plain read handles this: "

// iqPlainReadAlias is the IQ fork's one change to the reading hands: on a belt
// that carries both, read_document called on a plain-text file IS read — the
// same path, offset and limit, answered by read — instead of a refusal telling
// the model to call read. A small local model told "call read" keeps calling
// read_document and the turn goes round until the wall stops it; aliasing costs
// nothing, because the refusal fires before any paid rung is reached. Every
// other answer of read_document (a PDF, an image, a missing file, an
// unsupported format) passes through untouched.
//
// THE BASH BELT CARRIES NO read (bashbelt.go: a task worker reads through its
// shell), and it is the belt `codeaf do` runs on. Where the IQ seam has not
// granted Bash, that worker has no way left to open a text file, so a belt
// without read is handed one by fallback -- the same bare hand, built for the
// workspace -- used for this answer only and never put on the belt. A nil
// fallback, or a belt without read_document, leaves the belt as it came.
func iqPlainReadAlias(tools []bare.Tool, fallback func() bare.Tool) []bare.Tool {
	readAt, docAt := -1, -1
	for index, tool := range tools {
		switch tool.Name {
		case "read":
			readAt = index
		case "read_document":
			docAt = index
		}
	}
	if docAt < 0 || (readAt < 0 && fallback == nil) {
		return tools
	}
	var read func(context.Context, json.RawMessage) (string, bool, error)
	if readAt >= 0 {
		read = tools[readAt].Execute
	} else {
		read = fallback().Execute
	}
	document := tools[docAt].Execute
	tools[docAt].Execute = func(ctx context.Context, args json.RawMessage) (string, bool, error) {
		text, isError, err := document(ctx, args)
		if err != nil || !isError || !strings.HasPrefix(text, plainReadRedirect) {
			return text, isError, err
		}
		var parsed struct {
			Path   string `json:"path"`
			Offset *int   `json:"offset,omitempty"`
			Limit  *int   `json:"limit,omitempty"`
		}
		if decodeToolArguments(args, &parsed) != nil {
			return text, isError, err
		}
		forwarded, marshalErr := json.Marshal(parsed)
		if marshalErr != nil {
			return text, isError, err
		}
		return read(ctx, forwarded)
	}
	return tools
}

// iqWorkspaceRead is the bare read hand for this agent's workspace, at the
// belt's own result caps: the fallback the bash belt hands iqPlainReadAlias.
func (a *Agent) iqWorkspaceRead() bare.Tool {
	for _, tool := range bare.AllToolsCapped(a.config.Workspace, a.resultCaps()) {
		if tool.Name == "read" {
			return tool
		}
	}
	return bare.Tool{Name: "read", Execute: func(context.Context, json.RawMessage) (string, bool, error) {
		return "read is not available on this belt", true, nil
	}}
}
