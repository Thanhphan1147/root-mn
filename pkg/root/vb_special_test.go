package root

import (
	"strings"
	"testing"
)

func inDaylightVB(char string) *Game {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	g.SetupMode = false
	g.Current = VB
	g.Phase = "D"
	p := g.Players[VB]
	p.Character = char
	p.Pawn = "C1"
	p.Items = map[string]*ItemState{"torch#1": {Type: "torch", Zone: "satchel", FaceUp: true}}
	return g
}

func lastLine(g *Game) string { return g.RMNLog[len(g.RMNLog)-1] }

func pickAction(g *Game, pred func(Action) bool) (Action, bool) {
	for _, a := range g.LegalActions() {
		if pred(a) {
			return a, true
		}
	}
	return Action{}, false
}

// TestVagabondSpecialNotation checks each character's special action is logged
// with its own intent, not a generic notify.
func TestVagabondSpecialNotation(t *testing.T) {
	// Thief: Steal from a player (card is recorded and redactable).
	g := inDaylightVB("thief")
	g.Players[MC].Hand = []string{"R05"}
	g.Clearings["C1"].Warriors[MC] = 1 // a player to steal from must be present
	a, ok := pickAction(g, func(a Action) bool { return a.Kind == "vb-special" && a.Target == MC })
	if !ok {
		t.Fatal("no steal offered")
	}
	if err := g.Apply(a); err != nil {
		t.Fatal(err)
	}
	if l := lastLine(g); !strings.Contains(l, "V:steal target=MC") || !strings.Contains(l, "revealed=[R05]") {
		t.Fatalf("steal line = %q", l)
	}

	// Tinker: Day Labor takes a discard card.
	g2 := inDaylightVB("tinker")
	g2.Discard = []string{"B02"}
	a2, ok := pickAction(g2, func(a Action) bool { return a.Kind == "vb-special" && a.Card == "B02" })
	if !ok {
		t.Fatal("no day-labor offered")
	}
	if err := g2.Apply(a2); err != nil {
		t.Fatal(err)
	}
	if l := lastLine(g2); !strings.Contains(l, "V:day-labor card=B02") {
		t.Fatalf("day-labor line = %q", l)
	}

	// Ranger: Hideout.
	g3 := inDaylightVB("ranger")
	g3.Players[VB].Items["sword#1"] = &ItemState{Type: "sword", Zone: "damaged", Damaged: true}
	a3, ok := pickAction(g3, func(a Action) bool { return a.ID == "vb-special|hideout" })
	if !ok {
		t.Fatal("no hideout offered")
	}
	if err := g3.Apply(a3); err != nil {
		t.Fatal(err)
	}
	if l := lastLine(g3); !strings.Contains(l, "V:hideout") {
		t.Fatalf("hideout line = %q", l)
	}
}
