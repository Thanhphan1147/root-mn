package root

import "fmt"

// ---- Vagabond ----

func (g *Game) isForest(id string) bool {
	_, ok := GetMap("autumn").Forests[id]
	return ok
}

func (g *Game) forestsAdjacentTo(c string) []string {
	var out []string
	for name, clearings := range GetMap("autumn").Forests {
		for _, x := range clearings {
			if x == c {
				out = append(out, name)
				break
			}
		}
	}
	return out
}

func (g *Game) readyItem(p *Player, typ string) (string, bool) {
	for id, it := range p.Items {
		if it.Type == typ && !it.Damaged && it.FaceUp {
			return id, true
		}
	}
	return "", false
}

func (g *Game) hasItem(p *Player, typ string) bool {
	for _, it := range p.Items {
		if it.Type == typ && !it.Damaged {
			return true
		}
	}
	return false
}

func (g *Game) exhaustItem(p *Player, id string) {
	if it, ok := p.Items[id]; ok {
		it.FaceUp = false
		if it.Zone == "track" {
			it.Zone = "satchel"
		}
	}
}

func (g *Game) legalVBSlip() []Action {
	p := g.Players[VB]
	if g.VBSlipped {
		return nil
	}
	var acts []Action
	dest := []string{}
	if g.isForest(p.Pawn) {
		dest = append(dest, GetMap("autumn").Forests[p.Pawn]...)
	} else {
		dest = append(dest, g.forestsAdjacentTo(p.Pawn)...)
		dest = append(dest, g.Clearings[p.Pawn].Adj...)
	}
	for _, d := range dest {
		acts = append(acts, Action{
			ID: actID("vb-slip", d), Label: "Slip to " + d, Kind: "vb-slip", Faction: VB, To: d,
		})
	}
	return acts
}

