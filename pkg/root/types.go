package root

import (
	"fmt"
	"sort"
	"strings"
)

// Building is a placed building.
type Building struct {
	Owner Faction
	Type  string // sawmill, workshop, recruiter, roost, base-fox, base-rabbit, base-mouse
}

// Token is a placed token.
type Token struct {
	Owner Faction
	Type  string // keep, sympathy
}

// ItemState tracks a Vagabond item instance.
type ItemState struct {
	Type    string
	Zone    string // track, satchel, damaged
	FaceUp  bool
	Damaged bool
}

// Clearing is runtime state for one clearing.
type Clearing struct {
	ID        string
	Suit      Suit
	Slots     int
	Adj       []string
	Warriors  map[Faction]int
	Buildings []Building
	Tokens    []Token
	Wood      int
	Sympathy  Faction // "" or WA
	Ruin      bool
	RuinItem  string
}

// Player is per-faction state.
type Player struct {
	Faction Faction
	Name    string
	Seat    int
	VP      int

	Hand         []string
	Crafted      []string // persistent effect card ids in play
	CraftedItems []string // item types crafted (non-VB)
	UsedThisTurn map[string]bool
	Revealed     map[Faction]bool // hands this player has looked at this turn

	// Marquise
	WoodSupply   int
	Sawmills     int // remaining on track
	Workshops    int
	Recruiters   int
	KeepClearing string

	// Eyrie
	Leader         string
	Decree         map[string][]string
	Viziers        []string
	RetiredLeaders []string

	// Woodland Alliance
	Supporters []string
	Officers   int
	Bases      map[Suit]bool

	// Vagabond
	Character     string
	Pawn          string
	Items         map[string]*ItemState // instance id -> state
	ItemSeq       int
	Relationships map[Faction]string // hostile, indifferent, amiable, friendly, allied
	AidCount      map[Faction]int
	Quests        []string
	Coalition     Faction
}

// LogEntry is a human-readable annotation.
type LogEntry struct {
	Seq   int            `json:"seq"`
	Round int            `json:"round"`
	Phase string         `json:"phase"`
	Actor Faction        `json:"actor"`
	Kind  string         `json:"kind"`
	Text  string         `json:"text"`
	Data  map[string]any `json:"data,omitempty"`
}

// PendingKind enumerates deferred choices.
type PendingKind string

const (
	PendingBattleHits     PendingKind = "battle-hits"
	PendingBattleAmbush   PendingKind = "battle-ambush"
	PendingBattleEffects  PendingKind = "battle-effects"
	PendingDiscard        PendingKind = "discard-down"
	PendingFieldHospitals PendingKind = "field-hospitals"
)

// Pending is a deferred choice the engine needs before proceeding.
type Pending struct {
	Kind      PendingKind
	Player    Faction
	Remaining int
	Context   map[string]any
}

// Game is the full game state.
type Game struct {
	Map       string
	Clearings map[string]*Clearing
	Players   map[Faction]*Player
	Order     []Faction
	First     Faction

	Deck               []string
	Discard            []string
	AvailableDominance []string
	ItemSupply         map[string]int
	QuestDeck          []string
	QuestAvail         []string

	Round   int
	Phase   string // S, B, D, E
	Current Faction

	// turn bookkeeping
	ActionsLeft       int
	MarchMovesLeft    int
	RecruitedThisTurn bool
	MilitaryOpsLeft   int
	BirdSpent         int
	ExtraBattleReady  bool
	ExtraBattle       bool
	ExtraMove         bool
	TaxUsed           bool
	VBSlipped         bool

	Pending        *Pending
	Battle         *BattleState
	FH             []FHRec
	DecreeQueue    []DecreeItem
	EDAdded        int
	EDBirdAdded    bool
	EDNeedsLeader  bool
	EDTurmoilRest  bool
	EDDayStage     string
	MCDayStage     string
	WAEveningDrawn bool
	Winner         []Faction
	Log            []LogEntry
	RMNLog         []string
	Seq            int
	RngSeed        uint64
	Relaxed        bool
	SetupMode      bool
	SetupStage     string
	SetupPlaced    map[string]bool

	// last removal for Outrage
	LastRemoveSeq       int
	LastRemovedSympathy map[string]Faction // clearing -> offender (unused, kept simple)

	// DrawnThisAction collects cards drawn during the current action so the RMN
	// line can record them as the `drawn` outcome (redaction hides them from
	// opponents). Without it, several different draw effects produced identical
	// `draw ...` lines and a replay could not tell them apart.
	DrawnThisAction []string

	// VBRefreshLeft is how many exhausted items the Vagabond may still flip up
	// this Birdsong (3 + 2 per tea on the Refresh track at the start).
	VBRefreshLeft int

	// VBStolenThisAction is the card the Vagabond's Thief stole this action, so
	// the RMN can record it as the `revealed` outcome.
	VBStolenThisAction string

	// SDStolenThisAction is the card Stand and Deliver! took this action (a
	// random pick from the target's hand); it is recorded so a replay takes the
	// same card instead of re-rolling.
	SDStolenThisAction string

	// NextSteal, when non-empty, supplies the card Stand and Deliver! must take
	// on replay.
	NextSteal string

	// NextRoll, when non-empty, supplies the next battle dice so a replayed
	// line reproduces the recorded roll (see Roll).
	NextRoll []int

	// NextShuffle holds the logged `after` order for a deck recycle that
	// happened mid-action. It is consumed by recycleDeck at the exact moment the
	// deck empties, so a replay draws the same cards before/after the recycle
	// instead of applying the shuffle too early (see AGENTS.md §3).
	NextShuffle [][]string
}

