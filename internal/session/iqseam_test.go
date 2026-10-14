// @use_case Development/UseCase/UC_MetaHarness
// @facet Agent
package session

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/sdk/go/ai"
)

type fakeSeam struct {
	mu       sync.Mutex
	verdict  map[string]any
	pushes   []map[string]any
	observes []map[string]any
}

func (f *fakeSeam) server(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		defer f.mu.Unlock()
		if strings.HasPrefix(r.URL.Path, "/observe") {
			f.observes = append(f.observes, body)
			_ = json.NewEncoder(w).Encode(map[string]any{"recorded": true})
			return
		}
		f.pushes = append(f.pushes, body)
		v := map[string]any{"operation_id": body["operationId"]}
		for k, x := range f.verdict {
			v[k] = x
		}
		_ = json.NewEncoder(w).Encode(v)
	}))
}

func iqReadCall(id string) ai.ToolCall {
	var c ai.ToolCall
	c.ID = id
	c.Function.Name = "read"
	c.Function.Arguments = `{"path":"DESIGN.md","contents":"never sent"}`
	return c
}

func TestIQSeamInertWithoutURL(t *testing.T) {
	t.Setenv("IQ_SEAM_URL", "")
	s := &iqSeam{agent: &Agent{id: "s1"}}
	if _, _, ok := s.PreAction(t.Context(), nil, nil, iqReadCall("c1")); !ok {
		t.Fatal("an ungoverned build must behave as upstream")
	}
}

func TestIQSeamAdmitsAndObserves(t *testing.T) {
	f := &fakeSeam{verdict: map[string]any{"admitted": true}}
	srv := f.server(t)
	defer srv.Close()
	t.Setenv("IQ_SEAM_URL", srv.URL)
	t.Setenv("IQ_HARNESS_SESSION", "tok")
	s := &iqSeam{agent: &Agent{id: "s1"}}
	call := iqReadCall("c1")
	if _, _, ok := s.PreAction(t.Context(), nil, nil, call); !ok {
		t.Fatal("admitted call was vetoed")
	}
	p := f.pushes[0]
	if p["operationId"] != "iq:op:af-s1-c1" || p["tool"] != "read" || p["session"] != "tok" || p["harness_session"] != "codeaf-s1" {
		t.Fatalf("push body %v", p)
	}
	if args := p["args"].(map[string]any); args["path"] != "DESIGN.md" || args["contents"] != nil {
		t.Fatalf("args must carry only path-ish keys: %v", args)
	}
	s.PostFeedback(t.Context(), nil, nil, []ai.ToolCall{call}, []toolResult{{text: "boom", isError: true}}, false)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		n := len(f.observes)
		f.mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.observes) != 1 || f.observes[0]["operationId"] != "iq:op:af-s1-c1" || f.observes[0]["ok"] != false || f.observes[0]["error"] != "boom" {
		t.Fatalf("observe %v", f.observes)
	}
}

func TestIQSeamRefusalIsAVeto(t *testing.T) {
	f := &fakeSeam{verdict: map[string]any{"admitted": false, "reason": "no_permit"}}
	srv := f.server(t)
	defer srv.Close()
	t.Setenv("IQ_SEAM_URL", srv.URL)
	_, res, ok := (&iqSeam{agent: &Agent{id: "s1"}}).PreAction(t.Context(), nil, nil, iqReadCall("c2"))
	if ok || !res.isError || !strings.Contains(res.text, "no_permit") {
		t.Fatalf("refusal must veto with the reason: ok=%v %q", ok, res.text)
	}
}

func TestIQSeamUnreachableFailsClosed(t *testing.T) {
	t.Setenv("IQ_SEAM_URL", "http://127.0.0.1:1")
	_, res, ok := (&iqSeam{agent: &Agent{id: "s1"}}).PreAction(t.Context(), nil, nil, iqReadCall("c3"))
	if ok || !strings.Contains(res.text, "IQ gate unreachable") {
		t.Fatalf("an unreachable seam must refuse: ok=%v %q", ok, res.text)
	}
}

func TestIQSeamAskWithNobodyAttachedIsRefused(t *testing.T) {
	f := &fakeSeam{verdict: map[string]any{"admitted": true, "ask": "IQ budget: this day has spent $30.00 of its $25.00 ceiling"}}
	srv := f.server(t)
	defer srv.Close()
	t.Setenv("IQ_SEAM_URL", srv.URL)
	_, res, ok := (&iqSeam{agent: &Agent{id: "s1"}}).PreAction(t.Context(), nil, nil, iqReadCall("c4"))
	if ok || !strings.Contains(res.text, "nobody to ask") || !strings.Contains(res.text, "$25.00") {
		t.Fatalf("a budget ask headless must refuse, naming the spend: ok=%v %q", ok, res.text)
	}
}
