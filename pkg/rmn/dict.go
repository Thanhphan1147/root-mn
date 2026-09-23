package rmn

import "strings"

// OperandDef declares one operand of an intent.
type OperandDef struct {
	Key      string
	Type     ValueType
	Optional bool
}

// OutcomeDef declares one outcome field of an intent.
type OutcomeDef struct {
	Key      string
	Type     ValueType
	Optional bool
}

// IntentDef is a dictionary entry from SPEC §10/§11.
type IntentDef struct {
	Name      string
	Namespace string // "" for core verbs
	Operands  []OperandDef
	Outcome   []OutcomeDef
	Entropy   string // "", "explicit", "optional"
	Setup     bool
	Extension bool
}

func (d *IntentDef) operand(key string) *OperandDef {
	for i := range d.Operands {
		if d.Operands[i].Key == key {
			return &d.Operands[i]
		}
	}
	return nil
}

func (d *IntentDef) outcome(key string) *OutcomeDef {
	for i := range d.Outcome {
		if d.Outcome[i].Key == key {
			return &d.Outcome[i]
		}
	}
	return nil
}

var dict = map[string]*IntentDef{}

func op(key string, t ValueType) OperandDef     { return OperandDef{Key: key, Type: t} }
func opOpt(key string, t ValueType) OperandDef  { return OperandDef{Key: key, Type: t, Optional: true} }
func out(key string, t ValueType) OutcomeDef    { return OutcomeDef{Key: key, Type: t} }
func outOpt(key string, t ValueType) OutcomeDef { return OutcomeDef{Key: key, Type: t, Optional: true} }

func core(name string, ops []OperandDef, outs []OutcomeDef, entropy string, setup bool) {
	dict[name] = &IntentDef{Name: name, Operands: ops, Outcome: outs, Entropy: entropy, Setup: setup}
}

func ext(ns, name string, ops []OperandDef, outs []OutcomeDef, entropy string, setup bool) {
	dict[ns+":"+name] = &IntentDef{Name: name, Namespace: ns, Operands: ops, Outcome: outs, Entropy: entropy, Setup: setup, Extension: true}
}

// namespaceKind maps an extension namespace to the faction kind that owns it.
var namespaceKind = map[string]string{
	"C": "marquise", "E": "eyrie", "A": "alliance", "V": "vagabond",
	"L": "cult", "O": "riverfolk", "D": "duchy", "P": "corvid",
	"H": "hundreds", "K": "keepers",
	"HIRE": "hireling", "LM": "landmark", "CARD": "card", "SPY": "spy", "HOMELAND": "homeland",
}

// reactionNamespaces are faction namespaces that may be invoked by a reacting
// or paying actor other than the owning faction (SPEC §9.5).
var reactionIntents = map[string]bool{
	"A:outrage": true, "A:off-turn-score": true, "D:price-of-failure": true,
	"P:extortion-score": true, "O:buy": true,
}

