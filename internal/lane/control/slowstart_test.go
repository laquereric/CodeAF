package control

import (
	"testing"
	"time"
)

// A plan with a FirstFloor holds the ceiling and the hedge off a request that has
// not answered yet, and acts the moment the floor is past.
func TestNothingIsActedOnBeforeTheFirstFloor(t *testing.T) {
	p := plan()
	p.First, p.Gap, p.Think = Survival{}, Survival{}, Survival{}
	p.Alts[0].First = Survival{}
	p.FirstFloor = 3 * p.Ceiling
	watch := New(p)
	for step := 0; step <= int(4*p.Ceiling/time.Millisecond); step += 50 {
		act := watch.Quiet(at(step))
		if act.Kind == None {
			continue
		}
		if want := int(p.FirstFloor / time.Millisecond); step < want {
			t.Fatalf("acted at %dms, inside the %dms first floor", step, want)
		}
		return
	}
	t.Fatal("the ceiling never fired once the first floor was past")
}

// Zero is every plan upstream builds, and it changes nothing.
func TestAZeroFirstFloorLeavesTheCeilingWhereItWas(t *testing.T) {
	p := plan()
	p.First, p.Gap, p.Think = Survival{}, Survival{}, Survival{}
	p.Alts[0].First = Survival{}
	watch := New(p)
	for step := 0; step <= 20_000; step += 50 {
		if act := watch.Quiet(at(step)); act.Kind != None {
			if want := int(p.Ceiling / time.Millisecond); step != want {
				t.Fatalf("acted at %dms, want the ceiling at %dms", step, want)
			}
			return
		}
	}
	t.Fatal("never acted")
}
