package rmn

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ValueType is a registry type name from SPEC §4.4.
type ValueType string

const (
	TInt          ValueType = "int"
	THex          ValueType = "hex"
	TBool         ValueType = "bool"
	TScalar       ValueType = "scalar"
	TSuit         ValueType = "suit"
	TEnum         ValueType = "enum"
	TRelStatus    ValueType = "rel-status"
	TText         ValueType = "text"
	TFaction      ValueType = "faction"
	TFactionCode  ValueType = "faction-code"
	THireling     ValueType = "hireling"
	TOwner        ValueType = "owner"
	TLocation     ValueType = "location"
	TLocationList ValueType = "location-list"
	TUnit         ValueType = "unit"
	TOptionalUnit ValueType = "optional-unit"
	TUnitGroup    ValueType = "unit-group"
	TIntList      ValueType = "int-list"
	TItemRef      ValueType = "item-ref"
	TItemList     ValueType = "item-list"
	TCard         ValueType = "card"
	TCardID       ValueType = "card-id"
	TCardIDList   ValueType = "card-id-list"
	TQuestID      ValueType = "quest-id"
	TRelic        ValueType = "relic"
	TPhase        ValueType = "phase"
	TFactionList  ValueType = "faction-list"
	TCodeList     ValueType = "code-list"
	TCommaList    ValueType = "comma-list"
)

// UnitAtom is one [qty]unit element of a UnitGroup.
type UnitAtom struct {
	Qty int    `json:"qty"`
	Ref string `json:"ref"`
}

// Value is a parsed operand or outcome value. Only the fields relevant to Type
// are populated.
type Value struct {
	Type  ValueType  `json:"-"`
	Int   int        `json:"-"`
	Bool  bool       `json:"-"`
	Str   string     `json:"-"`
	List  []Value    `json:"-"`
	Group []UnitAtom `json:"-"`
}

var (
	reCardID      = regexp.MustCompile(`^[FRMB][0-9]{2}$`)
	reQuestID     = regexp.MustCompile(`^q\.[FRMB][0-9]{2}$`)
	reItemRef     = regexp.MustCompile(`^i\.[a-z]+#([0-9]+|ruin)$`)
	reRelicRef    = regexp.MustCompile(`^r\.(figure|tablet|jewelry)(#[0-9]+)?(@[0-9]+)?$`)
	reHex         = regexp.MustCompile(`^0x[0-9a-fA-F]+$`)
	reFactionCode = regexp.MustCompile(`^[CEAVLODPHK]$`)
	reCardPat     = regexp.MustCompile(`^[FRMB]?#([a-z]+)?$`)
)

func (v Value) String() string {
	switch v.Type {
	case TInt:
		return strconv.Itoa(v.Int)
	case TBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case TLocationList, TIntList, TItemList, TCardIDList, TFactionList:
		parts := make([]string, len(v.List))
		for i, e := range v.List {
			parts[i] = e.raw()
		}
		return "[" + strings.Join(parts, ",") + "]"
	case TUnitGroup:
		return formatGroup(v.Group)
	default:
		return v.Str
	}
}

func (v Value) raw() string {
	if v.Type == TInt {
		return strconv.Itoa(v.Int)
	}
	return v.Str
}

// UnmarshalJSON decodes a JSON operand/outcome value. The declared registry
// type is not recoverable from JSON alone, so the type is inferred.
func (v *Value) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	switch {
	case s == "null" || s == "":
		return nil
	case s[0] == '"':
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		v.Type, v.Str = TScalar, str
	case s == "true" || s == "false":
		v.Type, v.Bool, v.Str = TBool, s == "true", s
	case s[0] == '[':
		var raw []json.RawMessage
		if err := json.Unmarshal(b, &raw); err != nil {
			return err
		}
		if len(raw) > 0 && strings.HasPrefix(strings.TrimSpace(string(raw[0])), "{") {
			v.Type = TUnitGroup
			for _, r := range raw {
				var a UnitAtom
				if err := json.Unmarshal(r, &a); err != nil {
					return err
				}
				v.Group = append(v.Group, a)
			}
			return nil
		}
		v.Type = TLocationList
		for _, r := range raw {
			var ev Value
			if err := ev.UnmarshalJSON(r); err != nil {
				return err
			}
			v.List = append(v.List, ev)
		}
	default:
		var n int
		if err := json.Unmarshal(b, &n); err == nil {
			v.Type, v.Int = TInt, n
		} else {
			v.Type, v.Str = TScalar, s
		}
	}
	return nil
}

