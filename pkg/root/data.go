// Package root implements a rules-complete base-game engine for the board game
// ROOT (Leder Games): Autumn map, standard 54-card deck, and the four base
// factions Marquise de Cat, Eyrie Dynasties, Woodland Alliance, and Vagabond.
package root

// Faction identifies a base-game faction.
type Faction string

// Base factions.
const (
	MC Faction = "MC"
	ED Faction = "ED"
	WA Faction = "WA"
	VB Faction = "VB"
)

// Kind returns the canonical faction kind name.
func (f Faction) Kind() string {
	switch f {
	case MC:
		return "marquise"
	case ED:
		return "eyrie"
	case WA:
		return "alliance"
	case VB:
		return "vagabond"
	}
	return string(f)
}

// IsBase reports whether f is one of the four base factions.
func (f Faction) IsBase() bool {
	switch f {
	case MC, ED, WA, VB:
		return true
	}
	return false
}

// Suit is a card/clearing suit.
type Suit string

// Suits.
const (
	Fox    Suit = "F"
	Rabbit Suit = "R"
	Mouse  Suit = "M"
	Bird   Suit = "B"
)

// CardKind classifies a card.
type CardKind string

const (
	KindAmbush    CardKind = "ambush"
	KindDominance CardKind = "dominance"
	KindFavor     CardKind = "favor"
	KindItem      CardKind = "item"
	KindEffect    CardKind = "effect"
)

// CardDef is a static card definition.
type CardDef struct {
	ID     string
	Suit   Suit
	Name   string
	Kind   CardKind
	Cost   []Suit // crafting icons; "Any" is represented by empty Suit list entry via AnyIcon
	Any    int    // number of "any suit" icons
	Item   string // item type produced (items)
	VP     int    // item crafting VP
	Effect string // effect key
	Pers   string // "", "pers", "once"
}

// ClearingDef is static map data for one clearing.
type ClearingDef struct {
	ID    string
	Suit  Suit
	Slots int
	Adj   []string
	Ruin  bool
}

// MapDef is a static map.
type MapDef struct {
	Name      string
	Clearings map[string]ClearingDef
	Forests   map[string][]string
	Corners   [][2]string
	Ruins     []string
}

// CharacterDef is a Vagabond character.
type CharacterDef struct {
	Name     string
	Start    []string // item types
	Ability  string
	Ability2 string
}

// QuestDef is a Vagabond quest.
type QuestDef struct {
	ID    string
	Suit  Suit
	Name  string
	Items []string
}

var autumn = &MapDef{
	Name: "autumn",
	Clearings: map[string]ClearingDef{
		"C1":  {"C1", Fox, 1, []string{"C5", "C9", "C10"}, false},
		"C2":  {"C2", Mouse, 2, []string{"C5", "C6", "C10"}, false},
		"C3":  {"C3", Rabbit, 1, []string{"C6", "C7", "C11"}, false},
		"C4":  {"C4", Rabbit, 1, []string{"C8", "C9", "C12"}, false},
		"C5":  {"C5", Rabbit, 2, []string{"C1", "C2"}, false},
		"C6":  {"C6", Fox, 2, []string{"C2", "C3", "C11"}, true},
		"C7":  {"C7", Mouse, 2, []string{"C3", "C8", "C12"}, false},
		"C8":  {"C8", Fox, 2, []string{"C4", "C7"}, false},
		"C9":  {"C9", Mouse, 2, []string{"C1", "C4", "C12"}, false},
		"C10": {"C10", Rabbit, 2, []string{"C1", "C2", "C12"}, true},
		"C11": {"C11", Mouse, 3, []string{"C3", "C6", "C12"}, true},
		"C12": {"C12", Fox, 2, []string{"C4", "C7", "C9", "C10", "C11"}, true},
	},
	Forests: map[string][]string{
		"AutumnN":   {"C1", "C2", "C5", "C10"},
		"AutumnNW":  {"C1", "C9", "C10", "C12"},
		"Witchwood": {"C2", "C6", "C10", "C11", "C12"},
		"AutumnW":   {"C4", "C9", "C12"},
		"AutumnS":   {"C3", "C7", "C11", "C12"},
		"AutumnE":   {"C3", "C6", "C11"},
		"AutumnSW":  {"C4", "C7", "C8", "C12"},
	},
	Corners: [][2]string{{"C1", "C3"}, {"C2", "C4"}},
	Ruins:   []string{"C6", "C10", "C11", "C12"},
}

