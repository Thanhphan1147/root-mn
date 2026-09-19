package root

import "fmt"

// ---- Woodland Alliance ----

func (g *Game) legalWABirdsong() []Action {
	p := g.Players[WA]
	var acts []Action
	// Revolt options.
	for _, c := range g.clearingsSorted() {
		cl := g.Clearings[c]
		if cl.Sympathy != WA {
			continue
		}
		if g.buildingsOf(WA, c, "base-"+string(cl.Suit)) > 0 || p.Bases[cl.Suit] {
			continue
		}
		if !g.hasBaseInHand(p, cl.Suit) {
			continue
		}
		if g.matchingSupporters(p, cl.Suit) >= 2 {
			acts = append(acts, Action{
				ID: actID("revolt", c), Label: "Revolt at " + c, Kind: "revolt",
				Faction: WA, Clearing: c,
			})
		}
	}
	// Spread sympathy options.
	k := g.sympathyOnMap() + 1
	if k <= 10 {
		cost := SympathyCost[k]
		for _, c := range g.clearingsSorted() {
			cl := g.Clearings[c]
			if cl.Sympathy == WA {
				continue
			}
			if !g.Relaxed {
				if g.isKeep(c) {
					continue
				}
				if g.sympathyOnMap() > 0 && !g.adjacentToSympathy(c) {
					continue
				}
			}
			extra := 0
			if g.martialLaw(c) {
				extra = 1
			}
			if g.matchingSupporters(p, cl.Suit) >= cost+extra {
				acts = append(acts, Action{
					ID:    actID("spread", c),
					Label: fmt.Sprintf("Spread sympathy at %s (cost %d, %d VP)", c, cost+extra, SympathyVP[k]),
					Kind:  "spread", Faction: WA, Clearing: c,
				})
			}
		}
	}
	acts = append(acts, Action{ID: "pass", Label: "End Birdsong", Kind: "pass", Faction: WA})
	return acts
}

func (g *Game) hasBaseInHand(p *Player, s Suit) bool { return !p.Bases[s] }

func (g *Game) isKeep(c string) bool {
	for _, t := range g.Clearings[c].Tokens {
		if t.Type == "keep" {
			return true
		}
	}
	return false
}

func (g *Game) adjacentToSympathy(c string) bool {
	for _, n := range g.Clearings[c].Adj {
		if g.Clearings[n].Sympathy == WA {
			return true
		}
	}
	return false
}

func (g *Game) martialLaw(c string) bool {
	cl := g.Clearings[c]
	for f, n := range cl.Warriors {
		if f != WA && n >= 3 {
			return true
		}
	}
	return false
}

func (g *Game) matchingSupporters(p *Player, s Suit) int {
	n := 0
	for _, c := range p.Supporters {
		if matches(cardSuit(c), s) {
			n++
		}
	}
	return n
}

func (g *Game) spendSupporters(p *Player, s Suit, n int) {
	for i := 0; i < n; i++ {
		for j, c := range p.Supporters {
			if matches(cardSuit(c), s) {
				p.Supporters = append(p.Supporters[:j:j], p.Supporters[j+1:]...)
				g.Discard = append(g.Discard, c)
				break
			}
		}
	}
}

func (g *Game) legalWADaylight() []Action {
	p := g.Players[WA]
	acts := g.craftActions(p)
	for _, id := range p.Hand {
		acts = append(acts, Action{
			ID: actID("mobilize", id), Label: "Mobilize " + cardName(id), Kind: "mobilize",
			Faction: WA, Card: id,
		})
	}
	// Train: spend a card matching a base clearing suit.
	for _, id := range p.Hand {
		for _, c := range g.clearingsSorted() {
			if g.buildingsOf(WA, c, "base-"+string(g.Clearings[c].Suit)) > 0 && matches(cardSuit(id), g.Clearings[c].Suit) {
				acts = append(acts, Action{
					ID: actID("train", id, c), Label: fmt.Sprintf("Train: spend %s → officer", cardName(id)),
					Kind: "train", Faction: WA, Card: id, Clearing: c,
				})
			}
		}
	}
	return acts
}

func (g *Game) legalWAEvening() []Action {
	var acts []Action
	if g.MilitaryOpsLeft > 0 {
		acts = append(acts, g.moveActions(WA)...)
		acts = append(acts, g.battleActions(WA)...)
		for _, c := range g.clearingsSorted() {
			if g.buildingsOf(WA, c, "base-"+string(g.Clearings[c].Suit)) > 0 {
				acts = append(acts, Action{
					ID: actID("wa-recruit", c), Label: "Recruit at " + c, Kind: "wa-recruit",
					Faction: WA, Clearing: c,
				})
			}
		}
		k := g.sympathyOnMap() + 1
		if k <= 10 {
			for _, c := range g.clearingsSorted() {
				if g.Clearings[c].Sympathy == WA || g.Clearings[c].Warriors[WA] == 0 {
					continue
				}
				acts = append(acts, Action{
					ID:    actID("organize", c),
					Label: fmt.Sprintf("Organize at %s (%d VP)", c, SympathyVP[k]),
					Kind:  "organize", Faction: WA, Clearing: c,
				})
			}
		}
	}
	return acts
}

