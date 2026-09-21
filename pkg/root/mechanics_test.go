package root

import (
	"encoding/json"
	"math/rand"
	"strings"
	"testing"
)

func findAction(g *Game, kind string) *Action {
	for _, a := range g.LegalActions() {
		if a.Kind == kind {
			aa := a
			return &aa
		}
	}
	return nil
}

func TestMCBirdsongAndBuild(t *testing.T) {
	g := newTestGame(t)
	if g.Clearings["C1"].Wood != 1 {
		t.Fatalf("C1 wood after birdsong = %d, want 1", g.Clearings["C1"].Wood)
	}
	// Advance to Daylight, then finish the craft step.
	if err := g.Apply(Action{ID: "pass"}); err != nil {
		t.Fatal(err)
	}
	if err := g.Apply(Action{ID: "mc-done-crafting"}); err != nil {
		t.Fatal(err)
	}
	if g.Phase != "D" || g.ActionsLeft != 3 {
		t.Fatalf("phase=%s actions=%d", g.Phase, g.ActionsLeft)
	}
	ba := findAction(g, "mc-build")
	if ba == nil {
		t.Fatal("no build action available")
	}
	vpBefore := g.Players[MC].VP
	woodBefore := totalWood(g)
	if err := g.Apply(*ba); err != nil {
		t.Fatal(err)
	}
	if g.Players[MC].VP <= vpBefore {
		t.Fatalf("build did not score (VP %d)", g.Players[MC].VP)
	}
	if totalWood(g) >= woodBefore {
		t.Fatalf("build did not spend wood")
	}
	if g.ActionsLeft != 2 {
		t.Fatalf("actions left = %d", g.ActionsLeft)
	}
}

func totalWood(g *Game) int {
	n := 0
	for _, c := range g.Clearings {
		n += c.Wood
	}
	return n
}

func TestOverwork(t *testing.T) {
	g := newTestGame(t)
	_ = g.Apply(Action{ID: "pass"}) // Daylight
	_ = g.Apply(Action{ID: "mc-done-crafting"})
	p := g.Players[MC]
	// Give MC a card matching C1's suit (Fox) and clear its hand.
	c1suit := g.Clearings["C1"].Suit
	var card string
	for _, id := range []string{"F01", "F02", "F03"} {
		if cardSuit(id) == c1suit {
			card = id
		}
	}
	if card == "" {
		card = "B01" // bird matches any
	}
	p.Hand = []string{card}
	oa := findAction(g, "mc-overwork")
	if oa == nil {
		t.Fatal("no overwork action")
	}
	woodBefore := g.Clearings[oa.Clearing].Wood
	if err := g.Apply(*oa); err != nil {
		t.Fatal(err)
	}
	if g.Clearings[oa.Clearing].Wood != woodBefore+1 {
		t.Fatalf("overwork did not add wood")
	}
	if g.Players[MC].WoodSupply != 6 {
		t.Fatalf("wood supply = %d", g.Players[MC].WoodSupply)
	}
}

func TestWASympathyTrack(t *testing.T) {
	g := newTestGame(t)
	p := g.Players[WA]
	// Put WA on turn with known supporters.
	g.Current = WA
	g.beginTurn()
	p.Supporters = []string{"F04", "R04"} // both match Fox? R04 is Rabbit. Use matching.
	// Find a spread action; ensure affordability by stocking supporters of the clearing suit.
	sa := findAction(g, "spread")
	if sa == nil {
		// stock supporters matching any clearing suit
		p.Supporters = []string{"F04", "F05", "R04", "R05", "M04", "M05", "B04", "B05", "B11", "B12"}
		sa = findAction(g, "spread")
	}
	if sa == nil {
		t.Fatal("no spread action")
	}
	vpBefore := p.VP
	if err := g.Apply(*sa); err != nil {
		t.Fatal(err)
	}
	if g.sympathyOnMap() != 1 {
		t.Fatalf("sympathy on map = %d", g.sympathyOnMap())
	}
	// First sympathy scores 0 VP.
	if p.VP != vpBefore {
		t.Fatalf("first sympathy should score 0 VP, got %d", p.VP-vpBefore)
	}
}

func TestOutrage(t *testing.T) {
	g := newTestGame(t)
	// Place WA sympathy in C1, then have MC move in.
	g.Clearings["C1"].Sympathy = WA
	p := g.Players[MC]
	p.Hand = []string{"F01", "F02"}
	supBefore := len(g.Players[WA].Supporters)
	g.checkOutrageAfterMove(MC, "C1")
	if len(g.Players[WA].Supporters) != supBefore+1 {
		t.Fatalf("outrage did not add a supporter (%d -> %d)", supBefore, len(g.Players[WA].Supporters))
	}
}

func TestVBExplore(t *testing.T) {
	g := newTestGame(t)
	p := g.Players[VB]
	// Move pawn to a ruin clearing and give a torch.
	ruin := GetMap("autumn").Ruins[0]
	p.Pawn = ruin
	g.giveVBItem(p, "torch")
	g.Current = VB
	g.Phase = "D"
	ea := findAction(g, "vb-explore")
	if ea == nil {
		t.Fatal("no explore action")
	}
	itemsBefore := len(p.Items)
	vpBefore := p.VP
	if err := g.Apply(*ea); err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != itemsBefore+1 {
		t.Fatalf("explore did not gain an item")
	}
	if p.VP != vpBefore+1 {
		t.Fatalf("explore did not score 1 VP")
	}
	if g.Clearings[ruin].RuinItem != "" {
		t.Fatalf("ruin item not cleared")
	}
}

