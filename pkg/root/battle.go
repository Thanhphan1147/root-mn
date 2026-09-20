package root

import "fmt"

// BattleStage tracks the current phase of a battle.
type BattleStage string

const (
	StageDefAmbush BattleStage = "def-ambush"
	StageAtkFoil   BattleStage = "atk-foil"
	StageAtkPre    BattleStage = "atk-pre"
	StageDefPre    BattleStage = "def-pre"
	StageHits      BattleStage = "hits"
)

// BattleState is an in-progress battle.
type BattleState struct {
	Clearing   string
	Attacker   Faction
	Defender   Faction
	Stage      BattleStage
	DefAmbush  bool
	AtkFoil    bool
	AtkHits    int
	DefHits    int
	HitSide    Faction
	Remaining  int
	AtkExtra   int
	DefExtra   int
	AtkIgnore  int
	DefIgnore  int
	DespotDone bool
	AllyTarget Faction
	AllyHits   int
	ItemHits   int
	Step       int
	StepName   string
	D1, D2     int
	FH         []FHRec
}

// FHRec records Marquise warriors removed in a clearing (Field Hospitals).
type FHRec struct {
	Clearing string
	Count    int
}

// startBattle begins a battle and sets up the first pending stage.
func (g *Game) startBattle(a Action) error {
	att, def := a.Faction, a.Target
	if att == "" {
		att = g.Current
	}
	cl := a.Clearing
	if def == "" {
		enemies := g.enemiesIn(att, cl)
		if len(enemies) == 0 {
			return fmt.Errorf("no defender in %s", cl)
		}
		def = enemies[0]
	}
	b := &BattleState{Clearing: cl, Attacker: att, Defender: def, Stage: StageDefAmbush, AllyTarget: a.Ally, Step: 1, StepName: "Declare"}
	g.Battle = b
	if att == ED && g.Players[ED].Leader == "commander" {
		b.AtkExtra++
	}
	if att == WA && g.Phase == "E" && g.MilitaryOpsLeft > 0 {
		g.MilitaryOpsLeft--
	}
	if g.ExtraBattle && g.hasCrafted(att, "command-warren") {
		g.ExtraBattle = false
	} else if att == MC && g.ActionsLeft > 0 {
		g.ActionsLeft--
	}
	if att == VB {
		if id, ok := g.readyItem(g.Players[VB], "sword"); ok {
			g.exhaustItem(g.Players[VB], id)
		}
	}
	g.Logf(att, "battle", "%s attacks %s in %s", att, def, cl)
	// Defender ambush? Skip if attacker has Scouting Party.
	if !g.hasCrafted(att, "scouting-party") && g.canAmbush(def, cl) {
		b.Step, b.StepName = 2, "Defender ambush"
		g.Pending = &Pending{Kind: PendingBattleAmbush, Player: def}
		return nil
	}
	g.advanceBattleAfterAmbush()
	return nil
}

func (g *Game) hasCrafted(f Faction, effect string) bool {
	for _, id := range g.Players[f].Crafted {
		if c, ok := Card(id); ok && c.Effect == effect {
			return true
		}
	}
	return false
}

func (g *Game) canAmbush(f Faction, clearing string) bool {
	p := g.Players[f]
	s := g.Clearings[clearing].Suit
	for _, id := range p.Hand {
		if c, ok := Card(id); ok && c.Kind == KindAmbush && matches(c.Suit, s) {
			return true
		}
	}
	return false
}

func (g *Game) advanceBattleAfterAmbush() {
	b := g.Battle
	// After ambush resolution, offer pre-roll one-shots if available.
	if g.hasCrafted(b.Attacker, "brutal-tactics") || g.hasCrafted(b.Defender, "armorers") || g.hasCrafted(b.Defender, "sappers") {
		b.Stage = StageAtkPre
		b.Step, b.StepName = 4, "Effects"
		g.Pending = &Pending{Kind: PendingBattleEffects, Player: b.Attacker}
		return
	}
	g.resolveBattleRoll()
}