func formatGroup(g []UnitAtom) string {
	if len(g) == 1 {
		return formatAtom(g[0])
	}
	parts := make([]string, len(g))
	for i, a := range g {
		parts[i] = formatAtom(a)
	}
	return "(" + strings.Join(parts, "+") + ")"
}

func formatAtom(a UnitAtom) string {
	if a.Qty > 1 {
		return strconv.Itoa(a.Qty) + a.Ref
	}
	return a.Ref
}

// parseValue parses a single token as the declared registry type.
func parseValue(t ValueType, tok string) (Value, error) {
	switch t {
	case TInt:
		n, err := strconv.Atoi(tok)
		if err != nil || n < 0 {
			return Value{}, fmt.Errorf("expected int, got %q", tok)
		}
		return Value{Type: TInt, Int: n}, nil
	case THex:
		if !reHex.MatchString(tok) {
			return Value{}, fmt.Errorf("expected hex, got %q", tok)
		}
		return Value{Type: THex, Str: tok}, nil
	case TBool:
		if tok != "true" && tok != "false" {
			return Value{}, fmt.Errorf("expected bool, got %q", tok)
		}
		return Value{Type: TBool, Bool: tok == "true", Str: tok}, nil
	case TScalar:
		if isIdent(tok) || reHex.MatchString(tok) || isNumber(tok) {
			return Value{Type: TScalar, Str: tok}, nil
		}
		if len(tok) >= 2 && tok[0] == '"' {
			s, err := unquote(tok)
			if err != nil {
				return Value{}, err
			}
			return Value{Type: TScalar, Str: s}, nil
		}
		return Value{}, fmt.Errorf("expected scalar, got %q", tok)
	case TSuit:
		if !isSuit(tok) {
			return Value{}, fmt.Errorf("expected suit, got %q", tok)
		}
		return Value{Type: TSuit, Str: tok}, nil
	case TEnum, TPhase:
		if !isIdent(tok) && !(t == TPhase && isSuit(tok)) {
			return Value{}, fmt.Errorf("expected enum, got %q", tok)
		}
		return Value{Type: t, Str: tok}, nil
	case TRelStatus:
		if tok != "h" && tok != "0" && tok != "1" && tok != "2" && tok != "a" {
			return Value{}, fmt.Errorf("expected rel-status, got %q", tok)
		}
		return Value{Type: TRelStatus, Str: tok}, nil
	case TText:
		s, err := unquote(tok)
		if err != nil {
			return Value{}, err
		}
		return Value{Type: TText, Str: s}, nil
	case TFaction, TOwner:
		if !isIdent(tok) || tok == "SYS" {
			return Value{}, fmt.Errorf("expected faction, got %q", tok)
		}
		return Value{Type: t, Str: tok}, nil
	case TFactionCode:
		if !reFactionCode.MatchString(tok) {
			return Value{}, fmt.Errorf("expected faction-code, got %q", tok)
		}
		return Value{Type: TFactionCode, Str: tok}, nil
	case THireling:
		if !strings.HasPrefix(tok, "h.") || !isIdent(tok[2:]) {
			return Value{}, fmt.Errorf("expected hireling, got %q", tok)
		}
		return Value{Type: THireling, Str: tok}, nil
	case TLocation:
		if !isLocation(tok) {
			return Value{}, fmt.Errorf("expected location, got %q", tok)
		}
		return Value{Type: TLocation, Str: tok}, nil
	case TLocationList:
		if strings.HasPrefix(tok, "[") {
			return parseList(t, tok, TLocation)
		}
		if !isLocation(tok) {
			return Value{}, fmt.Errorf("expected location, got %q", tok)
		}
		return Value{Type: TLocationList, List: []Value{{Type: TLocation, Str: tok}}}, nil
	case TUnit:
		if !isUnit(tok) {
			return Value{}, fmt.Errorf("expected unit, got %q", tok)
		}
		return Value{Type: TUnit, Str: tok}, nil
	case TOptionalUnit:
		if tok == "none" {
			return Value{Type: TOptionalUnit, Str: "none"}, nil
		}
		if !isUnit(tok) {
			return Value{}, fmt.Errorf("expected unit or none, got %q", tok)
		}
		return Value{Type: TOptionalUnit, Str: tok}, nil
	case TUnitGroup:
		g, err := parseUnitGroup(tok)
		if err != nil {
			return Value{}, err
		}
		return Value{Type: TUnitGroup, Group: g}, nil
	case TIntList:
		return parseList(t, tok, TInt)
	case TItemRef:
		if !reItemRef.MatchString(tok) {
			return Value{}, fmt.Errorf("expected item-ref, got %q", tok)
		}
		return Value{Type: TItemRef, Str: tok}, nil
	case TItemList:
		return parseList(t, tok, TItemRef)
	case TCard:
		if !isCardRef(tok) {
			return Value{}, fmt.Errorf("expected card, got %q", tok)
		}
		return Value{Type: TCard, Str: tok}, nil
	case TCardID:
		if !reCardID.MatchString(tok) {
			return Value{}, fmt.Errorf("expected card-id, got %q", tok)
		}
		return Value{Type: TCardID, Str: tok}, nil
	case TCardIDList:
		return parseList(t, tok, TCardID)
	case TQuestID:
		if !reQuestID.MatchString(tok) {
			return Value{}, fmt.Errorf("expected quest-id, got %q", tok)
		}
		return Value{Type: TQuestID, Str: tok}, nil
	case TRelic:
		if !reRelicRef.MatchString(tok) {
			return Value{}, fmt.Errorf("expected relic, got %q", tok)
		}
		return Value{Type: TRelic, Str: tok}, nil
	case TFactionList:
		return parseList(t, tok, TFaction)
	case TCodeList:
		parts := splitTop(tok, ',')
		out := make([]Value, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if !reFactionCode.MatchString(p) {
				return Value{}, fmt.Errorf("expected faction-code, got %q", p)
			}
			out = append(out, Value{Type: TFactionCode, Str: p})
		}
		return Value{Type: TCodeList, List: out}, nil
	case TCommaList:
		parts := splitTop(tok, ',')
		out := make([]Value, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if !isIdent(p) {
				return Value{}, fmt.Errorf("expected identifier, got %q", p)
			}
			out = append(out, Value{Type: TScalar, Str: p})
		}
		return Value{Type: TCommaList, List: out}, nil
	}
	return Value{}, fmt.Errorf("unsupported value type %q", t)
}

