// @use_case Development/UseCase/UC_MetaHarness
// @facet Agent
//
// iqverify.go -- the IQ gate on the project's own checks. RunReading is the one place CodeAF runs a command the
// WORKSPACE chose (verify/discovery.go reads it from pytest.ini, conftest.py, package.json scripts, a Makefile...):
// the leaf's photographs before and after a change, the delivery gate's photographs, and the judge's reading. A
// worker that may write files but not run sh could otherwise have its own code run through a reading the seam
// never saw. So every reading is pushed to the seam as its own capability, codeaf:verify -- granted apart from
// a worker's shell -- before it runs, and observed after. Refused or unreachable -> the reading is not taken
// (RunReading's ok=false, the path a machine with no shell already takes) and the refusal is its reason.
// IQ_SEAM_URL unset -> upstream's reading, untouched.
package verify

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/Agent-Field/codeaf/internal/iqgate"
)

// iqVerifyTool is the step's name; iqVerifyAction the capability a reading is asked as, codeaf:verify -- CodeAF's
// own step, not an MCP tool. A seam that does not read action decides mcp:call:verify.
const (
	iqVerifyTool   = "verify"
	iqVerifyAction = "codeaf:verify"
)

var iqVerify struct {
	once    sync.Once
	session string
	mu      sync.Mutex
	seq     int
}

func iqVerifySession() string {
	iqVerify.once.Do(func() {
		b := make([]byte, 6)
		_, _ = rand.Read(b)
		iqVerify.session = "codeaf-verify-" + hex.EncodeToString(b)
	})
	return iqVerify.session
}

func iqVerifyOp() string {
	iqVerify.mu.Lock()
	defer iqVerify.mu.Unlock()
	iqVerify.seq++
	return fmt.Sprintf("iq:op:af-verify-%s-%d", iqVerifySession()[len("codeaf-verify-"):], iqVerify.seq)
}

// iqAdmitReading asks the seam whether strategy may run. refusal is non-empty when it may not (refused,
// unreachable, or a budget ask nobody can answer); otherwise done reports how it ended. Ungoverned -> "", a no-op.
func iqAdmitReading(workspace string, strategy Strategy) (done func(ok bool, errText string), refusal string) {
	base := iqgate.URL()
	if base == "" {
		return func(bool, string) {}, ""
	}
	dir := filepath.ToSlash(filepath.Clean(strategy.Workdir))
	if strategy.Workdir == "" {
		dir = "."
	}
	// The resource is the directory the command runs in; like sh, the command line itself is not sent.
	args, _ := json.Marshal(map[string]any{"path": dir})
	op, session := iqVerifyOp(), iqVerifySession()
	verdict, err := iqgate.PushAction(base, op, iqVerifyTool, iqVerifyAction, string(args), session)
	switch {
	case err != nil:
		return nil, fmt.Sprintf("IQ gate unreachable (%s): %v -- `%s` was not run", base, err, strategy.Command)
	case !verdict.Admitted:
		decided := verdict.Capability // what the seam actually decided: codeaf:verify, or mcp:call:verify before it reads action
		if decided == "" {
			decided = iqVerifyAction
		}
		return nil, fmt.Sprintf("IQ refusal: %s (%s) -- `%s` was not run", decided, verdict.Reason, strategy.Command)
	case verdict.Ask != "":
		return nil, fmt.Sprintf("%s -- nobody to ask; `%s` was not run", verdict.Ask, strategy.Command)
	}
	started := time.Now()
	return func(ok bool, errText string) {
		iqgate.Observe(base, op, ok, errText, time.Since(started), session)
	}, ""
}