// GetMap returns a built-in map by name.
func GetMap(name string) *MapDef {
	switch name {
	case "autumn", "fall", "":
		return autumn
	}
	return nil
}

// ForestList returns forest ids sorted by name for stable iteration.
func (m *MapDef) ForestList() []string {
	return []string{"AutumnE", "AutumnN", "AutumnNW", "AutumnS", "AutumnSW", "AutumnW", "Witchwood"}
}

// Adjacent reports whether a and b are adjacent clearings.
func (m *MapDef) Adjacent(a, b string) bool {
	for _, n := range m.Clearings[a].Adj {
		if n == b {
			return true
		}
	}
	return false
}

// ClearingList returns clearing ids in numeric order.
func (m *MapDef) ClearingList() []string {
	return []string{"C1", "C2", "C3", "C4", "C5", "C6", "C7", "C8", "C9", "C10", "C11", "C12"}
}

func buildDeck() []*CardDef {
	var d []*CardDef
	add := func(c CardDef) { d = append(d, &c) }
	// Fox (14)
	add(CardDef{ID: "F01", Suit: Fox, Name: "Ambush!", Kind: KindAmbush, Effect: "ambush"})
	add(CardDef{ID: "F02", Suit: Fox, Name: "Fox Dominance", Kind: KindDominance, Effect: "dom-fox"})
	add(CardDef{ID: "F03", Suit: Fox, Name: "Favor of the Foxes", Kind: KindFavor, Cost: []Suit{Fox, Fox, Fox}, Effect: "favor-fox"})
	add(CardDef{ID: "F04", Suit: Fox, Name: "Gently Used Knapsack", Kind: KindItem, Cost: []Suit{Mouse}, Item: "bag", VP: 1})
	add(CardDef{ID: "F05", Suit: Fox, Name: "Fox Root Tea", Kind: KindItem, Cost: []Suit{Mouse}, Item: "tea", VP: 2})
	add(CardDef{ID: "F06", Suit: Fox, Name: "Fox Travel Gear", Kind: KindItem, Cost: []Suit{Rabbit}, Item: "boot", VP: 1})
	add(CardDef{ID: "F07", Suit: Fox, Name: "Protection Racket", Kind: KindItem, Cost: []Suit{Rabbit, Rabbit}, Item: "coin", VP: 3})
	add(CardDef{ID: "F08", Suit: Fox, Name: "Foxfolk Steel", Kind: KindItem, Cost: []Suit{Fox, Fox}, Item: "sword", VP: 2})
	add(CardDef{ID: "F09", Suit: Fox, Name: "Anvil", Kind: KindItem, Cost: []Suit{Fox}, Item: "hammer", VP: 2})
	add(CardDef{ID: "F10", Suit: Fox, Name: "Stand and Deliver!", Kind: KindEffect, Cost: []Suit{Mouse, Mouse, Mouse}, Effect: "stand-deliver", Pers: "pers"})
	add(CardDef{ID: "F11", Suit: Fox, Name: "Stand and Deliver!", Kind: KindEffect, Cost: []Suit{Mouse, Mouse, Mouse}, Effect: "stand-deliver", Pers: "pers"})
	add(CardDef{ID: "F12", Suit: Fox, Name: "Tax Collector", Kind: KindEffect, Cost: []Suit{Fox, Rabbit, Mouse}, Effect: "tax-collector", Pers: "pers"})
	add(CardDef{ID: "F13", Suit: Fox, Name: "Tax Collector", Kind: KindEffect, Cost: []Suit{Fox, Rabbit, Mouse}, Effect: "tax-collector", Pers: "pers"})
	add(CardDef{ID: "F14", Suit: Fox, Name: "Tax Collector", Kind: KindEffect, Cost: []Suit{Fox, Rabbit, Mouse}, Effect: "tax-collector", Pers: "pers"})
	// Rabbit (13)
	add(CardDef{ID: "R01", Suit: Rabbit, Name: "Ambush!", Kind: KindAmbush, Effect: "ambush"})
	add(CardDef{ID: "R02", Suit: Rabbit, Name: "Rabbit Dominance", Kind: KindDominance, Effect: "dom-rabbit"})
	add(CardDef{ID: "R03", Suit: Rabbit, Name: "Favor of the Rabbits", Kind: KindFavor, Cost: []Suit{Rabbit, Rabbit, Rabbit}, Effect: "favor-rabbit"})
	add(CardDef{ID: "R04", Suit: Rabbit, Name: "Smuggler's Trail", Kind: KindItem, Cost: []Suit{Mouse}, Item: "bag", VP: 1})
	add(CardDef{ID: "R05", Suit: Rabbit, Name: "Rabbit Root Tea", Kind: KindItem, Cost: []Suit{Mouse}, Item: "tea", VP: 2})
	add(CardDef{ID: "R06", Suit: Rabbit, Name: "A Visit to Friends", Kind: KindItem, Cost: []Suit{Rabbit}, Item: "boot", VP: 1})
	add(CardDef{ID: "R07", Suit: Rabbit, Name: "Bake Sale", Kind: KindItem, Cost: []Suit{Rabbit, Rabbit}, Item: "coin", VP: 3})
	add(CardDef{ID: "R08", Suit: Rabbit, Name: "Command Warren", Kind: KindEffect, Cost: []Suit{Rabbit, Rabbit}, Effect: "command-warren", Pers: "pers"})
	add(CardDef{ID: "R09", Suit: Rabbit, Name: "Command Warren", Kind: KindEffect, Cost: []Suit{Rabbit, Rabbit}, Effect: "command-warren", Pers: "pers"})
	add(CardDef{ID: "R10", Suit: Rabbit, Name: "Better Burrow Bank", Kind: KindEffect, Cost: []Suit{Rabbit, Rabbit}, Effect: "burrow-bank", Pers: "pers"})
	add(CardDef{ID: "R11", Suit: Rabbit, Name: "Better Burrow Bank", Kind: KindEffect, Cost: []Suit{Rabbit, Rabbit}, Effect: "burrow-bank", Pers: "pers"})
	add(CardDef{ID: "R12", Suit: Rabbit, Name: "Cobbler", Kind: KindEffect, Cost: []Suit{Rabbit, Rabbit}, Effect: "cobbler", Pers: "pers"})
	add(CardDef{ID: "R13", Suit: Rabbit, Name: "Cobbler", Kind: KindEffect, Cost: []Suit{Rabbit, Rabbit}, Effect: "cobbler", Pers: "pers"})
	// Mouse (13)
	add(CardDef{ID: "M01", Suit: Mouse, Name: "Ambush!", Kind: KindAmbush, Effect: "ambush"})
	add(CardDef{ID: "M02", Suit: Mouse, Name: "Mouse Dominance", Kind: KindDominance, Effect: "dom-mouse"})
	add(CardDef{ID: "M03", Suit: Mouse, Name: "Favor of the Mice", Kind: KindFavor, Cost: []Suit{Mouse, Mouse, Mouse}, Effect: "favor-mouse"})
	add(CardDef{ID: "M04", Suit: Mouse, Name: "Mouse-in-a-Sack", Kind: KindItem, Cost: []Suit{Mouse}, Item: "bag", VP: 1})
	add(CardDef{ID: "M05", Suit: Mouse, Name: "Mouse Root Tea", Kind: KindItem, Cost: []Suit{Mouse}, Item: "tea", VP: 2})
	add(CardDef{ID: "M06", Suit: Mouse, Name: "Mouse Travel Gear", Kind: KindItem, Cost: []Suit{Rabbit}, Item: "boot", VP: 1})
	add(CardDef{ID: "M07", Suit: Mouse, Name: "Investments", Kind: KindItem, Cost: []Suit{Rabbit, Rabbit}, Item: "coin", VP: 3})
	add(CardDef{ID: "M08", Suit: Mouse, Name: "Sword", Kind: KindItem, Cost: []Suit{Fox, Fox}, Item: "sword", VP: 2})
	add(CardDef{ID: "M09", Suit: Mouse, Name: "Mouse Crossbow", Kind: KindItem, Cost: []Suit{Fox}, Item: "crossbow", VP: 1})
	add(CardDef{ID: "M10", Suit: Mouse, Name: "Scouting Party", Kind: KindEffect, Cost: []Suit{Mouse, Mouse}, Effect: "scouting-party", Pers: "pers"})
	add(CardDef{ID: "M11", Suit: Mouse, Name: "Scouting Party", Kind: KindEffect, Cost: []Suit{Mouse, Mouse}, Effect: "scouting-party", Pers: "pers"})
	add(CardDef{ID: "M12", Suit: Mouse, Name: "Codebreakers", Kind: KindEffect, Cost: []Suit{Mouse}, Effect: "codebreakers", Pers: "pers"})
	add(CardDef{ID: "M13", Suit: Mouse, Name: "Codebreakers", Kind: KindEffect, Cost: []Suit{Mouse}, Effect: "codebreakers", Pers: "pers"})
	// Bird (14)
	add(CardDef{ID: "B01", Suit: Bird, Name: "Ambush!", Kind: KindAmbush, Effect: "ambush"})
	add(CardDef{ID: "B02", Suit: Bird, Name: "Ambush!", Kind: KindAmbush, Effect: "ambush"})
	add(CardDef{ID: "B03", Suit: Bird, Name: "Bird Dominance", Kind: KindDominance, Effect: "dom-bird"})
	add(CardDef{ID: "B04", Suit: Bird, Name: "Royal Claim", Kind: KindEffect, Any: 4, Effect: "royal-claim", Pers: "once"})
	add(CardDef{ID: "B05", Suit: Bird, Name: "Armorers", Kind: KindEffect, Cost: []Suit{Fox}, Effect: "armorers", Pers: "once"})
	add(CardDef{ID: "B06", Suit: Bird, Name: "Armorers", Kind: KindEffect, Cost: []Suit{Fox}, Effect: "armorers", Pers: "once"})
	add(CardDef{ID: "B07", Suit: Bird, Name: "Sappers", Kind: KindEffect, Cost: []Suit{Mouse}, Effect: "sappers", Pers: "once"})
	add(CardDef{ID: "B08", Suit: Bird, Name: "Sappers", Kind: KindEffect, Cost: []Suit{Mouse}, Effect: "sappers", Pers: "once"})
	add(CardDef{ID: "B09", Suit: Bird, Name: "Brutal Tactics", Kind: KindEffect, Cost: []Suit{Fox, Fox}, Effect: "brutal-tactics", Pers: "once"})
	add(CardDef{ID: "B10", Suit: Bird, Name: "Brutal Tactics", Kind: KindEffect, Cost: []Suit{Fox, Fox}, Effect: "brutal-tactics", Pers: "once"})
	add(CardDef{ID: "B11", Suit: Bird, Name: "Birdy Bindle", Kind: KindItem, Cost: []Suit{Mouse}, Item: "bag", VP: 1})
	add(CardDef{ID: "B12", Suit: Bird, Name: "Woodland Runners", Kind: KindItem, Cost: []Suit{Rabbit}, Item: "boot", VP: 1})
	add(CardDef{ID: "B13", Suit: Bird, Name: "Arms Trader", Kind: KindItem, Cost: []Suit{Fox, Fox}, Item: "sword", VP: 2})
	add(CardDef{ID: "B14", Suit: Bird, Name: "Crossbow", Kind: KindItem, Cost: []Suit{Fox}, Item: "crossbow", VP: 1})
	return d
}