func (g *Game) legalVBDaylight() []Action {
	p := g.Players[VB]
	var acts []Action
	acts = append(acts, g.craftActions(p)...)

	// Move
	if _, ok := g.readyItem(p, "boot"); ok {
		var dest []string
		if g.isForest(p.Pawn) {
			dest = append(dest, GetMap("autumn").Forests[p.Pawn]...)
		} else {
			dest = append(dest, g.Clearings[p.Pawn].Adj...)
		}
		for _, d := range dest {
			acts = append(acts, Action{
				ID: actID("vb-move", d), Label: "Move to " + d, Kind: "vb-move", Faction: VB, To: d,
			})
			if !g.isForest(p.Pawn) {
				for _, al := range g.Order {
					if al == VB || p.Relationships[al] != "allied" {
						continue
					}
					if g.Clearings[p.Pawn].Warriors[al] > 0 {
						acts = append(acts, Action{
							ID:    actID("vb-move", d, string(al)),
							Label: fmt.Sprintf("Move to %s with %s warriors", d, al),
							Kind:  "vb-move", Faction: VB, To: d, Ally: al,
						})
					}
				}
			}
		}
	}
	// Battle
	if _, ok := g.readyItem(p, "sword"); ok {
		acts = append(acts, g.battleActions(VB)...)
		if cl := g.Clearings[p.Pawn]; cl != nil {
			for _, al := range g.Order {
				if al == VB || p.Relationships[al] != "allied" || cl.Warriors[al] == 0 {
					continue
				}
				for _, def := range g.enemiesIn(VB, p.Pawn) {
					if def == al {
						continue
					}
					acts = append(acts, Action{
						ID:    actID("vb-battle-ally", string(def), string(al)),
						Label: fmt.Sprintf("Battle %s with %s warriors", def, al),
						Kind:  "vb-battle-ally", Faction: VB, Target: def, Ally: al, Clearing: p.Pawn,
					})
				}
			}
		}
	}
	// Explore
	if cl := g.Clearings[p.Pawn]; cl != nil && cl.Ruin && cl.RuinItem != "" {
		if _, ok := g.readyItem(p, "torch"); ok {
			acts = append(acts, Action{
				ID: actID("vb-explore", p.Pawn), Label: "Explore ruin at " + p.Pawn,
				Kind: "vb-explore", Faction: VB, Clearing: p.Pawn,
			})
		}
	}
	// Aid
	if _, ok := g.readyItemAny(p); ok {
		for _, f := range g.Order {
			if f == VB || g.pieceCount(f, p.Pawn) == 0 {
				continue
			}
			s := g.Clearings[p.Pawn].Suit
			for _, id := range p.Hand {
				if !matches(cardSuit(id), s) {
					continue
				}
				acts = append(acts, Action{
					ID:    actID("vb-aid", string(f), id),
					Label: fmt.Sprintf("Aid %s with %s", f, cardName(id)),
					Kind:  "vb-aid", Faction: VB, Target: f, Card: id,
				})
			}
		}
	}
	// Quest
	if cl := g.Clearings[p.Pawn]; cl != nil {
		for _, qid := range g.QuestAvail {
			q, ok := QuestByID(qid)
			if !ok || q.Suit != cl.Suit {
				continue
			}
			if g.canCompleteQuest(p, q) {
				acts = append(acts, Action{
					ID: actID("vb-quest", qid, "vp"), Label: fmt.Sprintf("Complete quest %s (+VP)", q.Name),
					Kind: "vb-quest", Faction: VB, Quest: qid, Item: "vp",
				})
				acts = append(acts, Action{
					ID: actID("vb-quest", qid, "draw"), Label: fmt.Sprintf("Complete quest %s (draw 2)", q.Name),
					Kind: "vb-quest", Faction: VB, Quest: qid, Item: "draw",
				})
			}
		}
	}
	// Strike
	if _, ok := g.readyItem(p, "crossbow"); ok {
		for _, f := range g.enemiesIn(VB, p.Pawn) {
			acts = append(acts, Action{
				ID: actID("vb-strike", string(f)), Label: fmt.Sprintf("Strike %s at %s", f, p.Pawn),
				Kind: "vb-strike", Faction: VB, Target: f, Clearing: p.Pawn,
			})
		}
	}
	// Repair
	if _, ok := g.readyItem(p, "hammer"); ok {
		for id, it := range p.Items {
			if it.Damaged {
				acts = append(acts, Action{
					ID: actID("vb-repair", id), Label: "Repair " + id, Kind: "vb-repair",
					Faction: VB, Item: id,
				})
			}
		}
	}
	// Special action
	acts = append(acts, g.vbSpecialActions(p)...)
	// Coalition (>=10 VP, dominance card in hand).
	if p.VP >= 10 && p.Coalition == "" {
		for _, id := range p.Hand {
			if c, ok := Card(id); ok && c.Kind == KindDominance {
				for _, t := range g.coalitionTargets() {
					acts = append(acts, Action{
						ID:    actID("coalition", id, string(t)),
						Label: fmt.Sprintf("Coalition with %s (play %s)", t, c.Name),
						Kind:  "coalition", Faction: VB, Card: id, Target: t,
					})
				}
			}
		}
	}
	return acts
}

// coalitionTargets returns the lowest-VP non-Vagabond player(s) not already in
// a coalition.
func (g *Game) coalitionTargets() []Faction {
	low := 1 << 30
	var tied []Faction
	for _, f := range g.Order {
		if f == VB {
			continue
		}
		p := g.Players[f]
		if p.Coalition != "" {
			continue
		}
		if p.VP < low {
			low = p.VP
			tied = []Faction{f}
		} else if p.VP == low {
			tied = append(tied, f)
		}
	}
	return tied
}

func (g *Game) readyItemAny(p *Player) (string, bool) {
	for id, it := range p.Items {
		if !it.Damaged && it.FaceUp {
			return id, true
		}
	}
	return "", false
}

func (g *Game) canCompleteQuest(p *Player, q QuestDef) bool {
	counts := map[string]int{}
	for _, it := range p.Items {
		if !it.Damaged && it.FaceUp {
			counts[it.Type]++
		}
	}
	for _, need := range q.Items {
		if counts[need] <= 0 {
			return false
		}
		counts[need]--
	}
	return true
}

