package exec

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func iqNames(tools *Toolbox) map[string]bool {
	names := map[string]bool{}
	for _, definition := range tools.Definitions() {
		names[definition.Function.Name] = true
	}
	return names
}

// read_document on a plain-text file reads it -- numbered, paged -- instead of refusing it.
func TestIQReadDocumentReadsPlainText(t *testing.T) {
	space := workspace(t)
	if err := os.WriteFile(filepath.Join(space.Root(), "calc.py"), []byte("def add(a, b):\n    return a - b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := NewToolbox(space, "2", nil)
	result := tools.Execute(context.Background(), "read_document", `{"path":"calc.py"}`)
	if result.IsError || !strings.Contains(result.Content, "     2\t    return a - b") {
		t.Fatalf("plain text should be read: %+v", result)
	}
	var lines []string
	for i := 1; i <= 5; i++ {
		lines = append(lines, "line")
	}
	_ = os.WriteFile(filepath.Join(space.Root(), "long.txt"), []byte(strings.Join(lines, "\n")+"\n"), 0o644)
	paged := tools.Execute(context.Background(), "read_document", `{"path":"long.txt","offset":2,"limit":"2"}`)
	if paged.IsError || !strings.HasPrefix(paged.Content, "     2\tline\n     3\tline\n") || !strings.Contains(paged.Content, "offset=4") {
		t.Fatalf("paging: %q", paged.Content)
	}
	_ = os.WriteFile(filepath.Join(space.Root(), "blob.bin"), []byte{0x7f, 0, 1, 2}, 0o644)
	if binary := tools.Execute(context.Background(), "read_document", `{"path":"blob.bin"}`); !binary.IsError {
		t.Fatalf("a binary file is still refused: %+v", binary)
	}
}

// Ungoverned and with no document client, nothing new is offered; governed, the documents family is.
func TestIQDocumentsOfferedOnlyWhileGoverned(t *testing.T) {
	tools := NewToolbox(workspace(t), "2", nil)
	if names := iqNames(tools); names["capabilities"] || names["read_document"] {
		t.Fatalf("ungoverned, no client: nothing to arm, got %v", names)
	}
	t.Setenv("IQ_SEAM_URL", "http://127.0.0.1:1")
	if !iqNames(tools)["capabilities"] {
		t.Fatal("governed: capabilities is on offer")
	}
	if result := tools.execute(context.Background(), "capabilities", `{"need":"documents"}`); result.IsError {
		t.Fatalf("documents arm with no client: %+v", result)
	}
	if !iqNames(tools)["read_document"] {
		t.Fatal("armed read_document carries its schema")
	}
}

// A successful read_document is recorded under the leaf; a refused one is not.
func TestIQReadDocumentIsConsulted(t *testing.T) {
	space := workspace(t)
	_ = os.WriteFile(filepath.Join(space.Root(), "calc.py"), []byte("def add(a, b):\n"), 0o644)
	tools := NewToolbox(space, "7", nil)
	tools.Execute(context.Background(), "read_document", `{"path":"calc.py"}`)
	tools.Execute(context.Background(), "read_document", `{"path":"absent.py"}`)
	if got := space.Consulted("7"); len(got) != 1 || got[0] != "calc.py" {
		t.Fatalf("consulted = %v", got)
	}
}
