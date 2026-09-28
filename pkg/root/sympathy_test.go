package root

import "testing"

// TestBattleSympathyOnlyClearing: a sympathy token is a faction piece, so a
// clearing with only sympathy can be attacked, and the token can be removed as
// a battle hit (8.2.5, 8.2.6, 4.3).
func TestBattleSympathyOnlyClearing(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 3)
	Setup(g, SetupOptions{})

	g.addWarrior(MC, "C1", 1)
	g.placeSympathy("C1")

	if !g.hasPieceIn(WA, "C1") {
		t.Fatal("a sympathy token should count as a piece in a clearing")
	}
	found := false
	for _, a := range g.battleActions(MC) {
		if a.Clearing == "C1" && a.Target == WA {
			found = true
		}
	}
	if !found {
		t.Fatal("a battle against a sympathy-only clearing should be offered")
	}

	// As the defender, WA must be able to remove the sympathy token for a hit.
	g.Battle = &BattleState{Clearing: "C1", Attacker: MC, Defender: WA, HitSide: WA, Remaining: 1, Stage: StageHits}
	tokenHit := false
	for _, a := range g.legalBattleHits() {
		if a.Piece == "token" && a.Item == "sympathy" {
			tokenHit = true
		}
	}
	if !tokenHit {
		t.Fatal("the sympathy token should be assignable as a battle hit")
	}

	// Removing it clears the clearing's sympathy.
	if kind, ok := g.removePiece(MC, WA, "C1"); !ok || kind != "token" {
		t.Fatalf("removePiece = %q, %v; want token", kind, ok)
	}
	if g.Clearings["C1"].Sympathy == WA {
		t.Fatal("the clearing should no longer be sympathetic")
	}
}
