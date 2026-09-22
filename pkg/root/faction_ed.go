package root

import "fmt"

// ---- Eyrie Dynasties ----

func (g *Game) legalEDDecreeAdd() []Action {
	p := g.Players[ED]
	if g.EDNeedsLeader {
		return g.leaderActions()
	}
	if g.EDAdded >= 2 {
		return nil
	}
	var acts []Action
	for _, id := range p.Hand {
		if cardSuit(id) == Bird && g.EDBirdAdded {
			continue
		}
		for _, col := range DecreeColumns {
			acts = append(acts, Action{
				ID:    actID("decree-add", id, col),
				Label: fmt.Sprintf("Decree: add %s to %s", cardName(id), col),
				Kind:  "decree-add", Faction: ED, Card: id, Column: col,
			})
		}
	}
	return acts
}

// EDHasBird reports whether a bird card has already been added this Birdsong.
func (g *Game) EDHasBird() bool {
	p := g.Players[ED]
	for _, col := range DecreeColumns {
		for _, c := range p.Decree[col] {
			if c != "VIZIER" && cardSuit(c) == Bird {
				// approximate: any bird in decree counts
				return true
			}
		}
	}
	return false
}

func (g *Game) leaderActions() []Action {
	var acts []Action
	retired := map[string]bool{}
	for _, r := range g.Players[ED].RetiredLeaders {
		retired[r] = true
	}
	for name := range Leaders {
		if retired[name] {
			continue
		}
		acts = append(acts, Action{
			ID: actID("leader", name), Label: "Choose leader: " + name,
			Kind: "leader", Faction: ED, Leader: name,
		})
	}
	return acts
}

func (g *Game) legalEDDaylight() []Action {
	if g.EDNeedsLeader {
		return g.leaderActions()
	}
	p := g.Players[ED]
	// Daylight step 1: Craft (before resolving the Decree).
	if g.EDDayStage == "craft" {
		acts := g.craftActions(p)
		acts = append(acts, Action{
			ID: "ed-done-crafting", Label: "Done crafting — resolve the Decree",
			Kind: "ed-done-crafting", Faction: ED,
		})
		return acts
	}
	// Daylight step 2: Resolve the Decree.
	if len(g.DecreeQueue) == 0 {
		return nil
	}
	// Columns resolve left to right; within a column, cards may resolve in any order.
	col := g.DecreeQueue[0].Column
	var cards []DecreeItem
	for _, it := range g.DecreeQueue {
		if it.Column == col {
			cards = append(cards, it)
		}
	}
	var acts []Action
	for _, it := range cards {
		suit := decreeSuit(it.Card)
		switch col {
		case "RECRUIT":
			// A recruit that the supply cannot cover is unresolvable, so the
			// only legal move for the column becomes Turmoil.
			need := 1
			if p.Leader == "charismatic" {
				need = 2
			}
			if g.warriorSupplyLeft(ED) < need {
				continue
			}
			for _, c := range g.clearingsSorted() {
				if g.buildingsOf(ED, c, "roost") > 0 && matches(g.Clearings[c].Suit, suit) {
					acts = append(acts, Action{
						ID:    actID("decree-recruit", c, it.Card),
						Label: fmt.Sprintf("Decree Recruit at %s (%s)", c, cardName(it.Card)),
						Kind:  "decree-recruit", Faction: ED, Clearing: c, Card: it.Card, Column: col,
					})
				}
			}
		case "MOVE":
			for _, from := range g.clearingsSorted() {
				if !matches(g.Clearings[from].Suit, suit) || g.Clearings[from].Warriors[ED] == 0 {
					continue
				}
				for _, to := range g.Clearings[from].Adj {
					if !g.Relaxed && !g.Rules(ED, from) && !g.Rules(ED, to) {
						continue
					}
					n := g.Clearings[from].Warriors[ED]
					for q := 1; q <= n; q++ {
						acts = append(acts, Action{
							ID:    actID("decree-move", from, to, it.Card, itoa(q)),
							Label: fmt.Sprintf("Decree Move %d %s → %s (%s)", q, from, to, cardName(it.Card)),
							Kind:  "decree-move", Faction: ED, From: from, To: to, Card: it.Card, Amount: q, Column: col,
						})
					}
				}
			}
		case "BATTLE":
			for _, c := range g.clearingsSorted() {
				if !matches(g.Clearings[c].Suit, suit) || g.Clearings[c].Warriors[ED] == 0 {
					continue
				}
				for _, def := range g.enemiesIn(ED, c) {
					acts = append(acts, Action{
						ID:    actID("decree-battle", c, string(def), it.Card),
						Label: fmt.Sprintf("Decree Battle %s in %s (%s)", def, c, cardName(it.Card)),
						Kind:  "decree-battle", Faction: ED, Clearing: c, Target: def, Card: it.Card, Column: col,
					})
				}
			}
		case "BUILD":
			for _, c := range g.clearingsSorted() {
				if !matches(g.Clearings[c].Suit, suit) || !g.Rules(ED, c) {
					continue
				}
				if g.isKeep(c) {
					continue // only the Marquise may place pieces in the keep clearing
				}
				if g.buildingsOf(ED, c, "roost") > 0 || g.Clearings[c].FreeSlots() <= 0 {
					continue
				}
				if g.totalBuildings(ED, "roost") >= 7 {
					continue
				}
				acts = append(acts, Action{
					ID:    actID("decree-build", c, it.Card),
					Label: fmt.Sprintf("Decree Build roost at %s (%s)", c, cardName(it.Card)),
					Kind:  "decree-build", Faction: ED, Clearing: c, Card: it.Card, Column: col,
				})
			}
		}
	}
	// If no card in the current column can be resolved, Turmoil is the only way.
	if len(acts) == 0 {
		acts = append(acts, Action{
			ID:    "ed-turmoil",
			Label: "Turmoil — cannot resolve the Decree (lose VP per bird, purge, new leader)",
			Kind:  "ed-turmoil", Faction: ED,
		})
	}
	return acts
}

