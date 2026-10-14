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
// unsupported format) passes through untouched, and a belt without read (or
// without read_document) is returned as it came.
func iqPlainReadAlias(tools []bare.Tool) []bare.Tool {
	readAt, docAt := -1, -1
	for index, tool := range tools {
		switch tool.Name {
		case "read":
			readAt = index
		case "read_document":
			docAt = index
		}
	}
	if readAt < 0 || docAt < 0 {
		return tools
	}
	read, document := tools[readAt].Execute, tools[docAt].Execute
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