func TestCraftItem(t *testing.T) {
	g := newTestGame(t)
	g.Current = MC
	g.Phase = "D"
	g.ActionsLeft = 3
	g.MCDayStage = "craft"
	// Give MC a craftable item card with a matching workshop.
	// C5 has a workshop (Rabbit suit). Give a card costing 1 Rabbit (R06 boot).
	g.Players[MC].Hand = []string{"R06"}
	ca := findAction(g, "craft")
	if ca == nil {
		t.Fatal("no craft action")
	}
	vpBefore := g.Players[MC].VP
	if err := g.Apply(*ca); err != nil {
		t.Fatal(err)
	}
	if g.Players[MC].VP != vpBefore+1 {
		t.Fatalf("crafting boot should score 1 VP, got %d", g.Players[MC].VP-vpBefore)
	}
	found := false
	for _, it := range g.Players[MC].CraftedItems {
		if it == "boot" {
			found = true
		}
	}
	if !found {
		t.Fatal("boot not added to crafted items")
	}
}

func TestBattleResolution(t *testing.T) {
	g := newTestGame(t)
	// MC attacks ED in C3.
	g.Current = MC
	g.Phase = "D"
	g.ActionsLeft = 3
	g.addWarrior(MC, "C3", 3)
	ba := findAction(g, "battle")
	if ba == nil {
		t.Fatal("no battle action")
	}
	if err := g.Apply(*ba); err != nil {
		t.Fatal(err)
	}
	// Resolve any pending (ambush/effects/hits) deterministically.
	steps := 0
	for g.Pending != nil && steps < 50 {
		acts := g.LegalActions()
		if len(acts) == 0 {
			break
		}
		// prefer skip for ambush, and first hit option
		pick := acts[0]
		for _, a := range acts {
			if a.Kind == "battle-skip" {
				pick = a
				break
			}
		}
		if err := g.Apply(pick); err != nil {
			t.Fatal(err)
		}
		steps++
	}
	if g.Battle != nil {
		t.Fatal("battle did not end")
	}
}

func TestLongSelfPlay(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	winners := map[Faction]int{}
	for game := 0; game < 60; game++ {
		g := NewGame([]Faction{MC, ED, WA, VB}, MC, uint64(game*104729+3))
		Setup(g, SetupOptions{Seed: uint64(game*104729 + 3)})
		steps := 0
		for steps < 6000 {
			if w, ok := g.WinnerFaction(); ok {
				winners[w]++
				break
			}
			acts := g.LegalActions()
			if len(acts) == 0 {
				t.Fatalf("game %d: no legal actions (round %d phase %s current %s)", game, g.Round, g.Phase, g.Current)
			}
			if err := g.Apply(acts[rng.Intn(len(acts))]); err != nil {
				t.Fatalf("game %d step %d: %v", game, steps, err)
			}
			steps++
		}
	}
	t.Logf("winners across 60 random games: %v", winners)
	if len(winners) < 2 {
		t.Logf("note: only %d distinct winner(s) under random play", len(winners))
	}
}

func TestDominanceGoesAvailable(t *testing.T) {
	g := newTestGame(t)
	_ = g.Apply(Action{ID: "pass"}) // Daylight
	_ = g.Apply(Action{ID: "mc-done-crafting"})
	g.Players[MC].Hand = []string{"F02"} // Fox Dominance, matches C1 (Fox) sawmill
	oa := findAction(g, "mc-overwork")
	if oa == nil {
		t.Fatal("no overwork action")
	}
	if err := g.Apply(*oa); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range g.AvailableDominance {
		if d == "F02" {
			found = true
		}
	}
	if !found {
		t.Fatalf("dominance card not moved to available area: %v", g.AvailableDominance)
	}
}

func TestNewClutch(t *testing.T) {
	g := newTestGame(t)
	p := g.Players[ED]
	p.RetiredLeaders = []string{"builder", "charismatic", "commander", "despot"}
	g.turmoil()
	if len(p.RetiredLeaders) != 0 {
		t.Fatalf("A New Clutch should flip all leaders face up, got %v", p.RetiredLeaders)
	}
}

func TestVBAllyBattle(t *testing.T) {
	g := newTestGame(t)
	p := g.Players[VB]
	p.Pawn = "C3"
	p.Relationships[MC] = "allied"
	g.addWarrior(MC, "C3", 3)
	g.giveVBItem(p, "sword")
	g.Current = VB
	g.Phase = "D"
	aa := findAction(g, "vb-battle-ally")
	if aa == nil {
		t.Fatal("no ally battle action")
	}
	if err := g.Apply(*aa); err != nil {
		t.Fatal(err)
	}
	if g.Battle == nil || g.Battle.AllyTarget != MC {
		t.Fatalf("battle ally target not set: %+v", g.Battle)
	}
	if got := g.maxHits(VB, "C3"); got != 4 {
		t.Fatalf("maxHits with ally = %d, want 4 (1 sword + 3 ally)", got)
	}
}

