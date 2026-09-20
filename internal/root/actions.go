package root

import (
	"fmt"
	"sort"
)

// Action is a fully-specified legal move. The UI posts the ID back; the engine
// re-derives legal actions and matches by ID.
type Action struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Kind      string   `json:"kind"`
	Faction   Faction  `json:"faction"`
	Clearing  string   `json:"clearing,omitempty"`
	From      string   `json:"from,omitempty"`
	To        string   `json:"to,omitempty"`
	Card      string   `json:"card,omitempty"`
	Cards     []string `json:"cards,omitempty"`
	Building  string   `json:"building,omitempty"`
	Target    Faction  `json:"target,omitempty"`
	Ally      Faction  `json:"ally,omitempty"`
	Column    string   `json:"column,omitempty"`
	Quest     string   `json:"quest,omitempty"`
	Item      string   `json:"item,omitempty"`
	Items     []string `json:"items,omitempty"`
	Amount    int      `json:"amount,omitempty"`
	Piece     string   `json:"piece,omitempty"`
	Character string   `json:"character,omitempty"`
	Leader    string   `json:"leader,omitempty"`
}

func actID(kind string, parts ...string) string {
	id := kind
	for _, p := range parts {
		id += "|" + p
	}
	return id
}

// LegalActions returns every legal action for the current state.
func (g *Game) LegalActions() []Action {
	if g.SetupMode {
		return g.legalSetup()
	}
	if len(g.Winner) > 0 {
		return nil
	}
	if g.Pending != nil {
		return g.legalPending()
	}
	f := g.Current
	var acts []Action
	switch g.Phase {
	case "B":
		acts = g.legalBirdsong(f)
	case "D":
		acts = g.legalDaylight(f)
	case "E":
		acts = g.legalEvening(f)
	}
	if g.passAllowed() {
		acts = append(acts, Action{ID: "pass", Label: "End " + phaseName(g.Phase), Kind: "pass", Faction: f})
	}
	return acts
}

// passAllowed reports whether the generic end-of-phase action is legal. The
// Eyrie may not end Daylight until the Decree is fully resolved, and may not
// end Birdsong before adding cards to the Decree.
func (g *Game) passAllowed() bool {
	if g.SetupMode || g.Pending != nil || len(g.Winner) > 0 {
		return false
	}
	if g.Current == ED {
		if g.EDNeedsLeader {
			return false
		}
		switch g.Phase {
		case "B":
			// Must add 1-2 cards; if the hand is empty (Emergency Orders could
			// not draw), allow proceeding so the game cannot deadlock.
			return g.EDAdded >= 1 || len(g.Players[ED].Hand) == 0
		case "D":
			if g.EDDayStage == "craft" {
				return false
			}
			return len(g.DecreeQueue) == 0
		}
	}
	return true
}

func phaseName(p string) string {
	switch p {
	case "B":
		return "Birdsong"
	case "D":
		return "Daylight"
	case "E":
		return "Evening"
	}
	return p
}

func (g *Game) legalPending() []Action {
	p := g.Pending
	switch p.Kind {
	case PendingDiscard:
		pl := g.Players[p.Player]
		var acts []Action
		for _, c := range pl.Hand {
			acts = append(acts, Action{
				ID: actID("discard", c), Label: "Discard " + cardName(c), Kind: "discard",
				Faction: p.Player, Card: c,
			})
		}
		return acts
	case PendingBattleHits:
		return g.legalBattleHits()
	case PendingBattleAmbush:
		return g.legalBattleAmbush()
	case PendingBattleEffects:
		return g.legalBattleEffects()
	case PendingFieldHospitals:
		return g.legalFieldHospitals()
	}
	return nil
}

func (g *Game) legalFieldHospitals() []Action {
	var acts []Action
	seen := map[string]bool{}
	for _, rec := range g.FH {
		if seen[rec.Clearing] {
			continue
		}
		seen[rec.Clearing] = true
		s := g.Clearings[rec.Clearing].Suit
		for _, id := range g.Players[MC].Hand {
			if matches(cardSuit(id), s) {
				acts = append(acts, Action{
					ID:    actID("field-hospitals", rec.Clearing, id),
					Label: fmt.Sprintf("Field Hospitals: spend %s → save warriors at %s", cardName(id), rec.Clearing),
					Kind:  "field-hospitals", Faction: MC, Clearing: rec.Clearing, Card: id,
				})
			}
		}
	}
	acts = append(acts, Action{ID: "battle-skip", Label: "Decline Field Hospitals", Kind: "battle-skip", Faction: MC})
	return acts
}

