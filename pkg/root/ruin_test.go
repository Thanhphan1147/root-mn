package root

import (
	"strings"
	"testing"
)

// TestRuinNotationAndRedaction checks the ruin contents are recorded in the log
// but hidden from every player's view.
func TestRuinNotationAndRedaction(t *testing.T) {
	g := NewGame([]Faction{MC, ED}, MC, 7)
	BeginSetup(g)
	found := false
	for _, l := range g.RMNLog {
		if strings.Contains(l, "assign-ruins") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("RMN log should record assign-ruins, got %v", g.RMNLog)
	}
	for _, viewer := range []string{"", "MC", "ED"} {
		snap := Redact(g, viewer)
		for _, l := range snap["rmn"].([]string) {
			if strings.Contains(l, "assign-ruins") && !strings.Contains(l, "items=[?]") {
				t.Fatalf("viewer %q leaked ruin items: %s", viewer, l)
			}
		}
		for _, c := range snap["clearings"].(map[string]*Clearing) {
			if c.RuinItem != "" {
				t.Fatalf("viewer %q leaked ruin item %q", viewer, c.RuinItem)
			}
		}
	}
}
