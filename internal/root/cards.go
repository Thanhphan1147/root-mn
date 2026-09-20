package root

import "strings"

// resolveFavor removes all enemy pieces in clearings of the favor's suit.
func (g *Game) resolveFavor(p *Player, c *CardDef) {
	suit := Fox
	switch c.Effect {
	case "favor-rabbit":
		suit = Rabbit
	case "favor-mouse":
		suit = Mouse
	}
	vp := 0
	for _, cl := range g.clearingsSorted() {
		if g.Clearings[cl].Suit != suit {
			continue
		}
		for _, f := range g.Order {
			if f == p.Faction || !g.areEnemies(p.Faction, f) {
				continue
			}
			if f == VB && g.Players[VB].Pawn == cl {
				// Vagabond damages 3 items.
				d := 0
				for _, it := range g.Players[VB].Items {
					if !it.Damaged && d < 3 {
						it.Damaged = true
						it.Zone = "damaged"
						d++
					}
				}
				continue
			}
			for {
				kind, ok := g.removePiece(p.Faction, f, cl)
				if !ok {
					break
				}
				if kind == "building" || kind == "token" {
					vp++
				}
			}
		}
	}
	g.Score(p.Faction, vp)
	g.Logf(p.Faction, "favor", "Resolved %s: %d enemy building/token(s) removed (+%d VP)", c.Name, vp, vp)
}

// isBaseBuilding reports whether a building type is a Woodland Alliance base.
func isBaseBuilding(t string) bool { return strings.HasPrefix(t, "base-") }

// baseSuit extracts the suit of a base building type.
func baseSuit(t string) Suit {
	switch t {
	case "base-fox":
		return Fox
	case "base-rabbit":
		return Rabbit
	case "base-mouse":
		return Mouse
	}
	return ""
}