var cardIndex = func() map[string]*CardDef {
	m := map[string]*CardDef{}
	for _, c := range buildDeck() {
		m[c.ID] = c
	}
	return m
}()

// Card returns a card definition by id.
func Card(id string) (*CardDef, bool) {
	c, ok := cardIndex[id]
	return c, ok
}

// ItemCounts is the base item supply by type.
var ItemCounts = map[string]int{
	"boot": 2, "bag": 2, "sword": 2, "hammer": 1, "tea": 2, "coin": 2, "crossbow": 1,
}

// RuinItems are the four ruin items.
var RuinItems = []string{"bag", "boot", "hammer", "sword"}

// StartingItemSupply is the total physical item count by type.
var StartingItemSupply = map[string]int{
	"boot": 4, "bag": 4, "sword": 4, "hammer": 3, "tea": 3, "coin": 2, "crossbow": 2, "torch": 1,
}

// Characters are the base-game Vagabond characters.
var Characters = map[string]CharacterDef{
	"thief":  {"Thief", []string{"boot", "torch", "tea", "sword"}, "steal", ""},
	"tinker": {"Tinker", []string{"boot", "torch", "bag", "hammer"}, "day-labor", ""},
	"ranger": {"Ranger", []string{"boot", "torch", "crossbow", "sword"}, "hideout", ""},
}

