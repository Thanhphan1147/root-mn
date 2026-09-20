//go:build js && wasm

// Command rootwasm exposes the ROOT engine to the browser as WebAssembly. The
// browser owns the game state (persisted in localStorage), so the analysis tool
// needs no server and can be hosted as a static site (e.g. GitHub Pages).
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/Thanhphan1147/root-mn/pkg/root"
)

var current *root.Game

func snapshotJSON() string {
	if current == nil {
		return "{}"
	}
	b, err := json.Marshal(root.Snapshot(current))
	if err != nil {
		return "{}"
	}
	return string(b)
}

func withError(msg string) string {
	b, _ := json.Marshal(map[string]any{"error": msg})
	return string(b)
}

func main() {
	api := map[string]any{
		"newGame": js.FuncOf(func(this js.Value, args []js.Value) any {
			seed := uint64(0)
			if len(args) > 0 {
				seed = uint64(args[0].Int())
			}
			current = root.NewGame([]root.Faction{root.MC, root.ED, root.WA, root.VB}, root.MC, seed)
			root.BeginSetup(current)
			return snapshotJSON()
		}),
		"autoSetup": js.FuncOf(func(this js.Value, args []js.Value) any {
			seed := uint64(0)
			if len(args) > 0 {
				seed = uint64(args[0].Int())
			}
			current = root.NewGame([]root.Faction{root.MC, root.ED, root.WA, root.VB}, root.MC, seed)
			root.Setup(current, root.SetupOptions{Seed: seed})
			return snapshotJSON()
		}),
		"apply": js.FuncOf(func(this js.Value, args []js.Value) any {
			if current == nil || len(args) < 1 {
				return withError("no game")
			}
			id := args[0].String()
			err := current.Apply(root.Action{ID: id})
			out := map[string]any{}
			_ = json.Unmarshal([]byte(snapshotJSON()), &out)
			if err != nil {
				out["error"] = err.Error()
			}
			b, _ := json.Marshal(out)
			return string(b)
		}),
		"save": js.FuncOf(func(this js.Value, args []js.Value) any {
			if current == nil {
				return ""
			}
			b, err := json.Marshal(current)
			if err != nil {
				return ""
			}
			return string(b)
		}),
		"load": js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) < 1 {
				return withError("no state")
			}
			g := &root.Game{}
			if err := json.Unmarshal([]byte(args[0].String()), g); err != nil {
				return withError(err.Error())
			}
			current = g
			return snapshotJSON()
		}),
		"snapshot": js.FuncOf(func(this js.Value, args []js.Value) any {
			return snapshotJSON()
		}),
	}
	js.Global().Set("RootEngine", js.ValueOf(api))
	select {}
}
