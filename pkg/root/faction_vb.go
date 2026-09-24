package root

import (
	"fmt"
	"sort"
)

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
	var ids []string
	for id, it := range p.Items {
		if it.Type == typ && !it.Damaged && it.FaceUp {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return "", false
	}
	sort.Strings(ids) // deterministic: map order would vary between runs
	return ids[0], true
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
				seenItems := map[string]bool{}
				for _, it := range g.Players[f].CraftedItems {
					if seenItems[it] {
						continue // one action per crafted item type
					}
					seenItems[it] = true
					acts = append(acts, Action{
						ID:    actID("vb-aid", string(f), id, it),
						Label: fmt.Sprintf("Aid %s with %s, take %s", f, cardName(id), it),
						Kind:  "vb-aid", Faction: VB, Target: f, Card: id, Item: it,
					})
				}
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
	var ids []string
	for id, it := range p.Items {
		if !it.Damaged && it.FaceUp {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return "", false
	}
	sort.Strings(ids) // deterministic: map order would vary between runs
	return ids[0], true
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
	case "vb-refresh":
		if it, ok := p.Items[a.Item]; ok && !it.FaceUp && !it.Damaged && g.VBRefreshLeft > 0 {
			it.FaceUp = true
			g.placeFaceUpItem(p, it)
			g.VBRefreshLeft--
			g.Logf(VB, "refresh", "Refreshed %s", a.Item)
		}
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
		boots := make([]string, 0, len(p.Items))
		for id, it := range p.Items {
			if it.Type == "boot" && it.FaceUp && !it.Damaged {
				boots = append(boots, id)
			}
		}
		sort.Strings(boots) // deterministic choice
		for _, id := range boots {
			if exhausted >= cost {
				break
			}
			g.exhaustItem(p, id)
			exhausted++
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
		if a.Item != "" {
			tp := g.Players[a.Target]
			if takeStr(&tp.CraftedItems, a.Item) {
				g.giveVBItem(p, a.Item)
				g.Logf(VB, "aid", "Took a %s from %s's crafted items", a.Item, a.Target)
			}
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
		kind, ok := g.removePiece(VB, a.Target, a.Clearing)
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
		var ids []string
		for id, it := range p.Items {
			if it.Type == n && !it.Damaged && it.FaceUp {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			continue
		}
		sort.Strings(ids)
		g.exhaustItem(p, ids[0])
	}
}

func (g *Game) hostileWarriorsAt(c string) bool {
	vb := g.Players[VB]
	if vb == nil {
		return false
	}
	for f, n := range g.Clearings[c].Warriors {
		if f != VB && n > 0 && vb.Relationships[f] == "hostile" {
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
		// 9.2.9.I: to advance one space you must Aid the number of times listed
		// between the current and next space, in the same turn, and each Aid
		// counts toward only one improvement. So the count is per improvement
		// (1, then 2, then 3), resetting when the marker advances.
		p.AidCount[target]++
		need := 0
		switch rel {
		case "indifferent":
			need = 1
		case "amiable":
			need = 2
		case "friendly":
			need = 3
		}
		if p.AidCount[target] < need {
			return
		}
		p.AidCount[target] = 0
		switch rel {
		case "indifferent":
			p.Relationships[target] = "amiable"
			g.Score(VB, 1)
		case "amiable":
			p.Relationships[target] = "friendly"
			g.Score(VB, 2)
		case "friendly":
			p.Relationships[target] = "allied"
			g.Score(VB, 2)
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
		var dmg []string
		for id, it := range p.Items {
			if it.Damaged {
				dmg = append(dmg, id)
			}
		}
		sort.Strings(dmg)
		for _, id := range dmg {
			if repaired >= 3 {
				break
			}
			it := p.Items[id]
			it.Damaged = false
			if it.Zone == "damaged" {
				it.Zone = "satchel"
			}
			repaired++
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
			g.VBStolenThisAction = c
			g.Logf(VB, "special", "Stole a card from %s", a.Target)
		}
	}
}

// legalVBRefresh offers flipping one exhausted item up, while the Birdsong
// refresh allowance remains (9.4.1). The player chooses which items.
func (g *Game) legalVBRefresh() []Action {
	if g.VBRefreshLeft <= 0 {
		return nil
	}
	p := g.Players[VB]
	var ids []string
	for id, it := range p.Items {
		if !it.FaceUp && !it.Damaged {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	acts := make([]Action, 0, len(ids))
	for _, id := range ids {
		acts = append(acts, Action{
			ID: actID("vb-refresh", id), Label: "Refresh " + id, Kind: "vb-refresh", Faction: VB, Item: id,
		})
	}
	return acts
}

// exhaustedItemLeft reports whether a refresh is still possible/needed.
func (g *Game) hasExhaustedItem(p *Player) bool {
	for _, it := range p.Items {
		if !it.FaceUp && !it.Damaged {
			return true
		}
	}
	return false
}

// placeFaceUpItem sends a freshly refreshed tea/coin/bag to its track if there
// is room (9.2.5.I).
func (g *Game) placeFaceUpItem(p *Player, it *ItemState) {
	if (it.Type == "tea" || it.Type == "coin" || it.Type == "bag") && trackCount(p, it.Type) < 3 {
		it.Zone = "track"
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
			it.FaceUp = true
			if (it.Type == "tea" || it.Type == "coin" || it.Type == "bag") && trackCount(p, it.Type) < 3 {
				it.Zone = "track"
			} else {
				it.Zone = "satchel"
			}
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
