package root

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// This file is the notation-driven engine: it parses an RMN line, dispatches it
// to a declared handler (grouped by faction in handlers_*.go), and mutates the
// game state. Handlers build a canonical Action from the line's declared
// operands; Apply validates and executes it with the shared rules code.
//
// Deps: the per-intent handlers live in handlers_generic.go, handlers_mc.go,
// handlers_ed.go, handlers_wa.go and handlers_vb.go and register themselves in
// rmnHandlers.

// rmnEvent is a parsed RMN line.
type rmnEvent struct {
	Seq    int
	Round  int
	Phase  string
	Actor  Faction
	Intent string
	Ops    map[string]string
	Out    map[string]string
}

func (e rmnEvent) op(k string) string { return e.Ops[k] }
func (e rmnEvent) has(k string) bool  { _, ok := e.Ops[k]; return ok }
func (e rmnEvent) outcome(k string) string {
	return e.Out[k]
}

// rmnHandler builds the Action for an event, optionally arming system outcomes
// (dice, etc.) for replay. Returning ok=false falls through to the resolver.
type rmnHandler func(g *Game, ev rmnEvent) (Action, bool, error)

var rmnHandlers = map[string]rmnHandler{}

// register adds a handler for an intent.
func register(intent string, h rmnHandler) { rmnHandlers[intent] = h }

// ApplyRMN parses one RMN line and mutates the game.
func (g *Game) ApplyRMN(line string) error {
	ev, err := parseRMNLine(line)
	if err != nil {
		return err
	}
	// System / structural events are driven by the engine, not a player.
	if ev.Actor == "SYS" || isSystemIntent(ev.Intent) {
		return g.applySystemEvent(ev)
	}
	if h, ok := rmnHandlers[ev.Intent]; ok {
		a, ok, err := h(g, ev)
		if err != nil {
			return err
		}
		if ok {
			rmnHandlerHits++
			return g.applyEvent(a)
		}
	}
	// Safety net while handler coverage grows: resolve the line to the legal
	// action that produces it, then apply.
	if a, ok := g.resolveByLine(line); ok {
		rmnFallbackHits++
		rmnFallbackIntents[ev.Intent]++
		return g.Apply(a)
	}
	return fmt.Errorf("ApplyRMN: no handler matched %q", line)
}

// RMNCoverage reports how many lines were handled by a declared handler vs the
// line resolver.
func RMNCoverage() (viaHandler, viaFallback int) { return rmnHandlerHits, rmnFallbackHits }

// RMNFallbackIntents counts, per intent, how many lines still use the resolver.
func RMNFallbackIntents() map[string]int { return rmnFallbackIntents }

var (
	rmnHandlerHits     int
	rmnFallbackHits    int
	rmnFallbackIntents = map[string]int{}
)

// ResolveRMN reports whether the line can be handled (handler or resolver).
func (g *Game) ResolveRMN(line string) (viaHandler bool, err error) {
	ev, err := parseRMNLine(line)
	if err != nil {
		return false, err
	}
	if ev.Actor == "SYS" || isSystemIntent(ev.Intent) {
		return true, nil
	}
	if _, ok := rmnHandlers[ev.Intent]; ok {
		return true, nil
	}
	if _, ok := g.resolveByLine(line); ok {
		return false, nil
	}
	return false, fmt.Errorf("unresolved: %q", line)
}

func isSystemIntent(intent string) bool {
	switch intent {
	case "assign-ruins", "shuffle", "deal", "roll", "rng":
		return true
	}
	return false
}

// resolveByLine finds the legal action whose RMN line equals line, in stable
// (ID) order so replay is not map-order dependent.
func (g *Game) resolveByLine(line string) (Action, bool) {
	base := len(g.RMNLog)
	legal := g.LegalActions()
	sort.Slice(legal, func(i, j int) bool { return legal[i].ID < legal[j].ID })
	for i := range legal {
		c := g.Clone()
		if err := c.Apply(legal[i]); err != nil {
			continue
		}
		if len(c.RMNLog) == base+1 && c.RMNLog[base] == line {
			return legal[i], true
		}
	}
	return Action{}, false
}

// applyEvent executes an action built from an RMN line without the ID lookup
// Apply performs (the line is authoritative). It still runs the end-of-action
// hooks and records the RMN line.
func (g *Game) applyEvent(a Action) error {
	defer g.checkWin()
	defer g.maybeFieldHospitals()
	round, phase := g.Round, g.Phase
	err := g.applyResolved(a)
	g.NextRoll = nil // don't leak armed dice into the next action
	if err == nil {
		g.recordRMN(a, round, phase)
	}
	return err
}

// applySystemEvent handles SYS/structural events (assign-ruins, shuffle, deal,
// roll, rng). The deck and ruins are seeded, so during replay these are
// informational; they exist so a log is self-describing.
func (g *Game) applySystemEvent(ev rmnEvent) error {
	return nil
}

