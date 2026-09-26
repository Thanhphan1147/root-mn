package root

import (
	"fmt"
	"testing"
)

// TestBuildRMNLine checks that a shorthand line gains the sequence, round.phase
// and acting player inferred from the current state, and that a full line is
// passed through untouched.
func TestBuildRMNLine(t *testing.T) {
	g := NewGame([]Faction{MC, ED}, MC, 7)
	BeginSetup(g)

	got, err := g.BuildRMNLine("place group=MC.b.keep#1 to=C1")
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("%d 0.S MC place group=MC.b.keep#1 to=C1", len(g.RMNLog)+1)
	if got != want {
		t.Fatalf("shorthand built %q, want %q", got, want)
	}

	full := "99 12.D ED pass"
	if got, err := g.BuildRMNLine(full); err != nil || got != full {
		t.Fatalf("full line %q -> %q, %v; want unchanged", full, got, err)
	}

	if _, err := g.BuildRMNLine("   "); err == nil {
		t.Fatal("an empty line should error")
	}
}

// TestTryRMNShorthand applies a legal action typed as just its intent and
// operands, and checks the recorded line carries the inferred prefix.
func TestTryRMNShorthand(t *testing.T) {
	g := NewGame([]Faction{MC, ED}, MC, 7)
	BeginSetup(g)

	base := len(g.RMNLog)
	if err := g.TryRMN("place group=MC.b.keep#1 to=C1"); err != nil {
		t.Fatalf("shorthand apply failed: %v", err)
	}
	if len(g.RMNLog) != base+1 {
		t.Fatalf("expected one new RMN line, got %d", len(g.RMNLog)-base)
	}
	want := fmt.Sprintf("%d 0.S MC place group=MC.b.keep#1 to=C1", base+1)
	if g.RMNLog[base] != want {
		t.Fatalf("recorded %q, want %q", g.RMNLog[base], want)
	}
}

// TestTryRMNErrors checks the UI-friendly errors for malformed and illegal lines.
func TestTryRMNErrors(t *testing.T) {
	g := NewGame([]Faction{MC, ED}, MC, 7)
	BeginSetup(g)

	if err := g.TryRMN(""); err == nil || err.Error() != "empty RMN" {
		t.Fatalf("empty -> %v, want \"empty RMN\"", err)
	}
	if err := g.TryRMN("this is not rmn"); err == nil {
		t.Fatal("garbage should not be a legal move")
	}
}
