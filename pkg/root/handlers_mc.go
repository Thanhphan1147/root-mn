package root

import "strings"

// Marquise de Cat handlers.

func init() {
	register("C:build", func(g *Game, ev rmnEvent) (Action, bool, error) {
		b := strings.TrimSuffix(strings.TrimPrefix(ev.op("building"), "MC.b."), "#1")
		return Action{Kind: "mc-build", Faction: MC, Building: rmnBuildingInverse(b), Clearing: ev.op("at")}, true, nil
	})
	register("C:recruit", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "mc-recruit", Faction: MC}, true, nil
	})
	register("C:overwork", func(g *Game, ev rmnEvent) (Action, bool, error) {
		return Action{Kind: "mc-overwork", Faction: MC, Clearing: ev.op("at"), Card: ev.op("spend")}, true, nil
	})
}