// NewGame creates a game for the given factions (2-4, base only) on autumn.
func NewGame(factions []Faction, first Faction, seed uint64) *Game {
	g := &Game{
		Map:       "autumn",
		Relaxed:   false,
		Clearings: map[string]*Clearing{},
		Players:   map[Faction]*Player{},
		Deck:      []string{},
		RngSeed:   seed,
	}
	m := GetMap("autumn")
	for _, id := range m.ClearingList() {
		cd := m.Clearings[id]
		g.Clearings[id] = &Clearing{
			ID: id, Suit: cd.Suit, Slots: cd.Slots, Adj: append([]string{}, cd.Adj...),
			Warriors: map[Faction]int{}, Ruin: cd.Ruin,
		}
	}
	for _, f := range factions {
		g.Players[f] = &Player{
			Faction: f, Hand: []string{}, Decree: map[string][]string{},
			Bases: map[Suit]bool{}, Items: map[string]*ItemState{},
			Relationships: map[Faction]string{}, AidCount: map[Faction]int{},
			UsedThisTurn: map[string]bool{}, Revealed: map[Faction]bool{},
		}
		g.Order = append(g.Order, f)
	}
	g.First = first
	g.Current = first
	g.Round = 0
	g.Phase = "S"
	g.resetDeck()
	for _, f := range factions {
		if f == MC {
			p := g.Players[f]
			p.WoodSupply = 8
			p.Sawmills, p.Workshops, p.Recruiters = 6, 6, 6
		}
	}
	return g
}

func (g *Game) resetDeck() {
	g.Deck = nil
	for _, c := range buildDeck() {
		g.Deck = append(g.Deck, c.ID)
	}
	// Remove dominance for 2-player games.
	if len(g.Players) == 2 {
		var kept []string
		for _, id := range g.Deck {
			c, _ := Card(id)
			if c.Kind != KindDominance {
				kept = append(kept, id)
			}
		}
		g.Deck = kept
	}
	before := cloneSlice(g.Deck)
	g.shuffle()
	after := cloneSlice(g.Deck)
	g.recordSystemLine("shuffle", "zone=DECK reason=setup", fmt.Sprintf("before=[%s] after=[%s]", strings.Join(before, ","), strings.Join(after, ",")))
}

// shuffle uses a deterministic xorshift seeded by RngSeed.
func (g *Game) shuffle() {
	for i := len(g.Deck) - 1; i > 0; i-- {
		g.RngSeed ^= g.RngSeed << 13
		g.RngSeed ^= g.RngSeed >> 7
		g.RngSeed ^= g.RngSeed << 17
		j := int(g.RngSeed % uint64(i+1))
		g.Deck[i], g.Deck[j] = g.Deck[j], g.Deck[i]
	}
}

// Roll returns two battle dice (0..3). If outcomes were armed from a replay
// line (see NextRoll) they are consumed here, so replay reproduces the recorded
// dice instead of re-rolling.
func (g *Game) Roll() (int, int) {
	next := func() int {
		g.RngSeed ^= g.RngSeed << 13
		g.RngSeed ^= g.RngSeed >> 7
		g.RngSeed ^= g.RngSeed << 17
		return int(g.RngSeed % 4)
	}
	a, b := next(), next()
	if len(g.NextRoll) >= 2 {
		// A replay line supplied the dice; use them but keep the RNG advancing so
		// later draws/shuffles stay in sync.
		a, b = g.NextRoll[0], g.NextRoll[1]
		g.NextRoll = g.NextRoll[2:]
	}
	return a, b
}

