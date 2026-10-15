// @use_case Development/UseCase/UC_MetaHarness
// @facet Agent
//
// iqseam.go -- the IQ gate inside CodeAF (dropped into internal/session by UC_MetaHarness
// adapters/codeaf/build-codeaf.sh, registered FIRST on the session's control plane, ahead of the approval gate).
//
// CodeAF has no external hook: its approval policy is an in-process table and several worker paths build an
// allow-all policy. Every tool call of a session agent -- the chat, `codeaf do`, a task node's worker -- does pass
// the control plane's pre-action chain, so this citizen is where the common CpCp seam (UC_MetaHarness
// seam_server.py, IQ_SEAM_URL) is asked, exactly as the Claude Code hook and the OpenCode plugin ask it:
//
//	PreAction     POST /push {operationId, tool, args (path-ish keys only), session, harness_session}.
//	              refused -> the call does not run and the model is told why; the seam unreachable -> refused
//	              (fail closed); a budget `ask` -> CodeAF's own consent question when a person is attached,
//	              refused when nobody is (a headless run cannot answer it).
//	PostFeedback  POST /observe per call {operationId, ok, error, duration_ms, harness_session} -- never blocks.
//
// The operationId is iq:op:af-<session>-<tool call id>, so push and outcome join with no shared state beyond
// the start time kept here. IQ_SEAM_URL unset -> the citizen is inert and CodeAF behaves as upstream built it.
// The credential is IQ_HARNESS_SESSION, else ~/.iq/harness-token (the box's paired token).
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Agent-Field/agentfield/sdk/go/ai"
	"github.com/Agent-Field/codeaf/internal/approval"
	"github.com/Agent-Field/codeaf/internal/iqgate"
)

type iqSeam struct {
	agent *Agent
	mu    sync.Mutex
	open  map[string]iqOpen // tool call id -> its operation
	seq   int
}

type iqOpen struct {
	op      string
	started time.Time
}

type iqVerdict struct {
	OperationID string `json:"operation_id"`
	Admitted    bool   `json:"admitted"`
	Reason      string `json:"reason"`
	Ask         string `json:"ask"`
}

func (*iqSeam) Name() string { return "iq-seam" }

// The seam client is internal/iqgate, shared with the leaf toolbox (internal/exec/iqgate.go).
func iqSeamURL() string    { return iqgate.URL() }
func iqCredential() string { return iqgate.Credential() }
func iqPost(base, path string, body any) (map[string]any, error) {
	return iqgate.Post(base, path, body)
}
func iqSmallArgs(raw string) map[string]any { return iqgate.SmallArgs(raw) }

func (s *iqSeam) sessionKey() string {
	if s.agent != nil && strings.TrimSpace(s.agent.id) != "" {
		return "codeaf-" + strings.TrimSpace(s.agent.id)
	}
	return "codeaf"
}

func (s *iqSeam) opFor(call ai.ToolCall) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := strings.TrimSpace(call.ID)
	if id == "" {
		s.seq++
		id = fmt.Sprintf("n%d-%d", s.seq, time.Now().UnixNano())
	}
	op := "iq:op:af-" + strings.TrimPrefix(s.sessionKey(), "codeaf-") + "-" + id
	if s.open == nil {
		s.open = map[string]iqOpen{}
	}
	s.open[call.ID] = iqOpen{op: op, started: time.Now()}
	return op
}

func (s *iqSeam) PreAction(ctx context.Context, ep *episode, hub *eventHub, call ai.ToolCall) (ai.ToolCall, toolResult, bool) {
	base := iqSeamURL()
	if base == "" {
		return call, toolResult{}, true // not governed: upstream behaviour
	}
	op := s.opFor(call)
	raw, err := iqPost(base, "/push", map[string]any{
		"operationId": op, "session": iqCredential(), "tool": call.Function.Name,
		"args": iqSmallArgs(call.Function.Arguments), "harness_session": s.sessionKey(),
	})
	if err != nil {
		return call, refusal("IQ gate unreachable (" + base + "): " + err.Error()), false // fail closed
	}
	b, _ := json.Marshal(raw)
	var v iqVerdict
	_ = json.Unmarshal(b, &v)
	if !v.Admitted {
		reason := v.Reason
		if reason == "" {
			reason = "denied"
		}
		return call, refusal("IQ refusal: mcp:call:" + call.Function.Name + " (" + reason + ")"), false
	}
	if v.Ask != "" {
		if s.agent == nil || !s.agent.config.AskConsent || hub == nil {
			return call, refusal(v.Ask + " -- nobody to ask; raise it in ~/.iq/harness-budget.json"), false
		}
		ok, askErr := s.agent.ask(ctx, hub, call, approval.Decision{Action: approval.ActionPrompt, Rule: v.Ask})
		if askErr != nil || !ok {
			return call, refusal("not approved: " + v.Ask), false
		}
	}
	return call, toolResult{}, true
}

func (s *iqSeam) PostFeedback(ctx context.Context, ep *episode, hub *eventHub, calls []ai.ToolCall, results []toolResult, visibleText bool) {
	base := iqSeamURL()
	if base == "" {
		return
	}
	type row struct {
		body map[string]any
	}
	var rows []row
	s.mu.Lock()
	for i, call := range calls {
		o, ok := s.open[call.ID]
		if !ok || i >= len(results) {
			continue
		}
		delete(s.open, call.ID)
		r := results[i]
		if r.refusedBy != "" {
			continue // it never ran: the push row already says why
		}
		errText := ""
		if r.isError {
			errText = r.text
			if len(errText) > 500 {
				errText = errText[:500]
			}
		}
		rows = append(rows, row{map[string]any{
			"operationId": o.op, "session": iqCredential(), "ok": !r.isError, "error": errText,
			"duration_ms": time.Since(o.started).Milliseconds(), "harness_session": s.sessionKey(),
		}})
	}
	s.mu.Unlock()
	if len(rows) == 0 {
		return
	}
	go func() { // the effect has happened: reporting it never holds the turn
		for _, r := range rows {
			_, _ = iqPost(base, "/observe", r.body)
		}
	}()
}
