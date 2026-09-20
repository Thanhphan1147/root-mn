package rmn

import (
	"fmt"
	"strings"
)

// FoldResult reports non-fatal problems encountered while folding an event.
type FoldResult struct {
	Errors   []string
	Warnings []string
}

func (r *FoldResult) errf(format string, a ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, a...))
}
func (r *FoldResult) warnf(format string, a ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, a...))
}

// Fold applies one event to the state. It is total: it never panics and records
// problems in FoldResult.
func Fold(s *State, ev *Event) FoldResult {
	var res FoldResult
	s.Seq = ev.Seq
	if ev.Round > s.Round || (ev.Round == s.Round && phaseRank(ev.Phase) > phaseRank(s.Phase)) {
		s.Round, s.Phase = ev.Round, ev.Phase
	}
	switch ev.Intent {
	case "phase":
		if p, ok := ev.Str("phase"); ok {
			if p == "D" && s.Phase != "D" {
				s.ActionsUsed, s.BirdSpent = 0, 0
			}
			s.Phase = p
		}
	case "turn":
		if w, ok := ev.Str("who"); ok {
			s.Current = w
		}
	case "pass":
		// no-op
	case "place":
		foldPlace(s, ev, &res)
	case "move":
		foldMove(s, ev, &res)
	case "remove", "casualties":
		foldRemove(s, ev, &res)
	case "exile":
		foldRemove(s, ev, &res)
	case "build", "C:build", "E:build", "D:build":
		foldBuild(s, ev, &res)
	case "recruit", "C:recruit", "E:recruit", "D:recruit", "P:recruit":
		foldRecruit(s, ev, &res)
	case "battle", "E:battle", "L:crusade":
		s.ActionsUsed++
	case "ambush":
		// hits are explicit; no state change beyond the card leaving hand
		foldDiscardCard(s, ev, &res, "who", "card")
	case "draw":
		foldDraw(s, ev, &res)
	case "discard":
		foldDiscard(s, ev, &res)
	case "give":
		foldGive(s, ev, &res)
	case "craft":
		foldCraft(s, ev, &res)
	case "reveal", "L:reveal-outcast", "D:reveal-sway":
		// informational
	case "shuffle":
		if order, ok := ev.OutcomeList("order"); ok {
			s.Deck = nil
			for _, v := range order {
				s.Deck = append(s.Deck, v.Str)
			}
		}
	case "deal":
		foldDeal(s, ev, &res)
	case "assign-ruins":
		foldAssignRuins(s, ev, &res)
	case "score", "A:off-turn-score", "P:extortion-score":
		foldScore(s, ev, &res)
	case "lose-vp":
		foldLoseVP(s, ev, &res)
	case "move-vp-token":
		foldMoveVP(s, ev, &res)
	case "win":
		s.Winner = nil
		if v, ok := ev.Operands["factions"]; ok {
			for _, e := range v.List {
				s.Winner = append(s.Winner, e.Str)
			}
		}
	case "spend-bird":
		foldDiscardCard(s, ev, &res, "who", "card")
		s.BirdSpent++
	case "move-item", "V:move-item":
		foldMoveItem(s, ev, &res)
	case "set-item-state", "V:set-item-state":
		foldSetItemState(s, ev, &res)
	case "outrage", "A:outrage":
		foldOutrage(s, ev, &res)
	case "field-hospitals", "C:field-hospitals":
		foldFieldHospitals(s, ev, &res)
	case "commit":
		foldCommit(s, ev, &res)
	case "reveal-commit":
		foldRevealCommit(s, ev, &res)
	case "setup":
		// handled via specific setup intents
	case "C:birdsong-wood":
		foldBirdsongWood(s, ev, &res)
	case "C:overwork":
		foldOverwork(s, ev, &res)
	case "E:decree-add":
		foldDecreeAdd(s, ev, &res)
	case "E:discard-decree":
		foldDiscardDecree(s, ev, &res)
	case "E:appoint-leader":
		foldAppointLeader(s, ev, &res)
	case "E:add-vizier":
		foldAddVizier(s, ev, &res)
	case "E:score-roosts":
		foldScore(s, ev, &res)
	case "E:turmoil":
		// a lose-vp event follows
	case "A:mobilize":
		foldMobilize(s, ev, &res)
	case "A:spend-supporters":
		foldSpendSupporters(s, ev, &res)
	case "A:place-sympathy":
		foldPlaceSympathy(s, ev, &res)
	case "A:revolt":
		foldRevolt(s, ev, &res)
	case "A:train":
		foldDiscardCard(s, ev, &res, "", "card")
		s.Faction(ev.Actor).Officers++
	case "V:choose-character":
		foldChooseCharacter(s, ev, &res)
	case "V:move-pawn":
		foldMovePawn(s, ev, &res)
	case "V:aid":
		foldAid(s, ev, &res)
	case "V:explore":
		foldExplore(s, ev, &res)
	case "V:take-quest":
		foldTakeQuest(s, ev, &res)
	case "V:complete-quest":
		foldCompleteQuest(s, ev, &res)
	case "V:refresh", "V:repair", "V:rest":
		foldRefreshItems(s, ev, &res)
	case "V:exhaust":
		foldExhaustItems(s, ev, &res, true)
	case "V:damage":
		foldDamageItems(s, ev, &res)
	case "V:relationship":
		foldRelationship(s, ev, &res)
	case "V:coalition":
		foldCoalition(s, ev, &res)
	default:
		res.warnf("intent %s not folded (extension or informational)", ev.Intent)
	}
	return res
}

