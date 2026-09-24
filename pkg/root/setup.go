package root

import (
	"fmt"
	"strings"
)

// SetupOptions controls initial faction choices for the automatic setup.
type SetupOptions struct {
	MCCorner  string
	EDCorner  string
	EDLeader  string
	Character string
	Forest    string
	Seed      uint64
}

// BeginSetup initialises shared setup state and enters interactive setup mode.
func BeginSetup(g *Game) {
	m := GetMap("autumn")
	ruinClearings := append([]string{}, m.Ruins...)
	items := append([]string{}, RuinItems...)
	shuffleStr(items, &g.RngSeed)
	for i, c := range ruinClearings {
		g.Clearings[c].RuinItem = items[i%len(items)]
	}
	// Record which item is under each ruin (the players' RMN is the full-info
	// log; Redact hides this from everyone's view).
	var ruins, ruinItems []string
	for _, c := range ruinClearings {
		ruins = append(ruins, c)
		ruinItems = append(ruinItems, "i."+g.Clearings[c].RuinItem)
	}
	g.RMNLog = append(g.RMNLog, fmt.Sprintf("%d 0.S SYS assign-ruins -> {ruins=[%s], items=[%s]}",
		len(g.RMNLog)+1, strings.Join(ruins, ","), strings.Join(ruinItems, ",")))
	g.ItemSupply = map[string]int{}
	for k, v := range ItemCounts {
		g.ItemSupply[k] = v
	}
	g.QuestDeck = nil
	for _, q := range Quests {
		g.QuestDeck = append(g.QuestDeck, q.ID)
	}
	shuffleStr(g.QuestDeck, &g.RngSeed)
	for i := 0; i < 3 && len(g.QuestDeck) > 0; i++ {
		g.QuestAvail = append(g.QuestAvail, g.QuestDeck[0])
		g.QuestDeck = g.QuestDeck[1:]
	}
	g.SetupMode = true
	g.Round = 0
	g.Phase = "S"
	g.startSetup()
}

// has reports whether a faction is in this game.
func (g *Game) has(f Faction) bool {
	_, ok := g.Players[f]
	return ok
}

// startSetup begins the faction setup sequence for the factions present,
// skipping any that are not in the game.
func (g *Game) startSetup() {
	switch {
	case g.has(MC):
		g.SetupStage = "MC_KEEP"
		g.Current = MC
		g.Logf(MC, "setup", "Setup: Marquise chooses the keep clearing")
	case g.has(ED):
		g.SetupStage = "ED_CORNER"
		g.Current = ED
		g.Logf(ED, "setup", "Setup: Eyrie chooses a corner")
	case g.has(WA):
		g.waSetup()
	case g.has(VB):
		g.SetupStage = "VB_CHARACTER"
		g.Current = VB
		g.Logf(VB, "setup", "Setup: Vagabond chooses a character")
	default:
		g.finishSetup()
	}
}

func (g *Game) afterMC() {
	if g.has(ED) {
		g.SetupStage = "ED_CORNER"
		g.Current = ED
		g.Logf(ED, "setup", "Setup: Eyrie chooses a corner")
		return
	}
	g.afterED()
}

func (g *Game) afterED() {
	if g.has(WA) {
		g.waSetup()
		return
	}
	g.afterWA()
}

func (g *Game) afterWA() {
	if g.has(VB) {
		g.SetupStage = "VB_CHARACTER"
		g.Current = VB
		g.Logf(VB, "setup", "Setup: Vagabond chooses a character")
		return
	}
	g.finishSetup()
}

// Setup runs the standard automatic setup (used by tests and quick play).
func Setup(g *Game, opt SetupOptions) {
	BeginSetup(g)
	keep := opt.MCCorner
	if keep == "" {
		keep = "C1"
	}
	g.setupMCKeep(keep)
	allowed := append([]string{keep}, GetMap("autumn").Clearings[keep].Adj...)
	for _, typ := range []string{"sawmill", "workshop", "recruiter"} {
		placed := false
		for _, c := range allowed {
			if g.setupMCBuild(typ, c) {
				placed = true
				break
			}
		}
		if !placed {
			for _, c := range allowed {
				if g.setupMCBuild(typ, c) {
					break
				}
			}
		}
	}
	corner := opt.EDCorner
	if g.Players[MC] != nil && g.Players[MC].KeepClearing != "" {
		// The Eyrie must start diagonally opposite the Marquise.
		corner = oppositeCorner(g.Players[MC].KeepClearing)
	} else if corner == "" {
		corner = "C3"
	}
	g.setupEDCorner(corner)
	leader := opt.EDLeader
	if leader == "" {
		leader = "despot"
	}
	g.setupEDLeader(leader)
	ch := opt.Character
	if ch == "" {
		ch = "tinker"
	}
	g.setupVBCharacter(ch)
	forest := opt.Forest
	if forest == "" {
		forest = "AutumnN"
	}
	g.setupVBForest(forest)
}

