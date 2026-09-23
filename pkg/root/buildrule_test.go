package root

import "testing"

// TestMCBuildRules checks Marquise build rules the C9 playtest log raised: you
// may not stack a second building of the same type in a clearing, and a
// recruiter must be offered when wood and a slot are available.
func TestMCBuildRules(t *testing.T) {
	g := NewGame([]Faction{MC, ED}, MC, 7)
	BeginSetup(g)
	g.setupMCKeep("C1")
	for _, typ := range []string{"sawmill", "workshop", "recruiter"} {
		for _, c := range append([]string{"C1"}, GetMap("autumn").Clearings["C1"].Adj...) {
			if g.setupMCBuild(typ, c) {
				break
			}
		}
	}
	g.setupEDCorner(oppositeCorner("C1"))
	g.setupEDLeader("despot")

	// Pick a clearing MC rules with at least two free slots so there is room
	// for a second building after we add a workshop.
	target := ""
	for _, c := range g.clearingsSorted() {
		if g.Rules(MC, c) && g.Clearings[c].FreeSlots() >= 2 {
			target = c
			break
		}
	}
	if target == "" {
		t.Fatalf("no ruled clearing with two free slots")
	}
	g.Clearings[target].Buildings = append(g.Clearings[target].Buildings, Building{MC, "workshop"})

	for _, c := range g.Clearings {
		c.Wood = 9
	}
	p := g.Players[MC]
	acts := g.mcBuildActions(p)

	sawRecruiter, sawDupWorkshop := false, false
	for _, a := range acts {
		if a.Clearing != target {
			continue
		}
		switch a.Building {
		case "recruiter":
			sawRecruiter = true
		case "workshop":
			sawDupWorkshop = true
		}
	}
	if !sawRecruiter {
		t.Errorf("no recruiter build offered at %s (recruiters=%d, free=%d, ample wood)",
			target, p.Recruiters, g.Clearings[target].FreeSlots())
	}
	if sawDupWorkshop {
		t.Errorf("offered a second workshop at %s which already has one", target)
	}
}
