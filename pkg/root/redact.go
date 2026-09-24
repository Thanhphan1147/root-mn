package root

import "strings"

// hiddenLogKinds are log entries whose text can reveal hidden cards.
var hiddenLogKinds = map[string]bool{
	"deal": true, "mobilize": true, "aid": true,
	"stand-deliver": true, "special": true, "outrage": true,
}

// hiddenRMNIntents are RMN intents whose operands can reveal hidden cards.
var hiddenRMNIntents = map[string]bool{
	"draw": true, "deal": true, "A:mobilize": true,
	"V:aid": true, "V:day-labor": true, "V:steal": true, "stand-deliver": true,
}

// Redact returns the client-facing snapshot for one viewer: every other
// player's hand and supporters are hidden, the deck is removed, and hidden
// outcomes (draws/deals) are scrubbed from the log and the RMN log. viewer is a
// faction id (or "" for a spectator, which hides all hands).
func Redact(g *Game, viewer string) map[string]any {
	cp := g.Clone()
	viewerPlayer := cp.Players[Faction(viewer)]
	for f, p := range cp.Players {
		if string(f) == viewer {
			continue
		}
		// Hands the viewer has looked at (e.g. Codebreakers) stay visible.
		if viewerPlayer != nil && viewerPlayer.Revealed[f] {
			continue
		}
		for i := range p.Hand {
			p.Hand[i] = "??"
		}
		for i := range p.Supporters {
			p.Supporters[i] = "??"
		}
	}
	cp.Deck = nil
	// Ruin contents are hidden from everyone until explored.
	for _, c := range cp.Clearings {
		c.RuinItem = ""
	}

	snap := Snapshot(cp)
	snap["hash"] = ""
	// The acting player is the pending player when the engine is waiting on a
	// deferred choice, otherwise the turn player. Only they receive legal
	// actions; everyone can see whose choice it is, but hidden contexts are
	// stripped.
	actor := g.Current
	if g.Pending != nil {
		actor = g.Pending.Player
	}
	if string(actor) != viewer {
		snap["legal"] = []Action{}
	} else {
		// Legal actions must come from the true state: computing them on the
		// redacted clone would drop options that depend on hidden information
		// (e.g. the Vagabond exploring a ruin needs to see the ruin's item).
		snap["legal"] = g.LegalActions()
	}
	if g.Pending != nil {
		pend := *g.Pending
		pend.Context = nil
		snap["pending"] = &pend
	} else {
		snap["pending"] = nil
	}
	snap["log"] = redactLog(g.Log, viewer)
	snap["rmn"] = redactRMN(g.RMNLog, viewer)
	return snap
}

func redactLog(log []LogEntry, viewer string) []LogEntry {
	out := make([]LogEntry, len(log))
	copy(out, log)
	for i := range out {
		e := &out[i]
		if hiddenLogKinds[e.Kind] && string(e.Actor) != viewer {
			e.Text = redactCardIDs(e.Text)
			e.Data = nil
		}
	}
	return out
}

func redactRMN(lines []string, viewer string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[3] == "assign-ruins" {
			// Which item is under each ruin stays hidden from every player.
			out[i] = strings.Join(fields[:3], " ") + " SYS assign-ruins -> {items=[?]}"
			continue
		}
		if len(fields) < 4 {
			out[i] = line
			continue
		}
		intent := fields[3]
		if hiddenRMNIntents[intent] && rmnOwner(fields) != viewer {
			out[i] = redactCardIDs(line)
		} else {
			out[i] = line
		}
	}
	return out
}

func rmnOwner(fields []string) string {
	for _, f := range fields[4:] {
		if strings.HasPrefix(f, "who=") {
			return strings.TrimPrefix(f, "who=")
		}
	}
	return fields[2]
}

// redactCardIDs replaces card ids (a suit letter F/R/M/B followed by two
// digits) with "??". It scans manually instead of using regexp, which is large
// in the WebAssembly build.
func redactCardIDs(s string) string {
	if !strings.ContainsAny(s, "FRMB") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if i+2 < len(s) && isSuitLetter(s[i]) && isASCIIDigit(s[i+1]) && isASCIIDigit(s[i+2]) {
			b.WriteString("??")
			i += 2
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isSuitLetter(c byte) bool {
	return c == 'F' || c == 'R' || c == 'M' || c == 'B'
}

func isASCIIDigit(c byte) bool { return c >= '0' && c <= '9' }
