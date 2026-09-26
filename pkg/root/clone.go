package root

// Clone returns a deep copy of the game. It is a manual copy rather than a JSON
// round trip, so it is cheap enough to use inside a search.
func (g *Game) Clone() *Game { return g.clone(true) }

// CloneForSearch is Clone without the action logs, which a search never reads.
// It is the copy to use when expanding a search tree.
func (g *Game) CloneForSearch() *Game { return g.clone(false) }

func (g *Game) clone(withLogs bool) *Game {
	if g == nil {
		return nil
	}
	out := *g // scalars
	out.Clearings = make(map[string]*Clearing, len(g.Clearings))
	for k, c := range g.Clearings {
		out.Clearings[k] = c.clone()
	}
	out.Players = make(map[Faction]*Player, len(g.Players))
	for k, p := range g.Players {
		out.Players[k] = p.clone()
	}
	out.Order = cloneSlice(g.Order)
	out.Deck = cloneSlice(g.Deck)
	out.Discard = cloneSlice(g.Discard)
	out.AvailableDominance = cloneSlice(g.AvailableDominance)
	out.QuestDeck = cloneSlice(g.QuestDeck)
	out.QuestAvail = cloneSlice(g.QuestAvail)
	out.Winner = cloneSlice(g.Winner)
	out.ItemSupply = cloneMap(g.ItemSupply)
	out.SetupPlaced = cloneMap(g.SetupPlaced)
	out.FH = cloneSlice(g.FH)
	out.DecreeQueue = cloneSlice(g.DecreeQueue)
	out.NextShuffle = cloneSlice(g.NextShuffle)
	if g.Pending != nil {
		p := *g.Pending
		p.Context = cloneMap(g.Pending.Context)
		out.Pending = &p
	}
	if g.Battle != nil {
		b := *g.Battle
		b.FH = cloneSlice(g.Battle.FH)
		out.Battle = &b
	}
	if withLogs {
		out.Log = cloneSlice(g.Log)
		out.RMNLog = cloneSlice(g.RMNLog)
	} else {
		out.Log = nil
		out.RMNLog = nil
	}
	return &out
}

func (c *Clearing) clone() *Clearing {
	out := *c
	out.Adj = cloneSlice(c.Adj)
	out.Warriors = cloneMap(c.Warriors)
	out.Buildings = cloneSlice(c.Buildings)
	out.Tokens = cloneSlice(c.Tokens)
	return &out
}

func (p *Player) clone() *Player {
	out := *p
	out.Hand = cloneSlice(p.Hand)
	out.Crafted = cloneSlice(p.Crafted)
	out.CraftedItems = cloneSlice(p.CraftedItems)
	out.UsedThisTurn = cloneMap(p.UsedThisTurn)
	out.Revealed = cloneMap(p.Revealed)
	out.Viziers = cloneSlice(p.Viziers)
	out.RetiredLeaders = cloneSlice(p.RetiredLeaders)
	out.Supporters = cloneSlice(p.Supporters)
	out.Bases = cloneMap(p.Bases)
	out.Relationships = cloneMap(p.Relationships)
	out.AidCount = cloneMap(p.AidCount)
	out.Quests = cloneSlice(p.Quests)
	out.Decree = make(map[string][]string, len(p.Decree))
	for k, v := range p.Decree {
		out.Decree[k] = cloneSlice(v)
	}
	out.Items = make(map[string]*ItemState, len(p.Items))
	for k, it := range p.Items {
		if it == nil {
			out.Items[k] = nil
			continue
		}
		cp := *it
		out.Items[k] = &cp
	}
	return &out
}

func cloneSlice[T any](s []T) []T {
	if s == nil {
		return nil
	}
	out := make([]T, len(s))
	copy(out, s)
	return out
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
