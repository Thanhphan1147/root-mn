package rmn

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseLog parses a complete RMN v3 text document.
func ParseLog(data string) *Log {
	log := &Log{Header: &Header{Options: map[string]string{}}}
	data = strings.ReplaceAll(data, "\r\n", "\n")
	data = strings.ReplaceAll(data, "\r", "\n")
	lines := strings.Split(data, "\n")
	seenWan := false
	inHeader := true
	for i, raw := range lines {
		lineNo := i + 1
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "%") {
			if !seenWan {
				toks := tokenize(trimmed)
				if len(toks) == 0 || toks[0] != "%RMN" {
					log.Errors = append(log.Errors, ParseError{lineNo, "E1101", "first directive must be %RMN", raw})
					continue
				}
				seenWan = true
				inHeader = false
			}
			if err := parseDirective(log, trimmed, lineNo); err != nil {
				log.Errors = append(log.Errors, ParseError{lineNo, err.code, err.msg, raw})
			}
			continue
		}
		if !seenWan {
			log.Errors = append(log.Errors, ParseError{lineNo, "E1101", "event before %RMN", raw})
			continue
		}
		inHeader = false
		ev, perr := ParseEventLine(raw, lineNo)
		if perr != nil {
			log.Errors = append(log.Errors, *perr)
			continue
		}
		log.Events = append(log.Events, ev)
	}
	_ = inHeader
	if log.Header.Partial {
		// partial logs relax required-field checks
	}
	return log
}

type perr struct {
	code string
	msg  string
}

func (e *perr) Error() string { return e.code + ": " + e.msg }

func parseDirective(log *Log, line string, lineNo int) *perr {
	toks := tokenize(line)
	name := toks[0]
	args := toks[1:]
	h := log.Header
	kv := func(t string) (string, string, bool) {
		i := strings.IndexByte(t, '=')
		if i < 0 {
			return "", "", false
		}
		return t[:i], t[i+1:], true
	}
	switch name {
	case "%RMN":
		if len(args) != 1 {
			return &perr{"E1102", "%RMN requires a version"}
		}
		parts := strings.SplitN(args[0], ".", 2)
		if len(parts) != 2 || parts[0] != "3" {
			return &perr{"E1001", "unsupported RMN major " + args[0]}
		}
	case "%Game":
		h.Game = arg(args, 0)
	case "%Map":
		h.Map = strings.ToLower(arg(args, 0))
	case "%Clearings":
		for _, a := range args {
			c, s, ok := strings.Cut(a, ":")
			if !ok || !isSuit(s) {
				return &perr{"E1102", "malformed %Clearings entry " + a}
			}
			h.Clearings = append(h.Clearings, ClearingSuit{Clearing: c, Suit: s})
		}
	case "%Deck":
		h.Deck = strings.ToLower(arg(args, 0))
	case "%Faction":
		if len(args) < 2 {
			return &perr{"E1103", "%Faction requires id and kind"}
		}
		fd := FactionDecl{ID: args[0], Kind: args[1]}
		for _, a := range args[2:] {
			k, v, ok := kv(a)
			if !ok {
				return &perr{"E1103", "malformed %Faction field " + a}
			}
			switch k {
			case "seat":
				fd.Seat, _ = strconv.Atoi(v)
			case "name":
				s, _ := unquote(v)
				fd.Name = s
			}
		}
		h.Factions = append(h.Factions, fd)
	case "%Draft":
		d := &Draft{}
		for _, a := range args {
			k, v, _ := kv(a)
			switch k {
			case "pool":
				d.Pool = splitCodes(v)
			case "order":
				d.Order = splitCodes(v)
			case "method":
				d.Method = v
			}
		}
		h.Draft = d
	case "%First":
		h.First = arg(args, 0)
	case "%Winner":
		v, err := parseValue(TFactionList, arg(args, 0))
		if err != nil {
			return &perr{"E1102", err.Error()}
		}
		for _, e := range v.List {
			h.Winner = append(h.Winner, e.Str)
		}
	case "%Seed":
		s := &Seed{}
		for _, a := range args {
			k, v, _ := kv(a)
			if k == "algo" {
				s.Algo = v
			} else if k == "value" {
				s.Value = v
			}
		}
		h.Seed = s
	case "%Landmark":
		if len(args) < 2 {
			return &perr{"E1102", "%Landmark requires id and at="}
		}
		lm := Landmark{ID: args[0]}
		for _, a := range args[1:] {
			k, v, _ := kv(a)
			switch k {
			case "at":
				lm.At = v
			case "by":
				lm.By = v
			}
		}
		h.Landmarks = append(h.Landmarks, lm)
	case "%Hireling":
		if len(args) < 1 {
			return &perr{"E1102", "%Hireling requires an id"}
		}
		hl := Hireling{ID: args[0]}
		for _, a := range args[1:] {
			k, v, _ := kv(a)
			switch k {
			case "type":
				hl.Type = v
			case "controller":
				hl.Controller = v
			case "at":
				hl.At = v
			}
		}
		h.Hirelings = append(h.Hirelings, hl)
	case "%Option":
		if len(args) != 1 {
			return &perr{"E1102", "%Option requires key=value"}
		}
		k, v, ok := kv(args[0])
		if !ok {
			return &perr{"E1102", "malformed %Option"}
		}
		h.Options[k] = v
		if k == "partial" && v == "true" {
			h.Partial = true
		}
	case "%Checkpoint":
		cp := &Checkpoint{}
		for _, a := range args {
			k, v, _ := kv(a)
			switch k {
			case "seq":
				cp.Seq, _ = strconv.Atoi(v)
			case "hash":
				cp.Hash = v
			case "algo":
				cp.Algo = v
			}
		}
		log.Checkpoints = append(log.Checkpoints, cp)
	default:
		return &perr{"E1102", "unknown directive " + name}
	}
	return nil
}

