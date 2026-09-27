package root

import "testing"

// TestVBCraftTakesItem: Law 9.2.1/9.5 — when the Vagabond crafts an item it is
// taken face up into its Satchel (or matching track) and is immediately usable,
// not placed in the Crafted Items box.
func TestVBCraftTakesItem(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 3)
	Setup(g, SetupOptions{Character: "tinker"})
	p := g.Players[VB]

	if _, ok := Card("B13"); !ok {
		t.Fatal("test card B13 missing")
	}
	if c, _ := Card("B13"); c.Item != "sword" {
		t.Fatalf("B13 should craft a sword, got %q", c.Item)
	}
	p.Hand = append(p.Hand, "B13")
	g.giveVBItem(p, "hammer") // a crafting piece
	p.Pawn = "C5"             // craft in a clearing (suit comes from the clearing)

	if err := g.applyResolved(Action{Kind: "craft", Faction: VB, Card: "B13"}); err != nil {
		t.Fatal(err)
	}

	usable := false
	for _, it := range p.Items {
		if it.Type == "sword" && it.FaceUp && !it.Damaged {
			usable = true
		}
	}
	if !usable {
		t.Fatalf("crafted sword should be a usable item: %+v", p.Items)
	}
	if contains(p.CraftedItems, "sword") {
		t.Fatalf("a Vagabond item must not go to the Crafted Items box: %v", p.CraftedItems)
	}
	if _, ok := g.readyItem(p, "sword"); !ok {
		t.Fatal("the crafted sword should be ready for a Battle")
	}
}