func TestCodebreakers(t *testing.T) {
	g := newTestGame(t)
	g.Current = MC
	g.Phase = "D"
	g.ActionsLeft = 3
	p := g.Players[MC]
	p.Crafted = append(p.Crafted, "M12") // Codebreakers
	ca := findAction(g, "codebreakers")
	if ca == nil {
		t.Fatal("no codebreakers action")
	}
	if err := g.Apply(*ca); err != nil {
		t.Fatal(err)
	}
	if !p.UsedThisTurn["codebreakers"] {
		t.Fatal("codebreakers not marked used")
	}
}

func TestFirstActionTerminates(t *testing.T) {
	for seed := uint64(1); seed <= 15; seed++ {
		g := NewGame([]Faction{MC, ED, WA, VB}, MC, seed)
		Setup(g, SetupOptions{Seed: seed})
		steps := 0
		for steps < 8000 {
			if _, ok := g.WinnerFaction(); ok {
				break
			}
			acts := g.LegalActions()
			if len(acts) == 0 {
				t.Fatalf("seed %d: no legal actions at round %d phase %s", seed, g.Round, g.Phase)
			}
			if err := g.Apply(acts[0]); err != nil {
				t.Fatalf("seed %d step %d: %v", seed, steps, err)
			}
			steps++
		}
		if steps >= 8000 {
			t.Fatalf("seed %d did not terminate within 8000 steps (round %d)", seed, g.Round)
		}
	}
}

func TestInteractiveSetup(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	BeginSetup(g)
	if !g.SetupMode || g.SetupStage != "MC_KEEP" {
		t.Fatalf("expected setup mode at MC_KEEP, got %v %s", g.SetupMode, g.SetupStage)
	}
	applyID := func(id string) {
		for _, a := range g.LegalActions() {
			if a.ID == id {
				if err := g.Apply(a); err != nil {
					t.Fatalf("apply %s: %v", id, err)
				}
				return
			}
		}
		t.Fatalf("setup action %s not legal", id)
	}
	applyID("setup-mc-keep|C2")
	applyID("setup-mc-build|sawmill|C2")
	applyID("setup-mc-build|workshop|C2")
	applyID("setup-mc-build|recruiter|C5")
	applyID("setup-ed-corner|C4")
	applyID("setup-ed-leader|commander")
	applyID("setup-vb-character|thief")
	applyID("setup-vb-forest|AutumnE")
	if g.SetupMode {
		t.Fatalf("setup did not complete")
	}
	if g.Players[MC].KeepClearing != "C2" {
		t.Fatalf("keep = %s", g.Players[MC].KeepClearing)
	}
	if g.Players[ED].Leader != "commander" {
		t.Fatalf("ED leader = %s", g.Players[ED].Leader)
	}
	if g.Players[VB].Character != "thief" || g.Players[VB].Pawn != "AutumnE" {
		t.Fatalf("VB = %s at %s", g.Players[VB].Character, g.Players[VB].Pawn)
	}
	if len(g.Players[MC].Hand) != 3 || len(g.Players[VB].Hand) != 3 {
		t.Fatalf("starting hands wrong: %d %d", len(g.Players[MC].Hand), len(g.Players[VB].Hand))
	}
}

func TestStateRoundTrip(t *testing.T) {
	// Simulates localStorage persistence: marshal the game, restore it, and
	// verify it continues identically.
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 11)
	BeginSetup(g)
	// Run interactive setup with defaults.
	for _, id := range []string{
		"setup-mc-keep|C1", "setup-mc-build|sawmill|C1", "setup-mc-build|workshop|C5",
		"setup-mc-build|recruiter|C9", "setup-ed-corner|C3", "setup-ed-leader|despot",
		"setup-vb-character|tinker", "setup-vb-forest|AutumnN",
	} {
		for _, a := range g.LegalActions() {
			if a.ID == id {
				if err := g.Apply(a); err != nil {
					t.Fatalf("apply %s: %v", id, err)
				}
			}
		}
	}
	for i := 0; i < 120; i++ {
		acts := g.LegalActions()
		if len(acts) == 0 {
			break
		}
		if err := g.Apply(acts[0]); err != nil {
			t.Fatal(err)
		}
	}
	b, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	restored := &Game{}
	if err := json.Unmarshal(b, restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Legal action IDs must match after restore.
	before := g.LegalActions()
	after := restored.LegalActions()
	if len(before) != len(after) {
		t.Fatalf("legal action count differs after restore: %d vs %d", len(before), len(after))
	}
	for i := range before {
		if before[i].ID != after[i].ID {
			t.Fatalf("legal action %d differs: %s vs %s", i, before[i].ID, after[i].ID)
		}
	}
	// Both continue identically.
	if len(before) > 0 {
		if err := g.Apply(before[0]); err != nil {
			t.Fatal(err)
		}
		if err := restored.Apply(after[0]); err != nil {
			t.Fatal(err)
		}
		if stateDigest(g) != stateDigest(restored) {
			t.Fatalf("states diverged after restore")
		}
	}
}