// Leaders are the Eyrie leaders and the decree columns their viziers occupy.
var Leaders = map[string]struct {
	Viziers []string
	Ability string
}{
	"builder":     {[]string{"RECRUIT", "MOVE"}, "ignore-disdain"},
	"charismatic": {[]string{"RECRUIT", "BATTLE"}, "recruit-two"},
	"commander":   {[]string{"MOVE", "BATTLE"}, "extra-hit"},
	"despot":      {[]string{"MOVE", "BUILD"}, "despot-vp"},
}

// DecreeColumns is the fixed resolution order.
var DecreeColumns = []string{"RECRUIT", "MOVE", "BATTLE", "BUILD"}

// SympathyCost and SympathyVP are indexed by placement number 1..10.
var SympathyCost = []int{0, 1, 1, 1, 2, 2, 2, 3, 3, 3, 3}
var SympathyVP = []int{0, 0, 1, 1, 1, 2, 2, 3, 4, 4, 4}

// MC building tracks (index 1..6 = slot left to right).
var (
	MCBuildCost      = []int{0, 0, 1, 2, 3, 3, 4}
	MCSawmillVP      = []int{0, 0, 1, 2, 3, 4, 5}
	MCWorkshopVP     = []int{0, 0, 2, 2, 3, 4, 5}
	MCRecruiterVP    = []int{0, 0, 1, 2, 3, 3, 4}
	MCRecruiterBonus = []int{0, 0, 0, 1, 0, 1, 0}
)