func (g *Game) legalBattleAmbush() []Action {
	b := g.Battle
	p := g.Players[g.Pending.Player]
	var acts []Action
	s := g.Clearings[b.Clearing].Suit
	for _, id := range p.Hand {
		if c, ok := Card(id); ok && c.Kind == KindAmbush && matches(c.Suit, s) {
			kind := "battle-ambush"
			label := "Play ambush " + cardName(id)
			if b.Stage == StageAtkFoil {
				kind = "battle-foil"
				label = "Foil with " + cardName(id)
			}
			acts = append(acts, Action{ID: actID(kind, id), Label: label, Kind: kind, Faction: p.Faction, Card: id})
		}
	}
	label := "No ambush"
	acts = append(acts, Action{ID: "battle-skip", Label: label, Kind: "battle-skip", Faction: p.Faction})
	return acts
}

func (g *Game) legalBattleEffects() []Action {
	b := g.Battle
	f := g.Pending.Player
	p := g.Players[f]
	var acts []Action
	for _, id := range p.Crafted {
		c, ok := Card(id)
		if !ok || c.Pers != "once" {
			continue
		}
		relevant := false
		switch c.Effect {
		case "brutal-tactics":
			relevant = f == b.Attacker
		case "armorers", "sappers":
			relevant = f == b.Defender
		}
		if relevant {
			acts = append(acts, Action{ID: actID("battle-effect", id), Label: "Play " + c.Name, Kind: "battle-effect", Faction: f, Card: id})
		}
	}
	acts = append(acts, Action{ID: "battle-skip", Label: "Skip", Kind: "battle-skip", Faction: f})
	return acts
}

// ---- Birdsong ----

func (g *Game) legalBirdsong(f Faction) []Action {
	var acts []Action
	switch f {
	case ED:
		acts = g.legalEDDecreeAdd()
	case WA:
		acts = g.legalWABirdsong()
	case VB:
		acts = g.legalVBSlip()
	}
	acts = append(acts, g.persistentBirdsongActions(f)...)
	return acts
}

// ---- Daylight ----

func (g *Game) legalDaylight(f Faction) []Action {
	var acts []Action
	switch f {
	case MC:
		acts = g.legalMCDaylight()
	case ED:
		acts = g.legalEDDaylight()
	case WA:
		acts = g.legalWADaylight()
	case VB:
		acts = g.legalVBDaylight()
	}
	acts = append(acts, g.persistentDaylightActions(f)...)
	acts = append(acts, g.takeDominanceActions(f)...)
	if g.ExtraBattle {
		acts = append(acts, g.battleActions(f)...)
	}
	return acts
}

// takeDominanceActions lets a player spend a matching card to take an
// available (discarded/spent) Dominance card into hand.
func (g *Game) takeDominanceActions(f Faction) []Action {
	p := g.Players[f]
	var acts []Action
	for _, d := range g.AvailableDominance {
		dc, ok := Card(d)
		if !ok {
			continue
		}
		for _, c := range p.Hand {
			if dc.Effect == "dom-bird" {
				if cardSuit(c) != Bird {
					continue
				}
			} else if !matches(cardSuit(c), dc.Suit) {
				continue
			}
			acts = append(acts, Action{
				ID:    actID("take-dominance", c, d),
				Label: fmt.Sprintf("Take %s (spend %s)", dc.Name, cardName(c)),
				Kind:  "take-dominance", Faction: f, Card: c, Item: d,
			})
		}
	}
	return acts
}

// ---- Evening ----

func (g *Game) legalEvening(f Faction) []Action {
	var acts []Action
	switch f {
	case WA:
		acts = g.legalWAEvening()
	}
	acts = append(acts, g.persistentEveningActions(f)...)
	return acts
}

