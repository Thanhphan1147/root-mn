package root

import (
	"math/rand"
	"reflect"
	"testing"
)

// TestApplyFastMatchesApply checks that the search fast path produces exactly
// the same state as the validated Apply for every legal action it is given, so a
// search may substitute one for the other safely.
func TestApplyFastMatchesApply(t *testing.T) {
	norm := func(x *Game) *Game {
		c := x.Clone()
		c.Log = nil
		c.RMNLog = nil
		return c
	}
	for seed := 1; seed <= 25; seed++ {
		g := NewGame([]Faction{MC, ED}, MC, uint64(seed))
		BeginSetup(g)
		rng := rand.New(rand.NewSource(int64(seed)))
		for step := 0; step < 400; step++ {
			if len(g.Winner) > 0 {
				break
			}
			acts := g.LegalActions()
			if len(acts) == 0 {
				break
			}
			a := acts[rng.Intn(len(acts))]

			fast := g.CloneForSearch()
			if err := fast.ApplyFast(a); err != nil {
				t.Fatalf("seed %d step %d: ApplyFast(%s) error: %v", seed, step, a.ID, err)
			}
			slow := g.CloneForSearch()
			if err := slow.Apply(a); err != nil {
				t.Fatalf("seed %d step %d: Apply(%s) error: %v", seed, step, a.ID, err)
			}
			if !reflect.DeepEqual(norm(fast), norm(slow)) {
				t.Fatalf("seed %d step %d: ApplyFast and Apply diverged on %s", seed, step, a.ID)
			}
			g = slow
		}
	}
}
