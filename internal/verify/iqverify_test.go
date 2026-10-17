package verify

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

// A fake seam: admits mcp:call:verify only when allow is set, and keeps what it was asked.
type iqFakeSeam struct {
	allow    bool
	mu       sync.Mutex
	pushes   []map[string]any
	observes []map[string]any
}

func (f *iqFakeSeam) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body := map[string]any{}
	_ = json.NewDecoder(r.Body).Decode(&body)
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path == "/observe" {
		f.observes = append(f.observes, body)
		_, _ = w.Write([]byte(`{}`))
		return
	}
	f.pushes = append(f.pushes, body)
	reason := ""
	if !f.allow {
		reason = "no_permit"
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"admitted": f.allow, "reason": reason})
}

func iqReadingIn(t *testing.T, seam string) (string, Strategy) {
	t.Helper()
	t.Setenv("IQ_SEAM_URL", seam)
	t.Setenv("IQ_HARNESS_SESSION", "tok")
	space := t.TempDir()
	return space, Strategy{Command: "touch ran", Read: FormatPlain}
}

// A refused reading never runs the workspace's command, and says the refusal is why.
func TestIQAReadingTheSeamRefusesIsNeverRun(t *testing.T) {
	fake := &iqFakeSeam{}
	server := httptest.NewServer(fake)
	defer server.Close()
	space, strategy := iqReadingIn(t, server.URL)

	result, ok := RunReading(context.Background(), space, strategy, 10*time.Second)
	if ok {
		t.Fatalf("a refused reading is not taken: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(space, "ran")); err == nil {
		t.Fatal("the command ran although the seam refused it")
	}
	if !strings.Contains(result.Error, "IQ refusal: codeaf:verify (no_permit)") || !strings.Contains(result.Error, "`touch ran`") {
		t.Fatalf("the refusal is the reason: %q", result.Error)
	}
	if len(fake.pushes) != 1 || fake.pushes[0]["tool"] != "verify" || fake.pushes[0]["action"] != "codeaf:verify" || fake.pushes[0]["session"] != "tok" {
		t.Fatalf("asked as verify with the credential: %+v", fake.pushes)
	}
	if args, _ := fake.pushes[0]["args"].(map[string]any); args["path"] != "." {
		t.Fatalf("the resource is the directory it runs in: %+v", fake.pushes[0])
	}
	if hs, _ := fake.pushes[0]["harness_session"].(string); !strings.HasPrefix(hs, "codeaf-verify-") {
		t.Fatalf("harness_session: %q", hs)
	}

	// The photograph's reason is the refusal, not "no shell".
	if reading := photograph(context.Background(), space, Plan{}, []Strategy{strategy}, 10*time.Second); !strings.Contains(reading.Unread, "IQ refusal") {
		t.Fatalf("unread: %q", reading.Unread)
	}
}

// An admitted reading runs and is observed; an unreachable seam fails closed.
func TestIQAnAdmittedReadingRunsAndIsObserved(t *testing.T) {
	fake := &iqFakeSeam{allow: true}
	server := httptest.NewServer(fake)
	space, strategy := iqReadingIn(t, server.URL)

	if _, ok := RunReading(context.Background(), space, strategy, 10*time.Second); !ok {
		t.Fatal("an admitted reading is taken")
	}
	if _, err := os.Stat(filepath.Join(space, "ran")); err != nil {
		t.Fatal("the admitted command did not run")
	}
	if len(fake.observes) != 1 || fake.observes[0]["ok"] != true || fake.observes[0]["operationId"] != fake.pushes[0]["operationId"] {
		t.Fatalf("observed under the pushed operation: %+v / %+v", fake.observes, fake.pushes)
	}

	server.Close()
	_ = os.Remove(filepath.Join(space, "ran"))
	if result, ok := RunReading(context.Background(), space, strategy, 10*time.Second); ok || !strings.Contains(result.Error, "IQ gate unreachable") {
		t.Fatalf("an unreachable seam fails closed: %v %q", ok, result.Error)
	}
	if _, err := os.Stat(filepath.Join(space, "ran")); err == nil {
		t.Fatal("the command ran with the seam unreachable")
	}
}

// Ungoverned, a reading is upstream's.
func TestIQAnUngovernedReadingIsUpstreams(t *testing.T) {
	t.Setenv("IQ_SEAM_URL", "")
	space := t.TempDir()
	if _, ok := RunReading(context.Background(), space, Strategy{Command: "touch ran", Read: FormatPlain}, 10*time.Second); !ok {
		t.Fatal("ungoverned reading not taken")
	}
	if _, err := os.Stat(filepath.Join(space, "ran")); err != nil {
		t.Fatal("ungoverned command did not run")
	}
}
