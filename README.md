# Root Machine Notation (RMN)

[![CI](https://github.com/Thanhphan1147/root-mn/actions/workflows/ci.yml/badge.svg)](https://github.com/Thanhphan1147/root-mn/actions/workflows/ci.yml)
[![Deploy](https://github.com/Thanhphan1147/root-mn/actions/workflows/pages.yml/badge.svg)](https://github.com/Thanhphan1147/root-mn/actions/workflows/pages.yml)
[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Live web tool → https://thanhphan1147.github.io/root-mn/**

A machine-first notation, a rules-complete engine, and a serverless analysis
bench for the board game [ROOT](https://ledergames.com/products/root-a-game-of-woodland-might-and-right)
by Leder Games.

RMN is built for **deterministic, machine-consumable game logs**. Unlike
notations designed for hand-recording, it records typed *intents* with explicit
operands and explicit *outcomes* (dice, draws, reveals), so a log replays to an
identical state with no hidden randomness — and can be validated, diffed, and
streamed.

## What it is

- **`SPEC.md`** — the RMN v3.0 specification: grammar, JSON encoding, intent
  dictionaries, validation, replay, and an error taxonomy.
- **`pkg/rmn`** — a Go parser/validator for the RMN text and JSON forms.
- **`pkg/root`** — a rules-complete **base-game engine** for Marquise de
  Cat, Eyrie Dynasties, Woodland Alliance, and Vagabond.
- **`docs/`** — a serverless web tool: the engine compiled to WebAssembly with
  state in `localStorage`, so it is pure static files.

## What it solves

- **Replayability.** Every state-affecting choice (ambushes, supporter spends,
  quest items, dice, draws) is explicit, so a game can be replayed exactly.
- **Validation.** Actions are typed intents with operands, so a parser/validator
  can check legality instead of guessing intent from raw effects.
- **A lightweight client.** No backend: the whole engine runs in the browser
  and the game lives in the tab, which makes hosting trivial and latency zero.
- **A foundation for multiplayer.** Because the engine is deterministic and the
  log is canonical, a sync/relay layer is a small step rather than a rewrite.

## Features

- **RMN parser** — dictionary-driven, single-pass parsing of the text form with a
  typed operand registry, a JSON envelope, and canonical round-trip.
- **Rules engine** — setup, turn/phase flow, rule/control, movement, battle
  (declare → defender ambush → attacker ambush → effects → dice → resolve),
  crafting, all 54 standard-deck cards, dominance, and full faction mechanics:
  - **Marquise** wood, building tracks (costs/VP), recruit, overwork, Field
    Hospitals, keep.
  - **Eyrie** leaders, decree resolution (viziers included), turmoil, roosts.
  - **Woodland Alliance** sympathy, revolt, outrage, officers, military ops.
  - **Vagabond** items, explore, aid/relationships, quests, strike, rest,
    coalition, and ally support.
- **Legal-action generator** — the UI only offers legal moves, and the engine
  re-validates every action; game state is a deterministic fold over events.
- **Strict by default** — rule/control is enforced. An optional `Relaxed` mode
  lifts movement/build preconditions for open-mic table play.
- **Serverless web tool** — map with roads/forests, minimap graph on small
  screens, manual setup (choose keep corner, Eyrie corner/leader, Vagabond
  character/forest), per-clearing build slots, hand cards with effect text, a
  step-by-step battle UI, human-readable **Logs**, and the generated **RMN
  annotations**.

## Live web tool

Open **https://thanhphan1147.github.io/root-mn/** — no install, no server. Game
state is saved in your browser's `localStorage`; "New manual setup" starts fresh.

Run it locally:

```sh
bash scripts/build-web.sh                       # build docs/root.wasm
go run ./cmd/rmn serve --addr :8080 --web docs  # serve docs/ at http://localhost:8080
```

Or with Docker:

```sh
docker compose up -d --build
```

The `docs/` folder is fully static, so GitHub Pages can serve it directly (the
included workflow builds the wasm and deploys it).

## Layout

```
SPEC.md                 RMN v3.0 specification
cmd/rmn                 CLI (parse / validate / replay / canonical / apply / serve)
cmd/rootwasm            WebAssembly entry point exposing window.RootEngine
pkg/rmn            RMN parser, validator, and state replay
pkg/root           ROOT base-game rules engine
docs/                   static web tool (GitHub Pages root)
scripts/build-web.sh    rebuilds docs/root.wasm + wasm_exec.js
testdata/               RMN fixtures + real recorded games (Rootlog corpus)
```

## Build & test

```sh
go build ./...
go test ./...
gofmt -l .
go vet ./...
```

## CLI

```sh
go run ./cmd/rmn parse     testdata/worked.rmn
go run ./cmd/rmn validate  testdata/worked.rmn
go run ./cmd/rmn replay    testdata/worked.rmn
go run ./cmd/rmn canonical testdata/worked.rmn
```

## Notation example

```
%RMN 3.0
%Map autumn
%Deck standard
%Faction C marquise seat=1
%Faction E eyrie seat=2
%First C

19 1.B C phase phase=B
20 1.B C turn who=C
21 1.B C C:birdsong-wood at=[C1]
22 1.D C move group=2C.w from=C1 to=C11
23 1.D E battle attacker=E defender=C at=C11 -> {atk=2,def=1,extra_atk=0,extra_def=0}
24 1.D C casualties at=C11 group=2C.w ~23
```

## Status & roadmap

**Phase 1 (done):** RMN v3.0 spec, parser/validator, rules-complete base-game
engine, and the serverless web bench (single-device hotseat).

**Phase 2 (planned):** a room/session sync layer for live multiplayer, replay
import/export from the RMN log, and expansion factions (Riverfolk, Underworld,
Marauders, hirelings, landmarks).

## Contributing

Issues and PRs are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md). If you find
a rules bug, a short RMN snippet plus the expected behaviour is the fastest way
to a fix.

## Contact

- GitHub: [@Thanhphan1147](https://github.com/Thanhphan1147)
- Bugs & ideas: [open an issue](https://github.com/Thanhphan1147/root-mn/issues)

## Acknowledgements

- **ROOT** is designed by Cole Wehrle and published by Leder Games. This is an
  unofficial fan project, not affiliated with or endorsed by Leder Games.
- The `testdata/` corpus includes real games recorded in
  [Rootlog](https://github.com/Vagabottos/Rootlog); those files remain under
  their original terms.
- Rule constants were cross-checked against the Law of Root and the
  [haunt-roll-fail](https://github.com/haunt-roll-fail/haunt-roll-fail) engine.

## License

[MIT](LICENSE) © 2026 Thanh Phan.
