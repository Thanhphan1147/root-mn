package root

// Woodland Alliance handlers.

func init() {
	register("A:revolt", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "revolt", Faction: WA, Clearing: ev.op("at"), Cards: listItems(ev.op("cards"))}, true, nil
	})
	register("A:place-sympathy", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "spread", Faction: WA, Clearing: ev.op("at"), Cards: listItems(ev.op("spend"))}, true, nil
	})
	register("A:mobilize", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "mobilize", Faction: WA, Card: ev.op("cards")}, true, nil
	})
	register("A:train", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "train", Faction: WA, Card: ev.op("card")}, true, nil
	})
	register("A:organize", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "organize", Faction: WA, Clearing: ev.op("at")}, true, nil
	})
}
