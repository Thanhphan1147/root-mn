package root

import "testing"

// TestVagabondRefreshChoice checks the Birdsong refresh is an interactive choice
// (9.4.1): 3 + 2 per tea, the player picks a specific item each time, and a
// refreshed coin/tea/bag returns to its track.
func TestVagabondRefreshChoice(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	p := g.Players[VB]
	p.Pawn = "AutumnN"
	p.Items = map[string]*ItemState{
		"torch#1": {Type: "torch", Zone: "satchel", FaceUp: false},
		"coin#1":  {Type: "coin", Zone: "satchel", FaceUp: false},
		"boot#1":  {Type: "boot", Zone: "satchel", FaceUp: false},
		"boot#2":  {Type: "boot", Zone: "satchel", FaceUp: false},
		"tea#1":   {Type: "tea", Zone: "track", FaceUp: true},
	}
	g.SetupMode = false
	g.Current = VB
	g.Phase = "B"
	g.VBRefreshLeft = 3 + 2*trackCount(p, "tea")

	acts := g.legalVBRefresh()
	if len(acts) != 4 {
		t.Fatalf("refresh offered %d options, want 4", len(acts))
	}
	for _, a := range acts {
		if a.Item == "coin#1" {
			if err := g.Apply(a); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !p.Items["coin#1"].FaceUp || p.Items["coin#1"].Zone != "track" {
		t.Fatalf("refreshed coin should be face up on its track: %+v", p.Items["coin#1"])
	}
	if g.VBRefreshLeft != 4 {
		t.Fatalf("refresh allowance = %d, want 4", g.VBRefreshLeft)
	}
}

// TestVagabondRefreshIsNotAutomatic checks the refresh is not done for the
// player: after the Vagabond's turn begins, the exhausted item is still down and
// a refresh action is offered.
func TestVagabondRefreshIsNotAutomatic(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	g.SetupMode = false
	p := g.Players[VB]
	p.Pawn = "AutumnN"
	p.Items = map[string]*ItemState{"torch#1": {Type: "torch", Zone: "satchel", FaceUp: false}}
	g.Current = WA
	g.Phase = "E"
	g.endTurn() // advances to the Vagabond and begins its Birdsong

	if g.Current != VB {
		t.Fatalf("expected to advance to VB, got %s", g.Current)
	}
	if g.VBRefreshLeft != 3 {
		t.Fatalf("refresh allowance = %d, want 3", g.VBRefreshLeft)
	}
	if p.Items["torch#1"].FaceUp {
		t.Fatalf("refresh must be a choice, not automatic")
	}
	if len(g.legalVBRefresh()) == 0 {
		t.Fatalf("no refresh action offered")
	}
}
