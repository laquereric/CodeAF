package provider

import (
	"testing"
	"time"
)

func TestFirstTokenFloorUnsetLeavesUpstreamFigures(t *testing.T) {
	t.Setenv("CODEAF_FIRST_TOKEN_TIMEOUT", "")
	if got := atLeastFirstToken(90 * time.Second); got != 90*time.Second {
		t.Fatalf("unset must change nothing: %v", got)
	}
	for _, bad := range []string{"soon", "-5m", "0s"} {
		t.Setenv("CODEAF_FIRST_TOKEN_TIMEOUT", bad)
		if got := atLeastFirstToken(2 * time.Minute); got != 2*time.Minute {
			t.Fatalf("%q must change nothing: %v", bad, got)
		}
	}
}

func TestFirstTokenFloorOnlyRaises(t *testing.T) {
	t.Setenv("CODEAF_FIRST_TOKEN_TIMEOUT", "10m")
	if got := atLeastFirstToken(30 * time.Second); got != 10*time.Minute {
		t.Fatalf("a short bound must rise to the floor: %v", got)
	}
	if got := atLeastFirstToken(20 * time.Minute); got != 20*time.Minute {
		t.Fatalf("a longer bound must stay: %v", got)
	}
}

func TestFirstTokenFloorRaisesTheWatchdogsFirstBound(t *testing.T) {
	t.Setenv("CODEAF_FIRST_TOKEN_TIMEOUT", "")
	var role0 = boundsFor("")
	t.Setenv("CODEAF_FIRST_TOKEN_TIMEOUT", "15m")
	b := boundsFor("")
	if b.first != 15*time.Minute || b.gap != role0.gap {
		t.Fatalf("first must rise to 15m and the gap stay %v: %+v", role0.gap, b)
	}
}