func parseList(t ValueType, tok string, elem ValueType) (Value, error) {
	inner, ok := stripOuter(tok, '[', ']')
	if !ok {
		return Value{}, fmt.Errorf("expected list, got %q", tok)
	}
	inner = strings.TrimSpace(inner)
	if strings.HasPrefix(inner, "[") {
		return Value{}, fmt.Errorf("nested list")
	}
	v := Value{Type: t}
	if inner == "" {
		return v, nil
	}
	for _, p := range splitTop(inner, ',') {
		p = strings.TrimSpace(p)
		ev, err := parseValue(elem, p)
		if err != nil {
			return Value{}, err
		}
		v.List = append(v.List, ev)
	}
	return v, nil
}

func parseUnitGroup(tok string) ([]UnitAtom, error) {
	if inner, ok := stripOuter(tok, '(', ')'); ok {
		tok = inner
	}
	if strings.ContainsAny(tok, "()") {
		return nil, fmt.Errorf("nested unit group")
	}
	var out []UnitAtom
	for _, part := range splitTop(tok, '+') {
		if part == "" {
			return nil, fmt.Errorf("empty unit atom")
		}
		qty := 1
		ref := part
		i := 0
		for i < len(part) && isDigit(part[i]) {
			i++
		}
		if i > 0 {
			n, _ := strconv.Atoi(part[:i])
			qty = n
			ref = part[i:]
		}
		if !isUnit(ref) {
			return nil, fmt.Errorf("expected unit, got %q", ref)
		}
		out = append(out, UnitAtom{Qty: qty, Ref: ref})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty unit group")
	}
	return out, nil
}

