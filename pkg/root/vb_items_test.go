package root

import (
	"strings"
	"testing"
)

func satchelOrDamaged(p *Player) int {
	n := 0
	for _, it := range p.Items {
		if it.Zone == "satchel" || it.Zone == "damaged" {
			n++
		}
	}
	return n
}

func aidExhausting(g *Game, id string) Action {
	for _, a := range g.LegalActions() {
		if a.Kind == "vb-aid" && a.Exhaust == id && a.Item == "" {
			return a
		}
	}
	return Action{}
}

// TestExhaustedTrackItemGoesToSatchel checks that exhausting a tea/coin/bag on a
// track (which happens when the Vagabond Aids, 9.5.4) moves it to the Satchel,
// face down.
func TestExhaustedTrackItemGoesToSatchel(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	g.SetupMode = false
	g.Current = VB
	g.Phase = "D"
	p := g.Players[VB]
	p.Pawn = "C2"
	p.Items = map[string]*ItemState{
		"coin#1":  {Type: "coin", Zone: "track", FaceUp: true},
		"torch#1": {Type: "torch", Zone: "satchel", FaceUp: true},
	}
	p.Hand = []string{"B01"}
	g.Clearings["C2"].Warriors[MC] = 1

	a := aidExhausting(g, "coin#1")
	if a.ID == "" {
		t.Fatal("no Aid offered that exhausts the track coin")
	}
	if err := g.Apply(a); err != nil {
		t.Fatal(err)
	}
	it := p.Items["coin#1"]
	if it.Zone != "satchel" || it.FaceUp {
		t.Fatalf("exhausted track coin: zone=%q faceUp=%v, want satchel/face-down", it.Zone, it.FaceUp)
	}
	line := g.RMNLog[len(g.RMNLog)-1]
	if !strings.Contains(line, "exhaust=coin#1") {
		t.Fatalf("log should record the exhausted item: %q", line)
	}
}

// TestExhaustedTrackItemCountsTowardLimit checks the item moved into the Satchel
// counts against the item limit (6 + 2 per face-up bag) at the Evening check.
func TestExhaustedTrackItemCountsTowardLimit(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	g.SetupMode = false
	g.Current = VB
	g.Phase = "D"
	p := g.Players[VB]
	p.Pawn = "C2"
	p.Items = map[string]*ItemState{}
	for i := 0; i < 6; i++ { // satchel already at the limit of 6 (no bags)
		p.Items[string(rune('a'+i))] = &ItemState{Type: "boot", Zone: "satchel", FaceUp: true}
	}
	p.Items["coin#1"] = &ItemState{Type: "coin", Zone: "track", FaceUp: true}
	p.Hand = []string{"B01"}
	g.Clearings["C2"].Warriors[MC] = 1

	if got := satchelOrDamaged(p); got != 6 {
		t.Fatalf("setup satchel/damaged = %d, want 6", got)
	}
	if err := g.Apply(aidExhausting(g, "coin#1")); err != nil {
		t.Fatal(err)
	}
	// The coin is now in the Satchel: 7 items, over the limit of 6.
	if got := satchelOrDamaged(p); got != 7 {
		t.Fatalf("after exhausting the coin: satchel/damaged = %d, want 7", got)
	}
	g.vbCapacityCheck(p)
	if got := satchelOrDamaged(p); got != 6 {
		t.Fatalf("capacity check left %d items, want 6 (the limit)", got)
	}

	// With a face-up bag on the track the limit rises to 8, so nothing is removed.
	g2 := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	p2 := g2.Players[VB]
	p2.Items = map[string]*ItemState{
		"bag#1": {Type: "bag", Zone: "track", FaceUp: true},
	}
	for i := 0; i < 7; i++ {
		p2.Items[string(rune('a'+i))] = &ItemState{Type: "boot", Zone: "satchel", FaceUp: true}
	}
	limit := 6 + 2*trackCount(p2, "bag")
	if limit != 8 || satchelOrDamaged(p2) != 7 {
		t.Fatalf("limit=%d items=%d, want 8/7", limit, satchelOrDamaged(p2))
	}
	g2.vbCapacityCheck(p2)
	if got := satchelOrDamaged(p2); got != 7 {
		t.Fatalf("with a bag, capacity check removed items: %d left, want 7", got)
	}
}