func TestEyrieDecreeResolutionAndTurmoil(t *testing.T) {
	// A resolvable Build card offers resolution; an unresolvable one offers Turmoil.
	g := newTestGame(t)
	g.Relaxed = false // test canonical rule requirements
	g.Current = ED
	g.Phase = "D"
	p := g.Players[ED]
	p.Hand = nil

	// ED rules C3 (Rabbit roost). A Rabbit Recruit card is resolvable there.
	p.Decree = map[string][]string{"RECRUIT": {"R01"}}
	g.buildDecreeQueue()
	hasBuild, hasTurmoil := false, false
	for _, a := range g.LegalActions() {
		if a.Kind == "decree-recruit" {
			hasBuild = true
		}
		if a.Kind == "ed-turmoil" {
			hasTurmoil = true
		}
	}
	if !hasBuild {
		t.Fatal("resolvable Rabbit Recruit card should offer a decree-recruit option")
	}
	if hasTurmoil {
		t.Fatal("Turmoil must not be offered while a card can still be resolved")
	}

	// A Mouse Build card is unresolvable (ED rules no Mouse clearing).
	p.Decree = map[string][]string{"BUILD": {"M01"}}
	g.buildDecreeQueue()
	hasBuild, hasTurmoil = false, false
	for _, a := range g.LegalActions() {
		if a.Kind == "decree-build" {
			hasBuild = true
		}
		if a.Kind == "ed-turmoil" {
			hasTurmoil = true
		}
	}
	if hasBuild {
		t.Fatal("unresolvable card should not offer a resolution")
	}
	if !hasTurmoil {
		t.Fatal("unresolvable card must offer Turmoil")
	}

	// Applying Turmoil runs the penalties, then requires a new leader, then Evening.
	vpBefore := p.VP
	if err := g.Apply(Action{ID: "ed-turmoil", Kind: "ed-turmoil", Faction: ED}); err != nil {
		t.Fatalf("turmoil: %v", err)
	}
	if !g.EDNeedsLeader {
		t.Fatal("turmoil should require choosing a new leader")
	}
	if p.Decree["BUILD"] != nil && len(p.Decree["BUILD"]) > 0 {
		t.Fatalf("decree should be purged, got %v", p.Decree)
	}
	if p.VP > vpBefore {
		t.Fatal("turmoil should not increase VP")
	}
	// Choose a leader -> Evening.
	la := findAction(g, "leader")
	if la == nil {
		t.Fatal("expected a leader choice after turmoil")
	}
	if err := g.Apply(*la); err != nil {
		t.Fatalf("leader: %v", err)
	}
	if g.Phase != "E" {
		t.Fatalf("after turmoil + new leader, expected Evening, got %s", g.Phase)
	}
}

func TestMoveQuantities(t *testing.T) {
	g := newTestGame(t)
	g.Current = MC
	g.Phase = "D"
	g.ActionsLeft = 3
	// Put 3 Marquise warriors in C5 (adjacent to C1, which MC rules).
	g.Clearings["C5"].Warriors[MC] = 3

	amounts := map[int]bool{}
	for _, a := range g.LegalActions() {
		if a.Kind == "move" && a.From == "C5" && a.To == "C1" {
			amounts[a.Amount] = true
		}
	}
	for q := 1; q <= 3; q++ {
		if !amounts[q] {
			t.Fatalf("missing move option for quantity %d", q)
		}
	}
	// Apply a partial move of 2.
	var pick *Action
	for _, a := range g.LegalActions() {
		if a.Kind == "move" && a.From == "C5" && a.To == "C1" && a.Amount == 2 {
			aa := a
			pick = &aa
		}
	}
	if pick == nil {
		t.Fatal("no move-2 option")
	}
	before := g.Clearings["C1"].Warriors[MC]
	if err := g.Apply(*pick); err != nil {
		t.Fatal(err)
	}
	if g.Clearings["C5"].Warriors[MC] != 1 {
		t.Fatalf("C5 = %d, want 1", g.Clearings["C5"].Warriors[MC])
	}
	if g.Clearings["C1"].Warriors[MC] != before+2 {
		t.Fatalf("C1 = %d, want %d", g.Clearings["C1"].Warriors[MC], before+2)
	}

	// Eyrie decree Move offers every quantity too.
	g.Relaxed = false
	g.Current = ED
	g.Phase = "D"
	p := g.Players[ED]
	p.Decree = map[string][]string{"MOVE": {"R01"}} // Rabbit card, C3 is Rabbit
	g.buildDecreeQueue()
	dq := map[int]bool{}
	for _, a := range g.LegalActions() {
		if a.Kind == "decree-move" && a.From == "C3" {
			dq[a.Amount] = true
		}
	}
	n := g.Clearings["C3"].Warriors[ED]
	if n < 2 {
		t.Fatalf("expected several ED warriors at C3, got %d", n)
	}
	for q := 1; q <= n; q++ {
		if !dq[q] {
			t.Fatalf("decree move missing quantity %d of %d", q, n)
		}
	}
}

