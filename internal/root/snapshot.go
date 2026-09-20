package root

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// stateDigest returns a short deterministic digest of the game state.
func stateDigest(g *Game) string {
	b, err := json.Marshal(g)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return "0x" + hex.EncodeToString(sum[:8])
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
		"hash":       stateDigest(g),
	}
}
