// @use_case Development/UseCase/UC_MetaHarness
// @facet Agent
//
// Package iqgate is the IQ fork's client of the common CpCp seam (UC_MetaHarness seam_server.py, IQ_SEAM_URL),
// shared by CodeAF's two tool runtimes: the session control plane (internal/session/iqseam.go) and the leaf
// toolbox a task worker runs its tools through (internal/exec/iqgate.go). One client, so the two can never ask
// the seam different questions.
//
//	Push     POST /push {operationId, session, tool, args (path-ish keys only), harness_session} -> a Verdict.
//	Observe  POST /observe {operationId, session, ok, error, duration_ms, harness_session} -- best effort.
//
// IQ_SEAM_URL unset -> Enabled() is false and every caller behaves as upstream built it. The credential is
// IQ_HARNESS_SESSION, else ~/.iq/harness-token (the box's paired token).
package iqgate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Verdict is the seam's answer to a push.
type Verdict struct {
	OperationID string `json:"operation_id"`
	Admitted    bool   `json:"admitted"`
	Reason      string `json:"reason"`
	Ask         string `json:"ask"`
}

// URL is the seam's base, or "" when this process is not governed.
func URL() string { return strings.TrimRight(strings.TrimSpace(os.Getenv("IQ_SEAM_URL")), "/") }

// Enabled is true when a seam is configured.
func Enabled() bool { return URL() != "" }

// Credential is the paired token the seam authenticates a push by.
func Credential() string {
	if v := strings.TrimSpace(os.Getenv("IQ_HARNESS_SESSION")); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	b, err := os.ReadFile(filepath.Join(home, ".iq", "harness-token"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

var client = &http.Client{Timeout: 5 * time.Second}

// Post sends one JSON body to the seam and decodes its JSON answer.
func Post(base, path string, body any) (map[string]any, error) {
	buf, _ := json.Marshal(body)
	resp, err := client.Post(base+path, "application/json", bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("unreadable seam answer: %w", err)
	}
	return out, nil
}

// SmallArgs keeps only the path-ish keys the seam names an artifact by -- never contents or commands.
func SmallArgs(raw string) map[string]any {
	var args map[string]any
	if json.Unmarshal([]byte(raw), &args) != nil {
		return map[string]any{}
	}
	keep := map[string]any{}
	for _, k := range []string{"path", "file", "file_path", "filePath", "dir", "directory", "pattern", "glob"} {
		if v, ok := args[k]; ok {
			keep[k] = v
		}
	}
	return keep
}

// Push asks the seam whether one tool call may run. An error means the seam could not be asked: callers fail
// closed on it.
func Push(base, operationID, tool, rawArgs, harnessSession string) (Verdict, error) {
	raw, err := Post(base, "/push", map[string]any{
		"operationId": operationID, "session": Credential(), "tool": tool,
		"args": SmallArgs(rawArgs), "harness_session": harnessSession,
	})
	if err != nil {
		return Verdict{}, err
	}
	b, _ := json.Marshal(raw)
	var v Verdict
	_ = json.Unmarshal(b, &v)
	if !v.Admitted && v.Reason == "" {
		v.Reason = "denied"
	}
	return v, nil
}

// Observe reports how an admitted call ended. It never fails the caller.
func Observe(base, operationID string, ok bool, errText string, took time.Duration, harnessSession string) {
	if len(errText) > 500 {
		errText = errText[:500]
	}
	_, _ = Post(base, "/observe", map[string]any{
		"operationId": operationID, "session": Credential(), "ok": ok, "error": errText,
		"duration_ms": took.Milliseconds(), "harness_session": harnessSession,
	})
}
