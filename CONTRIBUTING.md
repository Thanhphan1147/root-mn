# Contributing

Thanks for taking a look at Root Machine Notation! This is an unofficial fan
project for the board game ROOT.

## Getting started

```sh
git clone https://github.com/Thanhphan1147/root-mn.git
cd root-mn
go test ./...
bash scripts/build-web.sh
go run ./cmd/rmn serve --addr :8080 --web docs   # http://localhost:8080
```

Requirements: Go 1.22+ (and optionally Docker). No Node toolchain is needed —
the web tool is vanilla JS plus the Go engine compiled to WebAssembly.

## What lives where

- `SPEC.md` — the notation spec. Changes here are design changes; open an issue
  first so we can talk through the grammar/JSON implications.
- `pkg/rmn` — parser, validator, replay for the notation.
- `pkg/root` — the ROOT rules engine. Faction rules should cite the Law of
  Root (or the wiki) in the PR description.
- `docs/` — the static web client. `docs/root.wasm` is generated; run
  `scripts/build-web.sh` after engine changes.

## Before you open a PR

```sh
gofmt -w .
go vet ./...
go test ./...
bash scripts/build-web.sh   # if the engine changed
```

CI runs the same checks. Please keep the engine strict by default and behind the
`Relaxed` flag for house rules.

## Reporting a rules bug

The fastest reports include:
1. the faction and phase,
2. the smallest RMN snippet (or UI steps) that reproduces it,
3. the expected behaviour and the rule reference.

## License

By contributing you agree that your contributions are licensed under the
project's [MIT License](LICENSE).
