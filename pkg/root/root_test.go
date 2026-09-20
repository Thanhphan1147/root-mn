package root

import (
	"math/rand"
	"testing"
)

func newTestGame(t *testing.T) *Game {
	t.Helper()
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 12345)
	Setup(g, SetupOptions{MCCorner: "C1", EDCorner: "C3", EDLeader: "despot", Character: "tinker", Seed: 12345})
	return g
}

func TestSetup(t *testing.T) {
	g := newTestGame(t)
	if g.totalBuildings(MC, "sawmill")+g.Players[MC].Sawmills != 6 {
		t.Fatalf("MC sawmills wrong")
	}
	if g.Players[MC].KeepClearing != "C1" {
		t.Fatalf("keep clearing = %s", g.Players[MC].KeepClearing)
	}
	if g.buildingsOf(ED, "C3", "roost") != 1 {
		t.Fatalf("ED roost missing")
	}
	if g.Clearings["C3"].Warriors[ED] != 6 {
		t.Fatalf("ED warriors = %d", g.Clearings["C3"].Warriors[ED])
	}
	if len(g.Players[WA].Supporters) != 3 {
		t.Fatalf("WA supporters = %d", len(g.Players[WA].Supporters))
	}
	if len(g.Players[VB].Items) != 4 {
		t.Fatalf("VB items = %d", len(g.Players[VB].Items))
	}
	if len(g.Players[MC].Hand) != 3 || len(g.Players[ED].Hand) != 3 {
		t.Fatalf("starting hands wrong")
	}
	for _, c := range GetMap("autumn").Ruins {
		if g.Clearings[c].RuinItem == "" {
			t.Fatalf("ruin %s has no item", c)
		}
	}
}

func TestRule(t *testing.T) {
	g := newTestGame(t)
	// MC has 1 warrior in C1 and a keep; ED none there.
	if !g.Rules(MC, "C1") {
		t.Fatalf("MC should rule C1")
	}
	// ED rules C3 with 6 warriors + roost.
	if !g.Rules(ED, "C3") {
		t.Fatalf("ED should rule C3")
	}
	// Tie: add equal warriors.
	g.addWarrior(ED, "C1", 1)
	// MC has 1 warrior + 1 building (keep is a token, not building; but setup placed buildings in C1? maybe)
}

func TestSelfPlay(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for game := 0; game < 20; game++ {
		g := NewGame([]Faction{MC, ED, WA, VB}, MC, uint64(game*7919+1))
		Setup(g, SetupOptions{MCCorner: "C1", EDCorner: "C3", EDLeader: "despot", Character: "tinker", Seed: uint64(game*7919 + 1)})
		steps := 0
		for steps < 4000 {
			if w, ok := g.WinnerFaction(); ok {
				t.Logf("game %d: winner %s in %d steps, round %d", game, w, steps, g.Round)
				break
			}
			acts := g.LegalActions()
			if len(acts) == 0 {
				t.Fatalf("game %d: no legal actions at round %d phase %s current %s pending %v",
					game, g.Round, g.Phase, g.Current, g.Pending)
			}
			a := acts[rng.Intn(len(acts))]
			if err := g.Apply(a); err != nil {
				t.Fatalf("game %d step %d: apply %q: %v", game, steps, a.ID, err)
			}
			steps++
		}
		if steps >= 4000 {
			t.Logf("game %d: reached step cap at round %d (VP %d/%d/%d/%d)",
				game, g.Round, g.Players[MC].VP, g.Players[ED].VP, g.Players[WA].VP, g.Players[VB].VP)
		}
	}
}
