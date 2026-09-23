package root

import "fmt"

// ---- Marquise de Cat ----

func (g *Game) legalMCDaylight() []Action {
	p := g.Players[MC]
	// Daylight step 1: craft with workshops before taking actions.
	if g.MCDayStage == "craft" {
		acts := g.craftActions(p)
		acts = append(acts, Action{
			ID: "mc-done-crafting", Label: "Done crafting — take actions",
			Kind: "mc-done-crafting", Faction: MC,
		})
		return acts
	}
	var acts []Action
	acts = append(acts, g.spendBirdActions(p)...)

	// Dominance activation (>=10 VP).
	if p.VP >= 10 {
		for _, id := range p.Hand {
			if c, ok := Card(id); ok && c.Kind == KindDominance {
				acts = append(acts, Action{
					ID: actID("dominance", id), Label: "Activate " + c.Name + " (Dominance)",
					Kind: "dominance", Faction: MC, Card: id,
				})
			}
		}
	}

	if g.ActionsLeft > 0 || g.MarchMovesLeft > 0 {
		acts = append(acts, g.moveActions(MC)...)
	}
	if g.ActionsLeft > 0 {
		acts = append(acts, g.battleActions(MC)...)
		if !g.RecruitedThisTurn {
			if g.totalBuildings(MC, "recruiter") > 0 {
				acts = append(acts, Action{ID: "mc-recruit", Label: "Recruit (1 per recruiter)", Kind: "mc-recruit", Faction: MC})
			}
		}
		acts = append(acts, g.mcBuildActions(p)...)
		acts = append(acts, g.mcOverworkActions(p)...)
	}
	return acts
}

func (g *Game) buildingsOfTotal(f Faction, typ string) int { return g.totalBuildings(f, typ) }

func (g *Game) mcBuildActions(p *Player) []Action {
	var acts []Action
	types := []string{"sawmill", "workshop", "recruiter"}
	for _, c := range g.clearingsSorted() {
		cl := g.Clearings[c]
		if !g.Rules(MC, c) || cl.FreeSlots() <= 0 {
			continue
		}
		for _, typ := range types {
			if g.buildingsOf(MC, c, typ) > 0 {
				continue // only one of each building type per clearing
			}
			remaining := p.Sawmills
			switch typ {
			case "workshop":
				remaining = p.Workshops
			case "recruiter":
				remaining = p.Recruiters
			}
			if remaining <= 0 {
				continue
			}
			slot := 7 - remaining
			cost := MCBuildCost[slot]
			if g.availableWood(p, c) < cost {
				continue
			}
			acts = append(acts, Action{
				ID:    actID("mc-build", typ, c),
				Label: fmt.Sprintf("Build %s at %s (cost %d wood, %d VP)", typ, c, cost, mcVP(typ, slot)),
				Kind:  "mc-build", Faction: MC, Clearing: c, Building: typ,
			})
		}
	}
	return acts
}

func mcVP(typ string, slot int) int {
	switch typ {
	case "sawmill":
		return MCSawmillVP[slot]
	case "workshop":
		return MCWorkshopVP[slot]
	case "recruiter":
		return MCRecruiterVP[slot]
	}
	return 0
}

func (g *Game) mcOverworkActions(p *Player) []Action {
	if p.WoodSupply <= 0 {
		return nil
	}
	var acts []Action
	for _, c := range g.clearingsSorted() {
		if g.buildingsOf(MC, c, "sawmill") == 0 {
			continue
		}
		for _, id := range p.Hand {
			if !matches(cardSuit(id), g.Clearings[c].Suit) {
				continue
			}
			acts = append(acts, Action{
				ID:    actID("mc-overwork", c, id),
				Label: fmt.Sprintf("Overwork: spend %s → wood at %s", cardName(id), c),
				Kind:  "mc-overwork", Faction: MC, Clearing: c, Card: id,
			})
		}
	}
	return acts
}

// availableWood counts wood in the chosen clearing plus ruled clearings
// connected to it through ruled clearings.
func (g *Game) availableWood(p *Player, start string) int {
	if g.Relaxed {
		total := g.Clearings[start].Wood
		for _, n := range g.Clearings[start].Adj {
			total += g.Clearings[n].Wood
		}
		return total
	}
	seen := map[string]bool{start: true}
	queue := []string{start}
	total := g.Clearings[start].Wood
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, n := range g.Clearings[c].Adj {
			if seen[n] {
				continue
			}
			if !g.Rules(MC, n) {
				continue
			}
			seen[n] = true
			total += g.Clearings[n].Wood
			queue = append(queue, n)
		}
	}
	return total
}

func (g *Game) applyMC(a Action) error {
	p := g.Players[MC]
	switch a.Kind {
	case "mc-done-crafting":
		g.MCDayStage = "actions"
		g.Logf(MC, "daylight", "Finished crafting; taking actions")
	case "mc-recruit":
		n := g.totalBuildings(MC, "recruiter")
		for _, c := range g.clearingsSorted() {
			r := g.buildingsOf(MC, c, "recruiter")
			if r > 0 {
				g.addWarrior(MC, c, r)
			}
		}
		g.RecruitedThisTurn = true
		g.ActionsLeft--
		g.Logf(MC, "recruit", "Recruited %d warrior(s) across recruiter clearings", n)
	case "mc-build":
		remaining := p.Sawmills
		switch a.Building {
		case "workshop":
			remaining = p.Workshops
		case "recruiter":
			remaining = p.Recruiters
		}
		slot := 7 - remaining
		cost := MCBuildCost[slot]
		g.spendWood(a.Clearing, cost)
		g.Clearings[a.Clearing].Buildings = append(g.Clearings[a.Clearing].Buildings, Building{MC, a.Building})
		switch a.Building {
		case "sawmill":
			p.Sawmills--
		case "workshop":
			p.Workshops--
		case "recruiter":
			p.Recruiters--
		}
		vp := mcVP(a.Building, slot)
		g.Score(MC, vp)
		g.ActionsLeft--
		g.Logf(MC, "build", "Built %s at %s, scored %d VP", a.Building, a.Clearing, vp)
	case "mc-overwork":
		takeStr(&p.Hand, a.Card)
		g.routeDiscard(a.Card)
		g.Clearings[a.Clearing].Wood++
		p.WoodSupply--
		g.ActionsLeft--
		g.Logf(MC, "overwork", "Overworked at %s: +1 wood", a.Clearing)
	case "spend-bird":
		takeStr(&p.Hand, a.Card)
		g.Discard = append(g.Discard, a.Card)
		g.ActionsLeft++
		g.BirdSpent++
		g.Logf(MC, "bird", "Spent bird card for +1 action")
	default:
		return fmt.Errorf("unknown MC action %s", a.Kind)
	}
	return nil
}

// spendWood removes wood from the chosen clearing and connected ruled clearings.
// Spent wood returns to the supply (the Marquise's wood is a fixed pool).
func (g *Game) spendWood(start string, n int) {
	if n <= 0 {
		return
	}
	p := g.Players[MC]
	// order: start first, then BFS ruled neighbours.
	order := []string{start}
	seen := map[string]bool{start: true}
	queue := []string{start}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, nb := range g.Clearings[c].Adj {
			if seen[nb] || !g.Rules(MC, nb) {
				continue
			}
			seen[nb] = true
			order = append(order, nb)
			queue = append(queue, nb)
		}
	}
	for _, c := range order {
		if n <= 0 {
			return
		}
		take := g.Clearings[c].Wood
		if take > n {
			take = n
		}
		g.Clearings[c].Wood -= take
		n -= take
		p.WoodSupply += take
	}
}