func (g *Game) setupMCKeep(clearing string) {
	p := g.Players[MC]
	p.KeepClearing = clearing
	opposite := oppositeCorner(clearing)
	for _, c := range GetMap("autumn").ClearingList() {
		if c == opposite {
			continue
		}
		g.addWarrior(MC, c, 1)
	}
	g.Clearings[clearing].Tokens = append(g.Clearings[clearing].Tokens, Token{MC, "keep"})
	g.SetupStage = "MC_BUILD"
	g.Logf(MC, "setup", "Marquise placed the keep at %s", clearing)
}

func (g *Game) setupMCBuild(typ, clearing string) bool {
	if g.SetupPlaced == nil {
		g.SetupPlaced = map[string]bool{}
	}
	if g.SetupPlaced[typ] {
		return false
	}
	cl := g.Clearings[clearing]
	if cl == nil || cl.FreeSlots() <= 0 {
		return false
	}
	cl.Buildings = append(cl.Buildings, Building{MC, typ})
	g.SetupPlaced[typ] = true
	g.Logf(MC, "setup", "Marquise placed a %s at %s", typ, clearing)
	if g.SetupPlaced["sawmill"] && g.SetupPlaced["workshop"] && g.SetupPlaced["recruiter"] {
		g.afterMC()
	}
	return true
}

func (g *Game) setupEDCorner(clearing string) {
	p := g.Players[ED]
	g.Clearings[clearing].Buildings = append(g.Clearings[clearing].Buildings, Building{ED, "roost"})
	g.addWarrior(ED, clearing, 6)
	p.Leader = ""
	g.SetupStage = "ED_LEADER"
	g.Logf(ED, "setup", "Eyrie placed a roost + 6 warriors at %s", clearing)
}

func (g *Game) setupEDLeader(leader string) {
	p := g.Players[ED]
	p.Leader = leader
	p.Viziers = nil
	for _, col := range Leaders[leader].Viziers {
		p.Decree[col] = append(p.Decree[col], "VIZIER")
		p.Viziers = append(p.Viziers, "VIZIER")
	}
	g.Logf(ED, "setup", "Eyrie chose leader %s", leader)
	g.afterED()
}

func (g *Game) waSetup() {
	g.drawCards(WA, 3)
	p := g.Players[WA]
	p.Supporters = append(p.Supporters, p.Hand...)
	p.Hand = nil
	g.Logf(WA, "setup", "Alliance drew 3 supporters")
	g.afterWA()
}

func (g *Game) setupVBCharacter(ch string) {
	g.Players[VB].Character = ch
	for _, t := range Characters[ch].Start {
		g.giveVBItem(g.Players[VB], t)
	}
	g.SetupStage = "VB_FOREST"
	g.Logf(VB, "setup", "Vagabond chose %s", ch)
}

func (g *Game) setupVBForest(forest string) {
	g.Players[VB].Pawn = forest
	for f := range g.Players {
		if f != VB {
			g.Players[VB].Relationships[f] = "indifferent"
		}
	}
	g.SetupStage = "DONE"
	g.finishSetup()
}

func (g *Game) finishSetup() {
	for _, f := range g.Order {
		g.drawCards(f, 3)
	}
	if p := g.Players[MC]; p != nil {
		p.Sawmills, p.Workshops, p.Recruiters = 5, 5, 5
		p.WoodSupply = 8
	}
	g.SetupMode = false
	g.Round = 1
	g.Phase = "B"
	g.Current = g.First
	g.Logf(g.First, "setup", "Setup complete")
	g.beginTurn()
}

