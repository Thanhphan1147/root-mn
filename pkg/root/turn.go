package root

// DecreeItem is one card queued for Eyrie decree resolution.
type DecreeItem struct {
	Column string
	Card   string
}

// beginTurn starts the Birdsong of the current player.
func (g *Game) beginTurn() {
	g.Phase = "B"
	g.ActionsLeft = 0
	g.MarchMovesLeft = 0
	g.RecruitedThisTurn = false
	g.BirdSpent = 0
	g.MilitaryOpsLeft = 0
	g.DecreeQueue = nil
	g.EDAdded = 0
	g.EDBirdAdded = false
	g.WAEveningDrawn = false
	g.ExtraBattle = false
	g.ExtraMove = false
	g.TaxUsed = false
	g.VBSlipped = false
	for _, p := range g.Players {
		p.UsedThisTurn = map[string]bool{}
		p.AidCount = map[Faction]int{}
		p.Revealed = map[Faction]bool{}
	}
	f := g.Current
	p := g.Players[f]
	switch f {
	case MC:
		g.birdsongWood(p)
	case ED:
		if len(p.Hand) == 0 {
			g.drawCards(ED, 1)
			g.Logf(ED, "birdsong", "Emergency Orders: drew 1 card (empty hand)")
		}
		if g.totalBuildings(ED, "roost") == 0 {
			g.newRoost(p)
		}
	case VB:
		// 9.4.1 Refresh: the Vagabond flips up 3 + 2 per tea (tea on the track
		// at the start, so tea flipped during this step does not count). The
		// player chooses which items, so this becomes interactive actions.
		g.VBRefreshLeft = 3 + 2*trackCount(p, "tea")
	}
	g.applyBetterBurrowBank(f)
	g.checkDominanceWin(f)
	g.Logf(f, "birdsong", "%s begins Birdsong", f)
}

// birdsongWood places one wood per sawmill in its clearing.
func (g *Game) birdsongWood(p *Player) {
	type src struct {
		clearing string
		count    int
	}
	var srcs []src
	for _, c := range g.clearingsSorted() {
		n := g.buildingsOf(MC, c, "sawmill")
		if n > 0 {
			srcs = append(srcs, src{c, n})
		}
	}
	for _, s := range srcs {
		for i := 0; i < s.count; i++ {
			if p.WoodSupply <= 0 {
				return
			}
			g.Clearings[s.clearing].Wood++
			p.WoodSupply--
		}
	}
	if len(srcs) > 0 {
		g.Logf(MC, "birdsong", "Placed wood at %d sawmill clearing(s)", len(srcs))
	}
}

// newRoost implements A New Roost: if the Eyrie has no roosts, place a roost and
// three warriors in the clearing with the fewest warriors where all of those
// pieces can be placed (i.e. the roost has a free building slot).
func (g *Game) newRoost(p *Player) {
	if g.totalBuildings(ED, "roost") >= 7 {
		return
	}
	best := ""
	bestN := 1 << 30
	for _, c := range g.clearingsSorted() {
		cl := g.Clearings[c]
		if cl.FreeSlots() <= 0 {
			continue
		}
		n := 0
		for _, w := range cl.Warriors {
			n += w
		}
		if n < bestN {
			bestN = n
			best = c
		}
	}
	if best == "" {
		return
	}
	g.Clearings[best].Buildings = append(g.Clearings[best].Buildings, Building{ED, "roost"})
	g.addWarrior(ED, best, 3)
	g.Logf(ED, "birdsong", "A New Roost: placed roost + 3 warriors at %s", best)
}

// Pass advances to the next phase (or ends the turn).
func (g *Game) Pass() {
	switch g.Phase {
	case "B":
		g.beginDaylight()
	case "D":
		g.beginEvening()
	case "E":
		if g.Current == WA && !g.WAEveningDrawn {
			g.waEveningDraw(g.Players[WA])
			g.WAEveningDrawn = true
			if g.Pending == nil {
				g.endTurn()
			}
			return
		}
		g.endTurn()
	}
}