// Clearing returns a clearing by id.
func (g *Game) Clearing(id string) *Clearing { return g.Clearings[id] }

// Player returns a player by faction.
func (g *Game) Player(f Faction) *Player { return g.Players[f] }

// Actor returns the faction that must act now: the pending player when the
// engine is waiting on a deferred choice, otherwise the current player.
func (g *Game) Actor() Faction {
	if g.Pending != nil {
		return g.Pending.Player
	}
	return g.Current
}

// Rules reports whether f rules clearing c (Eyrie wins ties).
func (g *Game) Rules(f Faction, c string) bool {
	cl := g.Clearings[c]
	if cl == nil {
		return false
	}
	val := func(x Faction) int {
		n := cl.Warriors[x]
		for _, b := range cl.Buildings {
			if b.Owner == x {
				n++
			}
		}
		return n
	}
	mine := val(f)
	best := 0
	for other := range g.Players {
		if other == f {
			continue
		}
		if v := val(other); v > best {
			best = v
		}
	}
	if f == ED {
		// Lords of the Forest: ED rules ties when it has a piece present.
		return mine >= best && mine > 0
	}
	return mine > best
}

// ruleValue is warriors+buildings.
func (g *Game) ruleValue(f Faction, c string) int {
	cl := g.Clearings[c]
	n := cl.Warriors[f]
	for _, b := range cl.Buildings {
		if b.Owner == f {
			n++
		}
	}
	return n
}

// FreeSlots returns the number of available building slots. A ruin occupies one
// slot until the Vagabond explores it.
func (cl *Clearing) FreeSlots() int {
	n := cl.Slots - len(cl.Buildings)
	if cl.Ruin {
		n--
	}
	if n < 0 {
		n = 0
	}
	return n
}

// pieceCount counts all pieces of a faction in a clearing.
func (g *Game) pieceCount(f Faction, c string) int {
	cl := g.Clearings[c]
	if cl == nil {
		return 0
	}
	n := cl.Warriors[f]
	for _, b := range cl.Buildings {
		if b.Owner == f {
			n++
		}
	}
	for _, t := range cl.Tokens {
		if t.Owner == f {
			n++
		}
	}
	return n
}

func (g *Game) buildingsOf(f Faction, c, typ string) int {
	n := 0
	for _, b := range g.Clearings[c].Buildings {
		if b.Owner == f && (typ == "" || b.Type == typ) {
			n++
		}
	}
	return n
}

func (g *Game) totalBuildings(f Faction, typ string) int {
	n := 0
	for _, c := range g.Clearings {
		for _, b := range c.Buildings {
			if b.Owner == f && (typ == "" || b.Type == typ) {
				n++
			}
		}
	}
	return n
}

func (g *Game) warriorsOnMap(f Faction) int {
	n := 0
	for _, c := range g.Clearings {
		n += c.Warriors[f]
	}
	return n
}

func (g *Game) sympathyOnMap() int {
	n := 0
	for _, c := range g.Clearings {
		if c.Sympathy == WA {
			n++
		}
	}
	return n
}

// DayStage reports the current faction's Daylight sub-step ("craft" or
// "actions"/"decree"), used by the client to explain why actions are limited.
func (g *Game) DayStage() string {
	switch g.Current {
	case MC:
		return g.MCDayStage
	case ED:
		return g.EDDayStage
	}
	return ""
}

// Logf appends a log entry (capped so persisted state stays small).
func (g *Game) Logf(actor Faction, kind, format string, args ...any) {
	g.Seq++
	g.Log = append(g.Log, LogEntry{
		Seq: g.Seq, Round: g.Round, Phase: g.Phase, Actor: actor, Kind: kind,
		Text: fmt.Sprintf(format, args...),
	})
	if len(g.Log) > 2000 {
		g.Log = g.Log[len(g.Log)-2000:]
	}
}

// WinnerFaction returns the winner if any.
func (g *Game) WinnerFaction() (Faction, bool) {
	if len(g.Winner) > 0 {
		return g.Winner[0], true
	}
	for _, f := range g.Order {
		p := g.Players[f]
		if p.VP >= 30 {
			g.Winner = []Faction{f}
			if vb := g.Players[VB]; vb != nil && vb.Coalition == f {
				g.Winner = append(g.Winner, VB)
			}
			return f, true
		}
	}
	return "", false
}

