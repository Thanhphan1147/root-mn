# Root Machine Notation (RMN)

A machine-first notation, rules engine, and analysis/play tool for the board
game [ROOT](https://ledergames.com/products/root-a-game-of-woodland-might-and-right)
by Leder Games.

RMN is designed for **deterministic, machine-consumable game logs** — unlike
human-recording notations, it records typed *intents* with explicit operands and
explicit *outcomes* (dice, draws, reveals), so a log replays to an identical
state with no hidden randomness.

This repository contains three things:

1. **`SPEC.md`** — the RMN v3.0 specification (grammar, JSON encoding, intent
   dictionaries, validation, replay, error taxonomy).
2. **`internal/rmn`** — a Go parser/validator for the RMN text and JSON forms.
3. **`internal/root`** — a rules-complete **base-game engine** for the four base
   factions (Marquise de Cat, Eyrie Dynasties, Woodland Alliance, Vagabond),
   plus **`docs/`** — a serverless web analysis/play tool that runs the engine
   as WebAssembly and stores game state in the browser.

## Features

- **RMN parser** — dictionary-driven, single-pass parser for the text form with
  a typed operand registry, plus a JSON envelope and canonical round-trip.
- **Rules engine** — setup, turn/phase flow, rule/control, movement, battle
  (declare → defender ambush → attacker ambush → effects → dice → resolve),
  crafting, all 54 cards, dominance, and faction mechanics:
  - **Marquise** wood, building tracks (costs/VP), recruit, overwork, Field
    Hospitals, keep.
  - **Eyrie** leaders, decree resolution, turmoil, roosts.
  - **Woodland Alliance** sympathy, revolt, outrage, officers, military ops.
  - **Vagabond** items, explore, aid/relationships, quests, strike, rest,
    coalition, and ally support.
- **Legal-action generator** — the UI only ever offers legal moves, and the
  engine re-validates every action.
- **Relaxed friends mode** — pedantic preconditions (movement rule, build rule,
  decree skip) are lifted by default for open-mic table play.
- **Serverless web tool** — the Go engine compiled to WebAssembly; state lives
  in `localStorage`, so the site is pure static files and can be hosted on
  GitHub Pages.

## Layout

```
SPEC.md                 RMN v3.0 specification
cmd/rmn                 CLI (parse / validate / replay / canonical / apply / serve)
cmd/rootwasm            WebAssembly entry point exposing window.RootEngine
internal/rmn            RMN parser, validator, and state replay
internal/root           ROOT base-game rules engine
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

## Web tool

```sh
# rebuild the wasm engine, then serve the static site locally
bash scripts/build-web.sh
go run ./cmd/rmn serve --addr :8080 --web docs
# open http://localhost:8080
```

Or with Docker:

```sh
docker compose up -d --build
```

### GitHub Pages

The site in `docs/` is fully static. Either point GitHub Pages at the `/docs`
folder, or use the included workflow (`.github/workflows/pages.yml`), which
builds the wasm and deploys `docs/`.

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

## Acknowledgements

- **ROOT** is designed by Cole Wehrle and published by Leder Games. This is an
  unofficial fan project and is not affiliated with or endorsed by Leder Games.
- The `testdata/` corpus includes real games recorded in
  [Rootlog](https://github.com/Vagabottos/Rootlog); those files remain under
  their original terms.
- Rule constants were cross-checked against the Law of Root and the
  [haunt-roll-fail](https://github.com/haunt-roll-fail/haunt-roll-fail) engine.

## License

MIT — see [LICENSE](LICENSE).
