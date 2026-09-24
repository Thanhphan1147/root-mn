package root

import "testing"

// TestVagabondRefresh checks the Birdsong refresh: flip 3 exhausted items face
// up, plus 2 for each tea face up on the Refresh Track.
func TestVagabondRefresh(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	p := g.Players[VB]
	p.Items = map[string]*ItemState{
		"torch#1": {Type: "torch", Zone: "satchel", FaceUp: false},
		"boot#1":  {Type: "boot", Zone: "satchel", FaceUp: false},
		"boot#2":  {Type: "boot", Zone: "satchel", FaceUp: false},
		"boot#3":  {Type: "boot", Zone: "satchel", FaceUp: false},
		"boot#4":  {Type: "boot", Zone: "satchel", FaceUp: false},
		"sword#1": {Type: "sword", Zone: "satchel", FaceUp: true},
	}
	g.vbRefresh(p)
	up := 0
	for _, it := range p.Items {
		if it.FaceUp {
			up++
		}
	}
	// 3 refreshed + the already-up sword = 4.
	if up != 4 {
		t.Fatalf("refresh flipped to %d face-up, want 4", up)
	}

	// With a tea on the track, 3+2 = 5 exhausted items refresh.
	p.Items["tea#1"] = &ItemState{Type: "tea", Zone: "track", FaceUp: true}
	for _, id := range []string{"torch#1", "boot#1", "boot#2", "boot#3", "boot#4"} {
		p.Items[id].FaceUp = false
	}
	g.vbRefresh(p)
	up = 0
	for _, it := range p.Items {
		if it.FaceUp {
			up++
		}
	}
	// tea + sword already up (2), plus 5 refreshed = 7.
	if up != 7 {
		t.Fatalf("with one tea, refresh flipped to %d face-up, want 7", up)
	}
}

// TestVagabondRefreshOnTurnStart checks the refresh actually runs when the
// Vagabond's turn begins.
func TestVagabondRefreshOnTurnStart(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	g.SetupMode = false
	p := g.Players[VB]
	p.Items = map[string]*ItemState{"torch#1": {Type: "torch", Zone: "satchel", FaceUp: false}}
	g.Current = WA
	g.Phase = "E"
	g.endTurn() // advances to VB and begins its turn
	if g.Current != VB {
		t.Fatalf("expected to advance to VB, got %s", g.Current)
	}
	if !p.Items["torch#1"].FaceUp {
		t.Fatalf("VB Birdsong did not refresh the exhausted torch")
	}
}
