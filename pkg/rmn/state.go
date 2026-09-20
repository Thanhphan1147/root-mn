package rmn

import (
	"sort"
	"strings"
)

// sortedClearings returns clearing IDs in sorted order.
func sortedClearings(s *State) []string {
	ids := make([]string, 0, len(s.Clearings))
	for id := range s.Clearings {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Piece is a placed building or token instance.
type Piece struct {
	Ref    string `json:"ref"`
	Owner  string `json:"owner"`
	Kind   string `json:"kind"`
	Serial int    `json:"serial,omitempty"`
}

// Clearing holds per-clearing occupants.
type Clearing struct {
	ID        string         `json:"id"`
	Suit      string         `json:"suit,omitempty"`
	Warriors  map[string]int `json:"warriors,omitempty"`
	Buildings []Piece        `json:"buildings,omitempty"`
	Tokens    []Piece        `json:"tokens,omitempty"`
	Wood      int            `json:"wood,omitempty"`
	Ruin      string         `json:"ruin,omitempty"`
}

// Item is a tracked item instance.
type Item struct {
	ID        string `json:"id"`
	Owner     string `json:"owner,omitempty"`
	Zone      string `json:"zone"`
	Exhausted bool   `json:"exhausted,omitempty"`
	Damaged   bool   `json:"damaged,omitempty"`
}

// Faction is per-faction state.
type Faction struct {
	ID           string              `json:"id"`
	Kind         string              `json:"kind"`
	Name         string              `json:"name,omitempty"`
	Seat         int                 `json:"seat,omitempty"`
	Hand         []string            `json:"hand,omitempty"`
	Supporters   []string            `json:"supporters,omitempty"`
	Decree       map[string][]string `json:"decree,omitempty"`
	Viziers      []string            `json:"viziers,omitempty"`
	Leader       string              `json:"leader,omitempty"`
	Character    string              `json:"character,omitempty"`
	Pawn         string              `json:"pawn,omitempty"`
	VP           int                 `json:"vp"`
	VPLocation   string              `json:"vp_location,omitempty"`
	Items        map[string]bool     `json:"items,omitempty"`
	Relationship map[string]string   `json:"relationship,omitempty"`
	Officers     int                 `json:"officers,omitempty"`
	Acolytes     int                 `json:"acolytes,omitempty"`
	Funds        int                 `json:"funds,omitempty"`
	Payments     int                 `json:"payments,omitempty"`
	Quests       []string            `json:"quests,omitempty"`
	Crafted      []string            `json:"crafted,omitempty"`
}

// Removal records the last piece removal for reaction preconditions.
type Removal struct {
	Seq    int      `json:"seq"`
	By     string   `json:"by"`
	At     string   `json:"at"`
	Pieces []string `json:"pieces,omitempty"`
}

// MoveRec records the last move for reaction preconditions.
type MoveRec struct {
	Seq    int      `json:"seq"`
	Who    string   `json:"who"`
	To     string   `json:"to"`
	Pieces []string `json:"pieces,omitempty"`
}

// State is the deterministic game state (SPEC §16.4).
type State struct {
	Round       int                  `json:"round"`
	Phase       string               `json:"phase"`
	Current     string               `json:"current,omitempty"`
	ActionsUsed int                  `json:"actions_used"`
	BirdSpent   int                  `json:"bird_spent"`
	Map         string               `json:"map"`
	Clearings   map[string]*Clearing `json:"clearings"`
	Factions    map[string]*Faction  `json:"factions"`
	Order       []string             `json:"order"`
	Deck        []string             `json:"deck,omitempty"`
	Discard     []string             `json:"discard,omitempty"`
	Items       map[string]*Item     `json:"items,omitempty"`
	Ruins       map[string]string    `json:"ruins,omitempty"`
	Supply      map[string]int       `json:"supply,omitempty"`
	QuestDeck   []string             `json:"quest_deck,omitempty"`
	QuestsAvail []string             `json:"quests_avail,omitempty"`
	Winner      []string             `json:"winner,omitempty"`
	LastRemoval *Removal             `json:"last_removal,omitempty"`
	LastMove    *MoveRec             `json:"last_move,omitempty"`
	Hidden      map[string]string    `json:"hidden,omitempty"`
	Seq         int                  `json:"seq"`
}

// NewState builds the initial state from a header.
func NewState(h *Header) *State {
	s := &State{
		Phase:     "S",
		Map:       h.Map,
		Clearings: map[string]*Clearing{},
		Factions:  map[string]*Faction{},
		Items:     map[string]*Item{},
		Ruins:     map[string]string{},
		Supply:    map[string]int{},
		Hidden:    map[string]string{},
	}
	for _, fd := range h.Factions {
		f := &Faction{ID: fd.ID, Kind: fd.Kind, Name: fd.Name, Seat: fd.Seat,
			Decree: map[string][]string{}, Items: map[string]bool{}, Relationship: map[string]string{}}
		s.Factions[fd.ID] = f
		s.Order = append(s.Order, fd.ID)
	}
	// seat order
	for i := 1; i < len(s.Order); i++ {
		for j := i; j > 0 && s.Factions[s.Order[j-1]].Seat > s.Factions[s.Order[j]].Seat; j-- {
			s.Order[j-1], s.Order[j] = s.Order[j], s.Order[j-1]
		}
	}
	s.Current = h.First

	// clearings + suits
	md := MapByName(h.Map)
	if md != nil {
		for id, suit := range md.Suits {
			s.Clearings[id] = &Clearing{ID: id, Suit: suit, Warriors: map[string]int{}}
		}
	}
	for _, cs := range h.Clearings {
		c := s.Clearing(cs.Clearing)
		c.Suit = cs.Suit
	}

	// deck
	for _, id := range DeckCards(h.Deck) {
		s.Deck = append(s.Deck, id)
	}

	// items to supply
	for _, id := range ItemCatalogue() {
		s.Items[id] = &Item{ID: id, Zone: "ISUPPLY"}
	}
	for _, id := range ItemCatalogue() {
		s.Supply[id] = 1
	}
	return s
}

// Clearing returns (creating if needed) a clearing.
func (s *State) Clearing(id string) *Clearing {
	c, ok := s.Clearings[id]
	if !ok {
		c = &Clearing{ID: id, Warriors: map[string]int{}}
		s.Clearings[id] = c
	}
	return c
}

// Faction returns (creating if needed) a faction.
func (s *State) Faction(id string) *Faction {
	f, ok := s.Factions[id]
	if !ok {
		f = &Faction{ID: id, Decree: map[string][]string{}, Items: map[string]bool{}, Relationship: map[string]string{}}
		s.Factions[id] = f
	}
	return f
}

// HomeZone returns the zone pieces of a given kind return to.
func HomeZone(ref string) string {
	if strings.HasPrefix(ref, "i.") {
		return "ISUPPLY"
	}
	return "SUPPLY"
}

func pieceKind(ref string) string {
	_, kind, ok := splitOwner(ref)
	if !ok {
		return ""
	}
	if i := strings.IndexByte(kind, '#'); i >= 0 {
		kind = kind[:i]
	}
	return kind
}

func pieceOwner(ref string) string {
	owner, _, _ := splitOwner(ref)
	return owner
}

// addWarrior adds n warriors of faction to a clearing (n may be negative).
func (s *State) addWarrior(faction, clearing string, n int) {
	c := s.Clearing(clearing)
	if c.Warriors == nil {
		c.Warriors = map[string]int{}
	}
	c.Warriors[faction] += n
	if c.Warriors[faction] <= 0 {
		delete(c.Warriors, faction)
	}
}

func (s *State) warriorCount(faction, clearing string) int {
	if c, ok := s.Clearings[clearing]; ok {
		return c.Warriors[faction]
	}
	return 0
}

func (s *State) addBuilding(owner, clearing, ref string) {
	c := s.Clearing(clearing)
	c.Buildings = append(c.Buildings, Piece{Ref: ref, Owner: owner, Kind: pieceKind(ref)})
}

func (s *State) addToken(owner, clearing, ref string) {
	c := s.Clearing(clearing)
	c.Tokens = append(c.Tokens, Piece{Ref: ref, Owner: owner, Kind: pieceKind(ref)})
}

func (s *State) removeBuilding(owner, clearing, kind string) bool {
	c, ok := s.Clearings[clearing]
	if !ok {
		return false
	}
	for i, p := range c.Buildings {
		if p.Owner == owner && (kind == "" || p.Kind == kind) {
			c.Buildings = append(c.Buildings[:i], c.Buildings[i+1:]...)
			return true
		}
	}
	return false
}

func (s *State) removeToken(owner, clearing, kind string) bool {
	c, ok := s.Clearings[clearing]
	if !ok {
		return false
	}
	for i, p := range c.Tokens {
		if p.Owner == owner && (kind == "" || p.Kind == kind) {
			c.Tokens = append(c.Tokens[:i], c.Tokens[i+1:]...)
			return true
		}
	}
	return false
}

// removeCard removes one occurrence of card from a slice.
func removeCard(list []string, card string) ([]string, bool) {
	for i, c := range list {
		if c == card {
			return append(list[:i:i], list[i+1:]...), true
		}
	}
	return list, false
}

func takeCard(list *[]string, card string) bool {
	out, ok := removeCard(*list, card)
	if ok {
		*list = out
	}
	return ok
}