func (g *Game) resolveBattleRoll() {
	b := g.Battle
	d1, d2 := g.Roll()
	b.D1, b.D2 = d1, d2
	b.Step, b.StepName = 4, "Dice roll"
	hi, lo := d1, d2
	if d1 < d2 {
		hi, lo = d2, d1
	}
	attDie, defDie := hi, lo
	if b.Defender == WA {
		attDie, defDie = lo, hi // Guerrilla War
	}
	wAtt := g.maxHits(b.Attacker, b.Clearing)
	wDef := g.maxHits(b.Defender, b.Clearing)
	attRolled := attDie
	if attRolled > wAtt {
		attRolled = wAtt
	}
	defRolled := defDie
	if defRolled > wDef {
		defRolled = wDef
	}
	// Defenseless.
	if g.isDefenseless(b.Defender, b.Clearing) {
		b.AtkExtra++
	}
	b.AtkHits = attRolled + b.AtkExtra - b.DefIgnore
	b.DefHits = defRolled + b.DefExtra - b.AtkIgnore
	if b.AtkHits < 0 {
		b.AtkHits = 0
	}
	if b.DefHits < 0 {
		b.DefHits = 0
	}
	b.Remaining = b.AtkHits
	b.HitSide = b.Defender
	b.Stage = StageHits
	b.Step, b.StepName = 5, "Resolve"
	g.Logf(b.Attacker, "battle", "Roll %d-%d: %s deals %d, %s deals %d", d1, d2, b.Attacker, b.AtkHits, b.Defender, b.DefHits)
	g.queueHits()
}

func (g *Game) maxHits(f Faction, clearing string) int {
	if f == VB {
		n := 0
		for _, it := range g.Players[VB].Items {
			if it.Type == "sword" && !it.Damaged {
				n++
			}
		}
		if g.Battle != nil && g.Battle.AllyTarget != "" {
			n += g.Clearings[g.Battle.Clearing].Warriors[g.Battle.AllyTarget]
		}
		return n
	}
	return g.Clearings[clearing].Warriors[f]
}

func (g *Game) isDefenseless(f Faction, clearing string) bool {
	if f == VB {
		for _, it := range g.Players[VB].Items {
			if it.Type == "sword" && !it.Damaged {
				return false
			}
		}
		return true
	}
	return g.Clearings[clearing].Warriors[f] == 0
}

// queueHits sets up hit assignment for the current side, then the other side.
func (g *Game) queueHits() {
	b := g.Battle
	for b.Remaining <= 0 {
		// move to next side
		if b.HitSide == b.Defender {
			b.HitSide = b.Attacker
			b.Remaining = b.DefHits
		} else {
			g.endBattle()
			return
		}
	}
	g.Pending = &Pending{Kind: PendingBattleHits, Player: b.HitSide}
}

func (g *Game) legalBattleHits() []Action {
	b := g.Battle
	side := b.HitSide
	var acts []Action
	if side == VB {
		for id, it := range g.Players[VB].Items {
			if !it.Damaged {
				acts = append(acts, Action{
					ID: actID("battle-hit", id), Label: "Damage item " + id, Kind: "battle-hit",
					Faction: VB, Item: id,
				})
			}
		}
		if b.AllyTarget != "" && g.Clearings[b.Clearing].Warriors[b.AllyTarget] > 0 {
			acts = append(acts, Action{
				ID:    actID("battle-hit", "ally", string(b.AllyTarget)),
				Label: fmt.Sprintf("Remove %s warrior (ally)", b.AllyTarget),
				Kind:  "battle-hit", Faction: VB, Piece: "ally-warrior", Target: b.AllyTarget,
			})
		}
		if len(acts) == 0 {
			acts = append(acts, Action{ID: "battle-skip", Label: "Ignore remaining hits", Kind: "battle-skip", Faction: VB})
		}
		return acts
	}
	cl := g.Clearings[b.Clearing]
	hasWarriors := cl.Warriors[side] > 0
	if hasWarriors {
		acts = append(acts, Action{
			ID: actID("battle-hit", "warrior", string(side)), Label: fmt.Sprintf("Remove %s warrior", side),
			Kind: "battle-hit", Faction: side, Piece: "warrior", Clearing: b.Clearing,
		})
		return acts
	}
	for i, bd := range cl.Buildings {
		if bd.Owner == side {
			acts = append(acts, Action{
				ID: actID("battle-hit", "building", itoa(i)), Label: "Remove " + bd.Type,
				Kind: "battle-hit", Faction: side, Piece: "building", Clearing: b.Clearing, Amount: i,
			})
		}
	}
	for i, t := range cl.Tokens {
		if t.Owner == side {
			acts = append(acts, Action{
				ID: actID("battle-hit", "token", itoa(i)), Label: "Remove " + t.Type,
				Kind: "battle-hit", Faction: side, Piece: "token", Clearing: b.Clearing, Amount: i,
			})
		}
	}
	if len(acts) == 0 {
		acts = append(acts, Action{ID: "battle-skip", Label: "Ignore remaining hits", Kind: "battle-skip", Faction: side})
	}
	return acts
}

