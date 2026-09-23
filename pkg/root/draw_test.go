package root

import "testing"

// TestEveningDrawBonuses checks each faction's Evening draw against the board's
// uncovered draw bonuses.
func TestEveningDrawBonuses(t *testing.T) {
	g := newTestGame(t)

	clearRoosts := func() {
		for _, c := range g.Clearings {
			kept := c.Buildings[:0:0]
			for _, b := range c.Buildings {
				if b.Owner != ED || b.Type != "roost" {
					kept = append(kept, b)
				}
			}
			c.Buildings = kept
		}
	}
	addRoosts := func(n int) {
		i := 0
		for cid := range g.Clearings {
			if i >= n {
				break
			}
			g.Clearings[cid].Buildings = append(g.Clearings[cid].Buildings, Building{ED, "roost"})
			i++
		}
	}

	// Eyrie: +1 at 3 roosts, +1 more at 6.
	clearRoosts()
	addRoosts(3)
	ed := g.Players[ED]
	ed.Hand = nil
	g.Current = ED
	g.Phase = "E"
	g.beginEvening()
	if len(ed.Hand) != 2 {
		t.Fatalf("Eyrie at 3 roosts drew %d cards, want 2", len(ed.Hand))
	}
	addRoosts(3)
	ed.Hand = nil
	g.beginEvening()
	if len(ed.Hand) != 3 {
		t.Fatalf("Eyrie at 6 roosts drew %d cards, want 3", len(ed.Hand))
	}

	// Marquise: +1 at 3 recruiters, +1 more at 5 (max 3 total).
	mc := g.Players[MC]
	mc.Recruiters = 3 // 6-3 = 3 on the map
	mc.Hand = nil
	g.Current = MC
	g.Phase = "E"
	g.beginEvening()
	if len(mc.Hand) != 2 {
		t.Fatalf("Marquise at 3 recruiters drew %d cards, want 2", len(mc.Hand))
	}
	mc.Recruiters = 1 // 5 on the map
	mc.Hand = nil
	g.beginEvening()
	if len(mc.Hand) != 3 {
		t.Fatalf("Marquise at 5 recruiters drew %d cards, want 3", len(mc.Hand))
	}

	// Woodland Alliance: +1 per base on the map.
	wa := g.Players[WA]
	wa.Officers = 0
	g.Clearings["C6"].Buildings = append(g.Clearings["C6"].Buildings, Building{WA, "base-fox"})
	g.Clearings["C8"].Buildings = append(g.Clearings["C8"].Buildings, Building{WA, "base-rabbit"})
	wa.Hand = nil
	g.Current = WA
	g.Phase = "E"
	g.beginEvening()
	if len(wa.Hand) != 3 {
		t.Fatalf("Alliance at 2 bases drew %d cards, want 3", len(wa.Hand))
	}

	// Vagabond: +1 per coin on the track.
	vb := g.Players[VB]
	vb.Items["coin#1"] = &ItemState{Type: "coin", Zone: "track", FaceUp: true}
	vb.Items["coin#2"] = &ItemState{Type: "coin", Zone: "track", FaceUp: true}
	vb.Hand = nil
	g.Current = VB
	g.Phase = "E"
	g.beginEvening()
	if len(vb.Hand) != 3 {
		t.Fatalf("Vagabond with 2 track coins drew %d cards, want 3", len(vb.Hand))
	}
}