func (g *Game) turmoil() {
	p := g.Players[ED]
	// Humiliate: lose 1 VP per bird card in decree incl. viziers.
	birds := 0
	for _, col := range DecreeColumns {
		for _, c := range p.Decree[col] {
			if c == "VIZIER" || cardSuit(c) == Bird {
				birds++
			}
		}
	}
	p.VP -= birds
	if p.VP < 0 {
		p.VP = 0
	}
	// Purge: discard non-vizier decree cards; viziers set aside.
	for _, col := range DecreeColumns {
		for _, c := range p.Decree[col] {
			if c != "VIZIER" {
				g.routeDiscard(c)
			}
		}
	}
	p.Decree = map[string][]string{}
	p.Viziers = nil
	// Depose.
	if p.Leader != "" {
		p.RetiredLeaders = append(p.RetiredLeaders, p.Leader)
	}
	p.Leader = ""
	if len(p.RetiredLeaders) >= len(Leaders) {
		p.RetiredLeaders = nil // A New Clutch: flip all retired leaders face up
	}
	g.DecreeQueue = nil
	g.EDNeedsLeader = true
	g.Logf(ED, "turmoil", "Turmoil! Lost %d VP; choose a new leader", birds)
}

func (g *Game) applyED(a Action) error {
	p := g.Players[ED]
	switch a.Kind {
	case "ed-done-crafting":
		g.EDDayStage = "decree"
		g.Logf(ED, "daylight", "Finished crafting; resolving the Decree")
	case "decree-add":
		if g.EDAdded >= 2 {
			return fmt.Errorf("already added 2 cards")
		}
		if cardSuit(a.Card) == Bird && g.EDBirdAdded {
			return fmt.Errorf("only one bird card may be added")
		}
		if !takeStr(&p.Hand, a.Card) {
			return fmt.Errorf("card not in hand")
		}
		p.Decree[a.Column] = append(p.Decree[a.Column], a.Card)
		g.EDAdded++
		if cardSuit(a.Card) == Bird {
			g.EDBirdAdded = true
		}
		g.Logf(ED, "decree", "Added %s to %s", cardName(a.Card), a.Column)
	case "leader":
		p.Leader = a.Leader
		p.Viziers = nil
		ld := Leaders[a.Leader]
		for _, col := range ld.Viziers {
			p.Decree[col] = append(p.Decree[col], "VIZIER")
			p.Viziers = append(p.Viziers, "VIZIER")
		}
		g.EDNeedsLeader = false
		g.Logf(ED, "leader", "Chose leader %s", a.Leader)
		if g.EDTurmoilRest {
			g.EDTurmoilRest = false
			g.beginEvening()
		}
	case "decree-recruit":
		n := 1
		if p.Leader == "charismatic" {
			n = 2
		}
		g.addWarrior(ED, a.Clearing, n)
		g.popDecree(a.Card, a.Column)
		g.Logf(ED, "decree", "Recruited %d at %s", n, a.Clearing)
	case "decree-move":
		n := a.Amount
		if n <= 0 || n > g.Clearings[a.From].Warriors[ED] {
			n = g.Clearings[a.From].Warriors[ED]
		}
		g.addWarrior(ED, a.From, -n)
		g.addWarrior(ED, a.To, n)
		g.popDecree(a.Card, a.Column)
		g.Logf(ED, "decree", "Moved %d %s → %s", n, a.From, a.To)
		g.checkOutrageAfterMove(ED, a.To)
	case "decree-battle":
		ba := Action{Kind: "battle", Faction: ED, Clearing: a.Clearing, Target: a.Target, Card: a.Card}
		if err := g.startBattle(ba); err != nil {
			return err
		}
		// pop decree after battle completes; store pending card in battle context
		g.popDecree(a.Card, a.Column)
	case "decree-build":
		g.Clearings[a.Clearing].Buildings = append(g.Clearings[a.Clearing].Buildings, Building{ED, "roost"})
		g.popDecree(a.Card, a.Column)
		g.Logf(ED, "decree", "Built roost at %s", a.Clearing)
	case "ed-turmoil":
		g.turmoil()
		g.EDTurmoilRest = true
		if !g.EDNeedsLeader {
			g.EDTurmoilRest = false
			g.beginEvening()
		}
	default:
		return fmt.Errorf("unknown ED action %s", a.Kind)
	}
	return nil
}

// decreeSuit returns the suit a Decree card is resolved as. Loyal Viziers are
// bird cards, so they match any clearing.
func decreeSuit(card string) Suit {
	if card == "VIZIER" {
		return Bird
	}
	return cardSuit(card)
}

func (g *Game) popDecree(card, column string) {
	for i, it := range g.DecreeQueue {
		if it.Card == card && it.Column == column {
			g.DecreeQueue = append(g.DecreeQueue[:i:i], g.DecreeQueue[i+1:]...)
			return
		}
	}
	if len(g.DecreeQueue) > 0 {
		g.DecreeQueue = g.DecreeQueue[1:]
	}
}
