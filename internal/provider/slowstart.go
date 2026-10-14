package provider

import (
	"os"
	"strings"
	"time"

	"github.com/Agent-Field/codeaf/internal/lane/control"
)

// A SLOW START IS NOT A DEAD ENDPOINT, on a lane the person has said is slow.
//
// Every bound in this package that judges a request before its first byte -- the
// watchdog's first-token silence (streamguard.go), the streaming transport's
// response-header deadline and the body's idle watchdog (transport.go) -- was
// argued from hosted endpoints, where ninety seconds is past every healthy first
// token. A local engine prefilling this harness's own long prompt on a laptop is
// not that: at seventy tokens a second, fifteen thousand tokens of prompt is three
// and a half minutes of honest silence before the first token, and each cut pays
// the whole prefill again on the retry.
//
// CODEAF_FIRST_TOKEN_TIMEOUT (a Go duration: "10m", "300s") is a FLOOR under those
// three, set by whoever knows the lane is slow -- IQ's launcher sets it when it
// points codeaf at its local model switch. It only ever raises a bound: unset,
// empty, unparseable or not positive, every figure is upstream's, unchanged. It
// also holds the lane controller's hedge and ceiling off a request that has not
// yet answered ([slowStartPlan]). It does not touch the mid-stream gap or the wall: a model that has started writing
// has finished prefilling, and those bounds were never about the start.
func firstTokenFloor() time.Duration {
	raw := strings.TrimSpace(os.Getenv("CODEAF_FIRST_TOKEN_TIMEOUT"))
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0
	}
	return d
}

// atLeastFirstToken is d, raised to the slow-start floor when one is set.
func atLeastFirstToken(d time.Duration) time.Duration {
	if f := firstTokenFloor(); f > d {
		return f
	}
	return d
}

// slowStartPlan is the plan with the slow-start floor in it: the lane
// controller acts on no silence before the first byte younger than the floor
// (control.Plan.FirstFloor). Unset, the plan is upstream's.
func slowStartPlan(plan control.Plan) control.Plan {
	plan.FirstFloor = firstTokenFloor()
	return plan
}
