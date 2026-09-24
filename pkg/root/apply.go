package root

import "fmt"

// Apply executes an action (by ID) after re-validating it against legal actions.
func (g *Game) Apply(a Action) (err error) {
	defer g.checkWin()
	defer g.maybeFieldHospitals()
	// Capture the round/phase before the action: it may end the phase or round.
	round, phase := g.Round, g.Phase
	legal := g.LegalActions()
	var found *Action
	for i := range legal {
		if legal[i].ID == a.ID {
			found = &legal[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("illegal action %q", a.ID)
	}
	a = *found
	defer func() {
		if err == nil {
			g.recordRMN(a, round, phase)
		}
	}()
	return g.applyResolved(a)
}

// ApplyFast executes an action that is already known to be legal — one taken
// straight from the parent's LegalActions — without re-deriving and scanning the
// legal set and without recording RMN. It is the path a search should use: the
// parent already produced this exact action, so re-validating it is wasted work.
func (g *Game) ApplyFast(a Action) error {
	defer g.checkWin()
	defer g.maybeFieldHospitals()
	return g.applyResolved(a)
}

// applyResolved dispatches an already-validated action.
func (g *Game) applyResolved(a Action) error {
	g.DrawnThisAction = nil
	if a.Kind == "battle-skip" && g.Pending != nil && g.Pending.Kind == PendingFieldHospitals {
		g.FH = nil
		g.Pending = nil
		return nil
	}
	if g.SetupMode {
		return g.applySetup(a)
	}
	switch a.Kind {
	case "pass":
		g.Pass()
		return nil
	case "discard":
		p := g.Players[g.Pending.Player]
		if takeStr(&p.Hand, a.Card) {
			g.routeDiscard(a.Card)
			g.Pending.Remaining--
		}
		if g.Pending != nil && g.Pending.Remaining <= 0 {
			g.Pending = nil
		}
		return nil
	case "craft":
		return g.applyCraft(a)
	case "move":
		return g.applyMove(a)
	case "battle":
		return g.startBattle(a)
	case "dominance":
		return g.applyDominance(a)
	case "mc-recruit", "mc-build", "mc-overwork", "spend-bird", "mc-done-crafting":
		return g.applyMC(a)
	case "decree-add", "decree-recruit", "decree-move", "decree-battle", "decree-build", "leader", "ed-turmoil", "ed-done-crafting":
		return g.applyED(a)
	case "revolt", "spread", "mobilize", "train", "organize", "wa-move", "wa-battle", "wa-recruit":
		return g.applyWA(a)
	case "vb-move", "vb-slip", "vb-refresh", "vb-explore", "vb-aid", "vb-quest", "vb-strike", "vb-repair", "vb-special", "vb-battle", "vb-battle-ally", "coalition":
		return g.applyVB(a)
	case "battle-ambush", "battle-foil", "battle-hit", "battle-skip", "battle-effect":
		return g.applyBattlePending(a)
	case "field-hospitals":
		return g.applyFieldHospitals(a)
	case "royal-claim", "stand-deliver", "tax-collector", "codebreakers":
		return g.applyPersistent(a)
	case "take-dominance":
		return g.applyTakeDominance(a)
	}
	return fmt.Errorf("unhandled action kind %q", a.Kind)
}

// applyMove performs a standard move.
// maybeFieldHospitals offers Field Hospitals after any removal of Marquise
// warriors outside of a battle (battles set the pending at their end).
func (g *Game) maybeFieldHospitals() {
	if len(g.FH) == 0 || g.Battle != nil || g.Pending != nil || g.SetupMode || len(g.Winner) > 0 {
		return
	}
	if !g.hasKeep() {
		g.FH = nil
		return
	}
	g.Pending = &Pending{Kind: PendingFieldHospitals, Player: MC}
}

// applyFieldHospitals returns every Marquise warrior removed in a clearing to
// the keep for a single card that matches the clearing's suit.
func (g *Game) applyFieldHospitals(a Action) error {
	p := g.Players[MC]
	saved := 0
	rest := g.FH[:0:0]
	for _, rec := range g.FH {
		if rec.Clearing == a.Clearing {
			saved += rec.Count
			continue
		}
		rest = append(rest, rec)
	}
	if saved == 0 {
		return nil
	}
	takeStr(&p.Hand, a.Card)
	g.Discard = append(g.Discard, a.Card)
	g.addWarrior(MC, p.KeepClearing, saved)
	g.FH = rest
	g.Logf(MC, "field-hospitals", "Spent %s to save %d warrior(s) to %s", cardName(a.Card), saved, p.KeepClearing)
	if len(g.FH) == 0 {
		g.Pending = nil
	}
	return nil
}

func (g *Game) applyMove(a Action) error {
	p := g.Players[a.Faction]
	if a.Faction == MC {
		if g.MarchMovesLeft > 0 {
			g.MarchMovesLeft--
		} else if g.ActionsLeft > 0 {
			g.ActionsLeft--
			g.MarchMovesLeft = 1 // second move of this March
		}
	}
	avail := g.Clearings[a.From].Warriors[a.Faction]
	if avail <= 0 {
		return fmt.Errorf("no warriors to move")
	}
	n := a.Amount
	if n <= 0 || n > avail {
		n = avail
	}
	if g.Phase == "E" && g.ExtraMove {
		g.ExtraMove = false
	} else if a.Faction == WA && g.Phase == "E" && g.MilitaryOpsLeft > 0 {
		g.MilitaryOpsLeft--
	}
	g.addWarrior(a.Faction, a.From, -n)
	g.addWarrior(a.Faction, a.To, n)
	g.Logf(a.Faction, "move", "Moved %d warrior(s) %s → %s", n, a.From, a.To)
	g.checkOutrageAfterMove(a.Faction, a.To)
	_ = p
	return nil
}

// removePiece removes one piece of owner in clearing, warriors first.
// remover is the faction causing the removal (for Hostile/Outrage triggers).
// Returns (kind, removed) where kind is "warrior","building","token".
func (g *Game) removePiece(remover, owner Faction, c string) (string, bool) {
	cl := g.Clearings[c]
	if cl.Warriors[owner] > 0 {
		cl.Warriors[owner]--
		if cl.Warriors[owner] == 0 {
			delete(cl.Warriors, owner)
		}
		if remover == VB {
			g.vbHostilityOnWarriorRemoval(owner)
		}
		if owner == MC {
			g.FH = append(g.FH, FHRec{Clearing: c, Count: 1})
		}
		return "warrior", true
	}
	for i, b := range cl.Buildings {
		if b.Owner == owner {
			cl.Buildings = append(cl.Buildings[:i], cl.Buildings[i+1:]...)
			if owner == WA && isBaseBuilding(b.Type) {
				g.waBaseRemoved(b.Type)
			}
			if owner == MC {
				// Destroyed buildings return to the rightmost empty track slot.
				p := g.Players[MC]
				switch b.Type {
				case "sawmill":
					p.Sawmills++
				case "workshop":
					p.Workshops++
				case "recruiter":
					p.Recruiters++
				}
			}
			return "building", true
		}
	}
	for i, t := range cl.Tokens {
		if t.Owner == owner {
			cl.Tokens = append(cl.Tokens[:i], cl.Tokens[i+1:]...)
			if t.Type == "sympathy" {
				cl.Sympathy = ""
				if remover != WA {
					g.outrage(remover, c)
				}
			}
			return "token", true
		}
	}
	return "", false
}

// routeDiscard sends a card to the discard pile, or to the available-dominance
// area if it is a Dominance card.
func (g *Game) routeDiscard(card string) {
	if c, ok := Card(card); ok && c.Kind == KindDominance {
		g.AvailableDominance = append(g.AvailableDominance, card)
		return
	}
	g.Discard = append(g.Discard, card)
}

// applyTakeDominance spends a matching card to take an available dominance.
func (g *Game) applyTakeDominance(a Action) error {
	p := g.Players[a.Faction]
	if !takeStr(&p.Hand, a.Card) {
		return fmt.Errorf("card not in hand")
	}
	g.routeDiscard(a.Card)
	takeStr(&g.AvailableDominance, a.Item)
	p.Hand = append(p.Hand, a.Item)
	g.Logf(a.Faction, "dominance", "Took available %s", cardName(a.Item))
	return nil
}

// applyCraft crafts a card for the current player.
func (g *Game) applyCraft(a Action) error {
	p := g.Players[a.Faction]
	c, ok := Card(a.Card)
	if !ok {
		return fmt.Errorf("unknown card")
	}
	g.markCraftingPieces(p, c)
	takeStr(&p.Hand, a.Card)
	if c.Kind == KindItem {
		g.ItemSupply[c.Item]--
		p.CraftedItems = append(p.CraftedItems, c.Item)
		vp := c.VP
		if a.Faction == ED && p.Leader != "builder" {
			vp = 1 // Disdain for Trade
		}
		g.Score(a.Faction, vp)
		g.Logf(a.Faction, "craft", "Crafted %s → %s item (+%d VP)", c.Name, c.Item, vp)
	} else if c.Kind == KindFavor {
		g.Discard = append(g.Discard, a.Card)
		g.resolveFavor(p, c)
	} else {
		// persistent effect
		p.Crafted = append(p.Crafted, a.Card)
		g.Logf(a.Faction, "craft", "Crafted persistent %s", c.Name)
	}
	return nil
}

func (g *Game) markCraftingPieces(p *Player, c *CardDef) {
	need := append([]Suit{}, c.Cost...)
	anyNeed := c.Any
	// simple greedy marking
	switch p.Faction {
	case MC:
		for _, cl := range g.clearingsSorted() {
			for i := 0; i < g.buildingsOf(MC, cl, "workshop"); i++ {
				key := "ws:" + cl + ":" + itoa(i)
				if p.UsedThisTurn[key] {
					continue
				}
				if consumeSuit(&need, &anyNeed, g.Clearings[cl].Suit) {
					p.UsedThisTurn[key] = true
				}
			}
		}
	case ED:
		for _, cl := range g.clearingsSorted() {
			if g.buildingsOf(ED, cl, "roost") == 0 || p.UsedThisTurn["roost:"+cl] {
				continue
			}
			if consumeSuit(&need, &anyNeed, g.Clearings[cl].Suit) {
				p.UsedThisTurn["roost:"+cl] = true
			}
		}
	case WA:
		for _, cl := range g.clearingsSorted() {
			if g.Clearings[cl].Sympathy != WA || p.UsedThisTurn["sym:"+cl] {
				continue
			}
			if consumeSuit(&need, &anyNeed, g.Clearings[cl].Suit) {
				p.UsedThisTurn["sym:"+cl] = true
			}
		}
	case VB:
		for _, it := range p.Items {
			if it.Type == "hammer" && it.Zone == "satchel" && it.FaceUp && !it.Damaged {
				it.FaceUp = false
				consumeSuit(&need, &anyNeed, g.Clearings[p.Pawn].Suit)
			}
		}
	}
}

func consumeSuit(need *[]Suit, anyNeed *int, have Suit) bool {
	for i, s := range *need {
		if matches(have, s) {
			*need = append((*need)[:i:i], (*need)[i+1:]...)
			return true
		}
	}
	if *anyNeed > 0 {
		*anyNeed--
		return true
	}
	return false
}

// applyDominance activates a dominance card (no more scoring).
func (g *Game) applyDominance(a Action) error {
	p := g.Players[a.Faction]
	if !takeStr(&p.Hand, a.Card) {
		return fmt.Errorf("card not in hand")
	}
	p.Crafted = append(p.Crafted, a.Card)
	g.Logf(a.Faction, "dominance", "Activated %s; no longer scores VP", cardName(a.Card))
	return nil
}