func (g *Game) vbSpecialActions(p *Player) []Action {
	var acts []Action
	if _, ok := g.readyItem(p, "torch"); !ok {
		return nil
	}
	switch p.Character {
	case "thief":
		for _, f := range g.Order {
			if f != VB && g.pieceCount(f, p.Pawn) > 0 && len(g.Players[f].Hand) > 0 {
				acts = append(acts, Action{
					ID: actID("vb-special", string(f)), Label: "Steal from " + string(f),
					Kind: "vb-special", Faction: VB, Target: f,
				})
			}
		}
	case "tinker":
		if cl := g.Clearings[p.Pawn]; cl != nil {
			s := cl.Suit
			for _, c := range g.Discard {
				if matches(cardSuit(c), s) {
					acts = append(acts, Action{
						ID: actID("vb-special", c), Label: "Day Labor: take " + cardName(c),
						Kind: "vb-special", Faction: VB, Card: c,
					})
				}
			}
		}
	case "ranger":
		acts = append(acts, Action{ID: "vb-special|hideout", Label: "Hideout: repair 3, end Daylight", Kind: "vb-special", Faction: VB})
	}
	return acts
}

func (g *Game) applyVB(a Action) error {
	p := g.Players[VB]
	switch a.Kind {
	case "vb-slip":
		p.Pawn = a.To
		g.VBSlipped = true
		g.Logf(VB, "slip", "Slipped to %s", a.To)
	case "vb-move":
		hostile := g.hostileWarriorsAt(a.To)
		cost := 1
		if hostile {
			cost = 2
		}
		exhausted := 0
		for id, it := range p.Items {
			if it.Type == "boot" && it.FaceUp && !it.Damaged && exhausted < cost {
				g.exhaustItem(p, id)
				exhausted++
			}
		}
		if a.Ally != "" {
			from := p.Pawn
			n := g.Clearings[from].Warriors[a.Ally]
			g.addWarrior(a.Ally, from, -n)
			g.addWarrior(a.Ally, a.To, n)
			g.Logf(VB, "move", "Moved with %d %s warrior(s) to %s", n, a.Ally, a.To)
		}
		p.Pawn = a.To
		g.Logf(VB, "move", "Moved to %s (exhausted %d boot)", a.To, exhausted)
	case "vb-explore":
		if id, ok := g.readyItem(p, "torch"); ok {
			g.exhaustItem(p, id)
		}
		cl := g.Clearings[a.Clearing]
		if cl.RuinItem != "" {
			g.giveVBItem(p, cl.RuinItem)
			cl.RuinItem = ""
			cl.Ruin = false
			g.Score(VB, 1)
			g.Logf(VB, "explore", "Explored %s: took item (+1 VP)", a.Clearing)
		}
	case "vb-aid":
		if id, ok := g.readyItemAny(p); ok {
			g.exhaustItem(p, id)
		}
		if takeStr(&p.Hand, a.Card) {
			g.Players[a.Target].Hand = append(g.Players[a.Target].Hand, a.Card)
		}
		g.vbAidRelationship(a.Target)
		g.Logf(VB, "aid", "Aided %s with %s", a.Target, cardName(a.Card))
	case "vb-quest":
		q, _ := QuestByID(a.Quest)
		g.exhaustQuestItems(p, q)
		p.Quests = append(p.Quests, a.Quest)
		takeStr(&g.QuestAvail, a.Quest)
		if len(g.QuestDeck) > 0 {
			g.QuestAvail = append(g.QuestAvail, g.QuestDeck[0])
			g.QuestDeck = g.QuestDeck[1:]
		}
		if a.Item == "draw" {
			g.drawCards(VB, 2)
			g.Logf(VB, "quest", "Completed %s: drew 2 cards", q.Name)
		} else {
			n := 0
			for _, qid := range p.Quests {
				if qq, ok := QuestByID(qid); ok && qq.Suit == q.Suit {
					n++
				}
			}
			g.Score(VB, n)
			g.Logf(VB, "quest", "Completed %s: +%d VP", q.Name, n)
		}
	case "vb-strike":
		kind, ok := g.removePiece(a.Target, a.Clearing)
		if ok && (kind == "building" || kind == "token") {
			g.Score(VB, 1)
		}
		if id, ok := g.readyItem(p, "crossbow"); ok {
			g.exhaustItem(p, id)
		}
		g.Logf(VB, "strike", "Struck %s at %s", a.Target, a.Clearing)
	case "vb-repair":
		if it, ok := p.Items[a.Item]; ok {
			it.Damaged = false
			if it.Zone == "damaged" {
				it.Zone = "satchel"
			}
		}
		if id, ok := g.readyItem(p, "hammer"); ok {
			g.exhaustItem(p, id)
		}
		g.Logf(VB, "repair", "Repaired %s", a.Item)
	case "vb-special":
		g.vbSpecial(a)
	case "vb-battle-ally":
		return g.startBattle(Action{Kind: "battle", Faction: VB, Clearing: a.Clearing, Target: a.Target, Ally: a.Ally})
	case "coalition":
		takeStr(&p.Hand, a.Card)
		p.Crafted = append(p.Crafted, a.Card)
		p.Coalition = a.Target
		g.Logf(VB, "coalition", "Formed a coalition with %s; the Vagabond no longer scores", a.Target)
	}
	return nil
}

