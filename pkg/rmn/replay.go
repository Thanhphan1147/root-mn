package rmn

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// StateHash returns the SHA-256 of the canonical JSON encoding of state.
func StateHash(s *State) string {
	b, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return "0x" + hex.EncodeToString(sum[:])
}

// ReplayResult is the outcome of folding a full log.
type ReplayResult struct {
	State    *State   `json:"state"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Events   int      `json:"events"`
	Hash     string   `json:"hash"`
}

// Replay folds every event of a log into a fresh state, validating structural
// invariants along the way.
func Replay(log *Log) *ReplayResult {
	res := &ReplayResult{}
	if log.Header == nil {
		log.Header = &Header{}
	}
	st := NewState(log.Header)
	res.State = st
	for _, pe := range log.Errors {
		res.Errors = append(res.Errors, fmt.Sprintf("line %d [%s] %s", pe.Line, pe.Code, pe.Msg))
	}
	prevSeq := 0
	prevRound, prevPhase := 0, 0
	seen := map[string]bool{}
	for _, fd := range log.Header.Factions {
		seen[fd.ID] = true
	}
	for _, ev := range log.Events {
		// seq
		if ev.Seq != prevSeq+1 {
			res.Errors = append(res.Errors, fmt.Sprintf("seq %d: expected %d (E2204)", ev.Seq, prevSeq+1))
		}
		prevSeq = ev.Seq
		// index monotonic
		r, p := ev.Round, phaseRank(ev.Phase)
		if r < prevRound || (r == prevRound && p < prevPhase) {
			res.Errors = append(res.Errors, fmt.Sprintf("seq %d: index %s decreases (E2406)", ev.Seq, ev.Index()))
		}
		prevRound, prevPhase = r, p
		// actor
		if ev.Actor != "SYS" && !seen[ev.Actor] {
			res.Errors = append(res.Errors, fmt.Sprintf("seq %d: actor %s not declared (E2206)", ev.Seq, ev.Actor))
		}
		// cause
		if ev.Cause != 0 && ev.Cause >= ev.Seq {
			res.Errors = append(res.Errors, fmt.Sprintf("seq %d: cause %d must reference an earlier seq (E2203)", ev.Seq, ev.Cause))
		}
		// setup ordering
		if ev.Round > 0 && len(log.Events) > 0 && ev.Seq < setupEnd(log.Events) {
			res.Errors = append(res.Errors, fmt.Sprintf("seq %d: non-setup event before setup completed (E2405)", ev.Seq))
		}
		fr := Fold(st, ev)
		res.Errors = append(res.Errors, fr.Errors...)
		res.Warnings = append(res.Warnings, fr.Warnings...)
		res.Events++
	}
	res.Hash = StateHash(st)
	// verify checkpoints
	for _, cp := range log.Checkpoints {
		if cp.Seq == res.Events && cp.Hash != "" && cp.Hash != res.Hash {
			res.Warnings = append(res.Warnings, fmt.Sprintf("checkpoint seq %d hash mismatch", cp.Seq))
		}
	}
	return res
}

func setupEnd(events []*Event) int {
	end := 0
	for _, ev := range events {
		if ev.Round == 0 {
			end = ev.Seq
		}
	}
	return end
}

// ValidateEvent checks schema-level constraints for a single event against a
// state, returning diagnostics. It does not fold.
func ValidateEvent(st *State, ev *Event) []string {
	var errs []string
	def, ok := lookupIntent(ev.Intent)
	if !ok {
		return []string{fmt.Sprintf("seq %d: unknown intent %s (E1301)", ev.Seq, ev.Intent)}
	}
	for _, od := range def.Operands {
		if od.Optional {
			continue
		}
		if _, present := ev.Operands[od.Key]; !present {
			errs = append(errs, fmt.Sprintf("seq %d: missing operand %s (E2210)", ev.Seq, od.Key))
		}
	}
	for k := range ev.Operands {
		if def.operand(k) == nil {
			errs = append(errs, fmt.Sprintf("seq %d: unknown operand %s (E2210)", ev.Seq, k))
		}
	}
	switch def.Entropy {
	case "explicit":
		if len(ev.Outcome) == 0 {
			errs = append(errs, fmt.Sprintf("seq %d: outcome required for %s (E2202)", ev.Seq, ev.Intent))
		}
	}
	return errs
}
