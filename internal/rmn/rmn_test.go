package rmn

import (
	"os"
	"strings"
	"testing"
)

func loadWorked(t *testing.T) *Log {
	t.Helper()
	b, err := os.ReadFile("../../testdata/worked.rmn")
	if err != nil {
		t.Fatal(err)
	}
	return ParseLog(string(b))
}

func TestParseWorked(t *testing.T) {
	log := loadWorked(t)
	if len(log.Errors) != 0 {
		t.Fatalf("parse errors: %v", log.Errors)
	}
	if len(log.Events) != 55 {
		t.Fatalf("events = %d, want 55", len(log.Events))
	}
	h := log.Header
	if h.Map != "autumn" || h.Deck != "standard" || h.First != "C" {
		t.Fatalf("header = %+v", h)
	}
	if len(h.Factions) != 4 {
		t.Fatalf("factions = %d", len(h.Factions))
	}
}

func TestParseEventKinds(t *testing.T) {
	cases := []struct {
		line   string
		intent string
		check  func(t *testing.T, e *Event)
	}{
		{"1 1.D C move group=2C.w from=C1 to=C11", "move", func(t *testing.T, e *Event) {
			g, _ := e.Group("group")
			if len(g) != 1 || g[0].Qty != 2 || g[0].Ref != "C.w" {
				t.Fatalf("group = %+v", g)
			}
			if from, _ := e.Str("from"); from != "C1" {
				t.Fatalf("from = %s", from)
			}
		}},
		{"2 1.D E battle attacker=E defender=C at=C11 -> {atk=2,def=1,extra_atk=0,extra_def=0}", "battle", func(t *testing.T, e *Event) {
			if v, _ := e.Outcome["atk"]; v.Int != 2 {
				t.Fatalf("atk = %+v", v)
			}
		}},
		{"3 0.S SYS assign-ruins -> {ruins=[C2,C3], items=[i.sword#1,i.hammer#1]}", "assign-ruins", func(t *testing.T, e *Event) {
			r, _ := e.OutcomeList("ruins")
			if len(r) != 2 || r[0].Str != "C2" {
				t.Fatalf("ruins = %+v", r)
			}
		}},
		{"4 0.S V place group=(i.boot#2+i.torch#1) to=BOARD:V:SATCHEL", "place", func(t *testing.T, e *Event) {
			g, _ := e.Group("group")
			if len(g) != 2 || g[0].Ref != "i.boot#2" {
				t.Fatalf("group = %+v", g)
			}
		}},
		{"5 1.D A A:place-sympathy at=C5 spend=M03", "A:place-sympathy", func(t *testing.T, e *Event) {
			if s, _ := e.Str("spend"); s != "M03" {
				t.Fatalf("spend = %s", s)
			}
		}},
		{"6 1.D V V:relationship target=C status=1", "V:relationship", func(t *testing.T, e *Event) {
			if s, _ := e.Str("status"); s != "1" {
				t.Fatalf("status = %s", s)
			}
		}},
		{"7 1.D C craft who=C card=F04 produce=i.coin#1", "craft", func(t *testing.T, e *Event) {
			if p, _ := e.Str("produce"); p != "i.coin#1" {
				t.Fatalf("produce = %s", p)
			}
		}},
		{"8 1.D C casualties at=C11 group=2C.w ~7", "casualties", func(t *testing.T, e *Event) {
			if e.Cause != 7 {
				t.Fatalf("cause = %d", e.Cause)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			e, perr := ParseEventLine(tc.line, 1)
			if perr != nil {
				t.Fatalf("parse: %v", perr)
			}
			if e.Intent != tc.intent {
				t.Fatalf("intent = %q", e.Intent)
			}
			tc.check(t, e)
		})
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		line string
		code string
	}{
		{"1 1.D C frobnicate x=1", "E1301"},
		{"1 1.D C move group=2C.w from=C1 extra", "E1302"},
		{"1 1.X C pass", "E1203"},
		{"1 1.D C move group=2C.w from=C1 to=C11 group=1C.w", "E1305"},
		{"x 1.D C pass", "E1203"},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			_, perr := ParseEventLine(tc.line, 1)
			if perr == nil {
				t.Fatalf("expected error %s", tc.code)
			}
			if perr.Code != tc.code {
				t.Fatalf("code = %s, want %s (%s)", perr.Code, tc.code, perr.Msg)
			}
		})
	}
}