func phaseRank(p string) int {
	switch p {
	case "S":
		return 0
	case "B":
		return 1
	case "D":
		return 2
	case "E":
		return 3
	}
	return 0
}

// expandUnits flattens a unit group into individual refs (repeating qty).
func expandUnits(g []UnitAtom) []string {
	var out []string
	for _, a := range g {
		for i := 0; i < a.Qty; i++ {
			out = append(out, a.Ref)
		}
	}
	return out
}

func destinations(ev *Event, key string) []string {
	v, ok := ev.Operands[key]
	if !ok {
		return nil
	}
	if len(v.List) > 0 {
		out := make([]string, len(v.List))
		for i, e := range v.List {
			out[i] = e.Str
		}
		return out
	}
	if v.Str != "" {
		return []string{v.Str}
	}
	return nil
}

func foldPlace(s *State, ev *Event, res *FoldResult) {
	g, _ := ev.Group("group")
	dests := destinations(ev, "to")
	if len(dests) == 0 {
		res.errf("place: no destination")
		return
	}
	units := expandUnits(g)
	if len(dests) == 1 {
		for _, u := range g {
			placeUnit(s, ev.Actor, u.Ref, u.Qty, dests[0], res)
		}
		return
	}
	for i, u := range units {
		if i >= len(dests) {
			break
		}
		placeUnit(s, ev.Actor, u, 1, dests[i], res)
	}
}

func placeUnit(s *State, actor, ref string, qty int, dest string, res *FoldResult) {
	owner, kind, ok := splitOwner(ref)
	if !ok {
		owner = actor
		kind = ref
	}
	switch {
	case kind == "w":
		if !strings.HasPrefix(dest, "C") {
			res.warnf("place: warriors to non-clearing %s", dest)
			return
		}
		s.addWarrior(owner, dest, qty)
	case strings.HasPrefix(kind, "b."):
		s.addBuilding(owner, dest, ref)
	case strings.HasPrefix(kind, "t."):
		if kind == "t.wood" {
			s.Clearing(dest).Wood += qty
			return
		}
		s.addToken(owner, dest, ref)
	case kind == "p":
		s.Faction(owner).Pawn = dest
	case strings.HasPrefix(ref, "i."):
		if it, ok := s.Items[ref]; ok {
			it.Zone = dest
			it.Owner = actor
		}
	case isCardRef(ref):
		if strings.HasPrefix(dest, "HAND:") {
			s.Faction(strings.TrimPrefix(dest, "HAND:")).Hand = append(s.Faction(strings.TrimPrefix(dest, "HAND:")).Hand, ref)
		} else if strings.HasPrefix(dest, "DISCARD") {
			s.Discard = append(s.Discard, ref)
		}
	default:
		res.warnf("place: unsupported unit %s", ref)
	}
}