func isSuit(s string) bool {
	return s == "F" || s == "R" || s == "M" || s == "B"
}

func isNumber(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return true
}

func isCardRef(s string) bool {
	return reCardID.MatchString(s) || reCardPat.MatchString(s) || s == "?"
}

// isUnit reports whether s is a well-formed unit reference.
func isUnit(s string) bool {
	if reCardID.MatchString(s) || reQuestID.MatchString(s) ||
		reItemRef.MatchString(s) || reRelicRef.MatchString(s) ||
		reCardPat.MatchString(s) || s == "?" {
		return true
	}
	return isPieceRef(s)
}

// isPieceRef reports whether s is Owner "." PieceKind.
func isPieceRef(s string) bool {
	owner, kind, ok := splitOwner(s)
	if !ok {
		return false
	}
	if !isOwner(owner) {
		return false
	}
	return isPieceKind(kind)
}

func isOwner(s string) bool {
	if strings.HasPrefix(s, "h.") {
		return isIdent(s[2:])
	}
	return isUpperIdent(s) && s != "SYS"
}

// splitOwner splits a piece ref into its owner and kind. Hireling owners are
// "h.<name>"; otherwise the owner is the first dot-separated segment.
func splitOwner(s string) (owner, kind string, ok bool) {
	if strings.HasPrefix(s, "h.") {
		rest := s[2:]
		i := strings.IndexByte(rest, '.')
		if i < 0 {
			return "", "", false
		}
		return "h." + rest[:i], rest[i+1:], true
	}
	i := strings.IndexByte(s, '.')
	if i < 0 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

func isPieceKind(k string) bool {
	switch k {
	case "w", "p", "vp", "mob", "warlord":
		return true
	}
	if strings.HasPrefix(k, "b.") || strings.HasPrefix(k, "t.") ||
		strings.HasPrefix(k, "m.") || strings.HasPrefix(k, "ldr.") ||
		strings.HasPrefix(k, "character.") {
		return true
	}
	if k == "plot" || strings.HasPrefix(k, "plot#") {
		return true
	}
	return false
}

var reLocation = regexp.MustCompile(
	`^(C[0-9]+|F[0-9]+(_[0-9]+)+|P[0-9]+_[0-9]+|BURROW(:[A-Za-z0-9_-]+)?|FERRY:[0-9]+_[0-9]+|` +
		`HAND:[A-Za-z0-9_-]+|DECK|DISCARD|SUPPLY|ISUPPLY|RUIN:C[0-9]+|QUEST:(AVAIL|DECK)|` +
		`BOARD:[A-Za-z0-9_.-]+:[A-Za-z0-9_:-]+|HOARD:[A-Za-z0-9_-]+|RETINUE:[A-Za-z0-9_-]+:[0-9]+|` +
		`HIRELING:h\.[A-Za-z0-9_-]+|EXILE)$`)

func isLocation(s string) bool { return reLocation.MatchString(s) }

func unquote(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("unterminated string")
	}
	var b strings.Builder
	for i := 1; i < len(s)-1; i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s)-1 {
			i++
			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte(s[i])
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String(), nil
}