func TestValidationMissingOperand(t *testing.T) {
	st := NewState(&Header{Map: "autumn", Deck: "standard", Factions: []FactionDecl{{ID: "C", Kind: "marquise", Seat: 1}}, First: "C"})
	ev, perr := ParseEventLine("1 1.D C move group=2C.w from=C1", 1)
	if perr != nil {
		t.Fatalf("parse: %v", perr)
	}
	errs := ValidateEvent(st, ev)
	if len(errs) == 0 {
		t.Fatal("expected missing-operand validation error")
	}
}

func TestReplayWorked(t *testing.T) {
	log := loadWorked(t)
	res := Replay(log)
	if len(res.Errors) != 0 {
		t.Fatalf("errors: %v", res.Errors)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("warnings: %v", res.Warnings)
	}
	st := res.State

	// Setup / board.
	c1 := st.Clearings["C1"]
	if c1 == nil || c1.Wood != 0 {
		t.Fatalf("C1 = %+v (wood should be spent)", c1)
	}
	if len(c1.Buildings) != 4 {
		t.Fatalf("C1 buildings = %+v", c1.Buildings)
	}
	if got := st.Clearings["C2"].Buildings; len(got) != 1 || got[0].Ref != "C.b.saw#2" {
		t.Fatalf("C2 buildings = %+v", got)
	}
	if n := st.Clearings["C12"].Warriors["E"]; n != 5 {
		t.Fatalf("C12 E = %d, want 5", n)
	}
	if n := st.Clearings["C11"].Warriors["C"]; n != 0 {
		t.Fatalf("C11 C = %d, want 0 (casualties)", n)
	}
	if n := st.Clearings["C11"].Warriors["E"]; n != 1 {
		t.Fatalf("C11 E = %d, want 1", n)
	}
	if tok := st.Clearings["C5"].Tokens; len(tok) != 1 || tok[0].Kind != "t.sym" {
		t.Fatalf("C5 tokens = %+v", tok)
	}
	// Ruins: C7 explored.
	if st.Clearings["C7"].Ruin != "" {
		t.Fatalf("C7 ruin should be cleared")
	}
	if st.Clearings["C2"].Ruin == "" {
		t.Fatalf("C2 ruin should remain")
	}
	// Factions.
	if st.Factions["A"].VP != 1 {
		t.Fatalf("A VP = %d, want 1", st.Factions["A"].VP)
	}
	if st.Factions["V"].VP != 1 {
		t.Fatalf("V VP = %d, want 1", st.Factions["V"].VP)
	}
	if st.Factions["E"].Leader != "E.ldr.despot" {
		t.Fatalf("E leader = %q", st.Factions["E"].Leader)
	}
	if st.Factions["V"].Character != "V.character.tinker" {
		t.Fatalf("V character = %q", st.Factions["V"].Character)
	}
	if got := st.Factions["E"].Decree["RECRUIT"]; len(got) != 1 || got[0] != "R03" {
		t.Fatalf("E decree RECRUIT = %v", got)
	}
	if st.Factions["V"].Relationship["C"] != "1" {
		t.Fatalf("V rel C = %q", st.Factions["V"].Relationship["C"])
	}
	// Cards.
	cHand := strings.Join(st.Factions["C"].Hand, ",")
	if !strings.Contains(cHand, "F06") || !strings.Contains(cHand, "B01") {
		t.Fatalf("C hand = %v", st.Factions["C"].Hand)
	}
	// Items.
	if it := st.Items["i.coin#1"]; it == nil || !strings.Contains(it.Zone, "TRACK") {
		t.Fatalf("i.coin#1 = %+v", it)
	}
	if it := st.Items["i.boot#1"]; it == nil || !strings.Contains(it.Zone, "SATCHEL") {
		t.Fatalf("i.boot#1 = %+v", it)
	}
	// Hash determinism.
	if res.Hash == "" || !strings.HasPrefix(res.Hash, "0x") {
		t.Fatalf("hash = %q", res.Hash)
	}
	res2 := Replay(loadWorked(t))
	if res2.Hash != res.Hash {
		t.Fatalf("hash not deterministic: %s != %s", res2.Hash, res.Hash)
	}
}

func TestCanonicalRoundTrip(t *testing.T) {
	log := loadWorked(t)
	text := log.CanonicalText()
	log2 := ParseLog(text)
	if len(log2.Errors) != 0 {
		t.Fatalf("reparse errors: %v", log2.Errors)
	}
	if len(log2.Events) != len(log.Events) {
		t.Fatalf("events %d != %d", len(log2.Events), len(log.Events))
	}
	for i := range log.Events {
		if log.Events[i].Canonical() != log2.Events[i].Canonical() {
			t.Fatalf("event %d round trip:\n%s\n%s", i, log.Events[i].Canonical(), log2.Events[i].Canonical())
		}
	}
}