func TestEyrieCraftBeforeDecree(t *testing.T) {
	g := newTestGame(t)
	g.Current = ED
	g.Phase = "D"
	p := g.Players[ED]
	p.Decree = map[string][]string{"RECRUIT": {"R01"}}
	g.buildDecreeQueue()
	g.EDDayStage = "craft"

	hasDone, hasDecree := false, false
	for _, a := range g.LegalActions() {
		if a.Kind == "ed-done-crafting" {
			hasDone = true
		}
		if a.Kind == "decree-recruit" {
			hasDecree = true
		}
	}
	if !hasDone {
		t.Fatal("craft step should offer 'done crafting'")
	}
	if hasDecree {
		t.Fatal("the Decree must not resolve during the craft step")
	}
	if err := g.Apply(Action{ID: "ed-done-crafting", Kind: "ed-done-crafting", Faction: ED}); err != nil {
		t.Fatal(err)
	}
	if g.EDDayStage != "decree" {
		t.Fatalf("stage = %q, want decree", g.EDDayStage)
	}
	hasDone, hasDecree = false, false
	for _, a := range g.LegalActions() {
		if a.Kind == "ed-done-crafting" {
			hasDone = true
		}
		if a.Kind == "decree-recruit" {
			hasDecree = true
		}
	}
	if hasDone {
		t.Fatal("'done crafting' should not persist into the decree step")
	}
	if !hasDecree {
		t.Fatal("expected decree resolution after crafting")
	}
}

func TestEyrieViziersResolve(t *testing.T) {
	g := newTestGame(t) // Despot: viziers in MOVE and BUILD
	g.Current = ED
	g.Phase = "D"
	g.EDDayStage = "decree"
	g.buildDecreeQueue()
	vcount := 0
	for _, it := range g.DecreeQueue {
		if it.Card == "VIZIER" {
			vcount++
		}
	}
	if vcount != 2 {
		t.Fatalf("expected 2 viziers in the resolution queue, got %d", vcount)
	}
	if g.DecreeQueue[0].Card != "VIZIER" || g.DecreeQueue[0].Column != "MOVE" {
		t.Fatalf("front decree item = %+v", g.DecreeQueue[0])
	}
	// A vizier is a bird card, so its Move resolves from any clearing (C3).
	found := false
	for _, a := range g.LegalActions() {
		if a.Kind == "decree-move" && a.From == "C3" {
			found = true
		}
	}
	if !found {
		t.Fatal("vizier Move should be resolvable from any clearing")
	}
}

func TestEyrieRoostRequiresRule(t *testing.T) {
	g := newTestGame(t)
	g.Current = ED
	g.Phase = "D"
	g.EDDayStage = "decree"
	p := g.Players[ED]

	// A Mouse Build card: ED rules no Mouse clearing, so no build option.
	p.Decree = map[string][]string{"BUILD": {"M01"}}
	g.buildDecreeQueue()
	for _, a := range g.LegalActions() {
		if a.Kind == "decree-build" {
			t.Fatalf("must not build a roost without ruling the clearing (%s)", a.Clearing)
		}
	}

	// Remove the setup roost at C3 (ED rules C3) and use a Rabbit card: now
	// the roost may be built there.
	cl := g.Clearings["C3"]
	for i, b := range cl.Buildings {
		if b.Owner == ED && b.Type == "roost" {
			cl.Buildings = append(cl.Buildings[:i], cl.Buildings[i+1:]...)
			break
		}
	}
	p.Decree = map[string][]string{"BUILD": {"R01"}}
	g.buildDecreeQueue()
	found := false
	for _, a := range g.LegalActions() {
		if a.Kind == "decree-build" && a.Clearing == "C3" {
			found = true
		}
	}
	if !found {
		t.Fatal("should be able to build a roost in a ruled, roost-free clearing")
	}
}

func TestEyrieCornerOppositeMC(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	BeginSetup(g)
	for _, a := range g.LegalActions() {
		if a.ID == "setup-mc-keep|C2" {
			if err := g.Apply(a); err != nil {
				t.Fatal(err)
			}
		}
	}
	for {
		placed := false
		for _, a := range g.LegalActions() {
			if a.Kind == "setup-mc-build" {
				if err := g.Apply(a); err != nil {
					t.Fatal(err)
				}
				placed = true
				break
			}
		}
		if !placed {
			break
		}
	}
	if g.SetupStage != "ED_CORNER" {
		t.Fatalf("stage = %s", g.SetupStage)
	}
	var corners []string
	for _, a := range g.LegalActions() {
		if a.Kind == "setup-ed-corner" {
			corners = append(corners, a.Clearing)
		}
	}
	if len(corners) != 1 || corners[0] != "C4" {
		t.Fatalf("with MC at C2, Eyrie must start at C4 (opposite); got %v", corners)
	}
}

func TestVBHostilityOnlyWhenVBActs(t *testing.T) {
	g := newTestGame(t)
	vb := g.Players[VB]
	g.addWarrior(ED, "C3", 1)
	// MC removes an ED warrior: ED must NOT become hostile to the Vagabond.
	g.removePiece(MC, ED, "C3")
	if vb.Relationships[ED] == "hostile" {
		t.Fatal("a warrior removed by another faction must not make that faction Hostile to the Vagabond")
	}
	// The Vagabond removes an MC warrior: MC becomes hostile.
	g.addWarrior(MC, "C5", 1)
	g.removePiece(VB, MC, "C5")
	if vb.Relationships[MC] != "hostile" {
		t.Fatal("Vagabond removing a warrior should make that faction Hostile")
	}
}

func TestOutrageOnSympathyRemoval(t *testing.T) {
	g := newTestGame(t)
	cl := g.Clearings["C5"]
	cl.Sympathy = WA
	cl.Tokens = append(cl.Tokens, Token{WA, "sympathy"})
	g.Players[MC].Hand = []string{"M01"} // Mouse; C5 is Rabbit, so no match
	before := len(g.Players[WA].Supporters)
	g.removePiece(MC, WA, "C5") // e.g. via a Favor/Revolt effect
	if cl.Sympathy == WA {
		t.Fatal("sympathy token was not removed")
	}
	if len(g.Players[WA].Supporters) != before+1 {
		t.Fatalf("Outrage should add a supporter when another player removes sympathy (%d -> %d)", before, len(g.Players[WA].Supporters))
	}
}

