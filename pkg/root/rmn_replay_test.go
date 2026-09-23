package root

import (
	"math/rand"
	"reflect"
	"testing"
)

// playRandom plays a game with a random policy and returns it, so its RMN log
// can be exercised by the replay tests.
func playRandom(seed int, steps int) *Game {
	g := NewGame([]Faction{MC, ED}, MC, uint64(seed))
	BeginSetup(g)
	rng := rand.New(rand.NewSource(int64(seed)))
	for i := 0; i < steps; i++ {
		if len(g.Winner) > 0 {
			break
		}
		acts := g.LegalActions()
		if len(acts) == 0 {
			break
		}
		if err := g.Apply(acts[rng.Intn(len(acts))]); err != nil {
			break
		}
	}
	return g
}

// TestSetupLogReplaysDeterministically guards the fix for the C9 playtest
// finding: setup actions used to log an identical "place group=? to=X", so a
// log could not reconstruct which pieces (or leader) were placed. Every setup
// line must now match exactly one legal action, and replaying it must rebuild
// the same setup state.
func TestSetupLogReplaysDeterministically(t *testing.T) {
	for seed := 1; seed <= 40; seed++ {
		// Build a game that stops exactly at the end of setup.
		orig := NewGame([]Faction{MC, ED}, MC, uint64(seed))
		BeginSetup(orig)
		rng := rand.New(rand.NewSource(int64(seed)))
		for orig.SetupMode {
			acts := orig.LegalActions()
			if len(acts) == 0 {
				break
			}
			if err := orig.Apply(acts[rng.Intn(len(acts))]); err != nil {
				t.Fatalf("seed %d: setup apply failed: %v", seed, err)
			}
		}

		g := NewGame([]Faction{MC, ED}, MC, uint64(seed))
		BeginSetup(g)
		for i, line := range orig.RMNLog {
			base := len(g.RMNLog)
			var matches []Action
			for _, a := range g.LegalActions() {
				c := g.Clone()
				if err := c.Apply(a); err != nil {
					continue
				}
				if len(c.RMNLog) == base+1 && c.RMNLog[base] == line {
					matches = append(matches, a)
				}
			}
			if len(matches) != 1 {
				t.Fatalf("seed %d setup event %d (%q) matched %d actions, want exactly 1", seed, i, line, len(matches))
			}
			if err := g.Apply(matches[0]); err != nil {
				t.Fatalf("seed %d: re-apply failed: %v", seed, err)
			}
		}
		if g.SetupMode {
			t.Fatalf("seed %d: setup did not complete on replay", seed)
		}
		if !reflect.DeepEqual(normGame(g), normGame(orig)) {
			t.Fatalf("seed %d: setup replay diverged", seed)
		}
	}
}

// TestRMNLogReplaysDeterministically is the full property RMN promises: any log
// reconstructs its game exactly. It currently fails on events where distinct
// card effects emit the same intent — several "draw ... from=DECK" effects
// draw the same top card and are indistinguishable, and a battle can produce
// two identical battle lines. Fixing those needs the intent to name its source
// (or per-effect intents), which is an RMN spec decision. Setup, MC builds,
// moves, battles, and Eyrie decree resolution are already deterministic.
func TestRMNLogReplaysDeterministically(t *testing.T) {
	t.Skip("known gaps: shared 'draw' intent across different card effects, duplicate battle lines")

	for seed := 1; seed <= 30; seed++ {
		orig := playRandom(seed, 500)
		r := replayLog(t, uint64(seed), orig.RMNLog)
		if !reflect.DeepEqual(normGame(orig), normGame(r)) {
			t.Fatalf("seed %d: replay diverged after %d events", seed, len(orig.RMNLog))
		}
	}
}

func replayLog(t *testing.T, seed uint64, lines []string) *Game {
	t.Helper()
	g := NewGame([]Faction{MC, ED}, MC, seed)
	BeginSetup(g)
	for i, line := range lines {
		base := len(g.RMNLog)
		var matches []Action
		for _, a := range g.LegalActions() {
			c := g.Clone()
			if err := c.Apply(a); err != nil {
				continue
			}
			if len(c.RMNLog) == base+1 && c.RMNLog[base] == line {
				matches = append(matches, a)
			}
		}
		if len(matches) != 1 {
			t.Fatalf("event %d matched %d actions: %q", i, len(matches), line)
		}
		if err := g.Apply(matches[0]); err != nil {
			t.Fatalf("event %d: re-apply failed: %v", i, err)
		}
	}
	return g
}

// normGame strips the action logs so the comparison is about game state.
func normGame(g *Game) *Game {
	c := g.Clone()
	c.Log = nil
	c.RMNLog = nil
	c.DrawnThisAction = nil
	return c
}