func foldMove(s *State, ev *Event, res *FoldResult) {
	from, _ := ev.Str("from")
	dests := destinations(ev, "to")
	g, _ := ev.Group("group")
	if from == "" || len(dests) == 0 {
		res.errf("move: missing from/to")
		return
	}
	var pieces []string
	for _, a := range g {
		owner, kind, _ := splitOwner(a.Ref)
		if kind == "w" {
			n := a.Qty
			if n > s.warriorCount(owner, from) {
				n = s.warriorCount(owner, from)
			}
			if len(dests) > 1 && len(dests) >= len(expandUnits(g)) {
				s.addWarrior(owner, from, -1)
				s.addWarrior(owner, dests[0], 1)
			} else {
				s.addWarrior(owner, from, -n)
				s.addWarrior(owner, dests[0], n)
			}
			for i := 0; i < n; i++ {
				pieces = append(pieces, a.Ref)
			}
		} else if kind == "p" {
			s.Faction(owner).Pawn = dests[0]
		} else {
			res.warnf("move: unsupported unit %s", a.Ref)
		}
	}
	s.LastMove = &MoveRec{Seq: ev.Seq, Who: ev.Actor, To: dests[0], Pieces: pieces}
	s.ActionsUsed++
}

func foldRemove(s *State, ev *Event, res *FoldResult) {
	at, _ := ev.Str("at")
	g, _ := ev.Group("group")
	var removed []string
	for _, a := range g {
		owner, kind, _ := splitOwner(a.Ref)
		switch {
		case kind == "w":
			n := a.Qty
			if n > s.warriorCount(owner, at) {
				n = s.warriorCount(owner, at)
			}
			s.addWarrior(owner, at, -n)
			for i := 0; i < n; i++ {
				removed = append(removed, a.Ref)
			}
		case strings.HasPrefix(kind, "b."):
			if s.removeBuilding(owner, at, kind) {
				removed = append(removed, a.Ref)
			}
		case strings.HasPrefix(kind, "t."):
			if s.removeToken(owner, at, kind) {
				removed = append(removed, a.Ref)
			}
		default:
			res.warnf("remove: unsupported unit %s", a.Ref)
		}
	}
	s.LastRemoval = &Removal{Seq: ev.Seq, By: ev.Actor, At: at, Pieces: removed}
}

func foldBuild(s *State, ev *Event, res *FoldResult) {
	at, _ := ev.Str("at")
	b, _ := ev.Str("building")
	owner := pieceOwner(b)
	if owner == "" {
		if w, ok := ev.Str("who"); ok {
			owner = w
		} else {
			owner = ev.Actor
		}
	}
	s.addBuilding(owner, at, b)
	if g, ok := ev.Group("cost"); ok {
		for _, a := range g {
			if pieceKind(a.Ref) == "t.wood" {
				spendWood(s, at, a.Qty, res)
			}
		}
	}
	s.ActionsUsed++
}

// spendWood removes wood tokens from the preferred clearing if possible,
// otherwise from any clearing that has wood.
func spendWood(s *State, preferred string, qty int, res *FoldResult) {
	remaining := qty
	if c, ok := s.Clearings[preferred]; ok && c.Wood > 0 {
		take := c.Wood
		if take > remaining {
			take = remaining
		}
		c.Wood -= take
		remaining -= take
	}
	for _, id := range sortedClearings(s) {
		if remaining <= 0 {
			break
		}
		c := s.Clearings[id]
		if c.Wood <= 0 {
			continue
		}
		take := c.Wood
		if take > remaining {
			take = remaining
		}
		c.Wood -= take
		remaining -= take
	}
	if remaining > 0 {
		res.warnf("build: %d wood unavailable", remaining)
	}
}

func foldRecruit(s *State, ev *Event, res *FoldResult) {
	dests := destinations(ev, "at")
	g, _ := ev.Group("group")
	owner := ev.Actor
	if w, ok := ev.Str("who"); ok {
		owner = w
	}
	if len(g) == 0 {
		for _, d := range dests {
			s.addWarrior(owner, d, 1)
		}
		return
	}
	units := expandUnits(g)
	for i, u := range units {
		d := dests[0]
		if len(dests) > 1 && i < len(dests) {
			d = dests[i]
		}
		_, kind, _ := splitOwner(u)
		if kind == "w" {
			s.addWarrior(owner, d, 1)
		}
	}
}