// EDRoostVP indexed by number of roosts on map 0..7.
var EDRoostVP = []int{0, 0, 1, 2, 3, 4, 4, 5}

// EDRoostDrawBonus indexed by roosts on map 0..7.
var EDRoostDrawBonus = []int{0, 0, 0, 1, 1, 1, 2, 2}

// WarriorSupply is the number of warrior pieces each faction has. Placing
// warriors beyond the supply does nothing — the pieces are not available.
var WarriorSupply = map[Faction]int{
	MC: 25,
	ED: 20,
	WA: 10,
}

// Quests is the base-game quest deck.
var Quests = []QuestDef{
	{"q.F01", Fox, "Fundraising", []string{"tea", "coin"}},
	{"q.F02", Fox, "Errand", []string{"tea", "boot"}},
	{"q.F03", Fox, "Logistic Help", []string{"boot", "bag"}},
	{"q.F04", Fox, "Repair a Shed", []string{"torch", "hammer"}},
	{"q.F05", Fox, "Give a Speech", []string{"torch", "tea"}},
	{"q.R01", Rabbit, "Guard Duty", []string{"torch", "sword"}},
	{"q.R02", Rabbit, "Errand", []string{"tea", "boot"}},
	{"q.R03", Rabbit, "Give a Speech", []string{"torch", "tea"}},
	{"q.R04", Rabbit, "Fend off a Bear", []string{"torch", "crossbow"}},
	{"q.R05", Rabbit, "Expel Bandits", []string{"sword", "sword"}},
	{"q.M01", Mouse, "Expel Bandits", []string{"sword", "sword"}},
	{"q.M02", Mouse, "Guard Duty", []string{"torch", "sword"}},
	{"q.M03", Mouse, "Fend off a Bear", []string{"torch", "crossbow"}},
	{"q.M04", Mouse, "Escort", []string{"boot", "boot"}},
	{"q.M05", Mouse, "Logistic Help", []string{"boot", "bag"}},
}

// QuestByID returns a quest definition.
func QuestByID(id string) (QuestDef, bool) {
	for _, q := range Quests {
		if q.ID == id {
			return q, true
		}
	}
	return QuestDef{}, false
}