func (g *Game) waEveningDraw(p *Player) {
	bases := g.totalBuildings(WA, "")
	draw := 1
	for _, c := range g.Clearings {
		for _, b := range c.Buildings {
			if b.Owner == WA && len(b.Type) >= 4 && b.Type[:4] == "base" {
				_ = b
			}
		}
	}
	draw = 1 + g.waBaseCount()
	g.drawCards(WA, draw)
	g.Logf(WA, "evening", "Drew %d card(s)", draw)
	g.queueDiscard(p)
	_ = bases
}

func (g *Game) waBaseCount() int {
	n := 0
	for _, c := range g.Clearings {
		for _, b := range c.Buildings {
			if b.Owner == WA && len(b.Type) > 5 && b.Type[:5] == "base-" {
				n++
			}
		}
	}
	return n
}

func (g *Game) addSupporter(p *Player, card string) {
	if g.waBaseCount() == 0 && len(p.Supporters) >= 5 {
		g.Discard = append(g.Discard, card)
		return
	}
	p.Supporters = append(p.Supporters, card)
}

func (g *Game) applyWA(a Action) error {
	p := g.Players[WA]
	switch a.Kind {
	case "revolt":
		cl := g.Clearings[a.Clearing]
		s := cl.Suit
		g.spendSupporters(p, s, 2)
		vp := 0
		for _, f := range g.Order {
			if f == WA {
				continue
			}
			for {
				kind, ok := g.removePiece(f, a.Clearing)
				if !ok {
					break
				}
				if kind == "building" || kind == "token" {
					vp++
				}
			}
		}
		g.Clearings[a.Clearing].Buildings = append(cl.Buildings, Building{WA, "base-" + string(s)})
		p.Bases[s] = true
		warriors := 0
		for _, c := range g.Clearings {
			if c.Sympathy == WA && c.Suit == s {
				warriors++
			}
		}
		g.addWarrior(WA, a.Clearing, warriors)
		p.Officers++
		g.Score(WA, vp)
		g.Logf(WA, "revolt", "Revolt at %s: base placed, %d enemy piece(s) removed (+%d VP)", a.Clearing, vp, vp)
	case "spread":
		cl := g.Clearings[a.Clearing]
		k := g.sympathyOnMap() + 1
		cost := SympathyCost[k]
		if g.martialLaw(a.Clearing) {
			cost++
		}
		g.spendSupporters(p, cl.Suit, cost)
		cl.Sympathy = WA
		g.Score(WA, SympathyVP[k])
		g.Logf(WA, "sympathy", "Spread sympathy at %s (+%d VP)", a.Clearing, SympathyVP[k])
	case "mobilize":
		if takeStr(&p.Hand, a.Card) {
			g.addSupporter(p, a.Card)
			g.Logf(WA, "mobilize", "Mobilized %s", cardName(a.Card))
		}
	case "train":
		if takeStr(&p.Hand, a.Card) {
			g.Discard = append(g.Discard, a.Card)
			p.Officers++
			g.Logf(WA, "train", "Trained an officer (now %d)", p.Officers)
		}
	case "organize":
		g.removeWarrior(WA, a.Clearing, 1)
		g.Clearings[a.Clearing].Sympathy = WA
		k := g.sympathyOnMap()
		if k > 10 {
			k = 10
		}
		g.Score(WA, SympathyVP[k])
		g.MilitaryOpsLeft--
		g.Logf(WA, "organize", "Organized at %s (+%d VP)", a.Clearing, SympathyVP[k])
	case "wa-recruit":
		g.addWarrior(WA, a.Clearing, 1)
		g.MilitaryOpsLeft--
		g.Logf(WA, "recruit", "Recruited at %s", a.Clearing)
	default:
		return fmt.Errorf("unknown WA action %s", a.Kind)
	}
	return nil
}

// checkOutrageAfterMove applies Outrage when offender moves warriors into a
// sympathetic clearing.
func (g *Game) checkOutrageAfterMove(offender Faction, clearing string) {
	if offender == WA {
		return
	}
	if g.Clearings[clearing].Sympathy != WA {
		return
	}
	g.outrage(offender, clearing)
}

func (g *Game) outrage(offender Faction, clearing string) {
	p := g.Players[offender]
	s := g.Clearings[clearing].Suit
	for _, c := range p.Hand {
		if matches(cardSuit(c), s) {
			takeStr(&p.Hand, c)
			g.addSupporter(g.Players[WA], c)
			g.Logf(WA, "outrage", "Outrage: %s gave %s", offender, cardName(c))
			return
		}
	}
	// no matching card: WA draws 1
	if len(g.Deck) == 0 {
		g.Deck = append(g.Deck, g.Discard...)
		g.Discard = nil
		g.shuffle()
	}
	if len(g.Deck) > 0 {
		c := g.Deck[0]
		g.Deck = g.Deck[1:]
		g.addSupporter(g.Players[WA], c)
		g.Logf(WA, "outrage", "Outrage: %s had no matching card; WA drew one", offender)
	}
}
