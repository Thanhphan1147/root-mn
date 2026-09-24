package root

import "testing"

// TestVagabondAidRelationshipCost checks advancing the relationship track costs
// one aid for the first step, two for the second, three for the third (9.2.9.I).
func TestVagabondAidRelationshipCost(t *testing.T) {
	g := NewGame([]Faction{MC, ED, WA, VB}, MC, 1)
	p := g.Players[VB]
	p.Relationships[MC] = "indifferent"
	p.AidCount = map[Faction]int{}
	vp0 := p.VP

	g.vbAidRelationship(MC) // 1st -> amiable (+1)
	if p.Relationships[MC] != "amiable" || p.VP != vp0+1 {
		t.Fatalf("after 1 aid: rel=%s vp=%d", p.Relationships[MC], p.VP)
	}
	g.vbAidRelationship(MC) // 2nd -> 1/2 toward friendly, no advance
	if p.Relationships[MC] != "amiable" || p.VP != vp0+1 {
		t.Fatalf("after 2 aids: rel=%s vp=%d (should not have advanced)", p.Relationships[MC], p.VP)
	}
	g.vbAidRelationship(MC) // 3rd -> friendly (+2)
	if p.Relationships[MC] != "friendly" || p.VP != vp0+3 {
		t.Fatalf("after 3 aids: rel=%s vp=%d", p.Relationships[MC], p.VP)
	}
	for i := 0; i < 2; i++ {
		g.vbAidRelationship(MC) // 4th,5th -> 2/3 toward allied
	}
	if p.Relationships[MC] != "friendly" {
		t.Fatalf("after 5 aids rel=%s, should still be friendly", p.Relationships[MC])
	}
	g.vbAidRelationship(MC) // 6th -> allied (+2)
	if p.Relationships[MC] != "allied" || p.VP != vp0+5 {
		t.Fatalf("after 6 aids: rel=%s vp=%d want allied/%d", p.Relationships[MC], p.VP, vp0+5)
	}
	g.vbAidRelationship(MC) // aiding an ally -> +2 each
	if p.VP != vp0+7 {
		t.Fatalf("aiding an ally: vp=%d want %d", p.VP, vp0+7)
	}
}