// craftableCards returns card ids in p's hand that p can craft now.
func (g *Game) craftableCards(p *Player) []string {
	var out []string
	for _, id := range p.Hand {
		c, ok := Card(id)
		if !ok || c.Kind == KindAmbush || c.Kind == KindDominance {
			continue
		}
		if c.Kind == KindItem && g.ItemSupply[c.Item] <= 0 {
			continue
		}
		if c.Pers != "" && hasPersistent(p, c.Effect) {
			continue
		}
		if g.canCraft(p, c) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func hasPersistent(p *Player, effect string) bool {
	for _, id := range p.Crafted {
		if c, ok := Card(id); ok && c.Effect == effect {
			return true
		}
	}
	return false
}

// pieceSuits returns the multiset of available crafting-piece suits for p.
func (g *Game) pieceSuits(p *Player) []Suit {
	var out []Suit
	switch p.Faction {
	case MC:
		for _, c := range g.clearingsSorted() {
			for i := 0; i < g.buildingsOf(MC, c, "workshop"); i++ {
				key := "ws:" + c + ":" + itoa(i)
				if !p.UsedThisTurn[key] {
					out = append(out, g.Clearings[c].Suit)
				}
			}
		}
	case ED:
		for _, c := range g.clearingsSorted() {
			if g.buildingsOf(ED, c, "roost") > 0 && !p.UsedThisTurn["roost:"+c] {
				out = append(out, g.Clearings[c].Suit)
			}
		}
	case WA:
		for _, c := range g.clearingsSorted() {
			if g.Clearings[c].Sympathy == WA && !p.UsedThisTurn["sym:"+c] {
				out = append(out, g.Clearings[c].Suit)
			}
		}
	case VB:
		for id, it := range p.Items {
			if it.Type == "hammer" && it.Zone == "satchel" && it.FaceUp && !it.Damaged {
				// VB hammer suit = pawn's clearing suit
				if cl := g.Clearings[p.Pawn]; cl != nil {
					out = append(out, cl.Suit)
				}
				_ = id
			}
		}
	}
	return out
}

// canCraft checks the icon multiset against available piece suits.
func (g *Game) canCraft(p *Player, c *CardDef) bool {
	avail := g.pieceSuits(p)
	need := append([]Suit{}, c.Cost...)
	anyNeed := c.Any
	if len(need)+anyNeed > len(avail) {
		return false
	}
	used := make([]bool, len(avail))
	for _, s := range need {
		found := false
		for i, a := range avail {
			if used[i] {
				continue
			}
			if matches(a, s) {
				used[i] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	// "any" icons take any remaining.
	return len(avail) >= len(need)+anyNeed
}

func (g *Game) craftActions(p *Player) []Action {
	var acts []Action
	for _, id := range g.craftableCards(p) {
		acts = append(acts, Action{
			ID: actID("craft", id), Label: "Craft " + cardName(id), Kind: "craft",
			Faction: p.Faction, Card: id,
		})
	}
	return acts
}

// spendBirdActions lets MC spend bird cards for extra actions.
func (g *Game) spendBirdActions(p *Player) []Action {
	if p.Faction != MC {
		return nil
	}
	var acts []Action
	for _, id := range p.Hand {
		if cardSuit(id) == Bird {
			acts = append(acts, Action{
				ID: actID("spend-bird", id), Label: "Spend bird " + cardName(id) + " (+1 action)",
				Kind: "spend-bird", Faction: MC, Card: id,
			})
		}
	}
	return acts
}

// battleActions enumerates battles the faction can initiate.
func (g *Game) battleActions(f Faction) []Action {
	var acts []Action
	for _, c := range g.clearingsSorted() {
		cl := g.Clearings[c]
		if f == VB {
			if g.Players[VB].Pawn != c {
				continue
			}
		} else if cl.Warriors[f] == 0 {
			continue
		}
		for _, def := range g.enemiesIn(f, c) {
			acts = append(acts, Action{
				ID:    actID("battle", c, string(def)),
				Label: fmt.Sprintf("Battle %s in %s", def, c), Kind: "battle",
				Faction: f, Clearing: c, Target: def,
			})
		}
	}
	return acts
}

func (g *Game) enemiesIn(f Faction, c string) []Faction {
	var out []Faction
	for _, other := range g.Order {
		if other == f {
			continue
		}
		if g.areEnemies(f, other) && g.pieceCount(other, c) > 0 {
			out = append(out, other)
		}
	}
	return out
}

func (g *Game) areEnemies(a, b Faction) bool {
	pa, pb := g.Players[a], g.Players[b]
	if pa != nil && pa.Coalition == b {
		return false
	}
	if pb != nil && pb.Coalition == a {
		return false
	}
	return true
}

// moveActions enumerates standard moves.
func (g *Game) moveActions(f Faction) []Action {
	var acts []Action
	if f == VB {
		// VB moves pawn by item; handled in VB file.
		return nil
	}
	for _, from := range g.clearingsSorted() {
		n := g.Clearings[from].Warriors[f]
		if n == 0 {
			continue
		}
		for _, to := range g.Clearings[from].Adj {
			if !g.Relaxed && !g.Rules(f, from) && !g.Rules(f, to) {
				continue
			}
			// Any number from 1 to N may move.
			for q := 1; q <= n; q++ {
				acts = append(acts, Action{
					ID:    actID("move", from, to, itoa(q)),
					Label: fmt.Sprintf("Move %d %s → %s", q, from, to), Kind: "move",
					Faction: f, From: from, To: to, Amount: q,
				})
			}
		}
	}
	return acts
}

func cardName(id string) string {
	if c, ok := Card(id); ok {
		return fmt.Sprintf("%s (%s)", c.Name, id)
	}
	if id == "VIZIER" {
		return "Loyal Vizier"
	}
	return id
}
