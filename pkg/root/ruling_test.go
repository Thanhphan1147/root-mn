package root

import "testing"

// TestRuling counts only warriors and buildings, per Law of Root 2.5: "A player
// rules a clearing if they have more total warriors and buildings in it than
// each other player. (Tokens and pawns do not contribute to rule.)" The
// Marquise's Keep is a token, so it must not grant rule; the Eyrie win ties
// (Lords of the Forest).
func TestRuling(t *testing.T) {
	g := NewGame([]Faction{MC, ED}, MC, 1)
	cl := g.Clearings["C1"]

	// MC: 1 warrior + 1 sawmill = 2 presence. ED: 2 warriors = 2 presence.
	cl.Warriors = map[Faction]int{MC: 1, ED: 2}
	cl.Buildings = []Building{{MC, "sawmill"}}
	if g.Rules(MC, "C1") {
		t.Errorf("MC must not rule a tie (more is required)")
	}
	if !g.Rules(ED, "C1") {
		t.Errorf("ED should rule the tie via Lords of the Forest")
	}

	// The keep is a token and must not count toward rule.
	cl.Tokens = []Token{{MC, "keep"}}
	if g.Rules(MC, "C1") {
		t.Errorf("the keep token must not grant rule")
	}

	// Adding a real building breaks the tie for MC (1+2 = 3 > 2).
	cl.Buildings = append(cl.Buildings, Building{MC, "workshop"})
	if !g.Rules(MC, "C1") {
		t.Errorf("MC with more warriors+buildings should rule")
	}
	if g.Rules(ED, "C1") {
		t.Errorf("ED must not rule when MC has more presence")
	}
}
