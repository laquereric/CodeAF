// @use_case Development/UseCase/UC_MetaHarness
// @facet Agent
//
// iqread.go -- read_document reads plain text on the leaf toolbox. The node belt's leaf has no plain read: a
// worker reads through sh, or through read_document once `capabilities` arms it, and read_document took only
// PDF/DOCX/PPTX. Under an IQ policy that refuses Bash (sh) that left a governed worker unable to open a
// source file at all. So a plain-text file sent to read_document is read here, locally and free, numbered and
// paged like the session's read; and while the IQ gate is on, the documents family is on offer even on a
// machine with no document client -- the plain-text half needs none. IQ_SEAM_URL unset -> what is offered
// is upstream's; the plain-text read itself holds either way (it only replaces an error).
package exec

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Agent-Field/agentfield/sdk/go/ai"
	"github.com/Agent-Field/codeaf/internal/iqgate"
)

const iqPlainReadDefaultLines = 2000

// iqPlainDocuments: the documents family is offered for plain text alone -- only while IQ governs.
func iqPlainDocuments() bool { return iqgate.Enabled() }

// documentsAvailable: read_document can serve something on this machine.
func (t *Toolbox) documentsAvailable() bool {
	return (t.media != nil && t.media.DocumentClient != nil) || iqPlainDocuments()
}

// readPlainText answers read_document on a plain-text file: lines offset.. (1-based), at most limit of them,
// numbered, with a closing line naming the next offset when the file goes on.
func (t *Toolbox) readPlainText(path, full string, args map[string]any) Result {
	offset := intArgOr(args, "offset", 1)
	if offset < 1 {
		offset = 1
	}
	limit := intArgOr(args, "limit", iqPlainReadDefaultLines)
	if limit < 1 {
		limit = iqPlainReadDefaultLines
	}
	file, err := os.Open(full)
	if err != nil {
		return errorf("could not read %s", filepath.ToSlash(path))
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	var out strings.Builder
	line, shown := 0, 0
	more := false
	for scanner.Scan() {
		line++
		if line < offset {
			continue
		}
		if shown == limit {
			more = true
			break
		}
		fmt.Fprintf(&out, "%6d\t%s\n", line, scanner.Text())
		shown++
	}
	if err := scanner.Err(); err != nil {
		return errorf("could not read %s: %v", filepath.ToSlash(path), err)
	}
	if shown == 0 {
		if line == 0 {
			return Result{Content: filepath.ToSlash(path) + " is empty"}
		}
		return Result{Content: fmt.Sprintf("%s has %d lines; offset %d is past the end", filepath.ToSlash(path), line, offset)}
	}
	if more {
		fmt.Fprintf(&out, "(more: read_document path=%s offset=%d)\n", filepath.ToSlash(path), offset+shown)
	}
	return Result{Content: out.String()}
}

// intArgOr reads an integer argument a model may send as a number or a numeric string.
func intArgOr(args map[string]any, key string, fallback int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		var n int
		if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &n); err == nil {
			return n
		}
	}
	return fallback
}

// iqLooksLikePlainText: valid UTF-8 with no NUL in the first 8 KB -- internal/session/tools_doc.go's test.
func iqLooksLikePlainText(full string) bool {
	file, err := os.Open(full)
	if err != nil {
		return false
	}
	defer file.Close()
	prefix := make([]byte, 8<<10)
	n, _ := file.Read(prefix)
	prefix = prefix[:max(n, 0)]
	if bytes.IndexByte(prefix, 0) >= 0 {
		return false
	}
	for len(prefix) > 0 && !utf8.Valid(prefix) && (prefix[len(prefix)-1]&0xc0) == 0x80 {
		prefix = prefix[:len(prefix)-1]
	}
	return utf8.Valid(prefix)
}

// iqPlainDocumentDefinition: while IQ governs, read_document's schema says it reads text files too, and is in
// the catalog with no document client at all (the plain-text half needs none).
func (t *Toolbox) iqPlainDocumentDefinition(catalog map[string]ai.ToolDefinition) {
	if !iqPlainDocuments() {
		return
	}
	catalog["read_document"] = define("read_document", "Read a text or source file (numbered lines; page with offset and limit), or a PDF, DOCX or PPTX into text. Free and local for text; scanned documents escalate to OCR through the rail. Repeat document reads are cached.", map[string]any{
		"path":     prop("string", "workspace-relative path: a text or source file, or a PDF, DOCX or PPTX"),
		"offset":   prop("integer", "text files: first line to show, default 1"),
		"limit":    prop("integer", "text files: how many lines, default 2000"),
		"pages":    prop("string", "optional PDF page range such as 1-5"),
		"question": prop("string", "optional question the worker will answer from the extracted text"),
	}, "path")
}

// iqNoteRead records a successful read_document under this leaf, for the run's checklist (Outcome.Consulted).
func (t *Toolbox) iqNoteRead(args map[string]any) {
	path := strings.TrimSpace(stringArg(args, "path"))
	if path == "" || t.workspace == nil {
		return
	}
	if full, err := t.workspace.Resolve(path); err == nil {
		t.workspace.RecordConsulted(t.leaf, full)
	}
}

// RecordConsulted notes a file a leaf read; Consulted lists them, workspace-relative and sorted.
func (w *Workspace) RecordConsulted(leaf, full string) {
	relative, err := filepath.Rel(w.root, full)
	if err != nil || strings.HasPrefix(relative, "..") {
		if relative, err = filepath.Rel(w.real, full); err != nil || strings.HasPrefix(relative, "..") {
			return
		}
	}
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.iqConsulted == nil {
		w.iqConsulted = map[string]map[string]bool{}
	}
	if w.iqConsulted[leaf] == nil {
		w.iqConsulted[leaf] = map[string]bool{}
	}
	w.iqConsulted[leaf][relative] = true
}

func (w *Workspace) Consulted(leaf string) []string {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	paths := make([]string, 0, len(w.iqConsulted[leaf]))
	for path := range w.iqConsulted[leaf] {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
