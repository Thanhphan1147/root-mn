# AGENTS.md — root-mn (`pkg/root` + `pkg/rmn`)

**This file is THE reference for how the ROOT engine and the RMN notation work and
MUST evolve. When code and this document disagree, this document wins — fix the
code (or, if the design genuinely changes, update this file in the same change).**

---

## 1. What this repo is

`root-mn` is a rules-complete, deterministic engine for the board game ROOT, plus
RMN (Root Machine Notation): a machine-first, replayable log of a game.

Three repos work together (same author):

- **root-mn** (this repo): the engine (`pkg/root`) and the notation
  parser/tooling (`pkg/rmn`), the WASM analysis bench (`cmd/rootwasm`, `docs/`).
- **root-bot**: bots (`pkg/bot`, `pkg/eval`, `pkg/search`), a serverless demo
  (`cmd/botwasm`, `docs/`) and a replay viewer, and the corpus generator
  (`cmd/gencorpus`).
- **root-multiplayer**: a correspondence server that embeds the engine.

## 2. Core philosophy: the notation IS the engine

The engine is **event-sourced**. A game is a sequence of RMN lines; each line is a
declared command with declared parameters; applying a line mutates the global
state of the game.

- The canonical entrypoint is `Game.ApplyRMN(line)`.
- A line parses to `(round, phase, actor, intent, operands, outcomes)` and
  dispatches to a **declared handler** for that intent.
- Handlers are grouped by faction in dedicated files:
  - `handlers_generic.go` — intents shared by all factions (pass, move, craft,
    battle, discard, setup `place`, `notify` dispatch, `remove`/battle hits, …).
  - `handlers_mc.go` / `handlers_ed.go` / `handlers_wa.go` / `handlers_vb.go` —
    faction intents (`C:*`, `E:*`, `A:*`, `V:*`).
- `rmn_engine.go` holds the parser, the registry, `ApplyRMN`, `applyEvent`,
  `TryRMN`, system-event handling and the operational helpers.

Rules implementations are the existing, validated `applyX` functions. Handlers
build the canonical `Action` and `applyEvent` runs it. **We do not reimplement
rules in the handlers** — rewriting validated rules is how silent rules bugs are
introduced.

`applyEvent(a)` executes a handler-built action without the ID re-lookup `Apply`
does (the line is authoritative), and still runs the end-of-action hooks
(`checkWin`, `maybeFieldHospitals`) and records the RMN line.

`RMN` has a separate, standalone parser/folder in `pkg/rmn` (its own `State`) used
for validation/analysis. It is independent of `pkg/root` and MUST NOT import it.

## 3. System (chance) events — the intended design

Randomness (dice, shuffles, card draws, ruins, quest order) MUST be logged as
concrete outcomes during play, and **applied** (not regenerated) during replay,
so a game is reconstructible from the log alone, **independent of the seed**.

- **Dice**: a `battle … -> {atk=…,def=…}` line arms `Game.NextRoll`; `Roll()`
  consumes it and STILL advances `RngSeed`, so later shuffles/draws stay in sync
  during seeded play.
- **Shuffles**: log `SYS shuffle zone=DECK|QUESTS before=[…] after=[…]`.
  Replay applies `after` (sets `Deck = after`, `Discard = nil`; for `QUESTS`
  sets the first 3 as `QuestAvail`, rest as `QuestDeck`). `before` is for
  integrity checking. The random order is produced during play and only replayed.
- **Draws**: log the concrete drawn cards and inject them on replay, so hands do
  not depend on deck order. (Needed because a shuffle can occur *inside* one
  action — a draw that empties and recycles the deck — so a standalone shuffle
  line cannot be applied at the right moment.)
- **Ruins**: `SYS assign-ruins -> {ruins=[…], items=[…]}`; replay sets each
  clearing's `RuinItem`.
- **Quest order**: covered by the `zone=QUESTS` shuffle.

State hash MUST be order-insensitive for hidden piles (hash `Deck`/`Discard` as
sorted multisets) so reconstructed order need not match byte-for-byte.

System lines are informational during play and authoritative during replay.
`isSystemIntent` / `ev.Actor == "SYS"` route to `applySystemEvent`.

## 4. Determinism — hard rules

The whole system is worthless if the same log replays to a different state.

- **NEVER** let game state depend on Go map iteration order. When iterating a map
  to choose or remove something (items, cards, pieces), collect keys and
  **`sort`** them first. (Past bugs: Favor cards damaging VB items; VB capacity
  removal; VB boot/item selection; quest exhaustion; repair.)
- Legal actions must be enumerated deterministically. `enemiesIn` iterates
  `Order`, not the player map. Deduplicate like-intents (WA `spread`/`revolt`
  auto vs explicit; WA `train` per-base; a duplicate `pass`).