// legalSetup returns setup actions for the current stage.
func (g *Game) legalSetup() []Action {
	var acts []Action
	switch g.SetupStage {
	case "MC_KEEP":
		for _, c := range []string{"C1", "C2", "C3", "C4"} {
			acts = append(acts, Action{
				ID: actID("setup-mc-keep", c), Label: "Place keep at " + c,
				Kind: "setup-mc-keep", Faction: MC, Clearing: c,
			})
		}
	case "MC_BUILD":
		keep := g.Players[MC].KeepClearing
		allowed := append([]string{keep}, GetMap("autumn").Clearings[keep].Adj...)
		for _, typ := range []string{"sawmill", "workshop", "recruiter"} {
			if g.SetupPlaced[typ] {
				continue
			}
			for _, c := range allowed {
				cl := g.Clearings[c]
				if cl == nil || cl.FreeSlots() <= 0 {
					continue
				}
				acts = append(acts, Action{
					ID:    actID("setup-mc-build", typ, c),
					Label: "Place " + typ + " at " + c,
					Kind:  "setup-mc-build", Faction: MC, Building: typ, Clearing: c,
				})
			}
		}
	case "ED_CORNER":
		if mc := g.Players[MC]; mc != nil && mc.KeepClearing != "" {
			// Must be diagonally opposite the Marquise's keep.
			c := oppositeCorner(mc.KeepClearing)
			acts = append(acts, Action{
				ID: actID("setup-ed-corner", c), Label: "Place roost at " + c + " (opposite the Marquise)",
				Kind: "setup-ed-corner", Faction: ED, Clearing: c,
			})
		} else {
			for _, c := range []string{"C1", "C2", "C3", "C4"} {
				acts = append(acts, Action{
					ID: actID("setup-ed-corner", c), Label: "Place roost at " + c,
					Kind: "setup-ed-corner", Faction: ED, Clearing: c,
				})
			}
		}
	case "ED_LEADER":
		for name := range Leaders {
			acts = append(acts, Action{
				ID: actID("setup-ed-leader", name), Label: "Choose leader: " + name,
				Kind: "setup-ed-leader", Faction: ED, Leader: name,
			})
		}
	case "VB_CHARACTER":
		for name := range Characters {
			acts = append(acts, Action{
				ID: actID("setup-vb-character", name), Label: "Choose character: " + name,
				Kind: "setup-vb-character", Faction: VB, Character: name,
			})
		}
	case "VB_FOREST":
		for _, f := range GetMap("autumn").ForestList() {
			acts = append(acts, Action{
				ID: actID("setup-vb-forest", f), Label: "Start in forest " + f,
				Kind: "setup-vb-forest", Faction: VB, To: f,
			})
		}
	}
	return acts
}

func (g *Game) applySetup(a Action) error {
	switch a.Kind {
	case "setup-mc-keep":
		g.setupMCKeep(a.Clearing)
	case "setup-mc-build":
		g.setupMCBuild(a.Building, a.Clearing)
	case "setup-ed-corner":
		g.setupEDCorner(a.Clearing)
	case "setup-ed-leader":
		g.setupEDLeader(a.Leader)
	case "setup-vb-character":
		g.setupVBCharacter(a.Character)
	case "setup-vb-forest":
		g.setupVBForest(a.To)
	}
	return nil
}

func placeMC(g *Game, typ, clearing string) {
	cl := g.Clearings[clearing]
	if cl.FreeSlots() <= 0 {
		return
	}
	cl.Buildings = append(cl.Buildings, Building{MC, typ})
}

func oppositeCorner(c string) string {
	switch c {
	case "C1":
		return "C3"
	case "C3":
		return "C1"
	case "C2":
		return "C4"
	case "C4":
		return "C2"
	}
	return "C3"
}

func shuffleStr(s []string, seed *uint64) {
	for i := len(s) - 1; i > 0; i-- {
		*seed ^= *seed << 13
		*seed ^= *seed >> 7
		*seed ^= *seed << 17
		j := int(*seed % uint64(i+1))
		s[i], s[j] = s[j], s[i]
	}
}

// giveVBItem adds an item instance to the Vagabond in the correct zone.
func (g *Game) giveVBItem(p *Player, typ string) {
	p.ItemSeq++
	id := itemID(typ, p.ItemSeq)
	st := &ItemState{Type: typ, Zone: "satchel", FaceUp: true}
	if typ == "tea" || typ == "coin" || typ == "bag" {
		if trackCount(p, typ) < 3 {
			st.Zone = "track"
		}
	}
	p.Items[id] = st
}

func itemID(typ string, n int) string {
	return typ + "#" + itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func trackCount(p *Player, typ string) int {
	n := 0
	for _, it := range p.Items {
		if it.Type == typ && it.Zone == "track" && it.FaceUp {
			n++
		}
	}
	return n
}