// applyBattlePending handles ambush/effects/hit pending actions.
func (g *Game) applyBattlePending(a Action) error {
	b := g.Battle
	switch a.Kind {
	case "battle-ambush":
		p := g.Players[b.Defender]
		takeStr(&p.Hand, a.Card)
		g.Discard = append(g.Discard, a.Card)
		b.DefAmbush = true
		b.DefExtra += 2
		g.Logf(b.Defender, "ambush", "Played ambush %s (+2 hits)", cardName(a.Card))
		// attacker may foil
		if g.canAmbush(b.Attacker, b.Clearing) {
			b.Stage = StageAtkFoil
			b.Step, b.StepName = 3, "Attacker ambush"
			g.Pending = &Pending{Kind: PendingBattleAmbush, Player: b.Attacker}
			return nil
		}
		g.advanceBattleAfterAmbush()
	case "battle-foil":
		p := g.Players[b.Attacker]
		takeStr(&p.Hand, a.Card)
		g.Discard = append(g.Discard, a.Card)
		b.DefExtra -= 2
		g.Logf(b.Attacker, "ambush", "Foiled ambush with %s", cardName(a.Card))
		g.advanceBattleAfterAmbush()
	case "battle-skip":
		if b.Stage == StageDefAmbush && !b.DefAmbush {
			g.advanceBattleAfterAmbush()
			return nil
		}
		if b.Stage == StageAtkFoil {
			g.advanceBattleAfterAmbush()
			return nil
		}
		if b.Stage == StageAtkPre || b.Stage == StageDefPre {
			g.nextPreStage()
			return nil
		}
		// hits: ignore remaining
		b.Remaining = 0
		g.queueHits()
	case "battle-effect":
		p := g.Players[a.Faction]
		takeStr(&p.Crafted, a.Card)
		g.Discard = append(g.Discard, a.Card)
		c, _ := Card(a.Card)
		switch c.Effect {
		case "brutal-tactics":
			b.AtkExtra++
			g.Score(b.Defender, 1)
		case "armorers":
			b.DefIgnore = 99
		case "sappers":
			b.DefExtra++
		}
		g.Logf(a.Faction, "effect", "Played %s", c.Name)
		g.nextPreStage()
	case "battle-hit":
		g.applyBattleHit(a)
	}
	return nil
}

func (g *Game) nextPreStage() {
	b := g.Battle
	if b.Stage == StageAtkPre {
		if g.hasCrafted(b.Defender, "armorers") || g.hasCrafted(b.Defender, "sappers") {
			b.Stage = StageDefPre
			g.Pending = &Pending{Kind: PendingBattleEffects, Player: b.Defender}
			return
		}
	}
	g.resolveBattleRoll()
}

