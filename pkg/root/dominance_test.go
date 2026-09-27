package root

import "testing"

// TestSlipIntoForestEndsTurn checks that a Vagabond Slip into a forest ends the
// turn immediately (no Daylight actions).
func TestSlipIntoForestEndsTurn(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 3)
	Setup(g, SetupOptions{})

	g.Current = VB
	g.Phase = "B"
	g.Players[VB].Pawn = "C6"
	g.VBSlipped = false

	var slip Action
	found := false
	for _, a := range g.legalVBSlip() {
		if g.isForest(a.To) {
			slip, found = a, true
			break
		}
	}
	if !found {
		t.Fatal("expected a Slip-into-forest destination from C6")
	}
	if err := g.applyResolved(slip); err != nil {
		t.Fatal(err)
	}
	if g.Current == VB {
		t.Fatal("the Vagabond's turn should have ended after slipping into a forest")
	}
}

// TestDominanceZone checks that a played dominance card lands in the public zone
// (with its activator), and that anyone can swap it back for a matching card.
func TestDominanceZone(t *testing.T) {
	g := NewGame([]Faction{MC, ED}, MC, 5)
	g.Players[MC].Hand = []string{"F02", "F01"} // Fox dominance + a Fox card

	if err := g.applyResolved(Action{Kind: "dominance", Faction: MC, Card: "F02"}); err != nil {
		t.Fatal(err)
	}
	if !contains(g.AvailableDominance, "F02") {
		t.Fatalf("played dominance not in the zone: %v", g.AvailableDominance)
	}
	if g.DominanceActive["F02"] != MC {
		t.Fatalf("dominance activator = %q, want MC", g.DominanceActive["F02"])
	}
	if !g.hasDominance(MC) {
		t.Fatal("MC should no longer score after activating a dominance")
	}
	if contains(g.Players[MC].Crafted, "F02") {
		t.Fatal("a played dominance should not be in the crafted area")
	}

	// Swap it back with a matching Fox card.
	if err := g.applyResolved(Action{Kind: "take-dominance", Faction: MC, Card: "F01", Item: "F02"}); err != nil {
		t.Fatal(err)
	}
	if contains(g.AvailableDominance, "F02") {
		t.Fatal("the dominance should have left the zone")
	}
	if _, ok := g.DominanceActive["F02"]; ok {
		t.Fatal("taking a dominance should clear its activation")
	}
	if !contains(g.Players[MC].Hand, "F02") {
		t.Fatalf("dominance should be back in hand: %v", g.Players[MC].Hand)
	}
	if g.hasDominance(MC) {
		t.Fatal("MC should score again once the dominance is taken back")
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
