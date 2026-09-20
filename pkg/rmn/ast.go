package rmn

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// ParseError is a parse-layer diagnostic.
type ParseError struct {
	Line int    `json:"line"`
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Raw  string `json:"raw,omitempty"`
}

func (e ParseError) Error() string {
	return fmt.Sprintf("line %d [%s]: %s", e.Line, e.Code, e.Msg)
}

// Header holds the static game metadata (SPEC §4.2, §5.2).
type Header struct {
	Game      string            `json:"game,omitempty"`
	Map       string            `json:"map,omitempty"`
	Clearings []ClearingSuit    `json:"clearings,omitempty"`
	Deck      string            `json:"deck,omitempty"`
	Factions  []FactionDecl     `json:"factions,omitempty"`
	Draft     *Draft            `json:"draft,omitempty"`
	First     string            `json:"first,omitempty"`
	Winner    []string          `json:"winner,omitempty"`
	Seed      *Seed             `json:"seed,omitempty"`
	Landmarks []Landmark        `json:"landmarks,omitempty"`
	Hirelings []Hireling        `json:"hirelings,omitempty"`
	Options   map[string]string `json:"options,omitempty"`
	Partial   bool              `json:"-"`
}

// ClearingSuit assigns a suit to a clearing.
type ClearingSuit struct {
	Clearing string `json:"clearing"`
	Suit     string `json:"suit"`
}

// FactionDecl declares a faction instance.
type FactionDecl struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Seat int    `json:"seat,omitempty"`
	Name string `json:"name,omitempty"`
}

// Draft records the ADSET pool and order.
type Draft struct {
	Pool   []string `json:"pool,omitempty"`
	Order  []string `json:"order,omitempty"`
	Method string   `json:"method,omitempty"`
}

// Seed records an optional RNG seed.
type Seed struct {
	Algo  string `json:"algo"`
	Value string `json:"value"`
}

// Landmark records a placed landmark (extension).
type Landmark struct {
	ID string `json:"id"`
	At string `json:"at"`
	By string `json:"by,omitempty"`
}

// Hireling records a hired hireling (extension).
type Hireling struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Controller string `json:"controller,omitempty"`
	At         string `json:"at,omitempty"`
}

// Event is a single RMN v3 event (SPEC §9).
type Event struct {
	Seq          int              `json:"seq"`
	Round        int              `json:"round"`
	Phase        string           `json:"phase"`
	Actor        string           `json:"actor"`
	Intent       string           `json:"intent"`
	Operands     map[string]Value `json:"operands"`
	Outcome      map[string]Value `json:"outcome,omitempty"`
	Cause        int              `json:"cause,omitempty"`
	Note         string           `json:"note,omitempty"`
	Opaque       bool             `json:"opaque,omitempty"`
	Line         int              `json:"-"`
	Raw          string           `json:"-"`
	operandOrder []string
}

// Checkpoint records a state hash at a sequence position.
type Checkpoint struct {
	Seq  int    `json:"seq"`
	Hash string `json:"state_hash"`
	Algo string `json:"algo,omitempty"`
}

// Log is a parsed RMN v3 document.
type Log struct {
	Header      *Header
	Events      []*Event
	Checkpoints []*Checkpoint
	Errors      []ParseError
}

// Index renders the event index, e.g. "1.D".
func (e *Event) Index() string { return strconv.Itoa(e.Round) + "." + e.Phase }

// Str returns an operand's string value.
func (e *Event) Str(key string) (string, bool) {
	v, ok := e.Operands[key]
	if !ok {
		return "", false
	}
	return v.Str, true
}

// Int returns an operand's integer value.
func (e *Event) Int(key string) (int, bool) {
	v, ok := e.Operands[key]
	if !ok {
		return 0, false
	}
	return v.Int, true
}

// Group returns an operand's unit group.
func (e *Event) Group(key string) ([]UnitAtom, bool) {
	v, ok := e.Operands[key]
	if !ok {
		return nil, false
	}
	return v.Group, true
}

// OutcomeStr returns an outcome field's string value.
func (e *Event) OutcomeStr(key string) (string, bool) {
	v, ok := e.Outcome[key]
	if !ok {
		return "", false
	}
	return v.Str, true
}

// OutcomeList returns an outcome field's list.
func (e *Event) OutcomeList(key string) ([]Value, bool) {
	v, ok := e.Outcome[key]
	if !ok {
		return nil, false
	}
	return v.List, true
}

// operandKeys returns operand keys in insertion (dictionary) order.
func (e *Event) operandKeys() []string { return e.operandOrder }

// Canonical renders the event in canonical text form.
func (e *Event) Canonical() string {
	var b []byte
	b = strconv.AppendInt(b, int64(e.Seq), 10)
	b = append(b, ' ')
	b = append(b, e.Index()...)
	b = append(b, ' ')
	b = append(b, e.Actor...)
	b = append(b, ' ')
	b = append(b, e.Intent...)
	for _, k := range e.operandOrder {
		b = append(b, ' ')
		b = append(b, k...)
		b = append(b, '=')
		b = append(b, e.Operands[k].String()...)
	}
	if len(e.Outcome) > 0 {
		b = append(b, " -> {"...)
		keys := make([]string, 0, len(e.Outcome))
		for k := range e.Outcome {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				b = append(b, ',')
			}
			b = append(b, k...)
			b = append(b, '=')
			b = append(b, e.Outcome[k].String()...)
		}
		b = append(b, '}')
	}
	if e.Cause > 0 {
		b = append(b, " ~"...)
		b = strconv.AppendInt(b, int64(e.Cause), 10)
	}
	if e.Note != "" {
		b = append(b, " // "...)
		b = append(b, e.Note...)
	}
	return string(b)
}

// MarshalJSON renders the event per SPEC §5.3.
func (e *Event) MarshalJSON() ([]byte, error) {
	type ev struct {
		Seq      int            `json:"seq"`
		Round    int            `json:"round"`
		Phase    string         `json:"phase"`
		Actor    string         `json:"actor"`
		Intent   string         `json:"intent"`
		Operands map[string]any `json:"operands"`
		Outcome  map[string]any `json:"outcome,omitempty"`
		Cause    int            `json:"cause,omitempty"`
		Note     string         `json:"note,omitempty"`
		Opaque   bool           `json:"opaque,omitempty"`
	}
	out := ev{Seq: e.Seq, Round: e.Round, Phase: e.Phase, Actor: e.Actor, Intent: e.Intent, Cause: e.Cause, Note: e.Note, Opaque: e.Opaque}
	out.Operands = map[string]any{}
	for k, v := range e.Operands {
		out.Operands[k] = jsonValue(v)
	}
	if len(e.Outcome) > 0 {
		out.Outcome = map[string]any{}
		for k, v := range e.Outcome {
			out.Outcome[k] = jsonValue(v)
		}
	}
	return json.Marshal(out)
}

// jsonValue converts a Value to its JSON form (SPEC §4.4).
func jsonValue(v Value) any {
	switch v.Type {
	case TInt:
		return v.Int
	case TBool:
		return v.Bool
	case TUnitGroup:
		type atom struct {
			Qty int    `json:"qty"`
			Ref string `json:"ref"`
		}
		out := make([]atom, len(v.Group))
		for i, a := range v.Group {
			out[i] = atom{Qty: a.Qty, Ref: a.Ref}
		}
		return out
	case TLocationList, TIntList, TItemList, TCardIDList, TFactionList, TCodeList, TCommaList:
		out := make([]any, len(v.List))
		for i, e := range v.List {
			out[i] = jsonValue(e)
		}
		return out
	default:
		return v.Str
	}
}