func arg(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

func splitCodes(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ParseEventLine parses one event line (without the leading turn/actor split).
func ParseEventLine(raw string, lineNo int) (*Event, *ParseError) {
	body, note := splitNote(raw)
	toks := tokenize(body)
	if len(toks) < 4 {
		return nil, &ParseError{lineNo, "E1304", "expected 'seq index actor intent'", raw}
	}
	seq, err := strconv.Atoi(toks[0])
	if err != nil || seq < 1 {
		return nil, &ParseError{lineNo, "E1203", "malformed seq", raw}
	}
	round, phase, err := parseIndex(toks[1])
	if err != nil {
		return nil, &ParseError{lineNo, "E1203", err.Error(), raw}
	}
	actor := toks[2]
	intent := toks[3]
	if actor != "SYS" && !isUpperIdent(actor) {
		return nil, &ParseError{lineNo, "E1103", "malformed actor " + actor, raw}
	}
	def, ok := lookupIntent(intent)
	if !ok {
		if ns, _ := splitIntent(intent); ns != "" {
			return nil, &ParseError{lineNo, "E2301", "unregistered namespace " + ns, raw}
		}
		return nil, &ParseError{lineNo, "E1301", "unknown intent " + intent, raw}
	}

	ev := &Event{Seq: seq, Round: round, Phase: phase, Actor: actor, Intent: intent,
		Operands: map[string]Value{}, Outcome: map[string]Value{}, Note: note, Line: lineNo, Raw: raw}

	rest := toks[4:]
	i := 0
	// operands
	for i < len(rest) {
		t := rest[i]
		if t == "->" || strings.HasPrefix(t, "~") {
			break
		}
		k, vraw, ok := strings.Cut(t, "=")
		if !ok {
			return nil, &ParseError{lineNo, "E1302", "malformed operand " + t, raw}
		}
		if _, dup := ev.Operands[k]; dup {
			return nil, &ParseError{lineNo, "E1305", "duplicate operand " + k, raw}
		}
		od := def.operand(k)
		var (
			val Value
			pe  error
		)
		if od != nil {
			val, pe = parseValue(od.Type, vraw)
		} else {
			val, pe = parseAny(vraw)
		}
		if pe != nil {
			return nil, &ParseError{lineNo, "E1307", fmt.Sprintf("operand %s: %v", k, pe), raw}
		}
		ev.Operands[k] = val
		i++
	}
	// outcome
	if i < len(rest) && rest[i] == "->" {
		i++
		if i >= len(rest) || !strings.HasPrefix(rest[i], "{") {
			return nil, &ParseError{lineNo, "E1206", "malformed outcome", raw}
		}
		inner, ok := stripOuter(rest[i], '{', '}')
		if !ok {
			return nil, &ParseError{lineNo, "E1206", "unterminated outcome", raw}
		}
		i++
		if strings.TrimSpace(inner) != "" {
			for _, f := range splitTop(inner, ',') {
				f = strings.TrimSpace(f)
				k, vraw, ok := strings.Cut(f, "=")
				if !ok {
					return nil, &ParseError{lineNo, "E1206", "malformed outcome field " + f, raw}
				}
				var val Value
				var pe error
				if od := def.outcome(k); od != nil {
					val, pe = parseValue(od.Type, vraw)
				} else {
					val, pe = parseAny(vraw)
				}
				if pe != nil {
					return nil, &ParseError{lineNo, "E1307", fmt.Sprintf("outcome %s: %v", k, pe), raw}
				}
				ev.Outcome[k] = val
			}
		}
	}
	// cause
	if i < len(rest) && strings.HasPrefix(rest[i], "~") {
		n, err := strconv.Atoi(rest[i][1:])
		if err != nil {
			return nil, &ParseError{lineNo, "E1303", "malformed cause", raw}
		}
		ev.Cause = n
		i++
	}
	if i != len(rest) {
		return nil, &ParseError{lineNo, "E1304", "trailing tokens", raw}
	}
	// canonical operand order = dictionary order, then extras
	for _, od := range def.Operands {
		if _, ok := ev.Operands[od.Key]; ok {
			ev.operandOrder = append(ev.operandOrder, od.Key)
		}
	}
	for k := range ev.Operands {
		found := false
		for _, od := range def.Operands {
			if od.Key == k {
				found = true
				break
			}
		}
		if !found {
			ev.operandOrder = append(ev.operandOrder, k)
		}
	}
	ev.Opaque = def.Extension && !isExtensionNamespace(def.Namespace)
	return ev, nil
}

func parseIndex(s string) (int, string, error) {
	i := strings.IndexByte(s, '.')
	if i < 0 {
		return 0, "", fmt.Errorf("malformed index %q", s)
	}
	round, err := strconv.Atoi(s[:i])
	if err != nil {
		return 0, "", fmt.Errorf("malformed index %q", s)
	}
	phase := s[i+1:]
	if phase != "S" && phase != "B" && phase != "D" && phase != "E" {
		return 0, "", fmt.Errorf("malformed phase %q", phase)
	}
	return round, phase, nil
}

// parseAny parses a token whose declared type is unknown, best-effort.
func parseAny(tok string) (Value, error) {
	switch {
	case strings.HasPrefix(tok, "{"):
		return Value{Type: TScalar, Str: tok}, nil
	case strings.HasPrefix(tok, "["):
		return parseList(TScalar, tok, TScalar)
	case strings.ContainsAny(tok, "+()"):
		g, err := parseUnitGroup(tok)
		if err != nil {
			return Value{Type: TScalar, Str: tok}, nil
		}
		return Value{Type: TUnitGroup, Group: g}, nil
	case isNumber(tok):
		n, _ := strconv.Atoi(tok)
		return Value{Type: TInt, Int: n}, nil
	case tok == "true" || tok == "false":
		return Value{Type: TBool, Bool: tok == "true", Str: tok}, nil
	default:
		if len(tok) >= 2 && tok[0] == '"' {
			s, err := unquote(tok)
			if err != nil {
				return Value{}, err
			}
			return Value{Type: TScalar, Str: s}, nil
		}
		return Value{Type: TScalar, Str: tok}, nil
	}
}