func TestVBAidTakeItem(t *testing.T) {
	g := newTestGame(t)
	p := g.Players[VB]
	p.Pawn = "C3" // ED roost + warriors here (Rabbit)
	p.Hand = []string{"R01"}
	g.giveVBItem(p, "hammer")
	g.Players[ED].CraftedItems = []string{"sword"}
	g.Current = VB
	g.Phase = "D"

	var pick *Action
	for _, a := range g.LegalActions() {
		if a.Kind == "vb-aid" && a.Item == "sword" {
			aa := a
			pick = &aa
		}
	}
	if pick == nil {
		t.Fatal("Aid should offer taking a crafted item")
	}
	if err := g.Apply(*pick); err != nil {
		t.Fatal(err)
	}
	if len(g.Players[ED].CraftedItems) != 0 {
		t.Fatalf("crafted item should have been taken: %v", g.Players[ED].CraftedItems)
	}
	found := false
	for _, it := range p.Items {
		if it.Type == "sword" {
			found = true
		}
	}
	if !found {
		t.Fatal("Vagabond did not receive the sword")
	}
}

func TestFieldHospitalsOnEffectRemoval(t *testing.T) {
	g := newTestGame(t)
	g.addWarrior(MC, "C5", 2)
	g.Players[MC].Hand = []string{"R01"} // Rabbit, matches C5
	g.removePiece(ED, MC, "C5")          // as if removed by a card effect
	if len(g.FH) == 0 {
		t.Fatal("Field Hospitals record not created")
	}
	g.maybeFieldHospitals()
	if g.Pending == nil || g.Pending.Kind != PendingFieldHospitals {
		t.Fatalf("Field Hospitals should be offered after a non-battle removal, pending=%+v", g.Pending)
	}
	var fh *Action
	for _, a := range g.LegalActions() {
		if a.Kind == "field-hospitals" {
			aa := a
			fh = &aa
		}
	}
	if fh == nil {
		t.Fatal("no Field Hospitals action offered")
	}
	if err := g.Apply(*fh); err != nil {
		t.Fatal(err)
	}
	if g.Clearings[g.Players[MC].KeepClearing].Warriors[MC] < 1 {
		t.Fatal("saved warrior did not reach the keep clearing")
	}
}

func TestMCCraftBeforeActions(t *testing.T) {
	g := newTestGame(t)
	if err := g.Apply(Action{ID: "pass"}); err != nil {
		t.Fatal(err)
	}
	if g.MCDayStage != "craft" {
		t.Fatalf("stage = %q, want craft", g.MCDayStage)
	}
	hasDone, hasBuild := false, false
	for _, a := range g.LegalActions() {
		if a.Kind == "mc-done-crafting" {
			hasDone = true
		}
		if a.Kind == "mc-build" {
			hasBuild = true
		}
	}
	if !hasDone {
		t.Fatal("craft step should offer 'done crafting'")
	}
	if hasBuild {
		t.Fatal("no actions should be available during the craft step")
	}
	if err := g.Apply(Action{ID: "mc-done-crafting"}); err != nil {
		t.Fatal(err)
	}
	if g.MCDayStage != "actions" {
		t.Fatalf("stage = %q, want actions", g.MCDayStage)
	}
}

func TestMCBuildingReturnsToTrack(t *testing.T) {
	g := newTestGame(t)
	p := g.Players[MC]
	total := p.Sawmills + p.Workshops + p.Recruiters
	// Clear the garrison warrior at C1 so a building can be removed.
	for g.Clearings["C1"].Warriors[MC] > 0 {
		g.removePiece(ED, MC, "C1")
	}
	if _, ok := g.removePiece(ED, MC, "C1"); !ok {
		t.Fatal("expected a building at C1")
	}
	if got := p.Sawmills + p.Workshops + p.Recruiters; got != total+1 {
		t.Fatalf("destroyed building should return to its track: %d -> %d", total, got)
	}
}

func TestWABaseRemovedInBattle(t *testing.T) {
	g := newTestGame(t)
	p := g.Players[WA]
	g.Clearings["C5"].Buildings = append(g.Clearings["C5"].Buildings, Building{WA, "base-rabbit"})
	p.Bases[Rabbit] = true
	p.Officers = 3
	p.Supporters = []string{"R01", "R02", "M01"}
	idx := -1
	for i, b := range g.Clearings["C5"].Buildings {
		if b.Owner == WA {
			idx = i
		}
	}
	g.Battle = &BattleState{Clearing: "C5", Attacker: MC, Defender: WA, Stage: StageHits, HitSide: WA, Remaining: 1}
	g.applyBattleHit(Action{Kind: "battle-hit", Piece: "building", Amount: idx, Faction: WA})
	if p.Officers != 1 {
		t.Fatalf("officers should be 1 after losing half (rounded up) of 3, got %d", p.Officers)
	}
	for _, c := range p.Supporters {
		if matches(cardSuit(c), Rabbit) {
			t.Fatalf("matching supporter should be discarded on base removal: %s", c)
		}
	}
	if p.Bases[Rabbit] {
		t.Fatal("base should no longer be considered placed")
	}
}

