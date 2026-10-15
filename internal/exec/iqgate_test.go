package exec

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type iqFakeSeam struct {
	mu       sync.Mutex
	pushes   []map[string]any
	observes []map[string]any
	refuse   map[string]bool
}

func (f *iqFakeSeam) serve(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		defer f.mu.Unlock()
		switch r.URL.Path {
		case "/push":
			f.pushes = append(f.pushes, body)
			tool, _ := body["tool"].(string)
			if f.refuse[tool] {
				_ = json.NewEncoder(w).Encode(map[string]any{"admitted": false, "reason": "no_permit"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"admitted": true})
		case "/observe":
			f.observes = append(f.observes, body)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// A refused tool does not run: the shell the seam refuses writes nothing.
func TestIQGateRefusesALeafShellBeforeItRuns(t *testing.T) {
	seam := &iqFakeSeam{refuse: map[string]bool{"sh": true}}
	t.Setenv("IQ_SEAM_URL", seam.serve(t))
	t.Setenv("IQ_HARNESS_SESSION", "tok")
	space := workspace(t)
	tools := NewToolbox(space, "2", nil)

	result := tools.Execute(context.Background(), "sh", `{"cmd":"echo ran > ran.txt"}`)
	if !result.IsError || !strings.Contains(result.Content, "IQ refusal: mcp:call:sh (no_permit)") {
		t.Fatalf("a refused call should come back as the refusal: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(space.Root(), "ran.txt")); err == nil {
		t.Fatal("the refused shell ran")
	}
	if len(seam.pushes) != 1 || seam.pushes[0]["session"] != "tok" {
		t.Fatalf("one push with the credential, got %v", seam.pushes)
	}
	if args, _ := seam.pushes[0]["args"].(map[string]any); len(args) != 0 {
		t.Fatalf("a command never leaves for the seam: %v", args)
	}
}

// An admitted tool runs and is observed under the same operation and session.
func TestIQGateObservesAnAdmittedLeafCall(t *testing.T) {
	seam := &iqFakeSeam{}
	t.Setenv("IQ_SEAM_URL", seam.serve(t))
	space := workspace(t)
	tools := NewToolbox(space, "2", nil)

	result := tools.Execute(context.Background(), "write", `{"path":"a.txt","content":"hi"}`)
	if result.IsError {
		t.Fatalf("admitted write failed: %s", result.Content)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		seam.mu.Lock()
		n := len(seam.observes)
		seam.mu.Unlock()
		if n == 1 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	seam.mu.Lock()
	defer seam.mu.Unlock()
	if len(seam.observes) != 1 {
		t.Fatalf("one observe row, got %d", len(seam.observes))
	}
	push, observe := seam.pushes[0], seam.observes[0]
	if push["operationId"] != observe["operationId"] || push["harness_session"] != observe["harness_session"] {
		t.Fatalf("push and observe must join: %v / %v", push, observe)
	}
	if args, _ := push["args"].(map[string]any); args["path"] != "a.txt" {
		t.Fatalf("the path is the artifact name: %v", push["args"])
	}
	if observe["ok"] != true {
		t.Fatalf("observed ok: %v", observe)
	}
	if !strings.HasPrefix(push["harness_session"].(string), "codeaf-2-") {
		t.Fatalf("harness_session names the leaf: %v", push["harness_session"])
	}
}

// A seam that cannot be reached refuses (fail closed).
func TestIQGateFailsClosed(t *testing.T) {
	t.Setenv("IQ_SEAM_URL", "http://127.0.0.1:1")
	space := workspace(t)
	result := NewToolbox(space, "2", nil).Execute(context.Background(), "write", `{"path":"b.txt","content":"x"}`)
	if !result.IsError || !strings.Contains(result.Content, "IQ gate unreachable") {
		t.Fatalf("unreachable seam must refuse: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(space.Root(), "b.txt")); err == nil {
		t.Fatal("the write ran with no seam to ask")
	}
}

// The package's suite never asks a live seam a shell happens to export; the gate's tests set their own.
func TestMain(m *testing.M) {
	os.Unsetenv("IQ_SEAM_URL")
	os.Unsetenv("IQ_HARNESS_SESSION")
	os.Exit(m.Run())
}