func foldDraw(s *State, ev *Event, res *FoldResult) {
	who, _ := ev.Str("who")
	f := s.Faction(who)
	if cards, ok := ev.OutcomeList("drawn"); ok {
		for _, v := range cards {
			if _, found := removeCard(s.Deck, v.Str); !found {
				res.warnf("draw: %s not in deck", v.Str)
			}
			f.Hand = append(f.Hand, v.Str)
		}
		return
	}
	qty, _ := ev.Int("qty")
	for i := 0; i < qty && len(s.Deck) > 0; i++ {
		f.Hand = append(f.Hand, s.Deck[0])
		s.Deck = s.Deck[1:]
	}
}

func foldDiscard(s *State, ev *Event, res *FoldResult) {
	who, _ := ev.Str("who")
	if who == "" {
		who = ev.Actor
	}
	f := s.Faction(who)
	if g, ok := ev.Group("cards"); ok {
		for _, a := range g {
			for i := 0; i < a.Qty; i++ {
				if takeCard(&f.Hand, a.Ref) {
					s.Discard = append(s.Discard, a.Ref)
				} else if takeCard(&f.Supporters, a.Ref) {
					s.Discard = append(s.Discard, a.Ref)
				} else {
					res.warnf("discard: %s not held by %s", a.Ref, who)
					s.Discard = append(s.Discard, a.Ref)
				}
			}
		}
	}
}

func foldDiscardCard(s *State, ev *Event, res *FoldResult, whoKey, cardKey string) {
	who := ev.Actor
	if whoKey != "" {
		if w, ok := ev.Str(whoKey); ok {
			who = w
		}
	}
	card, ok := ev.Str(cardKey)
	if !ok {
		return
	}
	f := s.Faction(who)
	if !takeCard(&f.Hand, card) {
		res.warnf("%s: %s not in %s's hand", ev.Intent, card, who)
	}
	s.Discard = append(s.Discard, card)
}

func foldGive(s *State, ev *Event, res *FoldResult) {
	from, _ := ev.Str("from")
	to, _ := ev.Str("to")
	g, _ := ev.Group("cards")
	f := s.Faction(from)
	for _, a := range g {
		if !takeCard(&f.Hand, a.Ref) {
			res.warnf("give: %s not in %s's hand", a.Ref, from)
		}
		if strings.HasPrefix(to, "HAND:") {
			s.Faction(strings.TrimPrefix(to, "HAND:")).Hand = append(s.Faction(strings.TrimPrefix(to, "HAND:")).Hand, a.Ref)
		} else if strings.Contains(to, "SUPPORTERS") {
			owner := boardOwner(to)
			s.Faction(owner).Supporters = append(s.Faction(owner).Supporters, a.Ref)
		} else {
			s.Discard = append(s.Discard, a.Ref)
		}
	}
}

func foldCraft(s *State, ev *Event, res *FoldResult) {
	who, _ := ev.Str("who")
	if who == "" {
		who = ev.Actor
	}
	f := s.Faction(who)
	card, _ := ev.Str("card")
	if !takeCard(&f.Hand, card) {
		res.warnf("craft: %s not in %s's hand", card, who)
	}
	s.Discard = append(s.Discard, card)
	if produce, ok := ev.Str("produce"); ok {
		if strings.HasPrefix(produce, "i.") {
			if it, ok := s.Items[produce]; ok {
				it.Owner = who
				it.Zone = "BOARD:" + who + ":CRAFTED"
			}
		}
		f.Crafted = append(f.Crafted, produce)
	}
}

func foldDeal(s *State, ev *Event, res *FoldResult) {
	who, _ := ev.Str("who")
	f := s.Faction(who)
	if g, ok := ev.Group("cards"); ok {
		for _, a := range g {
			if _, found := removeCard(s.Deck, a.Ref); !found {
				// allow deal of explicit cards even if not tracked in deck
			}
			f.Hand = append(f.Hand, a.Ref)
		}
	}
}

func foldAssignRuins(s *State, ev *Event, res *FoldResult) {
	ruins, _ := ev.OutcomeList("ruins")
	items, _ := ev.OutcomeList("items")
	for i, r := range ruins {
		if i >= len(items) {
			break
		}
		c := s.Clearing(r.Str)
		c.Ruin = items[i].Str
		s.Ruins[r.Str] = items[i].Str
		if it, ok := s.Items[items[i].Str]; ok {
			it.Zone = "RUIN:" + r.Str
		}
	}
}

