package root

import "strings"

// Vagabond handlers.

func init() {
	register("V:choose-character", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "setup-vb-character", Faction: VB, Character: strings.TrimPrefix(ev.op("character"), "V.character.")}, true, nil
	})
	register("V:move-pawn", func(g *Game, ev rmnEvent) (Action, bool, error) {
		if g.SetupMode {
			return Action{Kind: "setup-vb-forest", Faction: VB, To: ev.op("to")}, true, nil
		}
		return Action{Kind: "vb-slip", Faction: VB, To: ev.op("to")}, true, nil
	})
	register("V:explore", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "vb-explore", Faction: VB, Clearing: ev.op("at")}, true, nil
	})
	register("V:aid", func(g *Game, ev rmnEvent) (Action, bool, error) {
		a := Action{Kind: "vb-aid", Faction: VB, Target: Faction(ev.op("target")), Card: ev.op("card"), Exhaust: ev.op("exhaust")}
		if t := ev.op("take"); t != "" && t != "none" {
			a.Item = unitItem(t)
		}
		return a, true, nil
	})
	register("V:complete-quest", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "vb-quest", Faction: VB, Quest: ev.op("quest"), Item: ev.op("reward")}, true, nil
	})
	register("V:repair", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "vb-repair", Faction: VB, Item: ev.op("items")}, true, nil
	})
	register("V:strike", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "vb-strike", Faction: VB, Target: Faction(ev.op("target")), Clearing: ev.op("at")}, true, nil
	})
	register("V:refresh", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "vb-refresh", Faction: VB, Item: ev.op("items")}, true, nil
	})
	register("V:day-labor", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "vb-special", Faction: VB, Card: ev.op("card")}, true, nil
	})
	register("V:steal", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "vb-special", Faction: VB, Target: Faction(ev.op("target"))}, true, nil
	})
	register("V:hideout", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "vb-special", ID: "vb-special|hideout", Faction: VB}, true, nil
	})
	register("V:damage", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "battle-hit", Faction: VB, Item: ev.op("items")}, true, nil
	})
}

func init() {
	register("V:coalition", func(g *Game, ev rmnEvent) (Action, bool, error) {
		// The coalition line records the target; the spent dominance card is the
		// one in the Vagabond's hand.
		card := ""
		for _, id := range g.Players[VB].Hand {
			if c, ok := Card(id); ok && c.Kind == KindDominance {
				card = id
				break
			}
		}
		return Action{Kind: "coalition", Faction: VB, Target: Faction(ev.op("target")), Card: card}, true, nil
	})
}
