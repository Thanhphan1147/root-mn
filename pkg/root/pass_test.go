package root

import "testing"

// TestWABirdsongSinglePass guards the fix for a duplicate "pass": the WA
// Birdsong added its own pass while LegalActions also added one via
// passAllowed, so two identical actions appeared and a log could not tell them
// apart.
func TestWABirdsongSinglePass(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	g.SetupMode = false
	g.Pending = nil
	g.Winner = nil
	g.Current = WA
	g.Phase = "B"
	g.Round = 1

	n := 0
	for _, a := range g.LegalActions() {
		if a.ID == "pass" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("WA Birdsong offered %d pass actions, want 1", n)
	}
}