// QuestInfo is the client-facing quest description (public information).
type QuestInfo struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Suit  string   `json:"suit"`
	Items []string `json:"items"`
}

// QuestInfoMap returns descriptions for every quest.
func QuestInfoMap() map[string]QuestInfo {
	out := make(map[string]QuestInfo, len(Quests))
	for _, q := range Quests {
		out[q.ID] = QuestInfo{ID: q.ID, Name: q.Name, Suit: string(q.Suit), Items: q.Items}
	}
	return out
}

// CardInfo is the client-facing card description.
type CardInfo struct {
	ID     string `json:"id"`
	Suit   string `json:"suit"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Cost   string `json:"cost,omitempty"`
	Item   string `json:"item,omitempty"`
	VP     int    `json:"vp,omitempty"`
	Effect string `json:"effect,omitempty"`
	Desc   string `json:"desc"`
}

// CardInfoMap returns descriptions for every card in the standard deck.
func CardInfoMap() map[string]CardInfo {
	out := map[string]CardInfo{}
	for _, c := range buildDeck() {
		out[c.ID] = CardInfo{
			ID: c.ID, Suit: string(c.Suit), Name: c.Name, Kind: string(c.Kind),
			Cost: costText(c), Item: c.Item, VP: c.VP, Effect: c.Effect,
			Desc: cardDesc(c),
		}
	}
	return out
}

func costText(c *CardDef) string {
	counts := map[Suit]int{}
	for _, s := range c.Cost {
		counts[s]++
	}
	names := map[Suit]string{Fox: "Fox", Rabbit: "Rabbit", Mouse: "Mouse", Bird: "Bird"}
	var parts []string
	for _, s := range []Suit{Fox, Rabbit, Mouse, Bird} {
		if counts[s] > 0 {
			if counts[s] == 1 {
				parts = append(parts, names[s])
			} else {
				parts = append(parts, itoa(counts[s])+" "+names[s])
			}
		}
	}
	if c.Any > 0 {
		parts = append(parts, "any×"+itoa(c.Any))
	}
	sep := " + "
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

func cardDesc(c *CardDef) string {
	suitName := map[Suit]string{Fox: "fox", Rabbit: "rabbit", Mouse: "mouse", Bird: "bird"}[c.Suit]
	switch c.Kind {
	case KindAmbush:
		return "Battle · defender may deal 2 hits in a " + suitName + " clearing (attacker may foil)."
	case KindDominance:
		if c.Effect == "dom-bird" {
			return "Activate at 10+ VP. Win by ruling 2 opposite-corner clearings at Birdsong."
		}
		return "Activate at 10+ VP. Win by ruling 3 " + suitName + " clearings at Birdsong."
	case KindFavor:
		return "Craft (" + costText(c) + "): remove all enemy pieces in every " + suitName + " clearing; +1 VP per building/token removed."
	case KindItem:
		return "Craft (" + costText(c) + "): take a " + c.Item + " (+" + itoa(c.VP) + " VP)."
	default:
		return effectDesc(c)
	}
}

func effectDesc(c *CardDef) string {
	switch c.Effect {
	case "stand-deliver":
		return "Persistent · Birdsong: take a random card from a player; they score 1 VP."
	case "tax-collector":
		return "Persistent · Daylight (once): remove one of your warriors to draw 1 card."
	case "command-warren":
		return "Persistent · Daylight: may fight a battle as a free action."
	case "burrow-bank":
		return "Persistent · Birdsong: you and another player each draw 1 card."
	case "cobbler":
		return "Persistent · Evening: may take a free move."
	case "scouting-party":
		return "Persistent · As attacker you are unaffected by Ambush cards."
	case "codebreakers":
		return "Persistent · Daylight (once): look at another player's hand."
	case "royal-claim":
		return "One-shot · Birdsong: discard to score 1 VP per clearing you rule."
	case "armorers":
		return "One-shot · Battle: discard to ignore all rolled hits taken."
	case "sappers":
		return "One-shot · Battle (defender): discard to deal 1 extra hit."
	case "brutal-tactics":
		return "One-shot · Battle (attacker): discard to deal 1 extra hit; the defender scores 1 VP."
	}
	return "Craft (" + costText(c) + ") for a persistent effect."
}
