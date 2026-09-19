package rmn

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// envelope is the canonical JSON form (SPEC §5.1).
type envelope struct {
	RMN         string        `json:"rmn"`
	Header      *Header       `json:"header"`
	Events      []*Event      `json:"events"`
	Checkpoints []*Checkpoint `json:"checkpoints,omitempty"`
}

// JSON renders the log as its canonical JSON envelope.
func (l *Log) JSON() ([]byte, error) {
	return json.MarshalIndent(envelope{RMN: "3.0", Header: l.Header, Events: l.Events, Checkpoints: l.Checkpoints}, "", "  ")
}

// CanonicalText renders the header and events as canonical text.
func (l *Log) CanonicalText() string {
	var b strings.Builder
	b.WriteString("%RMN 3.0\n")
	h := l.Header
	if h == nil {
		return b.String()
	}
	if h.Game != "" {
		fmt.Fprintf(&b, "%%Game %s\n", h.Game)
	}
	if h.Map != "" {
		fmt.Fprintf(&b, "%%Map %s\n", h.Map)
	}
	if len(h.Clearings) > 0 {
		b.WriteString("%Clearings")
		for _, c := range h.Clearings {
			fmt.Fprintf(&b, " %s:%s", c.Clearing, c.Suit)
		}
		b.WriteString("\n")
	}
	if h.Deck != "" {
		fmt.Fprintf(&b, "%%Deck %s\n", h.Deck)
	}
	for _, f := range h.Factions {
		fmt.Fprintf(&b, "%%Faction %s %s", f.ID, f.Kind)
		if f.Seat != 0 {
			fmt.Fprintf(&b, " seat=%d", f.Seat)
		}
		if f.Name != "" {
			fmt.Fprintf(&b, " name=%q", f.Name)
		}
		b.WriteString("\n")
	}
	if h.First != "" {
		fmt.Fprintf(&b, "%%First %s\n", h.First)
	}
	if h.Seed != nil {
		fmt.Fprintf(&b, "%%Seed algo=%s value=%s\n", h.Seed.Algo, h.Seed.Value)
	}
	// options sorted for determinism
	if len(h.Options) > 0 {
		keys := make([]string, 0, len(h.Options))
		for k := range h.Options {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "%%Option %s=%s\n", k, h.Options[k])
		}
	}
	for _, ev := range l.Events {
		b.WriteString(ev.Canonical())
		b.WriteByte('\n')
	}
	for _, cp := range l.Checkpoints {
		fmt.Fprintf(&b, "%%Checkpoint seq=%d hash=%s", cp.Seq, cp.Hash)
		if cp.Algo != "" {
			fmt.Fprintf(&b, " algo=%s", cp.Algo)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// DecodeJSON parses the JSON envelope back into a Log.
func DecodeJSON(data []byte) (*Log, error) {
	var e envelope
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	l := &Log{Header: e.Header, Checkpoints: e.Checkpoints}
	if l.Header == nil {
		l.Header = &Header{Options: map[string]string{}}
	}
	for _, ev := range e.Events {
		l.Events = append(l.Events, ev)
	}
	return l, nil
}