func (g *Game) applyBattleHit(a Action) {
	b := g.Battle
	side := b.HitSide
	if side == VB {
		if a.Piece == "ally-warrior" {
			if g.removeWarrior(a.Target, b.Clearing, 1) > 0 {
				b.AllyHits++
				b.Remaining--
				g.Logf(VB, "battle", "Removed %s warrior (ally) to absorb a hit", a.Target)
			}
			g.queueHits()
			return
		}
		if it, ok := g.Players[VB].Items[a.Item]; ok {
			it.Damaged = true
			it.Zone = "damaged"
			b.ItemHits++
			b.Remaining--
			g.Logf(VB, "battle", "Damaged %s", a.Item)
		}
		g.queueHits()
		return
	}
	cl := g.Clearings[b.Clearing]
	wasHostile := g.Players[VB].Relationships[side] == "hostile"
	removed := false
	kind := ""
	switch a.Piece {
	case "warrior":
		if cl.Warriors[side] > 0 {
			cl.Warriors[side]--
			if cl.Warriors[side] == 0 {
				delete(cl.Warriors, side)
			}
			removed, kind = true, "warrior"
			if side == MC {
				b.FH = append(b.FH, FHRec{Clearing: b.Clearing, Count: 1})
			}
		}
	case "building":
		if a.Amount < len(cl.Buildings) && cl.Buildings[a.Amount].Owner == side {
			bd := cl.Buildings[a.Amount]
			cl.Buildings = append(cl.Buildings[:a.Amount], cl.Buildings[a.Amount+1:]...)
			removed, kind = true, "building"
			if side == WA && isBaseBuilding(bd.Type) {
				g.waBaseRemoved(bd.Type)
			}
			if side == MC {
				p := g.Players[MC]
				switch bd.Type {
				case "sawmill":
					p.Sawmills++
				case "workshop":
					p.Workshops++
				case "recruiter":
					p.Recruiters++
				}
			}
		}
	case "token":
		if a.Amount < len(cl.Tokens) && cl.Tokens[a.Amount].Owner == side {
			tok := cl.Tokens[a.Amount]
			cl.Tokens = append(cl.Tokens[:a.Amount], cl.Tokens[a.Amount+1:]...)
			removed, kind = true, "token"
			if tok.Type == "sympathy" {
				cl.Sympathy = ""
				remover := b.Attacker
				if side == b.Attacker {
					remover = b.Defender
				}
				if remover != WA {
					g.outrage(remover, b.Clearing)
				}
			}
			if tok.Type == "keep" {
				// keep removed permanently
			}
		}
	}
	if removed {
		b.Remaining--
		// The opponent of the side that lost the piece is the remover.
		scorer := b.Attacker
		if side == b.Attacker {
			scorer = b.Defender
		}
		switch kind {
		case "warrior":
			if scorer == VB {
				if wasHostile {
					g.Score(VB, 1) // Infamy (not for the warrior that caused Hostility)
				}
				g.vbHostilityOnWarriorRemoval(side)
			}
		case "building", "token":
			g.Score(scorer, 1)
			if scorer == ED && g.Players[ED].Leader == "despot" && !b.DespotDone {
				g.Score(ED, 1)
				b.DespotDone = true
			}
			// Infamy: +1 per Hostile piece removed in battle on the Vagabond's turn.
			if scorer == VB && g.Current == VB && g.Players[VB].Relationships[side] == "hostile" {
				g.Score(VB, 1)
			}
			g.Logf(scorer, "battle", "Removed %s %s (+1 VP)", side, kind)
		}
	}
	g.queueHits()
}

func (g *Game) endBattle() {
	b := g.Battle
	if b != nil && b.AllyTarget != "" && b.AllyHits > b.ItemHits {
		g.Players[VB].Relationships[b.AllyTarget] = "hostile"
		g.Logf(VB, "battle", "%s becomes Hostile (allied warriors took more hits than items)", b.AllyTarget)
	}
	g.Battle = nil
	g.Pending = nil
	g.Logf(g.Current, "battle", "Battle ended")
	if b != nil && len(b.FH) > 0 && g.hasKeep() {
		g.FH = b.FH
		g.Pending = &Pending{Kind: PendingFieldHospitals, Player: MC}
	}
}

func (g *Game) hasKeep() bool {
	for _, c := range g.Clearings {
		for _, t := range c.Tokens {
			if t.Type == "keep" {
				return true
			}
		}
	}
	return false
}