func foldScore(s *State, ev *Event, res *FoldResult) {
	who := ev.Actor
	if w, ok := ev.Str("who"); ok {
		who = w
	}
	amount, _ := ev.Int("amount")
	if amount == 0 {
		amount = 1
	}
	s.Faction(who).VP += amount
}

func foldLoseVP(s *State, ev *Event, res *FoldResult) {
	who := ev.Actor
	if w, ok := ev.Str("who"); ok {
		who = w
	}
	amount, _ := ev.Int("amount")
	if amount == 0 {
		amount = 1
	}
	f := s.Faction(who)
	f.VP -= amount
	if f.VP < 0 {
		f.VP = 0
	}
}

func foldMoveVP(s *State, ev *Event, res *FoldResult) {
	who, _ := ev.Str("who")
	to, _ := ev.Str("to_board")
	s.Faction(who).VPLocation = to
}

func foldMoveItem(s *State, ev *Event, res *FoldResult) {
	to, _ := ev.Str("to")
	g, _ := ev.Group("items")
	for _, a := range g {
		if it, ok := s.Items[a.Ref]; ok {
			it.Zone = to
			it.Damaged = strings.Contains(to, "DAMAGED")
		}
	}
}

func foldSetItemState(s *State, ev *Event, res *FoldResult) {
	g, _ := ev.Group("items")
	ex, hasEx := ev.Operands["exhausted"]
	dmg, hasDmg := ev.Operands["damaged"]
	for _, a := range g {
		it, ok := s.Items[a.Ref]
		if !ok {
			res.warnf("set-item-state: unknown item %s", a.Ref)
			continue
		}
		if hasEx {
			it.Exhausted = ex.Bool
		}
		if hasDmg {
			it.Damaged = dmg.Bool
		}
	}
}

func foldOutrage(s *State, ev *Event, res *FoldResult) {
	from, _ := ev.Str("from")
	card, _ := ev.Str("card")
	f := s.Faction(from)
	if !takeCard(&f.Hand, card) {
		res.warnf("outrage: %s not in %s's hand", card, from)
	}
	s.Faction("A").Supporters = append(s.Faction("A").Supporters, card)
}

func foldFieldHospitals(s *State, ev *Event, res *FoldResult) {
	at, _ := ev.Str("at")
	to, _ := ev.Str("to")
	if card, ok := ev.Str("spend"); ok {
		f := s.Faction(ev.Actor)
		if !takeCard(&f.Hand, card) {
			res.warnf("field-hospitals: %s not in hand", card)
		}
		s.Discard = append(s.Discard, card)
	}
	g, _ := ev.Group("save")
	for _, a := range g {
		owner, kind, _ := splitOwner(a.Ref)
		if kind != "w" {
			continue
		}
		n := a.Qty
		if n > s.warriorCount(owner, at) {
			n = s.warriorCount(owner, at)
		}
		s.addWarrior(owner, at, -n)
		s.addWarrior(owner, to, n)
	}
}

func foldCommit(s *State, ev *Event, res *FoldResult) {
	who, _ := ev.Str("who")
	domain, _ := ev.Str("domain")
	hash, _ := ev.Str("hash")
	s.Hidden[who+":"+domain] = hash
}

func foldRevealCommit(s *State, ev *Event, res *FoldResult) {
	who, _ := ev.Str("who")
	domain, _ := ev.Str("domain")
	val, _ := ev.Str("value")
	s.Hidden[who+":"+domain+":value"] = val
}

func foldBirdsongWood(s *State, ev *Event, res *FoldResult) {
	dests := destinations(ev, "at")
	owner := ev.Actor
	if owner == "" {
		owner = "C"
	}
	for _, d := range dests {
		c := s.Clearing(d)
		n := 0
		for _, b := range c.Buildings {
			if b.Owner == owner && (b.Kind == "b.saw" || b.Kind == "b.sawmill") {
				n++
			}
		}
		if n > 0 {
			c.Wood += n
		}
	}
}

