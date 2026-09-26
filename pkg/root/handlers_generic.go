package root

import (
	"strconv"
	"strings"
)

// Generic handlers: intents shared by (or meaningful to) all factions.

func init() {
	register("pass", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "pass", Faction: ev.Actor}, true, nil
	})

	// move covers warrior moves and the Vagabond pawn; WA recruit uses "place".
	register("move", func(g *Game, ev rmnEvent) (Action, bool, error) {
		if ev.op("group") == "VB.p" {
			return Action{Kind: "vb-move", Faction: VB, To: ev.op("to")}, true, nil
		}
		n, f := warriorGroup(ev.op("group"))
		if f == "" {
			f = ev.Actor
		}
		return Action{Kind: "move", Faction: f, From: ev.op("from"), To: ev.op("to"), Amount: n}, true, nil
	})

	register("craft", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "craft", Faction: ev.Actor, Card: ev.op("card")}, true, nil
	})
	register("spend-bird", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "spend-bird", Faction: ev.Actor, Card: ev.op("card")}, true, nil
	})
	register("discard", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "discard", Faction: ev.Actor, Card: ev.op("cards")}, true, nil
	})
	register("play-dominance", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "dominance", Faction: ev.Actor, Card: ev.op("card")}, true, nil
	})
	register("field-hospitals", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "field-hospitals", Faction: MC, Clearing: ev.op("at"), Card: ev.op("spend")}, true, nil
	})
	register("score", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "royal-claim", Faction: ev.Actor}, true, nil
	})
	register("give", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "stand-deliver", Faction: ev.Actor, Target: Faction(ev.op("from"))}, true, nil
	})
	register("draw", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "tax-collector", Faction: ev.Actor}, true, nil
	})

	// battle records the dice as an outcome; arm them so replay reproduces the roll.
	register("battle", func(g *Game, ev rmnEvent) (Action, bool, error) {
		if a, ok := ev.intOut("atk"); ok {
			if d, ok := ev.intOut("def"); ok {
				g.NextRoll = []int{a, d}
			}
		}
		return Action{Kind: "battle", Faction: ev.Actor, Clearing: ev.op("at"), Target: Faction(ev.op("defender"))}, true, nil
	})
	register("ambush", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "battle-ambush", Faction: ev.Actor, Card: ev.op("card")}, true, nil
	})
	register("CARD:effect", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "battle-effect", Faction: ev.Actor, Card: ev.op("card")}, true, nil
	})

	// notify is the generic carrier for several distinct choices; dispatch on text.
	register("notify", notifyHandler)

	// remove is a battle hit on a warrior, building or token.
	register("remove", removeHandler)

	// place covers all setup placements and the WA recruit.
	register("place", placeHandler)
}

// removeHandler resolves a battle-hit removal. Building/token hits need the
// target's index in the clearing, which the RMN records by type.
func removeHandler(g *Game, ev rmnEvent) (Action, bool, error) {
	grp := ev.op("group")
	at := ev.op("at")
	if grp == "ally-warrior" {
		return Action{}, false, nil // needs the ally target; resolver handles it
	}
	if strings.HasSuffix(grp, ".w") {
		_, f := warriorGroup(grp)
		return Action{Kind: "battle-hit", Faction: f, Piece: "warrior", Clearing: at}, true, nil
	}
	if i := strings.Index(grp, ".b."); i > 0 {
		f := Faction(grp[:i])
		typ := rmnBuildingInverse(grp[i+3:])
		if idx := buildingIndex(g, f, at, typ); idx >= 0 {
			return Action{Kind: "battle-hit", Faction: f, Piece: "building", Building: typ, Amount: idx, Clearing: at}, true, nil
		}
		return Action{}, false, nil
	}
	if i := strings.Index(grp, ".t."); i > 0 {
		f := Faction(grp[:i])
		typ := grp[i+3:]
		if idx := tokenIndex(g, f, at, typ); idx >= 0 {
			return Action{Kind: "battle-hit", Faction: f, Piece: "token", Item: typ, Amount: idx, Clearing: at}, true, nil
		}
		return Action{}, false, nil
	}
	return Action{}, false, nil
}

func buildingIndex(g *Game, f Faction, c, typ string) int {
	cl := g.Clearings[c]
	if cl == nil {
		return -1
	}
	for i, b := range cl.Buildings {
		if b.Owner == f && b.Type == typ {
			return i
		}
	}
	return -1
}

func tokenIndex(g *Game, f Faction, c, typ string) int {
	cl := g.Clearings[c]
	if cl == nil {
		return -1
	}
	for i, t := range cl.Tokens {
		if t.Owner == f && t.Type == typ {
			return i
		}
	}
	return -1
}

func (ev rmnEvent) intOut(k string) (int, bool) {
	v, ok := ev.Out[k]
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	return n, err == nil
}

func notifyHandler(g *Game, ev rmnEvent) (Action, bool, error) {
	text := ev.op("text")
	switch {
	case text == "done crafting":
		if ev.Actor == ED {
			return Action{Kind: "ed-done-crafting", Faction: ED}, true, nil
		}
		return Action{Kind: "mc-done-crafting", Faction: MC}, true, nil
	case text == "skip":
		return Action{Kind: "battle-skip", Faction: ev.Actor}, true, nil
	case strings.HasPrefix(text, "took available"):
		f := strings.Fields(text) // took available <item> spending <card>
		a := Action{Kind: "take-dominance", Faction: ev.Actor}
		if len(f) >= 3 {
			a.Item = f[2]
		}
		for i := range f {
			if f[i] == "spending" && i+1 < len(f) {
				a.Card = f[i+1]
			}
		}
		return a, true, nil
	case strings.HasPrefix(text, "looked at"):
		f := strings.Fields(text) // looked at <faction> hand
		a := Action{Kind: "codebreakers", Faction: ev.Actor}
		if len(f) >= 3 {
			a.Target = Faction(f[2])
		}
		return a, true, nil
	}
	return Action{}, false, nil
}

// placeHandler resolves setup placements (by the group's piece) and WA recruit.
func placeHandler(g *Game, ev rmnEvent) (Action, bool, error) {
	to := ev.op("to")
	grp := ev.op("group")
	switch {
	case strings.HasPrefix(grp, "MC.b.keep"):
		return Action{Kind: "setup-mc-keep", Faction: MC, Clearing: to}, true, nil
	case strings.HasPrefix(grp, "MC.b."):
		b := strings.TrimSuffix(strings.TrimPrefix(grp, "MC.b."), "#1")
		return Action{Kind: "setup-mc-build", Faction: MC, Building: rmnBuildingInverse(b), Clearing: to}, true, nil
	case strings.Contains(grp, "ED.b.roost"):
		return Action{Kind: "setup-ed-corner", Faction: ED, Clearing: to}, true, nil
	case grp == "WA.w" || strings.HasSuffix(grp, "WA.w"):
		return Action{Kind: "wa-recruit", Faction: WA, Clearing: to}, true, nil
	}
	return Action{}, false, nil
}

func rmnBuildingInverse(t string) string {
	switch t {
	case "saw":
		return "sawmill"
	}
	return t
}
