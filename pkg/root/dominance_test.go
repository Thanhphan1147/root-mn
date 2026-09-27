package root

import "testing"

// TestSlipIntoForestKeepsTurn: Law 9.4.2 / 9.2.10 — slipping into a forest is
// allowed and does not end the turn.
func TestSlipIntoForestKeepsTurn(t *testing.T) {
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
	if g.Current != VB {
		t.Fatal("slipping into a forest must not end the turn")
	}
	if !g.VBSlipped {
		t.Fatal("the slip should be marked used")
	}
}

// TestDominanceActivation: Law 3.3.1/3.3.2 — activating moves the card to the
// play area (tracked in DominanceActive), removes the score marker, and cannot
// be replaced by another dominance.
func TestDominanceActivation(t *testing.T) {
	g := NewGame([]Faction{MC, ED}, MC, 5)
	g.Players[MC].Hand = []string{"F02", "M02"} // Fox + Mouse dominance
	g.Players[MC].VP = 12

	acts := g.dominanceActions(MC)
	if len(acts) != 2 {
		t.Fatalf("expected 2 activation options, got %d", len(acts))
	}
	if err := g.applyResolved(Action{Kind: "dominance", Faction: MC, Card: "F02"}); err != nil {
		t.Fatal(err)
	}
	if g.DominanceActive["F02"] != MC {
		t.Fatalf("activator = %q, want MC", g.DominanceActive["F02"])
	}
	if g.Players[MC].VP != 0 {
		t.Fatalf("score marker should be removed, VP = %d", g.Players[MC].VP)
	}
	if !g.hasDominance(MC) {
		t.Fatal("MC should no longer score")
	}
	if contains(g.AvailableDominance, "F02") {
		t.Fatal("an activated dominance is in the play area, not the available pile")
	}
	// 3.3.2: cannot be replaced with a different dominance.
	if acts := g.dominanceActions(MC); len(acts) != 0 {
		t.Fatalf("an active dominance must not be replaceable, got %d options", len(acts))
	}
	// It also cannot be taken (only available cards can).
	for _, a := range g.takeDominanceActions(ED) {
		if a.Item == "F02" {
			t.Fatal("an activated dominance must not be takeable")
		}
	}
}

// TestTakeAvailableDominance: Law 3.3.3/3.3.4 — a spent dominance goes near the
// map as available and can be taken by spending a matching card; a bird
// dominance can only be taken with a bird card.
func TestTakeAvailableDominance(t *testing.T) {
	g := NewGame([]Faction{MC, ED}, MC, 5)
	g.routeDiscard("F02") // Fox dominance becomes available
	g.routeDiscard("B03") // Bird dominance becomes available
	if !contains(g.AvailableDominance, "F02") || !contains(g.AvailableDominance, "B03") {
		t.Fatalf("dominance should be available: %v", g.AvailableDominance)
	}

	g.Players[MC].Hand = []string{"F01", "M01"}
	// A Fox card can take the Fox dominance, but not the Bird dominance.
	items := map[string]bool{}
	for _, a := range g.takeDominanceActions(MC) {
		items[a.Item] = true
	}
	if !items["F02"] {
		t.Fatal("a Fox card should be able to take Fox dominance")
	}
	if items["B03"] {
		t.Fatal("a Fox card must not be able to take Bird dominance")
	}

	if err := g.applyResolved(Action{Kind: "take-dominance", Faction: MC, Card: "F01", Item: "F02"}); err != nil {
		t.Fatal(err)
	}
	if contains(g.AvailableDominance, "F02") {
		t.Fatal("the taken dominance should have left the available pile")
	}
	if !contains(g.Players[MC].Hand, "F02") {
		t.Fatalf("dominance should be in hand: %v", g.Players[MC].Hand)
	}
}

// TestCoalitionWinIncludesVagabond: Law 9.2.8 — if the coalitioned player wins,
// the Vagabond wins too.
func TestCoalitionWinIncludesVagabond(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 9)
	g.Players[VB].Coalition = MC
	g.Players[MC].VP = 30
	g.checkWin()
	if len(g.Winner) != 2 || g.Winner[0] != MC || g.Winner[1] != VB {
		t.Fatalf("winner = %v, want [MC VB]", g.Winner)
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