// checkWin declares a VP winner at 30.
func (g *Game) checkWin() {
	if len(g.Winner) > 0 {
		return
	}
	for _, f := range g.Order {
		if g.Players[f].VP >= 30 {
			g.Winner = []Faction{f}
			if vb := g.Players[VB]; vb != nil && vb.Coalition == f {
				g.Winner = append(g.Winner, VB)
			}
			g.Logf(f, "win", "%s reached 30 VP and wins!", f)
			return
		}
	}
}

// Score adds VP (respecting dominance: a player with an active dominance no
// longer scores).
func (g *Game) Score(f Faction, n int) {
	if n <= 0 {
		return
	}
	p := g.Players[f]
	if p == nil || g.hasDominance(f) {
		return
	}
	p.VP += n
}

func (g *Game) hasDominance(f Faction) bool {
	p := g.Players[f]
	for _, id := range p.Crafted {
		if c, ok := Card(id); ok && c.Kind == KindDominance {
			return true
		}
	}
	return false
}

func (g *Game) addWarrior(f Faction, c string, n int) {
	if n > 0 {
		// Placements are limited by the faction's warrior supply.
		if room := g.warriorSupplyLeft(f); room < n {
			n = room
		}
		if n <= 0 {
			return
		}
	}
	cl := g.Clearings[c]
	cl.Warriors[f] += n
	if cl.Warriors[f] <= 0 {
		delete(cl.Warriors, f)
	}
}

// warriorSupplyLeft returns how many warrior pieces a faction can still place.
func (g *Game) warriorSupplyLeft(f Faction) int {
	supply := WarriorSupply[f]
	if supply <= 0 {
		return 1 << 30 // no supply limit (e.g. the Vagabond)
	}
	left := supply - g.warriorsOnMap(f)
	if left < 0 {
		left = 0
	}
	return left
}

func (g *Game) removeWarrior(f Faction, c string, n int) int {
	cl := g.Clearings[c]
	if n > cl.Warriors[f] {
		n = cl.Warriors[f]
	}
	cl.Warriors[f] -= n
	if cl.Warriors[f] <= 0 {
		delete(cl.Warriors, f)
	}
	return n
}

// clearingsSorted returns clearing ids sorted.
func (g *Game) clearingsSorted() []string {
	ids := make([]string, 0, len(g.Clearings))
	for id := range g.Clearings {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// playersOrdered returns players in seat order.
func (g *Game) playersOrdered() []*Player {
	out := make([]*Player, 0, len(g.Order))
	for _, f := range g.Order {
		out = append(out, g.Players[f])
	}
	return out
}

// recycleDeck refills an empty Deck from the Discard pile and shuffles it. When
// a replay logged the shuffle (`NextShuffle`), the recorded order is applied at
// this exact moment so draws before and after the recycle line up.
func (g *Game) recycleDeck() {
	if len(g.NextShuffle) > 0 {
		g.Deck = cloneSlice(g.NextShuffle[0])
		g.NextShuffle = g.NextShuffle[1:]
		g.Discard = nil
		return
	}
	g.Deck = append(g.Deck, g.Discard...)
	g.Discard = nil
	before := cloneSlice(g.Deck)
	g.shuffle()
	after := cloneSlice(g.Deck)
	g.recordSystemLine("shuffle", "zone=DECK reason=recycle", fmt.Sprintf("before=[%s] after=[%s]", strings.Join(before, ","), strings.Join(after, ",")))
}

// drawCards draws n cards for f, reshuffling as needed.
func (g *Game) drawCards(f Faction, n int) {
	p := g.Players[f]
	for i := 0; i < n; i++ {
		if len(g.Deck) == 0 {
			g.recycleDeck()
		}
		if len(g.Deck) == 0 {
			return
		}
		p.Hand = append(p.Hand, g.Deck[0])
		g.DrawnThisAction = append(g.DrawnThisAction, g.Deck[0])
		g.Deck = g.Deck[1:]
	}
}

func removeStr(list []string, s string) ([]string, bool) {
	for i, x := range list {
		if x == s {
			return append(list[:i:i], list[i+1:]...), true
		}
	}
	return list, false
}

func takeStr(list *[]string, s string) bool {
	out, ok := removeStr(*list, s)
	if ok {
		*list = out
	}
	return ok
}

// cardSuit returns a card's suit.
func cardSuit(id string) Suit {
	if c, ok := Card(id); ok {
		return c.Suit
	}
	return ""
}

// matches reports whether a card or piece suit matches the target suit
// (Bird is wild).
func matches(have, want Suit) bool { return have == want || have == Bird || want == Bird }