// foldOverwork handles C:overwork: discard the spent card, then produce either
// a wood token at `at` (produce is a wood token) or a victory point
// (produce is the faction's vp token).
func foldOverwork(s *State, ev *Event, res *FoldResult) {
	at, _ := ev.Str("at")
	owner := ev.Actor
	if owner == "" {
		owner = "C"
	}
	f := s.Faction(owner)
	if spend, ok := ev.Str("spend"); ok {
		if takeCard(&f.Hand, spend) {
			s.Discard = append(s.Discard, spend)
		} else {
			res.warnf("overwork: %s not in %s's hand", spend, owner)
		}
	}
	produce, _ := ev.Str("produce")
	switch {
	case strings.Contains(produce, "t.wood"):
		s.Clearing(at).Wood++
	case strings.Contains(produce, "vp") || produce == "VP":
		f.VP++
	default:
		res.warnf("overwork: unknown produce %q", produce)
	}
}

func foldDecreeAdd(s *State, ev *Event, res *FoldResult) {
	col, _ := ev.Str("column")
	f := s.Faction("E")
	if f.Decree == nil {
		f.Decree = map[string][]string{}
	}
	if g, ok := ev.Group("cards"); ok {
		for _, a := range g {
			if !takeCard(&f.Hand, a.Ref) {
				res.warnf("decree-add: %s not in hand", a.Ref)
			}
			f.Decree[col] = append(f.Decree[col], a.Ref)
		}
	}
}

func foldDiscardDecree(s *State, ev *Event, res *FoldResult) {
	f := s.Faction("E")
	for _, col := range []string{"RECRUIT", "MOVE", "BATTLE", "BUILD"} {
		s.Discard = append(s.Discard, f.Decree[col]...)
	}
	s.Discard = append(s.Discard, f.Viziers...)
	f.Decree = map[string][]string{}
	f.Viziers = nil
}

func foldAppointLeader(s *State, ev *Event, res *FoldResult) {
	if l, ok := ev.Str("leader"); ok {
		s.Faction(ev.Actor).Leader = l
	}
}

func foldAddVizier(s *State, ev *Event, res *FoldResult) {
	if c, ok := ev.Str("card"); ok {
		f := s.Faction(ev.Actor)
		takeCard(&f.Hand, c)
		f.Viziers = append(f.Viziers, c)
	}
}

func foldMobilize(s *State, ev *Event, res *FoldResult) {
	f := s.Faction("A")
	if g, ok := ev.Group("cards"); ok {
		for _, a := range g {
			if !takeCard(&f.Hand, a.Ref) {
				res.warnf("mobilize: %s not in hand", a.Ref)
			}
			f.Supporters = append(f.Supporters, a.Ref)
		}
	}
}

func foldSpendSupporters(s *State, ev *Event, res *FoldResult) {
	f := s.Faction("A")
	if g, ok := ev.Group("cards"); ok {
		for _, a := range g {
			if takeCard(&f.Supporters, a.Ref) {
				s.Discard = append(s.Discard, a.Ref)
			} else {
				res.warnf("spend-supporters: %s not a supporter", a.Ref)
			}
		}
	}
}

func foldPlaceSympathy(s *State, ev *Event, res *FoldResult) {
	at, _ := ev.Str("at")
	f := s.Faction("A")
	if g, ok := ev.Group("spend"); ok {
		for _, a := range g {
			for i := 0; i < a.Qty; i++ {
				if takeCard(&f.Supporters, a.Ref) {
					s.Discard = append(s.Discard, a.Ref)
				} else {
					res.warnf("place-sympathy: %s not a supporter", a.Ref)
				}
			}
		}
	} else if spend, ok := ev.Str("spend"); ok {
		if takeCard(&f.Supporters, spend) {
			s.Discard = append(s.Discard, spend)
		} else {
			res.warnf("place-sympathy: %s not a supporter", spend)
		}
	}
	ref := fmt.Sprintf("A.t.sym#%d", countTokens(s, "A", "t.sym")+1)
	s.addToken("A", at, ref)
}

func foldRevolt(s *State, ev *Event, res *FoldResult) {
	at, _ := ev.Str("at")
	base, _ := ev.Str("base")
	f := s.Faction("A")
	if g, ok := ev.Group("cards"); ok {
		for _, a := range g {
			if takeCard(&f.Supporters, a.Ref) {
				s.Discard = append(s.Discard, a.Ref)
			}
		}
	}
	s.addBuilding("A", at, base)
	s.addWarrior("A", at, 1)
}