func TestJSONRoundTrip(t *testing.T) {
	log := loadWorked(t)
	b, err := log.JSON()
	if err != nil {
		t.Fatal(err)
	}
	log2, err := DecodeJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(log2.Events) != len(log.Events) {
		t.Fatalf("events %d != %d", len(log2.Events), len(log.Events))
	}
}

func TestValueTypes(t *testing.T) {
	cases := []struct {
		typ  ValueType
		tok  string
		want string
	}{
		{TInt, "3", "3"},
		{TSuit, "F", "F"},
		{TLocation, "C12", "C12"},
		{TLocation, "F6_7_8", "F6_7_8"},
		{TLocation, "BOARD:V:SATCHEL", "BOARD:V:SATCHEL"},
		{TFactionList, "[C,E]", "[C,E]"},
		{TUnit, "C.b.saw#2", "C.b.saw#2"},
		{TUnit, "V.character.tinker", "V.character.tinker"},
		{TUnit, "i.boot#ruin", "i.boot#ruin"},
		{TUnitGroup, "(2C.w+1E.w)", "(2C.w+E.w)"},
		{TUnitGroup, "i.boot#2+i.torch#1", "(i.boot#2+i.torch#1)"},
		{TRelStatus, "0", "0"},
	}
	for _, tc := range cases {
		v, err := parseValue(tc.typ, tc.tok)
		if err != nil {
			t.Fatalf("%s %q: %v", tc.typ, tc.tok, err)
		}
		if got := v.String(); got != tc.want {
			t.Fatalf("%s %q => %q, want %q", tc.typ, tc.tok, got, tc.want)
		}
	}
}

func TestApplySingleEvent(t *testing.T) {
	log := loadWorked(t)
	st := Replay(log).State
	ev, perr := ParseEventLine("56 2.D C score who=C amount=2", 0)
	if perr != nil {
		t.Fatal(perr)
	}
	fr := Fold(st, ev)
	if len(fr.Errors) != 0 {
		t.Fatalf("errors: %v", fr.Errors)
	}
	if st.Factions["C"].VP != 2 {
		t.Fatalf("C VP = %d", st.Factions["C"].VP)
	}
}

func TestMarquiseActions(t *testing.T) {
	st := Replay(loadWorked(t)).State
	woodBefore := st.Clearings["C2"].Wood

	// Overwork -> produce wood at the chosen clearing.
	ev, perr := ParseEventLine("56 2.B C C:overwork at=C2 spend=F03 produce=C.t.wood#1", 0)
	if perr != nil {
		t.Fatal(perr)
	}
	if fr := Fold(st, ev); len(fr.Errors) != 0 {
		t.Fatalf("overwork errors: %v", fr.Errors)
	}
	if st.Clearings["C2"].Wood != woodBefore+1 {
		t.Fatalf("C2 wood = %d, want %d", st.Clearings["C2"].Wood, woodBefore+1)
	}
	if strings.Contains(strings.Join(st.Factions["C"].Hand, ","), "F03") {
		t.Fatalf("F03 should have been spent: %v", st.Factions["C"].Hand)
	}

	// Overwork -> score a point.
	ev2, _ := ParseEventLine("57 2.B C C:overwork at=C2 spend=F05 produce=C.vp", 0)
	if fr := Fold(st, ev2); len(fr.Errors) != 0 {
		t.Fatalf("overwork errors: %v", fr.Errors)
	}
	if st.Factions["C"].VP != 1 {
		t.Fatalf("C VP = %d, want 1", st.Factions["C"].VP)
	}

	// Birdsong places one wood per sawmill in the clearing.
	ev3, _ := ParseEventLine("58 3.B C C:birdsong-wood at=[C1]", 0)
	if fr := Fold(st, ev3); len(fr.Errors) != 0 {
		t.Fatalf("birdsong errors: %v", fr.Errors)
	}
	if st.Clearings["C1"].Wood != 1 {
		t.Fatalf("C1 wood = %d, want 1 (one sawmill)", st.Clearings["C1"].Wood)
	}
}

func TestMapAndDeck(t *testing.T) {
	if len(DeckCards("standard")) != 54 {
		t.Fatalf("deck = %d cards", len(DeckCards("standard")))
	}
	if _, ok := CardByID("F01"); !ok {
		t.Fatal("F01 missing from manifest")
	}
	if len(ItemCatalogue()) != 13 {
		t.Fatalf("items = %d", len(ItemCatalogue()))
	}
	md := MapByName("autumn")
	if md == nil || md.Suits["C1"] != "R" {
		t.Fatalf("autumn map = %+v", md)
	}
	if len(md.Adj["C1"]) != 2 {
		t.Fatalf("C1 adjacency = %v", md.Adj["C1"])
	}
}
