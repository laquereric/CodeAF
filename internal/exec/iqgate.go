// @use_case Development/UseCase/UC_MetaHarness
// @facet Agent
//
// iqgate.go -- the IQ gate on the leaf toolbox. A task worker on the node belt (`codeaf do` with
// CODEAF_TASK_BELT=node) runs its tools -- sh, job, write, edit, web, read_document, ... -- through Toolbox, not
// through the session control plane where internal/session/iqseam.go sits, so without this a governed run's
// worker ran a shell the seam had refused everywhere else. Every call is pushed to the seam before it runs
// (refused, unreachable, or a budget ask with nobody to answer it -> the call does not run and the model reads
// why) and observed after. IQ_SEAM_URL unset -> Execute is upstream's dispatch, untouched.
package exec

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Agent-Field/codeaf/internal/iqgate"
)

type iqLeaf struct {
	once    sync.Once
	session string
	mu      sync.Mutex
	seq     int
}

// iqSession is this toolbox's harness_session: codeaf-<leaf>-<random>, one per toolbox, so a run's rows join.
func (t *Toolbox) iqSession() string {
	t.iq.once.Do(func() {
		b := make([]byte, 6)
		_, _ = rand.Read(b)
		leaf := strings.TrimSpace(t.leaf)
		if leaf == "" {
			leaf = "leaf"
		}
		t.iq.session = "codeaf-" + leaf + "-" + hex.EncodeToString(b)
	})
	return t.iq.session
}

func (t *Toolbox) iqOp() string {
	t.iq.mu.Lock()
	defer t.iq.mu.Unlock()
	t.iq.seq++
	return fmt.Sprintf("iq:op:af-%s-%d", strings.TrimPrefix(t.iqSession(), "codeaf-"), t.iq.seq)
}

// Execute dispatches one call, governed by the IQ seam when one is configured.
func (t *Toolbox) Execute(ctx context.Context, name string, arguments string) Result {
	base := iqgate.URL()
	if base == "" {
		return t.execute(ctx, name, arguments) // not governed: upstream behaviour
	}
	op, session := t.iqOp(), t.iqSession()
	verdict, err := iqgate.Push(base, op, name, arguments, session)
	switch {
	case err != nil:
		return t.finishResult(errorf("IQ gate unreachable (%s): %v", base, err)) // fail closed
	case !verdict.Admitted:
		return t.finishResult(errorf("IQ refusal: mcp:call:%s (%s)", name, verdict.Reason))
	case verdict.Ask != "":
		// A leaf has nobody attached to answer a budget question.
		return t.finishResult(errorf("%s -- nobody to ask; raise it in ~/.iq/harness-budget.json", verdict.Ask))
	}
	started := time.Now()
	result := t.execute(ctx, name, arguments)
	errText := ""
	if result.IsError {
		errText = result.Content
	}
	go iqgate.Observe(base, op, !result.IsError, errText, time.Since(started), session) // never holds the turn
	return result
}
