package root

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// stateDigest returns a short deterministic digest of the game state. It hashes
// only position-relevant fields (not the RNG counter, logs or transient replay
// scratch), so the same position hashes the same regardless of how it was
// reached or replayed. The hidden piles (Deck/Discard) are hashed as
// order-insensitive multisets, so a reconstructed order need not match.
func stateDigest(g *Game) string {
	c := *g
	c.RngSeed = 0
	c.Seq = 0
	c.NextRoll = nil
	c.NextShuffle = nil
	c.DrawnThisAction = nil
	c.VBStolenThisAction = ""
	c.SDStolenThisAction = ""
	c.NextSteal = ""
	c.Log = nil
	c.RMNLog = nil
	c.Deck = sortedCopy(g.Deck)
	c.Discard = sortedCopy(g.Discard)
	b, err := json.Marshal(&c)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return "0x" + hex.EncodeToString(sum[:8])
}

func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

// Snapshot builds the client-facing JSON view of a game.
func Snapshot(g *Game) map[string]any {
	return map[string]any{
		"round":      g.Round,
		"phase":      g.Phase,
		"current":    g.Current,
		"winner":     g.Winner,
		"order":      g.Order,
		"clearings":  g.Clearings,
		"players":    g.Players,
		"legal":      g.LegalActions(),
		"log":        g.Log,
		"pending":    g.Pending,
		"battle":     g.Battle,
		"relaxed":    g.Relaxed,
		"setupMode":  g.SetupMode,
		"setupStage": g.SetupStage,
		"dayStage":   g.DayStage(),
		"cards":      CardInfoMap(),
		"quests":     QuestInfoMap(),
		"questAvail": g.QuestAvail,
		"rmn":        g.RMNLog,
		"hash":       stateDigest(g),
	}
}
