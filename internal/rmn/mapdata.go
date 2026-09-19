package rmn

// CardInfo describes a card in a deck manifest.
type CardInfo struct {
	ID   string
	Suit string
	Name string
	Kind string
}

// MapData describes a map's fixed suits, adjacency, forests, and ruins.
type MapData struct {
	Name      string
	FixedSuit bool
	Suits     map[string]string
	Adj       map[string][]string
	Forests   map[string][]string
	Ruins     []string
}

var standardDeck = buildStandardDeck()

func buildStandardDeck() map[string]CardInfo {
	m := map[string]CardInfo{}
	names := []struct{ name, kind string }{
		{"ambush", "ambush"}, {"dominance", "dominance"}, {"favor", "favor"},
		{"betterburrowbank", "normal"}, {"cobbler", "normal"}, {"commandwarren", "normal"},
		{"codebreakers", "normal"}, {"royalclaim", "normal"}, {"sappers", "normal"},
		{"scoutingparty", "normal"}, {"standanddeliver", "normal"}, {"taxcollector", "normal"},
		{"armorers", "normal"}, {"brutaltactics", "normal"},
	}
	for _, suit := range []string{"F", "R", "M"} {
		for i, n := range names {
			id := suit + pad2(i+1)
			m[id] = CardInfo{ID: id, Suit: suit, Name: n.name, Kind: n.kind}
		}
	}
	bird := []struct{ name, kind string }{
		{"ambush", "ambush"}, {"ambush", "ambush"}, {"dominance", "dominance"},
		{"royalclaim", "normal"}, {"sappers", "normal"}, {"scoutingparty", "normal"},
		{"standanddeliver", "normal"}, {"taxcollector", "normal"}, {"armorers", "normal"},
		{"brutaltactics", "normal"}, {"betterburrowbank", "normal"}, {"cobbler", "normal"},
	}
	for i, n := range bird {
		id := "B" + pad2(i+1)
		m[id] = CardInfo{ID: id, Suit: "B", Name: n.name, Kind: n.kind}
	}
	return m
}

func pad2(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

// CardByID returns manifest info for a card ID.
func CardByID(id string) (CardInfo, bool) {
	c, ok := standardDeck[id]
	return c, ok
}

// DeckCards returns the card IDs of a named deck in canonical order.
func DeckCards(deck string) []string {
	if deck != "standard" {
		return nil
	}
	order := []string{"F", "R", "M", "B"}
	counts := map[string]int{"F": 14, "R": 14, "M": 14, "B": 12}
	var out []string
	for _, s := range order {
		for i := 1; i <= counts[s]; i++ {
			out = append(out, s+pad2(i))
		}
	}
	return out
}

var itemCounts = map[string]int{
	"boot": 2, "bag": 2, "coin": 2, "crossbow": 2, "sword": 2,
	"hammer": 1, "tea": 1, "torch": 1,
}

// ItemCatalogue returns all base item instance IDs in canonical order.
func ItemCatalogue() []string {
	var out []string
	for _, t := range []string{"boot", "bag", "coin", "crossbow", "sword", "hammer", "tea", "torch"} {
		for i := 1; i <= itemCounts[t]; i++ {
			out = append(out, "i."+t+"#"+itoa(i))
		}
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

var autumnMap = &MapData{
	Name:      "autumn",
	FixedSuit: true,
	Suits: map[string]string{
		"C1": "R", "C2": "M", "C3": "F", "C4": "R", "C5": "M", "C6": "F",
		"C7": "R", "C8": "M", "C9": "F", "C10": "R", "C11": "M", "C12": "F",
	},
	Adj: map[string][]string{
		"C1": {"C2", "C11"}, "C2": {"C1", "C3", "C12"}, "C3": {"C2", "C4"},
		"C4": {"C3", "C5"}, "C5": {"C4", "C6"}, "C6": {"C5", "C7", "C8"},
		"C7": {"C6", "C9"}, "C8": {"C6", "C10"}, "C9": {"C7", "C10"},
		"C10": {"C8", "C9", "C11"}, "C11": {"C1", "C10", "C12"}, "C12": {"C2", "C11"},
	},
	Forests: map[string][]string{
		"F6_7_8":     {"C6", "C7", "C8"},
		"F1_2_11_12": {"C1", "C2", "C11", "C12"},
		"F3_4_5":     {"C3", "C4", "C5"},
		"F9_10_11":   {"C9", "C10", "C11"},
	},
	Ruins: []string{"C2", "C3", "C7", "C11"},
}

// MapByName returns built-in map data.
func MapByName(name string) *MapData {
	switch name {
	case "autumn", "fall":
		return autumnMap
	}
	return nil
}
