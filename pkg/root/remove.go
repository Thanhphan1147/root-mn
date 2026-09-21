package root

// RemoveFaction drops a faction from the game entirely: its pieces are taken
// off the board, it leaves the turn order, and any deferred state waiting on it
// is cleared. If it was mid-turn, play advances to the next faction's Birdsong.
//
// This backs the administrator's "eject an absent player" command, which is why
// it is far more aggressive than any in-game effect.
func (g *Game) RemoveFaction(f Faction) {
	if g == nil {
		return
	}
	if _, ok := g.Players[f]; !ok {
		return
	}

	for _, cl := range g.Clearings {
		delete(cl.Warriors, f)
		cl.Buildings = withoutBuildings(cl.Buildings, f)
		cl.Tokens = withoutTokens(cl.Tokens, f)
		if cl.Sympathy == f {
			cl.Sympathy = ""
		}
		if f == MC {
			cl.Wood = 0 // wood is the Marquise's
		}
	}

	wasCurrent := g.Current == f
	idx := -1
	for i, x := range g.Order {
		if x == f {
			idx = i
			break
		}
	}
	order := g.Order[:0]
	for _, x := range g.Order {
		if x != f {
			order = append(order, x)
		}
	}
	g.Order = order
	delete(g.Players, f)

	if g.First == f {
		g.First = ""
		if len(g.Order) > 0 {
			g.First = g.Order[0]
		}
	}

	if g.Pending != nil && g.Pending.Player == f {
		g.Pending = nil
	}
	if g.Battle != nil && (g.Battle.Attacker == f || g.Battle.Defender == f ||
		g.Battle.HitSide == f || g.Battle.AllyTarget == f) {
		g.Battle = nil
	}
	switch f {
	case MC:
		g.FH = nil
		g.MCDayStage = ""
	case ED:
		g.DecreeQueue = nil
		g.EDDayStage = ""
	case WA:
		g.WAEveningDrawn = false
	}

	if len(g.Winner) > 0 {
		var w []Faction
		for _, x := range g.Winner {
			if x != f {
				w = append(w, x)
			}
		}
		g.Winner = w
	}

	g.Logf(f, "admin", "%s was removed from the game", f)

	if len(g.Order) == 0 {
		g.Current = ""
		return
	}
	if wasCurrent {
		next := 0
		if idx >= 0 {
			next = idx % len(g.Order)
		}
		g.Current = g.Order[next]
		g.beginTurn()
	}
}

func withoutBuildings(list []Building, f Faction) []Building {
	out := list[:0]
	for _, b := range list {
		if b.Owner != f {
			out = append(out, b)
		}
	}
	return out
}

func withoutTokens(list []Token, f Faction) []Token {
	out := list[:0]
	for _, t := range list {
		if t.Owner != f {
			out = append(out, t)
		}
	}
	return out
}
