package root

import "strings"

// Eyrie Dynasties handlers.

func init() {
	register("E:decree-add", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "decree-add", Faction: ED, Card: ev.op("cards"), Column: ev.op("column")}, true, nil
	})
	register("E:recruit", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "decree-recruit", Faction: ED, Clearing: firstLoc(ev.op("at")), Card: ev.op("card"), Column: decreeColumn(g, ev.op("card"))}, true, nil
	})
	register("E:move", func(g *Game, ev rmnEvent) (Action, bool, error) {
		n, _ := warriorGroup(ev.op("group"))
		return Action{Kind: "decree-move", Faction: ED, From: ev.op("from"), To: ev.op("to"), Amount: n, Card: ev.op("card"), Column: decreeColumn(g, ev.op("card"))}, true, nil
	})
	register("E:battle", func(g *Game, ev rmnEvent) (Action, bool, error) {
		if a, ok := ev.intOut("atk"); ok {
			if d, ok := ev.intOut("def"); ok {
				g.NextRoll = []int{a, d}
			}
		}
		return Action{Kind: "decree-battle", Faction: ED, Clearing: ev.op("at"), Target: Faction(ev.op("defender")), Card: ev.op("card"), Column: decreeColumn(g, ev.op("card"))}, true, nil
	})
	register("E:build", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "decree-build", Faction: ED, Clearing: ev.op("at"), Card: ev.op("card"), Column: decreeColumn(g, ev.op("card"))}, true, nil
	})
	register("E:turmoil", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "ed-turmoil", Faction: ED}, true, nil
	})
	register("E:appoint-leader", func(g *Game, ev rmnEvent) (Action, bool, error) {
		leader := strings.TrimPrefix(ev.op("leader"), "ED.ldr.")
		if g.SetupMode {
			return Action{Kind: "setup-ed-leader", Faction: ED, Leader: leader}, true, nil
		}
		return Action{Kind: "leader", Faction: ED, Leader: leader}, true, nil
	})
}

func firstLoc(v string) string {
	v = strings.Trim(v, "[]")
	if i := strings.IndexByte(v, ','); i >= 0 {
		return v[:i]
	}
	return v
}

// decreeColumn finds which Decree column a card is queued in (the RMN line for
// decree resolution does not record it).
func decreeColumn(g *Game, card string) string {
	for _, it := range g.DecreeQueue {
		if it.Card == card {
			return it.Column
		}
	}
	return ""
}