func foldChooseCharacter(s *State, ev *Event, res *FoldResult) {
	if c, ok := ev.Str("character"); ok {
		s.Faction(ev.Actor).Character = c
	}
}

func foldMovePawn(s *State, ev *Event, res *FoldResult) {
	if to, ok := ev.Str("to"); ok {
		s.Faction(ev.Actor).Pawn = to
	}
}

func foldAid(s *State, ev *Event, res *FoldResult) {
	target, _ := ev.Str("target")
	card, _ := ev.Str("card")
	f := s.Faction(ev.Actor)
	if takeCard(&f.Hand, card) {
		s.Faction(target).Hand = append(s.Faction(target).Hand, card)
	}
	if take, ok := ev.Str("take"); ok && take != "none" {
		if it, ok := s.Items[take]; ok {
			it.Owner = ev.Actor
			it.Zone = "BOARD:" + ev.Actor + ":TRACK"
		}
	}
	f.Relationship[target] = "1"
	s.ActionsUsed++
}

func foldExplore(s *State, ev *Event, res *FoldResult) {
	at, _ := ev.Str("at")
	if item, ok := ev.OutcomeStr("item"); ok {
		c := s.Clearing(at)
		c.Ruin = ""
		delete(s.Ruins, at)
		if it, ok := s.Items[item]; ok {
			it.Owner = ev.Actor
			it.Zone = "BOARD:" + ev.Actor + ":SATCHEL"
		}
	}
	s.ActionsUsed++
}

func foldTakeQuest(s *State, ev *Event, res *FoldResult) {
	q, ok := ev.OutcomeStr("quest")
	if !ok {
		q, _ = ev.Str("quest")
	}
	if q != "" {
		f := s.Faction(ev.Actor)
		f.Quests = append(f.Quests, q)
	}
	s.ActionsUsed++
}

func foldCompleteQuest(s *State, ev *Event, res *FoldResult) {
	if q, ok := ev.Str("quest"); ok {
		f := s.Faction(ev.Actor)
		takeCard(&f.Quests, q)
	}
	if g, ok := ev.Group("items"); ok {
		for _, a := range g {
			if it, ok := s.Items[a.Ref]; ok {
				it.Exhausted = true
			}
		}
	}
	s.ActionsUsed++
}

func foldRefreshItems(s *State, ev *Event, res *FoldResult) {
	g, _ := ev.Group("items")
	for _, a := range g {
		if it, ok := s.Items[a.Ref]; ok {
			it.Exhausted = false
			it.Damaged = false
			if strings.Contains(ev.Intent, "repair") || strings.Contains(ev.Intent, "rest") {
				it.Zone = "BOARD:" + ev.Actor + ":SATCHEL"
			}
		}
	}
}

func foldExhaustItems(s *State, ev *Event, res *FoldResult, ex bool) {
	g, _ := ev.Group("items")
	for _, a := range g {
		if it, ok := s.Items[a.Ref]; ok {
			it.Exhausted = ex
		}
	}
}

func foldDamageItems(s *State, ev *Event, res *FoldResult) {
	g, _ := ev.Group("items")
	for _, a := range g {
		if it, ok := s.Items[a.Ref]; ok {
			it.Damaged = true
			it.Zone = "BOARD:" + ev.Actor + ":DAMAGED"
		}
	}
}

func foldRelationship(s *State, ev *Event, res *FoldResult) {
	target, _ := ev.Str("target")
	status, _ := ev.Str("status")
	s.Faction(ev.Actor).Relationship[target] = status
}

func foldCoalition(s *State, ev *Event, res *FoldResult) {
	target, _ := ev.Str("target")
	s.Faction(ev.Actor).VPLocation = target
}

func countTokens(s *State, faction, kind string) int {
	n := 0
	for _, c := range s.Clearings {
		for _, t := range c.Tokens {
			if t.Owner == faction && t.Kind == kind {
				n++
			}
		}
	}
	return n
}

func boardOwner(loc string) string {
	parts := strings.Split(loc, ":")
	if len(parts) >= 2 && parts[0] == "BOARD" {
		return parts[1]
	}
	return ""
}