func (g *Game) beginDaylight() {
	g.Phase = "D"
	switch g.Current {
	case MC:
		g.ActionsLeft = 3
		g.MarchMovesLeft = 0
		g.MCDayStage = "craft"
	case ED:
		g.buildDecreeQueue()
		g.EDDayStage = "craft"
	}
	g.grantCommandWarren(g.Current)
	g.Logf(g.Current, "daylight", "%s begins Daylight", g.Current)
}

func (g *Game) buildDecreeQueue() {
	p := g.Players[ED]
	g.DecreeQueue = nil
	for _, col := range DecreeColumns {
		for _, card := range p.Decree[col] {
			g.DecreeQueue = append(g.DecreeQueue, DecreeItem{col, card})
		}
	}
}

func (g *Game) beginEvening() {
	g.Phase = "E"
	p := g.Players[g.Current]
	switch g.Current {
	case MC:
		draw := 1 + g.mcDrawBonus(p)
		g.drawCards(MC, draw)
		g.Logf(MC, "evening", "Drew %d card(s)", draw)
		g.queueDiscard(p)
	case ED:
		n := g.totalBuildings(ED, "roost")
		if n > 7 {
			n = 7
		}
		vp := EDRoostVP[n]
		g.Score(ED, vp)
		draw := 1 + EDRoostDrawBonus[n]
		g.drawCards(ED, draw)
		g.Logf(ED, "evening", "Scored %d VP for %d roosts; drew %d", vp, n, draw)
		g.queueDiscard(p)
	case WA:
		g.WAEveningDrawn = false
		g.MilitaryOpsLeft = p.Officers
		g.Logf(WA, "evening", "Military Operations available: %d", g.MilitaryOpsLeft)
		if g.MilitaryOpsLeft == 0 {
			g.waEveningDraw(p)
			g.WAEveningDrawn = true
		}
	case VB:
		g.vbEveningRest(p)
		draw := 1 + trackCount(p, "coin")
		g.drawCards(VB, draw)
		g.Logf(VB, "evening", "Drew %d card(s)", draw)
		g.queueDiscard(p)
		g.vbCapacityCheck(p)
	}
	g.grantCobbler(g.Current)
}

func (g *Game) mcDrawBonus(p *Player) int {
	recruiters := 6 - p.Recruiters
	d := 0
	if recruiters >= 3 {
		d++
	}
	if recruiters >= 5 {
		d++
	}
	return d
}

// queueDiscard sets a pending discard if the player is over the hand limit.
func (g *Game) queueDiscard(p *Player) {
	if len(p.Hand) > 5 {
		g.Pending = &Pending{Kind: PendingDiscard, Player: p.Faction, Remaining: len(p.Hand) - 5}
	}
}

func (g *Game) endTurn() {
	g.Pending = nil
	idx := -1
	for i, f := range g.Order {
		if f == g.Current {
			idx = i
			break
		}
	}
	if idx < 0 || idx == len(g.Order)-1 {
		g.Round++
		g.Current = g.Order[0]
	} else {
		g.Current = g.Order[idx+1]
	}
	// Win check at start of a player's turn (dominance checked in beginTurn via checkDominance).
	g.beginTurn()
}

// checkDominanceWin is evaluated at the start of a player's Birdsong.
func (g *Game) checkDominanceWin(f Faction) bool {
	p := g.Players[f]
	for _, id := range p.Crafted {
		c, ok := Card(id)
		if !ok || c.Kind != KindDominance {
			continue
		}
		switch c.Effect {
		case "dom-fox":
			if g.ruledSuit(Fox) >= 3 {
				g.Winner = []Faction{f}
				return true
			}
		case "dom-rabbit":
			if g.ruledSuit(Rabbit) >= 3 {
				g.Winner = []Faction{f}
				return true
			}
		case "dom-mouse":
			if g.ruledSuit(Mouse) >= 3 {
				g.Winner = []Faction{f}
				return true
			}
		case "dom-bird":
			m := GetMap("autumn")
			for _, pair := range m.Corners {
				if g.Rules(f, pair[0]) && g.Rules(f, pair[1]) {
					g.Winner = []Faction{f}
					return true
				}
			}
		}
	}
	return false
}

func (g *Game) ruledSuit(s Suit) int {
	n := 0
	for c, cl := range g.Clearings {
		if cl.Suit == s && g.Rules(g.Current, c) {
			n++
		}
	}
	return n
}