// --- parsing ---

func parseRMNLine(line string) (rmnEvent, error) {
	f := splitRMN(strings.TrimSpace(line))
	if len(f) < 4 {
		return rmnEvent{}, fmt.Errorf("rmn: short line %q", line)
	}
	ev := rmnEvent{Ops: map[string]string{}, Out: map[string]string{}}
	ev.Seq, _ = strconv.Atoi(f[0])
	rp := strings.SplitN(f[1], ".", 2)
	ev.Round, _ = strconv.Atoi(rp[0])
	if len(rp) > 1 {
		ev.Phase = rp[1]
	}
	ev.Actor = Faction(f[2])
	ev.Intent = f[3]
	for _, tok := range f[4:] {
		if tok == "->" {
			continue
		}
		if strings.HasPrefix(tok, "{") {
			parseKVSet(strings.Trim(tok, "{}"), ev.Out)
			continue
		}
		if k, v, ok := cutKV(tok); ok {
			ev.Ops[k] = unquote(v)
		}
	}
	return ev, nil
}

// splitRMN splits on spaces but keeps quoted strings and bracketed groups whole.
func splitRMN(s string) []string {
	var out []string
	var b strings.Builder
	depth := 0
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQuote = !inQuote
			b.WriteByte(c)
		case !inQuote && (c == '(' || c == '[' || c == '{'):
			depth++
			b.WriteByte(c)
		case !inQuote && (c == ')' || c == ']' || c == '}'):
			depth--
			b.WriteByte(c)
		case !inQuote && c == ' ' && depth == 0:
			if b.Len() > 0 {
				out = append(out, b.String())
				b.Reset()
			}
		default:
			b.WriteByte(c)
		}
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

func cutKV(tok string) (string, string, bool) {
	i := strings.IndexByte(tok, '=')
	if i < 0 {
		return "", "", false
	}
	return tok[:i], tok[i+1:], true
}

func parseKVSet(s string, dst map[string]string) {
	for _, part := range strings.Split(s, ",") {
		if k, v, ok := cutKV(strings.TrimSpace(part)); ok {
			dst[k] = unquote(v)
		}
	}
}

func unquote(s string) string { return strings.Trim(s, `"`) }

// --- operand helpers ---

// unitItem turns "i.boot" into "boot" (an item type); other units pass through.
func unitItem(v string) string { return strings.TrimPrefix(v, "i.") }

// unitFaction parses a "<n><F>.w" or "<F>.w" warrior group into (amount, faction).
func warriorGroup(v string) (int, Faction) {
	v = strings.TrimSuffix(v, ".w")
	n := 1
	i := 0
	for i < len(v) && v[i] >= '0' && v[i] <= '9' {
		i++
	}
	if i > 0 {
		n, _ = strconv.Atoi(v[:i])
	}
	return n, Faction(v[i:])
}

// listItems parses "(a+b+c)" or "a" into a slice.
func listItems(v string) []string {
	v = strings.Trim(v, "()")
	if v == "" {
		return nil
	}
	return strings.Split(v, "+")
}

// TryRMN applies a player-entered RMN line, validating it against the rules.
// Errors are short and UI-friendly.
func (g *Game) TryRMN(line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return fmt.Errorf("empty RMN")
	}
	ev, err := parseRMNLine(line)
	if err != nil {
		return fmt.Errorf("malformed RMN")
	}
	if a, ok := g.resolveByLine(line); ok {
		return g.Apply(a)
	}
	if msg := g.explainIllegal(ev); msg != "" {
		return fmt.Errorf("%s", msg)
	}
	return fmt.Errorf("not a legal move")
}

// explainIllegal returns a short reason for a few common illegal intents.
func (g *Game) explainIllegal(ev rmnEvent) string {
	switch ev.Intent {
	case "C:build", "mc-build":
		b := rmnBuildingInverse(strings.TrimSuffix(strings.TrimPrefix(ev.op("building"), "MC.b."), "#1"))
		c := ev.op("at")
		cl := g.Clearings[c]
		if cl == nil {
			return "unknown clearing " + c
		}
		if !g.Rules(MC, c) {
			return "you do not rule " + c
		}
		if cl.FreeSlots() <= 0 {
			return "no more building slots at " + c
		}
		if g.buildingsOf(MC, c, b) > 0 {
			return "already a " + b + " at " + c
		}
		p := g.Players[MC]
		remaining := p.Sawmills
		switch b {
		case "workshop":
			remaining = p.Workshops
		case "recruiter":
			remaining = p.Recruiters
		}
		if remaining <= 0 {
			return "no " + b + " left"
		}
		if g.availableWood(p, c) < MCBuildCost[7-remaining] {
			return "not enough wood at " + c
		}
	case "move":
		if g.Clearings[ev.op("from")] == nil {
			return "unknown clearing " + ev.op("from")
		}
	}
	return ""
}