func TestMCBattleTargets(t *testing.T) {
	g := newTestGame(t)
	g.Current = MC
	g.Phase = "D"
	g.MCDayStage = "actions"
	g.ActionsLeft = 3

	// MC can battle the Eyrie once it has warriors in their clearing.
	g.addWarrior(MC, "C3", 2)
	// MC can battle the Vagabond pawn when it is in a clearing with MC warriors.
	g.Players[VB].Pawn = "C1"

	var vsED, vsVB bool
	for _, a := range g.LegalActions() {
		if a.Kind == "battle" && a.Clearing == "C3" && a.Target == ED {
			vsED = true
		}
		if a.Kind == "battle" && a.Clearing == "C1" && a.Target == VB {
			vsVB = true
		}
	}
	if !vsED {
		t.Fatal("MC should be able to battle the Eyrie where it has warriors")
	}
	if !vsVB {
		t.Fatal("MC should be able to battle the Vagabond pawn")
	}
}

func TestWASympathySpendRecorded(t *testing.T) {
	g := newTestGame(t)
	g.Current = WA
	g.beginTurn()
	p := g.Players[WA]
	// Two Fox supporters and a wild Bird; C6 is a Fox clearing with no sympathy.
	p.Supporters = []string{"F04", "F05", "B11"}

	found := map[string]bool{}
	for _, a := range g.LegalActions() {
		if a.Kind == "spread" && a.Clearing == "C6" && len(a.Cards) == 1 {
			found[a.Cards[0]] = true
		}
	}
	for _, want := range []string{"F04", "F05", "B11"} {
		if !found[want] {
			t.Fatalf("expected a spread option spending %s; got %v", want, found)
		}
	}

	// Spend the Bird specifically.
	var pick *Action
	for _, a := range g.LegalActions() {
		if a.Kind == "spread" && a.Clearing == "C6" && len(a.Cards) == 1 && a.Cards[0] == "B11" {
			aa := a
			pick = &aa
		}
	}
	if pick == nil {
		t.Fatal("no spread action spending the Bird")
	}
	if err := g.Apply(*pick); err != nil {
		t.Fatal(err)
	}
	for _, c := range p.Supporters {
		if c == "B11" {
			t.Fatal("the spent Bird should have been removed")
		}
	}
	if len(p.Supporters) != 2 {
		t.Fatalf("supporters = %v", p.Supporters)
	}
	// The log records which card was spent.
	last := g.Log[len(g.Log)-1].Text
	if !strings.Contains(last, "B11") {
		t.Fatalf("log should record the spent supporter: %q", last)
	}
	// And the RMN annotation records it.
	rmn := g.RMNLog[len(g.RMNLog)-1]
	if !strings.Contains(rmn, "spend=B11") {
		t.Fatalf("RMN should record spend=B11: %q", rmn)
	}
}

func TestRuinOccupiesBuildSlot(t *testing.T) {
	g := newTestGame(t)
	cl := g.Clearings["C6"] // Fox, 2 slots, has a ruin
	if !cl.Ruin || cl.Slots != 2 {
		t.Fatalf("C6 = %+v", cl)
	}
	if cl.FreeSlots() != 1 {
		t.Fatalf("with a ruin, free slots = %d, want 1", cl.FreeSlots())
	}
	cl.Ruin = false
	cl.RuinItem = ""
	if cl.FreeSlots() != 2 {
		t.Fatalf("after the ruin is removed, free slots = %d, want 2", cl.FreeSlots())
	}
	cl.Buildings = append(cl.Buildings, Building{MC, "sawmill"})
	if cl.FreeSlots() != 1 {
		t.Fatalf("free slots = %d, want 1", cl.FreeSlots())
	}
}

func TestSetupFactionSubsets(t *testing.T) {
	subsets := [][]Faction{
		{MC, ED}, {MC, ED, VB}, {MC, WA, VB}, {ED, WA, VB}, {MC, ED, WA}, {MC, ED, WA, VB},
	}
	for _, fs := range subsets {
		g := NewGame(fs, fs[0], 7)
		BeginSetup(g)
		steps := 0
		for g.SetupMode && steps < 60 {
			acts := g.LegalActions()
			if len(acts) == 0 {
				t.Fatalf("%v: stuck at stage %s", fs, g.SetupStage)
			}
			if err := g.Apply(acts[0]); err != nil {
				t.Fatalf("%v: apply %s: %v", fs, acts[0].ID, err)
			}
			steps++
		}
		if g.SetupMode {
			t.Fatalf("%v: setup did not finish", fs)
		}
		// Play a few turns to be sure nothing else assumes all four factions.
		for i := 0; i < 40; i++ {
			acts := g.LegalActions()
			if len(acts) == 0 || len(g.Winner) > 0 {
				break
			}
			if err := g.Apply(acts[0]); err != nil {
				t.Fatalf("%v: turn apply: %v", fs, err)
			}
		}
	}
}

