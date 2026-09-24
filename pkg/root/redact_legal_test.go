package root

import "testing"

// TestRedactKeepsActorLegalActions checks the acting player still receives their
// full legal actions even though the snapshot hides hidden info (a redacted ruin
// item must not remove the Explore option).
func TestRedactKeepsActorLegalActions(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	g.SetupMode = false
	g.Current = VB
	g.Phase = "D"
	p := g.Players[VB]
	p.Pawn = "C12"
	p.Character = "tinker"
	p.Items = map[string]*ItemState{"torch#1": {Type: "torch", Zone: "satchel", FaceUp: true}}
	g.Clearings["C12"].Ruin = true
	g.Clearings["C12"].RuinItem = "boot"

	snap := Redact(g, "VB")
	found := false
	for _, a := range snap["legal"].([]Action) {
		if a.Kind == "vb-explore" {
			found = true
		}
	}
	if !found {
		t.Fatalf("redacted snapshot lost the Explore action")
	}
	if c := snap["clearings"].(map[string]*Clearing)["C12"]; c.RuinItem != "" {
		t.Fatalf("ruin item leaked: %q", c.RuinItem)
	}
}