func (g *Game) exhaustQuestItems(p *Player, q QuestDef) {
	need := append([]string{}, q.Items...)
	for _, n := range need {
		for id, it := range p.Items {
			if it.Type == n && !it.Damaged && it.FaceUp {
				g.exhaustItem(p, id)
				break
			}
		}
	}
}

func (g *Game) hostileWarriorsAt(c string) bool {
	for f, n := range g.Clearings[c].Warriors {
		if f != VB && n > 0 && g.Players[VB].Relationships[f] == "hostile" {
			return true
		}
	}
	return false
}

func (g *Game) vbAidRelationship(target Faction) {
	p := g.Players[VB]
	rel := p.Relationships[target]
	switch rel {
	case "allied":
		g.Score(VB, 2)
	case "hostile":
		// no relationship change
	default:
		p.AidCount[target]++
		switch rel {
		case "indifferent":
			if p.AidCount[target] >= 1 {
				p.Relationships[target] = "amiable"
				g.Score(VB, 1)
			}
		case "amiable":
			if p.AidCount[target] >= 2 {
				p.Relationships[target] = "friendly"
				g.Score(VB, 2)
			}
		case "friendly":
			if p.AidCount[target] >= 3 {
				p.Relationships[target] = "allied"
				g.Score(VB, 2)
			}
		}
	}
}

func (g *Game) vbSpecial(a Action) {
	p := g.Players[VB]
	if a.ID == "vb-special|hideout" {
		if id, ok := g.readyItem(p, "torch"); ok {
			g.exhaustItem(p, id)
		}
		repaired := 0
		for _, it := range p.Items {
			if it.Damaged && repaired < 3 {
				it.Damaged = false
				if it.Zone == "damaged" {
					it.Zone = "satchel"
				}
				repaired++
			}
		}
		g.Logf(VB, "special", "Hideout: repaired %d items; Daylight ends", repaired)
		g.beginEvening()
		return
	}
	if a.Card != "" {
		// Tinker Day Labor
		if id, ok := g.readyItem(p, "torch"); ok {
			g.exhaustItem(p, id)
		}
		takeStr(&g.Discard, a.Card)
		p.Hand = append(p.Hand, a.Card)
		g.Logf(VB, "special", "Day Labor: took %s", cardName(a.Card))
		return
	}
	if a.Target != "" {
		// Thief Steal
		if id, ok := g.readyItem(p, "torch"); ok {
			g.exhaustItem(p, id)
		}
		tp := g.Players[a.Target]
		if len(tp.Hand) > 0 {
			c := tp.Hand[0]
			tp.Hand = tp.Hand[1:]
			p.Hand = append(p.Hand, c)
			g.Logf(VB, "special", "Stole a card from %s", a.Target)
		}
	}
}

// vbRefresh flips face-up 3 + 2*tea exhausted items.
func (g *Game) vbRefresh(p *Player) {
	tea := trackCount(p, "tea")
	n := 3 + 2*tea
	for _, it := range p.Items {
		if n <= 0 {
			break
		}
		if !it.FaceUp {
			it.FaceUp = true
			n--
		}
	}
}

// vbEveningRest repairs damaged items if the pawn is in a forest.
func (g *Game) vbEveningRest(p *Player) {
	if !g.isForest(p.Pawn) {
		return
	}
	for _, it := range p.Items {
		if it.Damaged {
			it.Damaged = false
			if it.Zone == "damaged" {
				it.Zone = "satchel"
			}
			it.FaceUp = true
		}
	}
	g.Logf(VB, "rest", "Evening's Rest in forest")
}

// vbHostilityOnWarriorRemoval is called whenever any warrior is removed.
func (g *Game) vbHostilityOnWarriorRemoval(owner Faction) {
	p := g.Players[VB]
	if p == nil || owner == VB {
		return
	}
	if _, ok := p.Relationships[owner]; ok {
		p.Relationships[owner] = "hostile"
	}
}