func init() {
	// ---- Universal (§10) ----
	core("move", []OperandDef{op("group", TUnitGroup), op("from", TLocation), op("to", TLocationList)}, nil, "", false)
	core("place", []OperandDef{op("group", TUnitGroup), op("to", TLocationList)}, nil, "", true)
	core("remove", []OperandDef{op("group", TUnitGroup), op("at", TLocation)}, nil, "", false)
	core("exile", []OperandDef{op("group", TUnitGroup), op("at", TLocation)}, nil, "", false)
	core("route", []OperandDef{op("group", TUnitGroup), op("from", TLocation), op("via", TLocation), op("to", TLocation)}, nil, "", false)
	core("close-path", []OperandDef{op("path", TLocation)}, nil, "", false)
	core("open-path", []OperandDef{op("path", TLocation)}, nil, "", false)
	core("move-item", []OperandDef{op("items", TUnitGroup), op("to", TLocation)}, nil, "", false)
	core("set-item-state", []OperandDef{op("items", TUnitGroup), opOpt("exhausted", TBool), opOpt("damaged", TBool)}, nil, "", false)

	core("draw", []OperandDef{op("who", TFaction), op("qty", TInt), op("from", TLocation)}, []OutcomeDef{out("drawn", TCardIDList)}, "explicit", false)
	core("discard", []OperandDef{op("who", TFaction), op("cards", TUnitGroup), opOpt("from", TLocation)}, []OutcomeDef{out("revealed", TCardIDList)}, "optional", false)
	core("give", []OperandDef{op("from", TFaction), op("to", TLocation), op("cards", TUnitGroup)}, []OutcomeDef{out("revealed", TCardIDList)}, "optional", false)
	core("craft", []OperandDef{op("who", TFaction), op("card", TCard), op("produce", TUnit)}, nil, "optional", false)
	core("reveal", []OperandDef{op("who", TFaction), op("cards", TUnitGroup), op("to", TFaction)}, []OutcomeDef{out("revealed", TCardIDList)}, "optional", false)
	core("swap", []OperandDef{op("who", TFaction), op("card_out", TCard), op("card_in", TCard)}, nil, "optional", false)
	core("shuffle", []OperandDef{op("zone", TLocation)}, []OutcomeDef{out("order", TCardIDList)}, "explicit", true)
	core("deal", []OperandDef{op("who", TFaction), op("cards", TUnitGroup)}, nil, "", true)

	core("score", []OperandDef{op("who", TFaction), op("amount", TInt)}, nil, "", false)
	core("lose-vp", []OperandDef{op("who", TFaction), op("amount", TInt)}, nil, "", false)
	core("move-vp-token", []OperandDef{op("who", TFaction), op("to_board", TFaction)}, nil, "", false)
	core("win", []OperandDef{op("factions", TFactionList)}, nil, "", false)

	core("build", []OperandDef{op("who", TFaction), op("building", TUnit), op("at", TLocation), opOpt("cost", TUnitGroup)}, nil, "", false)
	core("recruit", []OperandDef{op("who", TFaction), op("group", TUnitGroup), op("at", TLocationList)}, nil, "", false)
	core("battle", []OperandDef{op("attacker", TFaction), op("defender", TFaction), op("at", TLocation)},
		[]OutcomeDef{out("atk", TInt), out("def", TInt), out("extra_atk", TInt), out("extra_def", TInt)}, "explicit", false)
	core("casualties", []OperandDef{op("at", TLocation), op("group", TUnitGroup)}, nil, "", false)
	core("ambush", []OperandDef{op("who", TFaction), op("at", TLocation), op("card", TCardID)}, []OutcomeDef{out("hits", TInt)}, "explicit", false)
	core("play-dominance", []OperandDef{op("who", TFaction), op("card", TCardID), opOpt("at", TLocation)}, []OutcomeDef{outOpt("target", TFaction)}, "optional", false)
	core("spend-bird", []OperandDef{op("who", TFaction), op("card", TCardID)}, nil, "", false)

	core("phase", []OperandDef{op("phase", TPhase)}, nil, "", false)
	core("turn", []OperandDef{op("who", TFaction)}, nil, "", false)
	core("pass", nil, nil, "", false)

	core("react", []OperandDef{op("who", TFaction), op("trigger_seq", TInt), op("effect", TEnum)}, nil, "optional", false)
	core("outrage", []OperandDef{op("from", TFaction), op("card", TCard), op("at", TLocation)}, []OutcomeDef{out("revealed", TCardIDList)}, "optional", false)
	core("field-hospitals", []OperandDef{op("at", TLocation), op("spend", TCard), op("save", TUnitGroup), op("to", TLocation)}, nil, "", false)
	core("price-of-failure", []OperandDef{op("who", TFaction), op("minister", TUnit)}, nil, "", false)
	core("extortion", []OperandDef{op("who", TFaction), op("at", TLocation)}, nil, "", false)
	core("off-turn-score", []OperandDef{op("who", TFaction), op("amount", TInt), op("trigger_seq", TInt)}, nil, "", false)

	core("commit", []OperandDef{op("who", TFaction), op("domain", TEnum), op("hash", THex)}, nil, "", false)
	core("reveal-commit", []OperandDef{op("who", TFaction), op("domain", TEnum), op("value", TScalar), op("nonce", THex)}, nil, "", false)
	core("expose", []OperandDef{op("who", TFaction), op("at", TLocation), op("guess", TEnum)}, []OutcomeDef{out("actual", TEnum)}, "explicit", false)
	core("notify", []OperandDef{op("who", TFaction), op("text", TText)}, nil, "", false)

	core("roll", []OperandDef{op("who", TFaction), op("dice", TInt), op("sides", TInt)}, []OutcomeDef{out("values", TIntList)}, "explicit", false)
	core("rng", []OperandDef{op("purpose", TEnum)}, []OutcomeDef{out("values", TIntList)}, "explicit", false)
	core("setup", []OperandDef{op("key", TEnum), op("value", TScalar)}, nil, "optional", true)
	core("assign-ruins", nil, []OutcomeDef{out("ruins", TLocationList), out("items", TItemList)}, "explicit", true)
	core("draft-pick", []OperandDef{op("who", TFaction), op("pick", TFactionCode)}, nil, "", true)
	core("place-landmark", []OperandDef{op("landmark", TEnum), op("at", TLocation), op("by", TFaction)}, nil, "", true)
	core("hire", []OperandDef{op("hireling", THireling), op("controller", TFaction), op("markers", TInt)}, nil, "", true)

	// ---- Marquise de Cat (§11.1) ----
	ext("C", "birdsong-wood", []OperandDef{op("at", TLocationList)}, nil, "", false)
	ext("C", "build", []OperandDef{op("who", TFaction), op("building", TUnit), op("at", TLocation), op("cost", TUnitGroup)}, nil, "", false)
	ext("C", "recruit", []OperandDef{op("who", TFaction), op("at", TLocationList)}, nil, "", false)
	ext("C", "field-hospitals", []OperandDef{op("at", TLocation), op("spend", TCard), op("save", TUnitGroup), op("to", TLocation)}, nil, "", false)
	ext("C", "overwork", []OperandDef{op("at", TLocation), op("spend", TCard), op("produce", TUnit)}, nil, "", false)

	// ---- Eyrie Dynasties (§11.2) ----
	ext("E", "decree-add", []OperandDef{op("column", TEnum), op("cards", TUnitGroup)}, nil, "", false)
	ext("E", "discard-decree", nil, nil, "", false)
	ext("E", "appoint-leader", []OperandDef{op("leader", TUnit)}, nil, "", true)
	ext("E", "turmoil", []OperandDef{op("reason", TEnum)}, nil, "", false)
	ext("E", "recruit", []OperandDef{op("at", TLocationList), opOpt("card", TCardID)}, nil, "", false)
	ext("E", "move", []OperandDef{op("group", TUnitGroup), op("from", TLocation), op("to", TLocation), opOpt("card", TCardID)}, nil, "", false)
	ext("E", "battle", []OperandDef{op("defender", TFaction), op("at", TLocation), opOpt("card", TCardID)},
		[]OutcomeDef{out("atk", TInt), out("def", TInt), out("extra_atk", TInt), out("extra_def", TInt)}, "explicit", false)
	ext("E", "build", []OperandDef{op("building", TUnit), op("at", TLocation), opOpt("card", TCardID)}, nil, "", false)
	ext("E", "add-vizier", []OperandDef{op("card", TCard)}, nil, "", false)
	ext("E", "score-roosts", []OperandDef{op("amount", TInt)}, nil, "", false)

	// ---- Woodland Alliance (§11.3) ----
	ext("A", "mobilize", []OperandDef{op("cards", TUnitGroup)}, []OutcomeDef{out("revealed", TCardIDList)}, "optional", true)
	ext("A", "spend-supporters", []OperandDef{op("cards", TUnitGroup)}, []OutcomeDef{out("revealed", TCardIDList)}, "optional", false)
	ext("A", "place-sympathy", []OperandDef{op("at", TLocation), op("spend", TUnitGroup)}, nil, "", false)
	ext("A", "revolt", []OperandDef{op("at", TLocation), op("base", TUnit), op("cards", TUnitGroup)}, nil, "", false)
	ext("A", "organize", []OperandDef{op("group", TUnitGroup), op("from", TLocation), op("to", TLocation)}, nil, "", false)
	ext("A", "train", []OperandDef{op("card", TCard)}, nil, "", false)
	ext("A", "outrage", []OperandDef{op("from", TFaction), op("card", TCard), op("at", TLocation)}, []OutcomeDef{out("revealed", TCardIDList)}, "optional", false)
	ext("A", "martial-law", []OperandDef{op("at", TLocation)}, nil, "", false)
	ext("A", "off-turn-score", []OperandDef{op("amount", TInt), op("trigger_seq", TInt)}, nil, "", false)

	// ---- Vagabond (§11.4) ----
	ext("V", "choose-character", []OperandDef{op("character", TUnit)}, nil, "", true)
	ext("V", "move-pawn", []OperandDef{op("to", TLocation)}, nil, "", true)
	ext("V", "aid", []OperandDef{op("target", TFaction), op("card", TCard), op("take", TOptionalUnit)}, nil, "", false)
	ext("V", "explore", []OperandDef{op("at", TLocation)}, []OutcomeDef{out("item", TItemRef)}, "explicit", false)
	ext("V", "take-quest", []OperandDef{op("quest", TQuestID)}, []OutcomeDef{out("quest", TQuestID)}, "explicit", false)
	ext("V", "complete-quest", []OperandDef{op("quest", TQuestID), op("items", TUnitGroup)}, nil, "", false)
	ext("V", "repair", []OperandDef{op("items", TUnitGroup)}, nil, "", false)
	ext("V", "rest", []OperandDef{op("items", TUnitGroup)}, nil, "", false)
	ext("V", "refresh", []OperandDef{op("items", TUnitGroup)}, nil, "", false)
	ext("V", "exhaust", []OperandDef{op("items", TUnitGroup)}, nil, "", false)
	ext("V", "damage", []OperandDef{op("items", TUnitGroup)}, nil, "", false)
	ext("V", "move-item", []OperandDef{op("items", TUnitGroup), op("to", TLocation)}, nil, "", false)
	ext("V", "set-item-state", []OperandDef{op("items", TUnitGroup), opOpt("exhausted", TBool), opOpt("damaged", TBool)}, nil, "", false)
	ext("V", "relationship", []OperandDef{op("target", TFaction), op("status", TRelStatus)}, nil, "", true)
	ext("V", "coalition", []OperandDef{op("target", TFaction)}, nil, "", false)
	ext("V", "day-labor", []OperandDef{op("card", TCard)}, []OutcomeDef{out("revealed", TCardIDList)}, "optional", false)

	// ---- Extension factions (§11.5–§11.9), dictionary-level ----
	ext("L", "set-outcast", []OperandDef{op("suit", TSuit)}, nil, "", true)
	ext("L", "set-hated-outcast", []OperandDef{op("suit", TSuit)}, nil, "", false)
	ext("L", "build-garden", []OperandDef{op("garden", TUnit), op("at", TLocation)}, nil, "", false)
	ext("L", "sanctify", []OperandDef{op("at", TLocation), op("building", TUnit)}, nil, "", false)
	ext("L", "convert", []OperandDef{op("at", TLocation), op("warrior", TUnit)}, nil, "", false)
	ext("L", "crusade", []OperandDef{op("defender", TFaction), op("at", TLocation)},
		[]OutcomeDef{out("atk", TInt), out("def", TInt), out("extra_atk", TInt), out("extra_def", TInt)}, "explicit", false)
	ext("L", "discard-lost-souls", []OperandDef{op("cards", TUnitGroup)}, []OutcomeDef{out("revealed", TCardIDList)}, "optional", false)
	ext("L", "reveal-outcast", []OperandDef{op("cards", TUnitGroup)}, []OutcomeDef{out("revealed", TCardIDList)}, "explicit", false)

	ext("O", "set-price", []OperandDef{op("service", TEnum), op("price", TInt)}, nil, "", true)
	ext("O", "set-funds", []OperandDef{op("amount", TInt)}, nil, "", false)
	ext("O", "establish-post", []OperandDef{op("post", TUnit), op("at", TLocation)}, nil, "", false)
	ext("O", "move-post", []OperandDef{op("post", TUnit), op("from", TLocation), op("to", TLocation)}, nil, "", false)
	ext("O", "buy", []OperandDef{op("buyer", TFaction), op("service", TEnum), op("cost", TInt), opOpt("card", TCardID), opOpt("at", TLocation), opOpt("via", TLocation)}, []OutcomeDef{out("drawn", TCardIDList)}, "optional", false)
	ext("O", "sell-card", []OperandDef{op("buyer", TFaction), op("card", TCard), op("cost", TInt)}, nil, "optional", false)
	ext("O", "dividends", []OperandDef{op("at", TLocationList)}, nil, "", false)
	ext("O", "export", []OperandDef{op("card", TCard)}, nil, "", false)
	ext("O", "protectionism", []OperandDef{op("spend", TInt)}, nil, "", false)

	ext("D", "sway-minister", []OperandDef{op("minister", TUnit), op("cards", TUnitGroup)}, []OutcomeDef{out("revealed", TCardIDList)}, "explicit", true)
	ext("D", "price-of-failure", []OperandDef{op("minister", TUnit)}, nil, "", false)
	ext("D", "build", []OperandDef{op("building", TUnit), op("at", TLocation)}, nil, "", false)
	ext("D", "recruit", []OperandDef{op("at", TLocationList)}, nil, "", false)
	ext("D", "march", []OperandDef{op("group", TUnitGroup), op("from", TLocation), op("to", TLocation)}, nil, "", false)
	ext("D", "move-tunnel", []OperandDef{op("tunnel", TUnit), op("from", TLocation), op("to", TLocation)}, nil, "", false)
	ext("D", "burrow-move", []OperandDef{op("group", TUnitGroup), op("to", TLocation)}, nil, "", false)
	ext("D", "reveal-sway", []OperandDef{op("cards", TUnitGroup)}, []OutcomeDef{out("revealed", TCardIDList)}, "explicit", false)

	ext("P", "place-plot", []OperandDef{op("at", TLocation)}, []OutcomeDef{out("plot", TEnum)}, "optional", false)
	ext("P", "flip-plot", []OperandDef{op("at", TLocation), op("plot_type", TEnum)}, nil, "", false)
	ext("P", "remove-plot", []OperandDef{op("at", TLocation)}, nil, "", false)
	ext("P", "trick", []OperandDef{op("at_a", TLocation), op("at_b", TLocation)}, nil, "", false)
	ext("P", "expose", []OperandDef{op("at", TLocation), op("guess", TEnum)}, []OutcomeDef{out("actual", TEnum)}, "explicit", false)
	ext("P", "recruit", []OperandDef{op("at", TLocationList)}, nil, "", false)
	ext("P", "extortion-score", []OperandDef{op("amount", TInt), op("trigger_seq", TInt)}, nil, "", false)
	ext("P", "buy-card", []OperandDef{op("seller", TFaction), op("card", TCard), op("cost", TInt)}, nil, "", false)

	ext("H", "choose-mood", []OperandDef{op("mood", TEnum)}, nil, "", false)
	ext("H", "recruit-mob", []OperandDef{op("at", TLocationList)}, nil, "", false)
	ext("H", "move-mob", []OperandDef{op("from", TLocation), op("to", TLocation)}, nil, "", false)
	ext("H", "roll-mob-die", nil, []OutcomeDef{out("values", TIntList)}, "explicit", false)
	ext("H", "raid", []OperandDef{op("at", TLocation)}, nil, "", false)
	ext("H", "advance", []OperandDef{op("at", TLocation)}, nil, "", false)
	ext("H", "oppress", []OperandDef{op("at", TLocation)}, nil, "", false)

	ext("K", "retinue-add", []OperandDef{op("column", TInt), op("cards", TUnitGroup)}, nil, "", false)
	ext("K", "retinue-move", []OperandDef{op("card", TCard), op("from", TInt), op("to", TInt)}, nil, "", false)
	ext("K", "retinue-remove", []OperandDef{op("card", TCard)}, nil, "", false)
	ext("K", "delve", []OperandDef{op("at", TLocation)}, []OutcomeDef{out("relic", TRelic)}, "explicit", false)
	ext("K", "recover", []OperandDef{op("relic", TRelic)}, nil, "", false)
	ext("K", "rebuild", []OperandDef{op("waystation", TUnit), op("at", TLocation), op("side", TEnum)}, nil, "", false)
	ext("K", "flip-relic", []OperandDef{op("relic", TRelic), op("points", TInt)}, nil, "", false)
	ext("K", "faithful", []OperandDef{op("card", TCard)}, nil, "", false)

	ext("HIRE", "activate", []OperandDef{op("who", TFaction), op("hireling", THireling), op("effect", TEnum)}, nil, "optional", false)
	ext("HIRE", "move", []OperandDef{op("group", TUnitGroup), op("from", TLocation), op("to", TLocation)}, nil, "", false)
	ext("HIRE", "promote", []OperandDef{op("hireling", THireling)}, nil, "", false)
	ext("HIRE", "demote", []OperandDef{op("hireling", THireling)}, nil, "", false)
	ext("HIRE", "control", []OperandDef{op("hireling", THireling), op("controller", TFaction), op("markers", TInt)}, nil, "", false)
	ext("HIRE", "return", []OperandDef{op("hireling", THireling)}, nil, "", false)
	ext("LM", "use", []OperandDef{op("landmark", TEnum), op("at", TLocation), op("effect", TEnum)}, nil, "optional", false)
	ext("LM", "close", []OperandDef{op("landmark", TEnum)}, nil, "", false)
	ext("CARD", "effect", []OperandDef{op("who", TFaction), op("card", TCard), op("effect", TEnum), opOpt("targets", TUnitGroup), opOpt("at", TLocation)}, []OutcomeDef{out("revealed", TCardIDList)}, "optional", false)
	ext("SPY", "play", []OperandDef{op("who", TFaction), op("target", TFaction), op("card", TCardID)}, nil, "", false)
	ext("SPY", "trigger", []OperandDef{op("who", TFaction), op("target", TFaction), op("effect", TEnum)}, nil, "optional", false)
	ext("SPY", "return", []OperandDef{op("who", TFaction), op("target", TFaction)}, nil, "", false)
	ext("HOMELAND", "op", []OperandDef{op("op", TEnum)}, nil, "optional", false)
}

// lookupIntent resolves "verb" or "Namespace:verb".
func lookupIntent(name string) (*IntentDef, bool) {
	d, ok := dict[name]
	return d, ok
}

// isExtensionNamespace reports whether ns is a known extension namespace.
func isExtensionNamespace(ns string) bool {
	_, ok := namespaceKind[ns]
	return ok
}

// splitIntent splits a namespaced intent into namespace and verb.
func splitIntent(s string) (ns, verb string) {
	if i := strings.IndexByte(s, ':'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return "", s
}