- Matching a logged line to a legal action MUST be done in stable ID order
  (`resolveByLine` sorts by ID).
- `beginTurn` resets per-turn scratch (`UsedThisTurn`, `AidCount`, `Revealed`, …).
- `Game.Roll` advances `RngSeed` exactly twice (discarding values when armed).

## 5. RMN format — conventions that MUST hold

- Line shape: `seq round.phase actor intent [operand=value …] [-> outcome …]`.
  Outcomes may be braced (`{a=1,b=2}`) or bare (`after=[…]`); everything after
  `->` is an outcome. Operand/outcome values may be `(a+b)`, `[a,b]`, quoted text.
- **Every event MUST be self-describing and unambiguous.** If two distinct legal
  actions can emit the same line, the notation is broken: add the distinguishing
  operand (e.g. Eyrie decree lines carry `card=`; `V:aid` carries
  `exhaust=`/`take=`; battle removals name the piece type; VP vs draw quest
  rewards differ).
- The `round.phase` of a line is the phase **at the moment the action was taken**,
  not the resulting state (capture before `applyResolved`).
- New intents MUST be added to `pkg/rmn/dict.go` with declared operands/outcomes.
- Prefer a specific intent over a generic `notify`; `notify` is only for truly
  generic, uniquely-texted messages, and the `notify` handler dispatches the few
  overloaded cases by text.

## 6. Hidden information & redaction

- RMN is the **full-information** log (it may contain draws/hands); redaction is a
  **view** concern (`pkg/root/redact.go`).
- `Redact(g, viewer)` hides other players' hands/supporters, removes the deck,
  scrubs hidden outcomes in the log/RMN (`hiddenLogKinds`, `hiddenRMNIntents`),
  and hides ruin contents from **everyone** (`RuinItem = ""`, `assign-ruins`
  items shown as `[?]`).
- **Legality MUST be computed on the true state, never the redacted clone.** A
  redacted snapshot once dropped the Vagabond's Explore because the (hidden)
  ruin item was cleared first. Acting-player legal actions come from `g`, not `cp`.

## 7. Validation — the yardstick (never regress)

- **Golden corpus**: `testdata/corpus/*.rmn` (+ `.hash`), ~100 games (2p + 4p).
  Regenerate with root-bot `cmd/gencorpus` after any change that alters behavior
  or the state digest.
- `TestApplyRMNGoldenCorpus`: replay every game through `ApplyRMN` and match the
  recorded final-state hash. It also reports handler vs resolver coverage — the
  target is **0 fallback** (every action line routed through a declared handler).
- `TestApplyRMNSeedIndependent` (the goal of the chance work): replay each game
  with a *different* seed and match the same hash.
- `pkg/root/rmn_replay_test.go`: setup logs replay deterministically.
- Any behavior change MUST keep the corpus tests green, or the corpus is updated
  deliberately and explained.

`Snapshot(g)["hash"]` is `stateDigest(g)`: a canonical digest of **position**
only — it excludes `RngSeed`, logs, `NextRoll`, `DrawnThisAction`,
`VBStolenThisAction`, `Seq`. Keep it that way.

## 8. Workflow: adding or changing an action

1. Legal generation in `actions.go` / `faction_*.go` — deterministic, ordered.
2. Apply logic in the existing `applyX` (rules live here).
3. RMN intent in `rmn.go` — unique, self-describing; update `pkg/rmn/dict.go`.
4. Handler in the right `handlers_*.go` — build the canonical `Action`.
5. If it consumes chance, log the concrete outcome and apply it on replay.
6. Regenerate the corpus; run the corpus tests until green (0 fallback).
7. Update this file if a philosophy/convention changed.

## 9. Repos, commands, resources

- Engine tests: `go test ./...` (in root-mn). Corpus: regenerate with
  `go run ./cmd/gencorpus -n 100 -out <root-mn>/testdata/corpus` (in root-bot;
  use a local `replace` while iterating, drop it before release).
- Releases: tag `vX.Y.Z`, push tag; bump consumers (`go get`).
- This host is resource-constrained: keep search/benchmarks small (low `-sims`,
  few games), use timeouts, avoid runaway processes. Do not restart
  `danang-photo-picker`.

## 10. Status / WIP

- Shipped: `ApplyRMN` + per-faction handlers (0 fallback on the corpus),
  dice-outcome injection, `TryRMN` (custom RMN input in both UIs with short
  errors: `malformed RMN`, `not a legal move`, specific build reasons).
- WIP (stashed in root-mn): full chance injection — `SYS shuffle before/after`,
  ruin replay, seed-independent state digest, and the seed-independence test.
  Finish per §3 (add draw logging + multiset pile digest), get the corpus green,
  then un-stash/commit.
