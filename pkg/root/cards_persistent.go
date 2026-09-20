package root

import "fmt"

// ---- Persistent card effects (base game) ----

// persistentBirdsongActions lists Birdsong actions granted by crafted cards.
func (g *Game) persistentBirdsongActions(f Faction) []Action {
	var acts []Action
	if g.hasCrafted(f, "royal-claim") {
		acts = append(acts, Action{
			ID: "royal-claim", Label: "Royal Claim: discard, score 1 VP per ruled clearing",
			Kind: "royal-claim", Faction: f,
		})
	}
	if g.hasCrafted(f, "stand-deliver") {
		for _, other := range g.Order {
			if other == f || len(g.Players[other].Hand) == 0 {
				continue
			}
			acts = append(acts, Action{
				ID:    actID("stand-deliver", string(other)),
				Label: fmt.Sprintf("Stand and Deliver!: take a card from %s", other),
				Kind:  "stand-deliver", Faction: f, Target: other,
			})
		}
	}
	return acts
}

// persistentDaylightActions lists Daylight actions granted by crafted cards.
func (g *Game) persistentDaylightActions(f Faction) []Action {
	var acts []Action
	if g.hasCrafted(f, "codebreakers") && !g.Players[f].UsedThisTurn["codebreakers"] {
		for _, other := range g.Order {
			if other == f || len(g.Players[other].Hand) == 0 {
				continue
			}
			acts = append(acts, Action{
				ID:    actID("codebreakers", string(other)),
				Label: fmt.Sprintf("Codebreakers: look at %s's hand", other),
				Kind:  "codebreakers", Faction: f, Target: other,
			})
		}
	}
	if g.hasCrafted(f, "tax-collector") && !g.TaxUsed {
		for _, c := range g.clearingsSorted() {
			if g.Clearings[c].Warriors[f] > 0 {
				acts = append(acts, Action{
					ID:    actID("tax-collector", c),
					Label: fmt.Sprintf("Tax Collector: remove a warrior at %s to draw 1", c),
					Kind:  "tax-collector", Faction: f, Clearing: c,
				})
			}
		}
	}
	return acts
}

// persistentEveningActions lists Evening actions granted by crafted cards.
func (g *Game) persistentEveningActions(f Faction) []Action {
	if g.ExtraMove && f != VB {
		return g.moveActions(f)
	}
	return nil
}

func (g *Game) applyPersistent(a Action) error {
	p := g.Players[a.Faction]
	switch a.Kind {
	case "royal-claim":
		// Discard the card, then score.
		found := false
		for _, id := range p.Crafted {
			if c, ok := Card(id); ok && c.Effect == "royal-claim" {
				takeStr(&p.Crafted, id)
				g.Discard = append(g.Discard, id)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("royal claim not crafted")
		}
		n := 0
		for c := range g.Clearings {
			if g.Rules(a.Faction, c) {
				n++
			}
		}
		g.Score(a.Faction, n)
		g.Logf(a.Faction, "royal-claim", "Royal Claim: scored %d VP for %d ruled clearings", n, n)
	case "stand-deliver":
		tp := g.Players[a.Target]
		if len(tp.Hand) == 0 {
			return fmt.Errorf("target has no cards")
		}
		idx := int(g.RngSeed % uint64(len(tp.Hand)))
		c := tp.Hand[idx]
		tp.Hand = append(tp.Hand[:idx], tp.Hand[idx+1:]...)
		p.Hand = append(p.Hand, c)
		g.Score(a.Target, 1)
		g.Logf(a.Faction, "stand-deliver", "Stand and Deliver!: took a card from %s (they scored 1 VP)", a.Target)
	case "codebreakers":
		p.UsedThisTurn["codebreakers"] = true
		g.Logf(a.Faction, "codebreakers", "Codebreakers: looked at %s's hand", a.Target)
	case "tax-collector":
		if g.TaxUsed {
			return fmt.Errorf("tax collector already used")
		}
		if g.removeWarrior(a.Faction, a.Clearing, 1) == 0 {
			return fmt.Errorf("no warrior to remove")
		}
		g.drawCards(a.Faction, 1)
		g.TaxUsed = true
		g.Logf(a.Faction, "tax-collector", "Tax Collector: removed a warrior at %s, drew 1", a.Clearing)
	default:
		return fmt.Errorf("unknown persistent action %s", a.Kind)
	}
	return nil
}

// applyBetterBurrowBank grants the Birdsong draws if crafted.
func (g *Game) applyBetterBurrowBank(f Faction) {
	if !g.hasCrafted(f, "burrow-bank") {
		return
	}
	g.drawCards(f, 1)
	for _, other := range g.Order {
		if other != f {
			g.drawCards(other, 1)
			g.Logf(f, "burrow-bank", "Better Burrow Bank: %s and %s each drew 1", f, other)
			return
		}
	}
}

// grantCommandWarren marks a free battle available at Daylight.
func (g *Game) grantCommandWarren(f Faction) {
	if g.hasCrafted(f, "command-warren") {
		g.ExtraBattle = true
	}
}

// grantCobbler marks a free Evening move available.
func (g *Game) grantCobbler(f Faction) {
	if g.hasCrafted(f, "cobbler") {
		g.ExtraMove = true
	}
}

// vbInfamy awards Infamy when the Vagabond removes a Hostile piece in battle.
func (g *Game) vbInfamy(vb Faction, victim Faction, kind string) {
	if vb != VB {
		return
	}
	if g.Players[VB].Relationships[victim] != "hostile" {
		return
	}
	if kind == "warrior" {
		// The warrior that made the faction Hostile is free; approximate by
		// only awarding for warriors when already hostile before this battle.
	}
	g.Score(VB, 1)
}

// waBaseRemoved applies the Woodland Alliance penalties when a base is removed.
func (g *Game) waBaseRemoved(baseType string) {
	p := g.Players[WA]
	s := baseSuit(baseType)
	p.Bases[s] = false // the base returns to the faction board
	// Discard all supporters matching the base suit, including birds.
	var kept []string
	for _, c := range p.Supporters {
		if matches(cardSuit(c), s) {
			g.Discard = append(g.Discard, c)
		} else {
			kept = append(kept, c)
		}
	}
	p.Supporters = kept
	// Remove half the officers (rounded up).
	loss := (p.Officers + 1) / 2
	p.Officers -= loss
	// If no bases remain, discard supporters down to 5.
	if g.waBaseCount() == 0 && len(p.Supporters) > 5 {
		g.Discard = append(g.Discard, p.Supporters[5:]...)
		p.Supporters = p.Supporters[:5]
	}
	g.Logf(WA, "base-removed", "Base removed: lost %d officer(s) and matching supporters", loss)
}

// vbCapacityCheck removes excess Vagabond items at Evening.
func (g *Game) vbCapacityCheck(p *Player) {
	limit := 6 + 2*trackCount(p, "bag")
	count := 0
	for _, it := range p.Items {
		if it.Zone == "satchel" || it.Zone == "damaged" {
			count++
		}
	}
	if count <= limit {
		return
	}
	remove := count - limit
	for id, it := range p.Items {
		if remove <= 0 {
			break
		}
		if it.Zone == "satchel" || it.Zone == "damaged" {
			delete(p.Items, id)
			remove--
		}
	}
	g.Logf(VB, "capacity", "Removed excess items (limit %d)", limit)
}