// TestBattleHitsWithoutVagabond covers the nil-pointer crash when a game has no
// Vagabond but a battle hit is assigned (battle.go read g.Players[VB] directly).
func TestBattleHitsWithoutVagabond(t *testing.T) {
	subsets := [][]Faction{{MC, ED}, {MC, WA}, {ED, WA}, {MC, ED, WA}}
	for _, fs := range subsets {
		g := NewGame(fs, fs[0], 5)
		BeginSetup(g)
		for g.SetupMode {
			acts := g.LegalActions()
			if len(acts) == 0 {
				t.Fatalf("%v: setup stalled", fs)
			}
			if err := g.Apply(acts[0]); err != nil {
				t.Fatalf("%v: setup: %v", fs, err)
			}
		}
		att, def := fs[0], fs[1]
		g.addWarrior(att, "C1", 3)
		g.addWarrior(def, "C1", 3)
		g.Battle = &BattleState{
			Clearing: "C1", Attacker: att, Defender: def,
			Stage: StageHits, HitSide: def, Remaining: 1, AtkHits: 1,
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%v: panic assigning a battle hit: %v", fs, r)
				}
			}()
			g.applyBattleHit(Action{Kind: "battle-hit", Faction: def, Piece: "warrior", Clearing: "C1"})
		}()
		if got := g.Clearings["C1"].Warriors[def]; got != 2 {
			t.Fatalf("%v: defender warriors = %d, want 2", fs, got)
		}
	}
}

// TestFullBattleWithoutVagabond drives a complete battle through the action
// pipeline in a two-player MC-vs-ED game (the reported crash).
func TestFullBattleWithoutVagabond(t *testing.T) {
	g := NewGame([]Faction{MC, ED}, MC, 99)
	BeginSetup(g)
	for g.SetupMode {
		acts := g.LegalActions()
		if len(acts) == 0 {
			t.Fatal("setup stalled")
		}
		if err := g.Apply(acts[0]); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	g.Current = MC
	g.Phase = "D"
	g.MCDayStage = "actions"
	g.ActionsLeft = 3
	g.addWarrior(MC, "C1", 5)
	g.addWarrior(ED, "C1", 5)

	var battle *Action
	for _, a := range g.LegalActions() {
		if a.Kind == "battle" && a.Clearing == "C1" && a.Target == ED {
			aa := a
			battle = &aa
			break
		}
	}
	if battle == nil {
		t.Fatal("no MC battle action against ED in C1")
	}
	if err := g.Apply(*battle); err != nil {
		t.Fatalf("start battle: %v", err)
	}
	steps := 0
	for g.Pending != nil && steps < 100 {
		acts := g.LegalActions()
		if len(acts) == 0 {
			break
		}
		pick := acts[0]
		for _, a := range acts {
			if a.Kind == "battle-skip" {
				pick = a
				break
			}
		}
		if err := g.Apply(pick); err != nil {
			t.Fatalf("battle step: %v", err)
		}
		steps++
	}
	if g.Battle != nil {
		t.Fatal("battle did not end")
	}
}

// TestRandomPlayAcrossSubsets fuzzes every 2-4 player faction combination,
// including battles, to catch any other player-count assumptions.
func TestRandomPlayAcrossSubsets(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	subsets := [][]Faction{
		{MC, ED}, {MC, WA}, {MC, VB}, {ED, WA}, {ED, VB}, {WA, VB},
		{MC, ED, WA}, {MC, ED, VB}, {MC, WA, VB}, {ED, WA, VB},
		{MC, ED, WA, VB},
	}
	for _, fs := range subsets {
		g := NewGame(fs, fs[0], uint64(len(fs)*17+3))
		BeginSetup(g)
		for step := 0; step < 2000; step++ {
			if _, ok := g.WinnerFaction(); ok {
				break
			}
			acts := g.LegalActions()
			if len(acts) == 0 {
				break
			}
			if err := g.Apply(acts[rng.Intn(len(acts))]); err != nil {
				t.Fatalf("%v step %d: %v", fs, step, err)
			}
		}
	}
}

func TestRemoveFaction(t *testing.T) {
	g := newTestGame(t)
	// ED is the current, mid-battle player with pieces on the board.
	g.addWarrior(ED, "C1", 3)
	g.Clearings["C1"].Buildings = append(g.Clearings["C1"].Buildings, Building{ED, "roost"})
	g.Current = ED
	g.Phase = "D"
	g.Pending = &Pending{Kind: PendingBattleHits, Player: ED}
	g.Battle = &BattleState{Clearing: "C1", Attacker: MC, Defender: ED, HitSide: ED, Remaining: 1}

	g.RemoveFaction(ED)

	if _, ok := g.Players[ED]; ok {
		t.Fatal("ED should be removed from Players")
	}
	for _, f := range g.Order {
		if f == ED {
			t.Fatal("ED should be removed from Order")
		}
	}
	if g.Battle != nil {
		t.Fatal("a battle referencing ED should be cleared")
	}
	if g.Pending != nil {
		t.Fatal("a pending choice for ED should be cleared")
	}
	cl := g.Clearings["C1"]
	if cl.Warriors[ED] != 0 {
		t.Fatalf("ED warriors should be gone: %v", cl.Warriors)
	}
	for _, b := range cl.Buildings {
		if b.Owner == ED {
			t.Fatal("ED buildings should be gone")
		}
	}
	if g.Current == ED || g.Current == "" {
		t.Fatalf("turn should advance to a remaining faction, got %q", g.Current)
	}
	if len(g.LegalActions()) == 0 {
		t.Fatal("game should continue with legal actions after removal")
	}
}
