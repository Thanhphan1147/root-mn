---
title: Root Machine Notation (RMN) v3.0 — Machine-First Action Log
version: 3.0-draft2
date_created: 2026-09-19
last_updated: 2026-09-19
owner: RAN project (RMN spec authoring)
tags: [schema, design, data, process, board-game, root, replay]
---

# Root Machine Notation (RMN) v3.0 — Specification Draft 2

RMN v3.0 is a machine-first action-log notation for the board game **ROOT**
(Leder Games). It defines one event schema with two equivalent encodings:

* a **canonical text form** (`*.rmn`), line-oriented, normative for logs;
* a **canonical JSON form** (`*.rmn.json`), normative for field names and
  types, used for network transport and storage.

This document supersedes RMN v2.0 (`Design.md`) and draft 1. Where they
disagree, this document wins. The authoritative coverage input is
`coverage.md`. Section 19 maps Rootlog v2.x into RMN v3.0.

## Status of sections: CORE vs EXTENSION

Per the v3.0 scope decision, the **normative core** covers the universal event
model plus the four base-game factions **C (Marquise), E (Eyrie), A
(Woodland Alliance), V (Vagabond, any number of instances)**. Everything for
the expansion factions **L, O, D, P**, the Marauder factions **H, K**,
hirelings, landmarks, Homeland, and spy cards is marked **[EXT]** and is
specified at *dictionary level* (names, operands, effects) only. Extension
rule tables may be incomplete; extension intents MUST NOT destabilize the core
event model. A core-only implementation MUST be able to parse, validate,
fold, and hash a core log without loading any extension dictionary.

Conventions:

* `MUST`, `MUST NOT`, `SHOULD`, `MAY` per RFC 2119.
* EBNF uses `::=`, `|`, `[ ]` (optional), `{ }` (zero or more), `" "` (literal).
* Tables marked **NORMATIVE** are part of the specification; **INFORMATIVE**
  tables are guidance.
* Section badges: **[CORE]**, **[EXT]**.
* All integers are unsigned decimal; there is no negative operand type.

---

## 1. Scope, goals, non-goals, terminology [CORE]

### 1.1 Scope

This specification defines:

1. lexical and syntactic grammar of the text form;
2. the JSON object schema and the 1:1 text↔JSON mapping;
3. header/metadata records;
4. the entity model, stable identifier (ID) schemes, and state fields;
5. a unified location grammar;
6. the event record model (seq / index / actor / intent / operands / outcome /
   causal parent);
7. the universal intent vocabulary;
8. per-faction intent dictionaries for **C, E, A, V** (core) and for
   **G/L/O/D/P/H/K/hirelings/landmarks/Homeland/spy** (extension);
9. the battle sub-grammar;
10. the entropy and hidden-information model;
11. setup, draft, and ADSET grammar;
12. the validation model, invariants, and error taxonomy;
13. replay, checkpoints, hashing, and determinism rules;
14. the extension mechanism;
15. a worked example, a Rootlog→RMN import mapping, and conformance vectors.

### 1.2 Goals (priority order — strict)

* **G1 — Unambiguous machine parsing.** A single-pass, **dictionary-driven**
  parser. The EBNF in §4 is a *descriptive* superset; the parser selects the
  value production for each operand from the intent's declared operand type.
  No backtracking, no semantic lookahead, no context-dependent defaults.
* **G2 — Machine validation of legality.** Typed intents with explicit
  operands and explicit entropy, so a validator can check preconditions and
  invariants from the state.
* **G3 — Fast deterministic state computation.** `state = fold(events)`,
  stable IDs, cheap per-event deltas, defined iteration order.
* **G4 — Bit-exact replay.** All entropy and all state-affecting player
  choices are explicit; canonical ordering; state hashes; no wall-clock.

### 1.3 Non-goals

* Not a UI format (rendering, localization, art).
* Not a complete ROOT rules engine. The spec defines the *minimum* typed
  preconditions needed for machine validation; full rule-data preconditions
  are loaded from the normative data tables in Appendices A–C or supplied by
  an implementation.
* Not a database, network protocol, or container format.

### 1.4 Terminology

| Term | Definition |
|---|---|
| **Event** | One atomic action record; the unit of the log. |
| **Intent** | The typed verb of an event (e.g. `move`, `E:decree-add`). |
| **Operand** | A typed named argument (`key=value`) of an intent. |
| **Outcome** | The explicit entropy and explicit choices consumed by an intent. |
| **Actor** | The faction instance performing the event, or `SYS`. |
| **Faction kind** | Rules identity: `marquise`, `eyrie`, `alliance`, `vagabond`, `cult`, `riverfolk`, `duchy`, `corvid`, `hundreds`, `keepers`. |
| **Faction instance** | A concrete seat playing a kind. Canonical ID is the registry ID declared in the header (`C`, `V`, `G`, `V2`, …). |
| **Index** | `round.phase`, e.g. `1.D`. Round 0 / phase `S` is setup. |
| **Seq** | Global monotonic event sequence number, starting at 1, unique, +1 per event. |
| **Cause** | `~<seq>`: the seq of the event that causally triggered this one. |
| **Zone** | A named container for pieces/cards/items. |
| **Fold** | Deterministic left fold `state_n = apply(state_{n-1}, event_n)`. |
| **Delta** | Canonical, ordered list of mutations produced by one event. |
| **Checkpoint** | Sidecar record `(seq, state_hash)` used to accelerate replay. |
| **Manifest** | Normative table of cards/items/quests for a deck. |
| **Value type** | A named production in the operand type registry (§4.4) selected by the dictionary. |

---

## 2. Versioning & compatibility rules [CORE]

* The first header directive MUST be `%RMN 3.0`. Version is `MAJOR.MINOR`.
* **MAJOR** bumps for grammar changes that make old logs unparseable, removal
  or rename of normative JSON fields, changes to ID semantics, or changes to
  the state-hash algorithm.
* **MINOR** bumps for additive intents, additive JSON fields, additive
  location zones, additive manifests, additive error codes. Old logs remain
  valid; new fields are optional.
* **PATCH** bumps for editorial fixes.
* An implementation MUST reject a log whose `MAJOR` exceeds its supported
  `MAJOR` (`E1001`).
* Unknown intents in a **registered namespace** MUST be rejected at validation
  (`E2301`) unless the namespace is listed by `%Option extensions=`; such
  opaque intents are preserved verbatim, set `opaque:true` in JSON, and MUST
  NOT alter core state.
* Unknown **operand keys** are `E2210`; unknown **outcome fields** are
  permitted and ignored unless `%Option hash_opaque=include`.
* `%Option requires=<feature>[,...]` gates replay; an implementation lacking a
  feature MUST fail `E1002`.
* Text and JSON forms MUST round-trip losslessly (§5.7). Canonical emission is
  deterministic.

---

## 3. Lexical rules [CORE]

### 3.1 Line structure

The text form is lines terminated by LF (`\n`), CRLF (`\r\n`), or CR (`\r`).
A line is exactly one of:

* **blank** — zero or more spaces/tabs only; ignored.
* **comment line** — optional whitespace then `#` then any characters to end
  of line; ignored. (`#` in column 1 of an event line cannot occur because
  event lines start with a digit.)
* **header directive** — starts with `%` in column 1.
* **event line** — starts with a decimal digit.

A trailing `//` on an event line is the event's **note** (§9.1). `#` and `//`
are therefore disjoint: `#` is an ignored full-line comment; `//` after an
event is preserved as the note. `//` at line start is not a comment and is a
parse error.

The first non-blank, non-comment line MUST be `%RMN <version>`.

### 3.2 Whitespace

Token separator is one or more spaces or tabs (`WS`). Leading/trailing
whitespace on a line is ignored. Tokens never span lines. A line is the unit
of parsing.

### 3.3 Comments and notes

* Comment lines and blank lines MUST NOT affect parsing, validation, folding,
  or hashing.
* An event note (`// ...`) is preserved in `note` and excluded from folding
  and hashing.

### 3.4 Identifiers

```
IDENT     ::= ALPHA { ALPHA | DIGIT | "_" | "-" }
ALPHA     ::= "A".."Z" | "a".."z"
DIGIT     ::= "0".."9"
```

Intent verbs are lowercase `IDENT`s (may contain `-`). Faction instance IDs,
namespaces, and zone names are uppercase `IDENT`s. Enum values are lowercase
unless enumerated otherwise. `SYS` is reserved and MUST NOT be declared as a
faction.

### 3.5 Numbers and hex

```
INT       ::= "0" | ( "1".."9" ) { DIGIT }
HEX       ::= "0x" HEXDIG { HEXDIG }
HEXDIG    ::= DIGIT | "a".."f" | "A".."F"
```

* Quantities are `INT >= 1`; a `0` quantity is `E2201`.
* Dice values are `INT` in `0..3`.
* Prices are `INT` in `1..4`; funds `INT >= 0`.
* `HEX` MUST have at least one digit. Field-specific lengths are validated
  (e.g. state hashes are 64 hex chars, seed values 16).

### 3.6 Strings

```
STRING    ::= '"' { CHAR | ESCAPE } '"'
CHAR      ::= ? any character except '"', '\', and EOL ?
ESCAPE    ::= "\" ( '"' | "\" | "n" | "t" | "u" HEXDIG HEXDIG HEXDIG HEXDIG )
```

Used for player names and notes. Canonical emission uses minimal escaping and
contains no raw newline.

### 3.7 Suits

```
SUIT      ::= "F" | "R" | "M" | "B"
```

`F`=Fox, `R`=Rabbit, `M`=Mouse, `B`=Bird. An empty suit in a card pattern
means "any/unspecified".

### 3.8 Other terminals

```
WS        ::= ( " " | "\t" ) { " " | "\t" }
EOL       ::= "\n" | "\r\n" | "\r"
ANY       ::= ? any character except EOL ?
```

### 3.9 Case sensitivity and forbidden characters

All syntax is case sensitive; implementations MUST NOT case-fold. Forbidden
outside strings: a raw newline; a raw `//` in an operand (it would start a
note); the characters `{ } ~ ->` in operand tokens (`{ }` reserve outcome
objects, `~` cause, `->` the outcome introducer); a raw `%` in column 1 of an
event line.

---

## 4. Grammar for the text form [CORE]

The EBNF in this section is **NORMATIVE as a description of the token shapes**,
but the grammar alone is **not LL(1)**: `OperandValue` has overlapping first
sets. The **parser is dictionary-driven**:

1. After the intent verb, the parser loads the intent's ordered operand schema
   from the intent dictionary (§10, §11).
2. For each operand it reads `key`, `=`, then parses exactly one value using
   the **declared value type** of that key. The value type selects a single
   production from the registry in §4.4. There is no ambiguity and no
   backtracking.
3. It stops at `->`, `~`, `//`, or end of line.
4. It validates required/unknown keys against the schema (layer E2).

The descriptive `AnyValue` union below is therefore a *superset*;
implementations
MUST use the type registry, not the union, for dispatch.

### 4.1 Top level

```
Log          ::= Header { Line } EOF
Line         ::= Blank | CommentLine | Event
Blank        ::= { " " | "\t" } EOL
CommentLine  ::= { " " | "\t" } "#" { ANY } EOL
```

`Blank` and `CommentLine` are disjoint by the required `#`.

### 4.2 Header

```
Header       ::= { Blank | CommentLine } HdrWan { Blank | CommentLine | HdrDirective EOL }
HdrWan       ::= "%RMN" WS Version EOL
HdrDirective ::= "%Game"       WS IDENT
               | "%Map"        WS IDENT
               | "%Clearings"  WS ClearingSuit { WS ClearingSuit }
               | "%Deck"       WS IDENT
               | "%Faction"    WS Faction WS FactionKind [ WS "seat=" INT ] [ WS "name=" STRING ]
               | "%Draft"      WS "pool=" CodeList [ WS "order=" CodeList ] [ WS "method=" IDENT ]
               | "%First"      WS Faction
               | "%Winner"     WS FactionList
               | "%Seed"       WS "algo=" IDENT WS "value=" HEX
               | "%Landmark"   WS IDENT WS "at=" Location [ WS "by=" Faction ]
               | "%Hireling"   WS HirelingID WS "type=" IDENT [ WS "controller=" Faction ] [ WS "at=" Location ]
               | "%Option"     WS IDENT "=" ( Scalar | CommaList )
               | "%Checkpoint" WS "seq=" INT WS "hash=" HEX [ WS "algo=" IDENT ]
Version      ::= INT "." INT
ClearingSuit ::= Clearing ":" SUIT
FactionList  ::= "[" Faction { "," [ WS ] Faction } "]"
FactionCode  ::= "C" | "E" | "A" | "V" | "L" | "O" | "D" | "P" | "H" | "K"
CodeList     ::= FactionCode { "," FactionCode }
```

`%Map`, `%Deck`, `%First`, and at least one `%Faction` are **REQUIRED** for a
complete log (this matches the required JSON fields, §5.2). `%Clearings` is
required unless the map declares fixed suits (Autumn, Fall). A fragment log MAY
set `%Option partial=true` to relax this; such a log is not replayable.

Canonical directive order: `%RMN`, `%Game`, `%Map`, `%Clearings`, `%Deck`, all
`%Faction`, `%Draft`, `%First`, `%Seed`, all `%Landmark`, all `%Hireling`, all
`%Option`, `%Winner`, all `%Checkpoint`. A parser MUST accept any order after
`%RMN`; a canonical emitter MUST use the order above.

### 4.3 Event line

```
Event        ::= INT WS Index WS Actor WS Intent [ WS Operands ] [ WS Outcome ] [ WS Cause ] [ WS Note ] EOL
Index        ::= INT "." Phase
Phase        ::= "S" | "B" | "D" | "E"
Actor        ::= Faction | "SYS"
Intent       ::= CoreVerb | Namespace ":" IDENT
CoreVerb     ::= IDENT
Namespace    ::= IDENT
Operands     ::= Operand { WS Operand }
Operand      ::= IDENT "=" AnyValue
Outcome      ::= "->" WS "{" [ OutcomeField { "," [ WS ] OutcomeField } ] "}"
OutcomeField ::= IDENT "=" AnyValue
Cause        ::= "~" INT
Note         ::= "//" { ANY }
```

Rules:

* `Intent` with a `:` is an extension intent; its namespace MUST be registered
  (§6.2) or listed in `%Option extensions=`.
* Operands MAY appear in any order; canonical emission uses dictionary order.
  Duplicate keys are `E1305`.
* `Outcome` is **present iff** the dictionary marks the intent `explicit`;
  **permitted iff** `explicit` or `optional`; and **required iff** the
  dictionary marks the operand as hidden (`E2202`, `E2410`).
* `Cause` MUST reference an earlier `seq` (`E2203`) and SHOULD be present when
  the actor differs from the current player (`E2205`) or the effect was
  triggered by another event.
* Fixed order: `Operands`, `Outcome`, `Cause`, `Note`.

### 4.4 Operand value type registry

The parser dispatches on the **declared type** of the operand key. The
following table is **NORMATIVE**; the `AnyValue` union after it is descriptive
only and MUST NOT be used to parse an event operand.

| Type | Production | JSON encoding |
|---|---|---|
| `int` | `INT` | number |
| `hex` | `HEX` | string |
| `bool` | `"true"` \| `"false"` | boolean |
| `scalar` | `Scalar` | string/number/bool by token |
| `suit` | `SUIT` | string |
| `enum` | `IDENT` (allowed set per key) | string |
| `rel-status` | `RelStatus` | string |
| `text` | `STRING` | string |
| `faction` | `Faction` | string |
| `faction-code` | `FactionCode` | string |
| `hireling` | `HirelingID` | string |
| `owner` | `Owner` | string |
| `location` | `Location` | string |
| `location-list` | `Location` \| `List(Location)` | array (always) |
| `unit` | `Unit` | string (canonical ref) |
| `optional-unit` | `"none"` \| `Unit` | string (canonical ref or `"none"`) |
| `unit-group` | `UnitGroup` | array of `{qty,ref}` |
| `int-list` | `List(INT)` | array |
| `item-ref` | `ItemRef` | string |
| `item-list` | `List(ItemRef)` | array |
| `card` | `CardRef` | string |
| `card-id` | `CardID` | string |
| `card-id-list` | `List(CardID)` | array |
| `quest-id` | `QuestID` | string |
| `relic` | `RelicRef` | string |
| `phase` | `Phase` | string |
| `faction-list` | `FactionList` | array |
| `code-list` | `CodeList` | array |
| `comma-list` | `CommaList` | array |

```
AnyValue     ::= INT | HEX | "true" | "false" | "none" | SUIT | IDENT | STRING
               | Faction | Owner | Location | Unit | UnitGroup
               | CardRef | CardID | QuestID | RelicRef | ItemRef | HirelingID
               | Phase | CodeList | CommaList
Scalar       ::= INT | HEX | IDENT | STRING
List(T)      ::= "[" [ T { "," [ WS ] T } ] "]"
CommaList    ::= IDENT { "," IDENT }

Faction      ::= IDENT                          // not "SYS"; must be declared
Owner        ::= Faction | HirelingID
HirelingID   ::= "h." IDENT
FactionKind  ::= "marquise" | "eyrie" | "alliance" | "vagabond" | "cult"
               | "riverfolk" | "duchy" | "corvid" | "hundreds" | "keepers"
Phase        ::= "S" | "B" | "D" | "E"
RelStatus    ::= "h" | "0" | "1" | "2" | "a"

CardID       ::= SUIT DIGIT DIGIT               // exactly two digits, e.g. F01
QuestID      ::= "q." SUIT DIGIT DIGIT
CardRef      ::= CardID | CardPattern
CardPattern  ::= [ SUIT ] "#" [ IDENT ] | "?"
ItemRef      ::= "i." IDENT "#" ( DIGIT { DIGIT } | "ruin" )
RelicRef     ::= "r." RelicType [ "#" INT ] [ "@" INT ]
ItemType     ::= "boot" | "bag" | "coin" | "crossbow" | "sword" | "hammer"
               | "tea" | "torch" | "club"        // catalogue of valid IDENTs
RelicType    ::= "figure" | "tablet" | "jewelry"

Unit         ::= PieceRef | ItemRef | CardRef | QuestID | RelicRef
PieceRef     ::= Owner "." PieceKind
PieceKind    ::= "w" | "p" | "vp" | "mob" | "warlord"
               | "b." IDENT [ "#" INT ]
               | "t." IDENT [ "#" INT ]
               | "m." IDENT
               | "ldr." IDENT
               | "plot" [ "#" INT ]
               | "character." IDENT
UnitAtom     ::= [ INT ] Unit
UnitGroup    ::= UnitAtom { "+" UnitAtom }
               | "(" UnitAtom { "+" UnitAtom } ")"
```

`List` is non-recursive: `T` is a value type that is not `List` or
`unit-group`. Nested lists are `E1207`. Parentheses in a `UnitGroup` MUST NOT
nest; nested parentheses are `E1205`. The `location-list` type accepts either a
single `Location` (text `to=C1`) or a bracketed list (text `to=[C1,C2]`); JSON
canonicalizes both to an array. All other list types require brackets.

### 4.5 Locations

```
Location     ::= Clearing | Forest | Path | Burrow | Ferry | Hand | Deck
               | Discard | Supply | ItemSupply | Ruin | QuestZone | Board
               | Hoard | Retinue | HirelingZone | Exile
Clearing     ::= "C" INT
Forest       ::= "F" INT "_" INT { "_" INT }
Path         ::= "P" INT "_" INT
Burrow       ::= "BURROW" [ ":" Faction ]
Ferry        ::= "FERRY:" INT "_" INT
Hand         ::= "HAND:" Faction
Deck         ::= "DECK"
Discard      ::= "DISCARD"
Supply       ::= "SUPPLY"
ItemSupply   ::= "ISUPPLY"
Ruin         ::= "RUIN:" Clearing
QuestZone    ::= "QUEST:AVAIL" | "QUEST:DECK"
Board        ::= "BOARD:" Owner ":" BoardZone
Hoard        ::= "HOARD:" Faction
Retinue      ::= "RETINUE:" Faction ":" INT
HirelingZone ::= "HIRELING:" HirelingID
Exile        ::= "EXILE"
BoardZone    ::= "WOOD" | "SUPPORTERS" | "OFFICERS" | "SATCHEL" | "TRACK"
               | "DAMAGED" | "CRAFTED" | "ACOLYTES" | "OUTCAST" | "HATED"
               | "LOSTSOULS" | "FUNDS" | "PAYMENTS" | "VIZIERS" | "LEADER"
               | "SWAYED" | "MINISTERS" | "PLOTS" | "MOODS" | "RELICS"
               | "VP" | "QUEST"
               | "DECREE:" DecreeColumn
               | "PRICE:" PriceKind
               | "REL:" Faction
DecreeColumn ::= "RECRUIT" | "MOVE" | "BATTLE" | "BUILD"
PriceKind    ::= "HAND" | "RIVERBOATS" | "MERCENARIES"
```

Location token shape errors are `E1204`; semantic errors (clearing absent from
map, `a >= b` in a path, non-adjacent forest set) are validation `E2409`.
`BURROW` without a qualifier is legal only when exactly one Duchy instance is
declared. The canonical retinue zone is `RETINUE:<F>:<n>`; `BOARD:...` has no
retinue zone. The canonical hoard zone is `HOARD:<F>`.

### 4.6 Worked grammar example

```
17 1.D C move group=2C.w from=C1 to=C11
18 1.D C build who=C building=C.b.saw#2 at=C2 cost=C.t.wood#1
19 1.D E battle attacker=E defender=C at=C11 -> {atk=2,def=1,extra_atk=0,extra_def=0}
20 1.D C casualties at=C11 group=2C.w ~19
21 1.D C remove group=A.t.sym#1 at=C5
22 1.D A A:outrage from=C card=M05 at=C5 ~21
23 1.E C draw who=C qty=1 from=DECK -> {drawn=[F06]}
```

Each line parses with the dictionary: `move` declares `group:unit-group,
from:location, to:location`; `battle` declares `attacker:faction,
defender:faction, at:location` with outcome fields `atk/def/extra_atk/extra_def`
(all `int`); `remove` declares `group:unit-group, at:location`; `A:outrage`
declares `from:faction, card:card, at:location`; `draw` declares `who:faction,
qty:int, from:location` with outcome `drawn:card-id-list`. The index is
non-decreasing (`1.D` through 22, then `1.E`), and `A:outrage` at C5 carries
`~21`, the sympathy removal by C at C5 that triggered it.

---

## 5. JSON encoding schema + mapping [CORE]

### 5.1 Envelope

```json
{ "rmn": "3.0", "header": { ... }, "events": [ ... ], "checkpoints": [ ... ] }
```

`rmn` and `checkpoints` occur **only at the envelope level** and MUST NOT
appear inside `header`. NDJSON form emits one `header` object then one event
object per line; `checkpoints` may be emitted as objects with `"type":
"checkpoint"`.

### 5.2 Header object

| JSON field | Type | Required | Text directive |
|---|---|---|---|
| `game` | string | no | `%Game` |
| `map` | string | yes | `%Map` |
| `clearings` | array `{clearing,suit}` | conditional | `%Clearings` |
| `deck` | string | yes | `%Deck` |
| `factions` | array Faction | yes | `%Faction` |
| `draft` | `{pool:string[],order?:string[],method?:string}` | no | `%Draft` |
| `first` | string | yes | `%First` |
| `winner` | string[] | no | `%Winner` |
| `seed` | `{algo:string,value:string}` | no | `%Seed` |
| `landmarks` | array `{id,at,by?}` | no | `%Landmark` |
| `hirelings` | array `{id,type,controller?,at?}` | no | `%Hireling` |
| `options` | object string→`Scalar` | no | `%Option` (last wins) |

`Faction` object: `{id:string, kind:string, seat?:integer, name?:string}`.
`clearings` is required unless the map declares fixed suits. `partial` is a
reserved `options` key.

### 5.3 Event object

| JSON field | Type | Required | Meaning |
|---|---|---|---|
| `seq` | integer ≥ 1 | yes | global monotonic sequence |
| `round` | integer ≥ 0 | yes | round (0 = setup) |
| `phase` | `"S"\|"B"\|"D"\|"E"` | yes | phase |
| `actor` | string | yes | faction instance or `"SYS"` |
| `intent` | string | yes | verb |
| `operands` | object | yes (may be `{}`) | named operands |
| `outcome` | object | conditional | explicit entropy/choices |
| `cause` | integer | no | causal parent seq |
| `note` | string | no | free text |
| `opaque` | boolean | no | derived: namespace is an undeclared extension |

`opaque` is **derived from the intent namespace** and is never emitted in text
(there is no text token for it). It is excluded from the state hash.

### 5.4 Checkpoint object

| JSON field | Type | Required | Text |
|---|---|---|---|
| `seq` | integer | yes | `seq=` |
| `state_hash` | string (64 hex) | yes | `hash=` |
| `algo` | string | no | `algo=` |
| `full_state` | object | no | — (no text form) |
| `inclusive` | boolean | no (default true) | — |

`inclusive=true` means the checkpoint is the state **after** folding event
`seq`. A hash-only checkpoint cannot resume a fold; an implementation MUST
require `full_state` or an earlier `full_state` checkpoint to resume.

### 5.5 Mapping rules

1. `seq`, `round`, `phase`, `actor`, `intent`, `cause`, `note` map directly.
2. Operand keys are identical in text and JSON. Text→JSON: collect the
   `key=value` pairs into `operands`. JSON→text: emit each present key.
   Missing required key = `E2210`; unknown operand key = `E2210`.
3. Outcome keys are identical in both forms. Unknown outcome keys are allowed
   and preserved; they are ignored by folding unless `hash_opaque=include`.
4. Scalars map byte-identically. `unit-group` → `[{qty,ref}]`; `List(T)` →
   array; `Location`/`CardRef` → canonical string.
5. Header directives map field-by-field:
   * `%Clearings C1:R C2:M` → `clearings:[{clearing:"C1",suit:"R"},…]`;
   * `%Faction C marquise seat=1 name="Cat"` → `{id:"C",kind:"marquise",seat:1,name:"Cat"}`;
   * `%Draft pool=C,E order=C,E method=snake` → `draft:{pool:["C","E"],order:["C","E"],method:"snake"}`;
   * `%Landmark treetop at=C5 by=C` → `{id:"treetop",at:"C5",by:"C"}`;
   * `%Hireling h.C type=forestpatrol controller=A at=C5` → `{id:"h.C",type:"forestpatrol",controller:"A",at:"C5"}`;
   * repeated `%Option k=v` → `options[k]=v` (last wins).
6. Canonical text emission: one space between tokens, LF endings, no trailing
   whitespace, directive order of §4.2, operand order = dictionary order.
7. Canonical JSON emission: schema key order, 2-space indent, no trailing
   spaces. For hashing only, keys are sorted (§16.4).

### 5.6 JSON example

```json
{
  "rmn": "3.0",
  "header": {
    "map": "autumn",
    "deck": "standard",
    "factions": [
      { "id": "C", "kind": "marquise", "seat": 1 },
      { "id": "E", "kind": "eyrie", "seat": 2 }
    ],
    "first": "C",
    "seed": { "algo": "rmn-rng-v1", "value": "0x9e3779b97f4a7c15" }
  },
  "events": [
    { "seq": 17, "round": 1, "phase": "D", "actor": "C", "intent": "move",
      "operands": { "group": [ { "qty": 2, "ref": "C.w" } ], "from": "C1", "to": "C11" } },
    { "seq": 18, "round": 1, "phase": "D", "actor": "C", "intent": "build",
      "operands": { "who": "C", "building": "C.b.saw#2", "at": "C2",
                    "cost": [ { "qty": 1, "ref": "C.t.wood#1" } ] } },
    { "seq": 19, "round": 1, "phase": "D", "actor": "E", "intent": "battle",
      "operands": { "attacker": "E", "defender": "C", "at": "C11" },
      "outcome": { "atk": 2, "def": 1, "extra_atk": 0, "extra_def": 0 } },
    { "seq": 20, "round": 1, "phase": "D", "actor": "C", "intent": "casualties",
      "operands": { "at": "C11", "group": [ { "qty": 2, "ref": "C.w" } ] }, "cause": 19 },
    { "seq": 22, "round": 1, "phase": "E", "actor": "C", "intent": "draw",
      "operands": { "who": "C", "qty": 1, "from": "DECK" },
      "outcome": { "drawn": ["F06"] } }
  ]
}
```

### 5.7 Round-trip

Given JSON `J`, `text(J)` parsed back MUST equal `J` except for insignificant
key ordering and optional-field omission. Given canonical text `T`, `json(T)`
re-emitted as canonical text MUST equal `T` byte-for-byte. §20 exercises this.

---

## 6. Header / metadata records [CORE]

### 6.1 Directives

| Directive | Semantics |
|---|---|
| `%RMN v` | Spec version; MUST be first. |
| `%Game id` | Opaque game identifier. |
| `%Map id` | `autumn` (core), `fall` (core), or extension `winter`, `lake`, `mountain`. |
| `%Clearings` | Explicit clearing→suit map; required unless the map has fixed suits. |
| `%Deck id` | `standard` (core) or extension `eap`. Selects the manifest (Appendix A). |
| `%Faction id kind ...` | Declares an instance; `kind` selects the dictionary; `seat` is 1-based order. |
| `%Draft` | ADSET pool and pick order. |
| `%First` | First-player instance. |
| `%Winner` | Winning instance list. |
| `%Seed` | Optional RNG algorithm and seed. Explicit outcomes take precedence and do not advance the RNG (§13.3). |
| `%Landmark` | Landmark placement. [EXT] |
| `%Hireling` | Hireling instance. [EXT] |
| `%Option` | Reserved keys §6.3; last wins. |
| `%Checkpoint` | Sidecar replay checkpoint; not an event. |

### 6.2 Faction kinds and registry

| Kind | Code | Base? | Notes |
|---|---|---|---|
| `marquise` | C | yes | |
| `eyrie` | E | yes | |
| `alliance` | A | yes | |
| `vagabond` | V | yes | Any number of instances (`V`, `G`, `V2`); all share the `V:` dictionary. |
| `cult` | L | [EXT] | |
| `riverfolk` | O | [EXT] | |
| `duchy` | D | [EXT] | |
| `corvid` | P | [EXT] | |
| `hundreds` | H | [EXT] | |
| `keepers` | K | [EXT] | |

Extension namespaces: `HIRE` (hirelings), `LM` (landmarks), `CARD`
(persistent-card effects), `SPY` (spy cards), `HOMELAND` (future factions).
`SYS` is reserved.

> Instance IDs are declared, not derived from letters, so a second Vagabond,
> a future Homeland faction, or a third Vagabond is added by declaration
> without a grammar change. Import preserves Rootlog letters as IDs.

### 6.3 Reserved `%Option` keys

| Key | Values | Meaning |
|---|---|---|
| `landmarks` | `on`/`off` | Landmarks in play. |
| `hirelings` | `on`/`off` | Hirelings in play. |
| `marauders` | `on`/`off` | H/K content. |
| `homeland` | `on`/`off` | Homeland content. |
| `spies` | `on`/`off` | Spy cards. |
| `partial` | `true`/`false` | Fragment log; relaxes required header fields. |
| `extensions` | `comma-list` | Opaque passthrough namespaces. |
| `requires` | `comma-list` | Replay aborts `E1002` if unsupported. |
| `hash_opaque` | `include`/`exclude` | Whether opaque intents/unknown outcomes enter the hash. |
| `entropy` | `explicit`/`commit`/`seed` | Preferred entropy regime. |
| `checkpoint_every` | `int` | Advisory checkpoint interval (default 64). |
| `first_player_rotation` | `seat`/`none` | Whether `%First` rotates each round (default `none`). |
| `source_sha256` | `hex` | Import provenance; excluded from the state hash. |

An unknown reserved-key value is `E2304`.

### 6.4 Header validity

* `%Faction` IDs unique; `%First`/`%Winner`/`%Draft.order` name declared
  instances/codes.
* `%Clearings` covers exactly the map's clearing set.
* `%Seed` names a registered algorithm (`rmn-rng-v1`) or `E2302`.

---

## 7. Entity model, IDs, state fields [CORE for C/E/A/V]

### 7.1 ID scheme

| Entity | ID form | Scope | Example |
|---|---|---|---|
| Faction instance | registry ID | log | `C`, `V`, `G` |
| Warrior | `<owner>.w` (count only) | per owner+zone | `C.w` |
| Pawn | `<owner>.p` | per instance | `V.p` |
| Building | `<owner>.b.<type>#<n>` | per owner type | `C.b.saw#2` |
| Token | `<owner>.t.<type>#<n>` | per owner type | `A.t.sym#2` |
| Plot | `<owner>.plot#<n>` | per owner | `P.plot#3` |
| Mob | `<owner>.mob` | per owner | `H.mob` |
| Warlord | `<owner>.warlord` | per owner | `H.warlord` |
| VP token | `<owner>.vp` | per owner | `C.vp` |
| Minister | `<owner>.m.<name>` | per owner | `D.m.marshal` |
| Leader | `<owner>.ldr.<name>` | per owner | `E.ldr.despot` |
| Character | `<owner>.character.<name>` | per owner | `V.character.tinker` |
| Item | `i.<type>#<n>` | global physical supply | `i.sword#1` |
| Card | `<suit><2-digit serial>` | per deck manifest | `F01` |
| Quest | `q.<suit><2-digit serial>` | per quest manifest | `q.F03` |
| Relic | `r.<type>#<n>[@<points>]` | global supply | `r.tablet#1@2` |
| Hireling | `h.<type>` | log | `h.C` |

Warriors are the only intentionally non-individuated entity: casualties are
`(<qty>)<owner>.w` at a location. All other limited entities have stable
per-instance IDs.

### 7.2 Base faction state schemas

| Kind | State fields | Board zones |
|---|---|---|
| `marquise` | `vp`, `wood{clearing:count}`, `buildings{type:count}`, `warriors{zone:count}`, `keep_location` | `BOARD:C:CRAFTED` |
| `eyrie` | `vp`, `leader`, `decree{recruit[],move[],battle[],build[]}` (ordered), `viziers[]`, `roosts`, `turmoil_count` | `BOARD:E:DECREE:{RECRUIT,MOVE,BATTLE,BUILD}`, `BOARD:E:VIZIERS`, `BOARD:E:LEADER` |
| `alliance` | `vp`, `supporters[]` (ordered), `officers`, `bases{clearing}`, `sympathy{clearing}`, `martial_law{clearing}` | `BOARD:A:SUPPORTERS`, `BOARD:A:OFFICERS` |
| `vagabond` | `vp`, `character`, `items{id:zone,state}`, `quests[]`, `relationships{faction:status}`, `coalition` | `BOARD:V:SATCHEL`, `BOARD:V:TRACK`, `BOARD:V:DAMAGED`, `BOARD:V:REL:<F>`, `BOARD:V:QUEST` |

### 7.3 Extension faction state schemas [EXT]

| Kind | State fields | Board zones |
|---|---|---|
| `cult` | `vp`, `acolytes`, `outcast`, `hated_outcast`, `lost_souls[]`, `gardens{clearing}` | `BOARD:L:ACOLYTES`, `BOARD:L:OUTCAST`, `BOARD:L:HATED`, `BOARD:L:LOSTSOULS` |
| `riverfolk` | `vp`, `prices{hand,riverboats,mercenaries}`, `funds`, `payments`, `posts{clearing,type}` | `BOARD:O:PRICE:{HAND,RIVERBOATS,MERCENARIES}`, `BOARD:O:FUNDS`, `BOARD:O:PAYMENTS` |
| `duchy` | `vp`, `ministers{name:swayed}`, `tunnels{clearing}`, `burrow` | `BOARD:D:SWAYED`, `BOARD:D:MINISTERS` |
| `corvid` | `vp`, `plots{id,clearing,face,type}`, `exposed` | `BOARD:P:PLOTS` |
| `hundreds` | `vp`, `warlord`, `mob{clearing:count}`, `moods[]`, `hoard[]` | `HOARD:H`, `BOARD:H:MOODS` |
| `keepers` | `vp`, `relics{id:location}`, `retinue[1..3]`, `waystations{clearing,side}` | `RETINUE:K:{1,2,3}`, `BOARD:K:RELICS` |

### 7.4 Items

Item zones: `ISUPPLY`, `RUIN:<clearing>`, `BOARD:<V>:SATCHEL/TRACK/DAMAGED`,
`BOARD:<F>:CRAFTED`, `EXILE`. State fields: `zone`, `exhausted` (bool),
`damaged` (bool). The normative catalogue is Appendix B; the base supply is
13 items: `boot#1..2`, `bag#1..2`, `coin#1..2`, `crossbow#1..2`, `sword#1..2`,
`hammer#1`, `tea#1`, `torch#1`. `club` is an extension item.

### 7.5 Cards, quests, relics

Card state: `id`, `suit`, `name`, `kind` (`normal`/`ambush`/`dominance`/
`favor`/`quest`), `owner`, `zone`, `face`. Card IDs are assigned by the
manifest (Appendix A) and are stable per deck ID. A `CardPattern` is legal only
where the dictionary declares a pattern type; a pattern that resolves to more
than one card in a hidden operation is an entropy event (`E2413` unless an
outcome/commit is present). Pattern resolution order is canonical (§16.2):
ascending `(suit, serial)`.

Quest state: `id`, `suit`, `name`, `location`, `owner`, `completed`. Relic
state: `id`, `type`, `points`, `location`.

---

## 8. Location model [CORE]

Locations are explicit, typed, and never overloaded. There are no
context-dependent defaults. The grammar is §4.5.

### 8.1 Rootlog overload resolution

| Ambiguous Rootlog token | Meaning A | Meaning B | RMN v3 tokens |
|---|---|---|---|
| `$_r` | Eyrie Recruit decree | Riverfolk riverboat price | `BOARD:E:DECREE:RECRUIT` vs `BOARD:O:PRICE:RIVERBOATS` |
| `$_m` | Eyrie Move decree | Riverfolk mercenary price | `BOARD:E:DECREE:MOVE` vs `BOARD:O:PRICE:MERCENARIES` |
| `$_x`/`$_b` | Eyrie Battle/Build decree | — | `BOARD:E:DECREE:BATTLE`/`:BUILD` |
| `$_` | Eyrie discard decree | Riverfolk set all prices | `E:discard-decree` vs `O:set-price service=all` |
| `$_o`/`$_ho` | Lizard outcast/hated | — | `BOARD:L:OUTCAST` / `BOARD:L:HATED` |
| `$_f` | Riverfolk funds | — | `BOARD:O:FUNDS` |
| `$_<Faction>` | Vagabond relationship | — | `BOARD:<V>:REL:<F>` |
| `$_<n>` | Keepers retinue column | — | `RETINUE:K:<n>` |
| `t` | wood/sympathy/plot/tunnel/post | — | `C.t.wood`, `A.t.sym`, `P.plot`, `D.t.tunnel`, `O.t.post` |
| `b_r` | Recruiter / Rabbit base / Rabbit garden | — | `C.b.recruiter`, `A.b.base_rabbit`, `L.b.garden_rabbit` |
| `b_m` | Mouse base / Mouse garden / Market | — | `A.b.base_mouse`, `L.b.garden_mouse`, `D.b.market` |
| `t_r` | Rabbit trade post / Raid plot | — | `O.t.post_rabbit`, `P.plot` |
| `a_b` | forest vs path | — | `F<a>_<b>` vs `P<a>_<b>` |
| `0` | burrow vs clearing 0 | — | `BURROW:D` (never `C0`) |
| `Q` | available quests vs quest deck | — | `QUEST:AVAIL` vs `QUEST:DECK` |
| `*` | discard pile as source | — | `DISCARD` |

### 8.2 Rule/control

Locations do not store rule or control; the validator derives
`rule(faction, clearing)` from piece counts and buildings. The derivation
algorithm is rule data (Appendix C map + ROOT rules) and is never an operand
default. Structural placement errors are `E2409`; rule/control failures are
`E2407`.

---

## 9. Event record model [CORE]

### 9.1 Fields

| Field | Text position | JSON field | Type | Required |
|---|---|---|---|---|
| Seq | 1 | `seq` | integer ≥ 1 | yes |
| Index | 2 | `round`, `phase` | integer, enum | yes |
| Actor | 3 | `actor` | faction ID or `SYS` | yes |
| Intent | 4 | `intent` | string | yes |
| Operands | 5.. | `operands` | object | yes |
| Outcome | after `->` | `outcome` | object | iff entropy |
| Cause | after `~` | `cause` | integer | conditional |
| Note | after `//` | `note` | string | no |

### 9.2 Seq rules

* `seq` starts at 1 and increases by exactly 1 per event. Gaps/duplicates are
  `E2204`. Header and checkpoint records do not consume `seq`.
* `seq` order IS canonical event order (§16.2).

### 9.3 Index rules

* `round` is 0 for setup, then 1, 2, 3, …; `phase` is `S` for setup, else
  `B`/`D`/`E`.
* The index is monotonic non-decreasing in canonical order; it MUST NOT
  decrease (`E2406`). It MAY repeat across many events.
* The turn pointer `(round, phase, current_player)` is part of state (§16.4).
  `turn` and `phase` markers update it.

### 9.4 Actor rules

* `actor` MUST be a declared instance or `SYS`.
* The current player is derived from seat order and the turn pointer.
* An event with `actor != current_player` MUST carry `cause` (`E2205`) except
  `SYS` events and setup events. Out-of-turn events (Riverfolk services,
  Outrage, Price of Failure, Extortion, off-turn scores, Field Hospitals) are
  first-class.

### 9.5 Intent rules

* Core intents are lowercase verbs; extension intents are `<Namespace>:<verb>`.
* The dictionary is selected by intent (namespace), not by actor. Two Vagabond
  instances share the `V:` dictionary.
* For a **faction-namespace** intent, the actor MUST be an instance of that
  kind, except the reaction/service intents `A:outrage`, `A:off-turn-score`,
  `D:price-of-failure`, `P:extortion-score`, and `O:buy`, which MAY be invoked
  with the reacting or paying actor. A violation is `E2407`.
* Unknown verbs are `E1301` (parser consults the dictionary); unknown
  namespaces are `E2301`.

### 9.6 Operand rules

* Operands are named in both text and JSON; binding is the intent dictionary.
* Required operands MUST be present; unknown keys are `E2210`. Optional
  operands are marked `?` in the dictionaries.
* A value that must express "no entity" uses the explicit token `none` (type
  `optional-unit`), never omission of a required key.

### 9.7 Outcome rules

* Outcome presence: **required** iff dictionary marks `explicit`; **permitted**
  iff `explicit` or `optional`; **forbidden** otherwise (`E2202`).
* For `optional` entropy, an outcome is required whenever the operation is
  hidden or resolves a pattern (`E2410`, `E2413`).
* Unknown outcome fields are permitted and ignored unless
  `hash_opaque=include`.

### 9.8 Cause rules

* `cause` MUST reference an earlier `seq` (`E2203`).
* Reactions/interrupts SHOULD set `cause` even when actor == current player.

---

## 10. Universal intent dictionary [CORE]

`Entropy` legend: `—` none; `E` explicit outcome required; `O` optional.
Operand suffix `?` = optional. Outcome fields are typed; `int-list`,
`item-list`, `optional-unit` are registry types. A `[S]` marker after an intent
name means the intent is **setup-legal**: it MAY appear at index `0.S`. Any
intent without `[S]` at `0.S` is `E2405`. The complete setup-legal set is
listed normatively in §14.2.

### 10.1 Movement, placement, items

| Intent | Operands | Entropy | Outcome | Preconditions | Effects |
|---|---|---|---|---|---|
| `move` | `group:unit-group, from:location, to:location-list` | — | — | source holds group; destination reachable; rule/control; no closed path; if `to` is a list, total group size == list length and one unit goes to each in order | remove from `from`, add to `to`; fire move triggers; cost 1 action |
| `place` [S] | `group:unit-group, to:location-list` | — | — | group in home zone; caps; multi-dest rule as `move` | move from home zone to `to`; cost 1 action (or free in setup) |
| `remove` | `group:unit-group, at:location` | — | — | group present | to home zone; fire removal triggers; record `last_removal` |
| `exile` | `group:unit-group, at:location` | — | — | group present | to `EXILE` |
| `route` | `group:unit-group, from:location, via:location, to:location` | — | — | path/ferry/tunnel open | explicit route-aware move |
| `close-path` | `path:location` | — | — | path open | close path |
| `open-path` | `path:location` | — | — | path closed | open path |
| `move-item` | `items:unit-group, to:location` | — | — | items on the mover's board | move items between satchel/track/damaged |
| `set-item-state` | `items:unit-group, exhausted?:bool, damaged?:bool` | — | — | items present | set item flags (does not move zones) |

`place`'s home zone is type-dependent (warriors/buildings/tokens → `SUPPLY`;
items → `ISUPPLY`; cards → `DECK`), which is deterministic, not
context-dependent.

### 10.2 Cards

| Intent | Operands | Entropy | Outcome | Preconditions | Effects |
|---|---|---|---|---|---|
| `draw` | `who:faction, qty:int, from:location` | E | `drawn:card-id-list` | `from` is `DECK`/`DISCARD`; deck non-empty or reshuffle first | cards to `HAND:who`; if deck empty, a `shuffle zone=DISCARD` MUST immediately precede |
| `discard` | `who:faction, cards:unit-group, from?:location` | O | `revealed:card-id-list` | cards in `from` (default: any `who` zone) | cards to `DISCARD` |
| `give` | `from:faction, to:location, cards:unit-group` | O | `revealed:card-id-list` | cards in `from`'s zones; transfer legal | cards to `to` (a hand, board zone, or supporters) |
| `craft` | `who:faction, card:card, produce:unit` | O | — | craft cost paid; crafting possible | discard card; item to `BOARD:who:CRAFTED` or resolve persistent effect |
| `reveal` | `who:faction, cards:unit-group, to:faction` | O | `revealed:card-id-list` | cards present | set face; outcome required if hidden |
| `swap` | `who:faction, card_out:card, card_in:card` | O | — | cards in relevant zones | exchange |
| `shuffle` [S] | `zone:location` | E | `order:card-id-list` | `zone` is `DECK`/`DISCARD` | reorder per outcome |
| `deal` [S] | `who:faction, cards:unit-group` | — | — | setup only | place specific cards into `HAND:who` |

### 10.3 Scoring and VP

| Intent | Operands | Entropy | Outcome | Preconditions | Effects |
|---|---|---|---|---|---|
| `score` | `who:faction, amount:int` | — | — | amount ≥ 1 | `who.vp += amount` |
| `lose-vp` | `who:faction, amount:int` | — | — | amount ≥ 1 | `who.vp = max(0, vp - amount)` |
| `move-vp-token` | `who:faction, to_board:faction` | — | — | coalition/dominance condition | set `who.vp.location` |
| `win` | `factions:faction-list` | — | — | victory condition | record winners |

### 10.4 Build, recruit, battle, dominance

| Intent | Operands | Entropy | Outcome | Preconditions | Effects |
|---|---|---|---|---|---|
| `build` | `who:faction, building:unit, at:location, cost?:unit-group` | — | — | rule; cost available; building in supply | spend; place building; cost 1 action |
| `recruit` | `who:faction, group:unit-group, at:location-list` | — | — | rule; capacity; supply | place warriors per destination |
| `battle` | `attacker:faction, defender:faction, at:location` | E | `atk:int, def:int, extra_atk:int, extra_def:int` | rule; attacker has warriors; defender present | declare battle; casualties are separate caused events; cost 1 action |
| `casualties` | `at:location, group:unit-group` | — | — | group present; caused by a battle | remove to supply; hit assignment explicit; cost 0 actions |
| `ambush` | `who:faction, at:location, card:card-id` | E | `hits:int` | open battle at `at`; card in hand; suit matches | play ambush; `hits` explicit |
| `play-dominance` | `who:faction, card:card-id, at?:location` | O | `target?:faction` | card in hand | card to board/clearing; record dominance target |
| `spend-bird` | `who:faction, card:card-id` | — | — | card is Bird in hand; Daylight | discard card; grant +1 action |

### 10.5 Turn / phase control

| Intent | Operands | Entropy | Outcome | Preconditions | Effects |
|---|---|---|---|---|---|
| `phase` | `phase:phase` | — | — | matches index | set turn pointer phase; entering `D` resets `actions_remaining=3` |
| `turn` | `who:faction` | — | — | seat order | set current player; set phase from index |
| `pass` | *(none)* | — | — | current player | no-op turn |

### 10.6 Reactions / interrupts

| Intent | Operands | Entropy | Outcome | Preconditions | Effects |
|---|---|---|---|---|---|
| `react` | `who:faction, trigger_seq:int, effect:enum` | O | — | trigger present | generic marker; effect-specific events follow |
| `outrage` | `from:faction, card:card, at:location` | O | `revealed:card-id-list` | `last_removal` shows `from` removed a sympathy token at `at` | card from `from` hand to `BOARD:A:SUPPORTERS` |
| `field-hospitals` | `at:location, spend:card, save:unit-group, to:location` | — | — | `last_removal` shows Marquise warriors removed at `at`; card in hand | discard card; place saved warriors at keep |
| `price-of-failure` | `who:faction, minister:unit` | — | — | `last_removal` shows a Duchy building removed | discard minister |
| `extortion` | `who:faction, at:location` | — | — | `last_removal` shows enemy moved into a Corvid plot clearing | score |
| `off-turn-score` | `who:faction, amount:int, trigger_seq:int` | — | — | trigger present | `who.vp += amount` |

### 10.7 Information

| Intent | Operands | Entropy | Outcome | Preconditions | Effects |
|---|---|---|---|---|---|
| `commit` | `who:faction, domain:enum, hash:hex` | — | — | — | store commitment |
| `reveal-commit` | `who:faction, domain:enum, value:scalar, nonce:hex` | — | — | commitment verifies | reveal; apply pending |
| `expose` | `who:faction, at:location, guess:enum` | E | `actual:enum` | facedown plot at `at` | compare; apply effect |
| `notify` | `who:faction, text:text` | — | — | — | non-state informational |

### 10.8 System intents

| Intent | Operands | Entropy | Outcome | Effects |
|---|---|---|---|---|
| `roll` | `who:faction, dice:int, sides:int` | E | `values:int-list` | record dice |
| `rng` | `purpose:enum` | E | `values:int-list` | generic entropy record |
| `setup` [S] | `key:enum, value:scalar` | O | — | initialize static state |
| `assign-ruins` [S] | *(none)* | E | `ruins:location-list, items:item-list` | assign items to ruins |
| `draft-pick` [S] | `who:faction, pick:faction-code` | — | — | record pick |
| `place-landmark` [S] | `landmark:enum, at:location, by:faction` | — | — | [EXT] place landmark |
| `hire` [S] | `hireling:hireling, controller:faction, markers:int` | — | — | [EXT] hire |

---

## 11. Per-faction intent dictionaries

A `[S]` marker after an intent name means the intent is setup-legal (MAY appear
at `0.S`); see §10 intro and §14.2. All other intents at `0.S` are `E2405`.

### 11.1 Marquise de Cat (`marquise`, `C`) [CORE]

| Intent | Operands | Entropy | Outcome | Preconditions | Effects |
|---|---|---|---|---|---|
| `C:birdsong-wood` | `at:location-list` | — | — | sawmills in/adjacent to `at` | place 1 wood token per sawmill at each `at` clearing |
| `C:build` | `who:faction, building:unit, at:location, cost:unit-group` | — | — | rule; wood cost | spend wood; place building; score |
| `C:recruit` | `who:faction, at:location-list` | — | — | recruiter rule; supply | place 1 warrior per clearing |
| `C:field-hospitals` | `at:location, spend:card, save:unit-group, to:location` | — | — | reaction; card in hand | discard card; save warriors |
| `C:overwork` | `at:location, spend:card, produce:unit` | — | — | workshop at `at` | discard card; produce wood/VP |

March is the universal `move`; battle is universal `battle`.

### 11.2 Eyrie Dynasties (`eyrie`, `E`) [CORE]

| Intent | Operands | Entropy | Outcome | Preconditions | Effects |
|---|---|---|---|---|---|
| `E:decree-add` | `column:enum, cards:unit-group` | — | — | Birdsong; cards in hand | append cards to `BOARD:E:DECREE:<COLUMN>` (order preserved) |
| `E:discard-decree` | *(none)* | — | — | Turmoil | discard all decree cards + viziers in canonical order; clear columns |
| `E:appoint-leader` [S] | `leader:unit` | — | — | Turmoil or setup | set `E.leader` |
| `E:turmoil` | `reason:enum` | — | — | decree could not be executed (rule-data) | increment `turmoil_count`; a `lose-vp` event follows |
| `E:recruit` | `at:location-list` | — | — | recruit column count | place warriors per recruit column |
| `E:move` | `group:unit-group, from:location, to:location` | — | — | move column count | move warriors; if impossible, Turmoil |
| `E:battle` | `defender:faction, at:location` | E | `atk,def,extra_atk,extra_def` | battle column count | as `battle`; if impossible, Turmoil |
| `E:build` | `building:unit, at:location` | — | — | build column count; roost | place roost |
| `E:add-vizier` | `card:card` | — | — | setup/ability | move card to `BOARD:E:VIZIERS` |
| `E:score-roosts` | `amount:int` | — | — | no cards in hand | score roosts |

### 11.3 Woodland Alliance (`alliance`, `A`) [CORE]

| Intent | Operands | Entropy | Outcome | Preconditions | Effects |
|---|---|---|---|---|---|
| `A:mobilize` [S] | `cards:unit-group` | O | `revealed:card-id-list` | Birdsong; cards in hand | move cards to `BOARD:A:SUPPORTERS` |
| `A:spend-supporters` | `cards:unit-group` | O | `revealed:card-id-list` | cost paid | supporters to `DISCARD` |
| `A:place-sympathy` | `at:location, spend:unit-group` | — | — | rule/adjacency; `spend` is a supporter card | discard `spend`; place sympathy; score |
| `A:revolt` | `at:location, base:unit, cards:unit-group` | — | — | 2+ supporters of suit; base in supply | spend `cards` to `DISCARD`; place base + warriors; remove enemy pieces; score |
| `A:organize` | `group:unit-group, from:location, to:location` | — | — | officers available | move warriors |
| `A:train` | `card:card` | — | — | spend supporter | gain officer |
| `A:outrage` | `from:faction, card:card, at:location` | O | `revealed:card-id-list` | as `outrage` | gain card to supporters/hand |
| `A:martial-law` | `at:location` | — | — | base present | place martial-law marker |
| `A:off-turn-score` | `amount:int, trigger_seq:int` | — | — | trigger | score |

### 11.4 Vagabond (`vagabond`, `V`; instances `V`, `G`, `V2`, …) [CORE]

| Intent | Operands | Entropy | Outcome | Preconditions | Effects |
|---|---|---|---|---|---|
| `V:choose-character` [S] | `character:unit` | — | — | setup only | set `V.character` |
| `V:move-pawn` [S] | `to:location` | — | — | pawn move legal | move pawn |
| `V:aid` | `target:faction, card:card, take:optional-unit` | — | — | card in hand; if `take` names an item it MUST be on `target`'s board | give card to target; move item to `BOARD:V:TRACK`; relationship +1; cost 1 action |
| `V:explore` | `at:location` | E | `item:item-ref` | ruin at `at`; torch | take ruin item to satchel/track; score; cost 1 action |
| `V:take-quest` | `quest:quest-id` | E | `quest:quest-id` | at quest location | quest to `BOARD:V:QUEST`; cost 1 action |
| `V:complete-quest` | `quest:quest-id, items:unit-group` | — | — | quest complete; `items` are the exhausted items | discard quest; score; exhaust `items`; cost 1 action |
| `V:repair` | `items:unit-group` | — | — | hammer; damaged items | damaged → satchel; refresh; cost 1 action |
| `V:rest` | `items:unit-group` | — | — | Evening; 3+ items | repair/refresh `items`; free |
| `V:refresh` | `items:unit-group` | — | — | Birdsong | clear `exhausted` |
| `V:exhaust` | `items:unit-group` | — | — | item available | set `exhausted=true`; free |
| `V:damage` | `items:unit-group` | — | — | item on board | move to damaged box |
| `V:move-item` | `items:unit-group, to:location` | — | — | items on `V`'s board | move satchel↔track↔damaged |
| `V:set-item-state` | `items:unit-group, exhausted?:bool, damaged?:bool` | — | — | items present | set flags |
| `V:relationship` [S] | `target:faction, status:rel-status` | — | — | legal change | set relationship (`h`,`0`,`1`,`2`,`a`) |
| `V:coalition` | `target:faction` | — | — | allied; VP condition | form coalition |
| `V:day-labor` | `card:card` | O | `revealed:card-id-list` | Tinker ability | draw named card from discard |

### 11.5 Lizard Cult (`cult`, `L`) [EXT]

| Intent | Operands | Entropy | Outcome | Effects |
|---|---|---|---|---|
| `L:set-outcast` [S] | `suit:suit` | — | — | set outcast; Lost Souls to discard |
| `L:set-hated-outcast` | `suit:suit` | — | — | set hated outcast |
| `L:build-garden` | `garden:unit, at:location` | — | — | place garden |
| `L:sanctify` | `at:location, building:unit` | — | — | remove building; gain acolyte/VP |
| `L:convert` | `at:location, warrior:unit` | — | — | remove enemy warrior; place own |
| `L:crusade` | `defender:faction, at:location` | E | `atk,def,extra_atk,extra_def` | as `battle` |
| `L:discard-lost-souls` | `cards:unit-group` | O | `revealed:card-id-list` | Lost Souls to discard; score |
| `L:reveal-outcast` | `cards:unit-group` | E | `revealed:card-id-list` | reveal for outcast |

### 11.6 Riverfolk Company (`riverfolk`, `O`) [EXT]

| Intent | Operands | Entropy | Outcome | Effects |
|---|---|---|---|---|
| `O:set-price` [S] | `service:enum, price:int` | — | — | set price(s); `service` ∈ `hand`,`riverboats`,`mercenaries`,`all` |
| `O:set-funds` | `amount:int` | — | — | set funds |
| `O:establish-post` | `post:unit, at:location` | — | — | place trade post + warrior |
| `O:move-post` | `post:unit, from:location, to:location` | — | — | move post |
| `O:buy` | `buyer:faction, service:enum, cost:int, card?:card-id, at?:location, via?:location` | O | `drawn:card-id-list` | one atomic event: buyer pays; service effect. `hand` requires `card`; `mercenaries` requires `at`; `riverboats` requires `via` |
| `O:sell-card` | `buyer:faction, card:card, cost:int` | O | — | card; funds |
| `O:dividends` | `at:location-list` | — | — | score per post |
| `O:export` | `card:card` | — | — | discard card; gain warrior in payments |
| `O:protectionism` | `spend:int` | — | — | spend funds; score |

### 11.7 Underground Duchy (`duchy`, `D`) [EXT]

| Intent | Operands | Entropy | Outcome | Effects |
|---|---|---|---|---|
| `D:sway-minister` [S] | `minister:unit, cards:unit-group` | E | `revealed:card-id-list` | move cards to `BOARD:D:SWAYED`; sway minister |
| `D:price-of-failure` | `minister:unit` | — | — | discard minister |
| `D:build` | `building:unit, at:location` | — | — | place citadel/market |
| `D:recruit` | `at:location-list` | — | — | place warriors |
| `D:march` | `group:unit-group, from:location, to:location` | — | — | move warriors/tunnels |
| `D:move-tunnel` | `tunnel:unit, from:location, to:location` | — | — | move tunnel |
| `D:burrow-move` | `group:unit-group, to:location` | — | — | move from `BURROW:D` |
| `D:reveal-sway` | `cards:unit-group` | E | `revealed:card-id-list` | reveal sway cards |

### 11.8 Corvid Conspiracy (`corvid`, `P`) [EXT]

| Intent | Operands | Entropy | Outcome | Effects |
|---|---|---|---|---|
| `P:place-plot` | `at:location` | E/O | `plot:enum` or `commit` | place facedown plot |
| `P:flip-plot` | `at:location, plot_type:enum` | — | — | reveal plot; apply effect; Bomb removes immediately |
| `P:remove-plot` | `at:location` | — | — | plot to supply |
| `P:trick` | `at_a:location, at_b:location` | — | — | swap plots |
| `P:expose` | `at:location, guess:enum` | E | `actual:enum` | compare; apply |
| `P:recruit` | `at:location-list` | — | — | place up to 4 warriors |
| `P:extortion-score` | `amount:int, trigger_seq:int` | — | — | score |
| `P:buy-card` | `seller:faction, card:card, cost:int` | — | — | buy card |

### 11.9 Marauder factions [EXT]

| Intent | Operands | Entropy | Outcome | Effects |
|---|---|---|---|---|
| `H:choose-mood` | `mood:enum` | — | — | set mood |
| `H:recruit-mob` | `at:location-list` | — | — | place mob warriors |
| `H:move-mob` | `from:location, to:location` | — | — | move mob |
| `H:roll-mob-die` | *(none)* | E | `values:int-list` | mob die (Rootlog omits it) |
| `H:raid` | `at:location` | — | — | raid; items to hoard |
| `H:advance` | `at:location` | — | — | advance warlord |
| `H:oppress` | `at:location` | — | — | build/score |
| `K:retinue-add` | `column:int, cards:unit-group` | — | — | append to retinue |
| `K:retinue-move` | `card:card, from:int, to:int` | — | — | move within retinue |
| `K:retinue-remove` | `card:card` | — | — | remove from retinue |
| `K:delve` | `at:location` | E | `relic:relic` | reveal/flip relic |
| `K:recover` | `relic:relic` | — | — | take relic |
| `K:rebuild` | `waystation:unit, at:location, side:enum` | — | — | build/rebuild waystation |
| `K:flip-relic` | `relic:relic, points:int` | — | — | set point side |
| `K:faithful` | `card:card` | — | — | play faithful retainer |

### 11.10 Hirelings, landmarks, card effects, spies, Homeland [EXT]

| Intent | Operands | Entropy | Outcome | Effects |
|---|---|---|---|---|
| `HIRE:activate` | `who:faction, hireling:hireling, effect:enum` | O | — | activate hireling ability |
| `HIRE:move` | `group:unit-group, from:location, to:location` | — | — | move hireling pieces |
| `HIRE:promote` | `hireling:hireling` | — | — | side up |
| `HIRE:demote` | `hireling:hireling` | — | — | side down |
| `HIRE:control` | `hireling:hireling, controller:faction, markers:int` | — | — | transfer control |
| `HIRE:return` | `hireling:hireling` | — | — | return to supply |
| `LM:use` | `landmark:enum, at:location, effect:enum` | O | — | landmark effect |
| `LM:close` | `landmark:enum` | — | — | exhaust/close |
| `CARD:effect` | `who:faction, card:card, effect:enum, targets?:unit-group, at?:location` | O | `revealed:card-id-list` | typed persistent-card activation |
| `SPY:play` | `who:faction, target:faction, card:card-id` | — | — | play spy |
| `SPY:trigger` | `who:faction, target:faction, effect:enum` | O | — | resolve spy |
| `SPY:return` | `who:faction, target:faction` | — | — | return spy |
| `HOMELAND:op` | `op:enum` | O | — | reserved opaque Homeland namespace |

### 11.11 Entropy and choice summary

| Intent family | Kind | Outcome fields |
|---|---|---|
| `draw` | deck order | `drawn` |
| `shuffle` | permutation | `order` |
| `battle` | dice | `atk`,`def`,`extra_atk`,`extra_def` |
| `ambush` | choice | `hits` |
| `V:explore` | ruin | `item` |
| `V:take-quest` | quest draw | `quest` |
| `expose`/`P:expose` | hidden plot | `actual` |
| `reveal`/`L:reveal-outcast`/`D:reveal-sway` | hidden cards | `revealed` |
| `assign-ruins` | assignment | `ruins`,`items` |
| `roll`/`rng` | dice | `values` |
| `A:revolt`/`A:place-sympathy`/`A:spend-supporters` | choice | `cards`/`spend` operands |
| `V:complete-quest` | choice | `items` operand |
| `V:aid` | choice | `take` operand |
| `O:buy` | choice | `card`/`at`/`via` operands |
| `commit` | commitment | (hash operand) |

---

## 12. Battle sub-grammar [CORE]

### 12.1 Canonical sequence

All events share the clearing and are chained by `cause`:

1. `battle attacker=A defender=D at=L -> {atk,def,extra_atk,extra_def}`.
2. Optional `ambush who=side at=L card=id -> {hits=N}` (defender first),
   caused by (1).
3. `casualties at=L group=...` caused by (1), one per side; defender assigns
   first. `group` names exactly the removed pieces.
4. Optional item damage / faction effects as separate caused events
   (`V:damage`, `A:outrage`, `D:price-of-failure`, `P:extortion-score`).
5. Optional `field-hospitals`/`L:crusade` follow-ons caused by (1).

### 12.2 Hit totals and the casualties invariant

```
attacker_hits = atk + extra_atk + Σ ambush.hits(attacker side)
defender_hits = def + extra_def + Σ ambush.hits(defender side)
```

For each battle `b`, let `removed(D)` be the total pieces removed in
`casualties` events with `cause=b` belonging to defender `D`, and
`removed(A)` likewise. The validator MUST check
`removed(D) == attacker_hits` and `removed(A) == defender_hits` (`E2412`).
Extra hits beyond the number of removable pieces are a rule-data concern and
MUST be recorded as `extra_*`.

### 12.3 Dice and ambush

* `atk`/`def` are raw dice in `0..3`; `extra_atk`/`extra_def` are explicit hit
  counts (default 0 if omitted). `ambush.hits` is explicit.
* At most one ambush per side; defender resolves first; `ambush` events MUST
  appear in that order and carry `cause` = battle seq.
* A battle with neither explicit dice nor a seed is `E2402`.

### 12.4 Multi-faction clearings

The attacker/defender pair is explicit; no inference.

---

## 13. Entropy & hidden-information model [CORE]

### 13.1 Principles

1. **Explicit outcomes and explicit choices are always sufficient.** A log in
   which every entropy intent has an outcome and every hidden/pattern
   operation is explicit replays with zero RNG.
2. **A seed is optional and subordinate.** If both seed and explicit outcome
   exist for a decision, the explicit outcome wins.
3. **Explicit outcomes do not advance the RNG.** When an outcome is supplied,
   the engine MUST NOT call `next()`; the RNG state is unchanged.
4. **Every state-affecting choice is recorded**: ambush hit count, supporter
   spends, quest items, aid take, `O:buy` targets, pattern resolution.
5. **Hidden information is committed or revealed**, never silently omitted.

### 13.2 Entropy sources and required records

| Source | Event | Required record |
|---|---|---|
| Deck order | `shuffle` | `order` |
| Reshuffle | `shuffle zone=DISCARD` | `order` |
| Draw | `draw` | `drawn` |
| Setup deal | `deal` | `cards` operand |
| Ruin contents | `assign-ruins` | `ruins`,`items` |
| Quest draw | `V:take-quest` | `quest` |
| Battle dice | `battle` | `atk`,`def`,`extra_atk`,`extra_def` |
| Ambush hits | `ambush` | `hits` |
| Plot hidden type | `P:place-plot` | `plot` or `commit` |
| Exposure | `P:expose`/`expose` | `actual` |
| Revealed hidden cards | `reveal`/`L:reveal-outcast`/`D:reveal-sway` | `revealed` |
| Mob die | `H:roll-mob-die` | `values` |
| ADSET randomness | `draft-pick`/`place-landmark`/`hire` | picks/placements |

### 13.3 Seed-derived entropy

If `%Seed algo=rmn-rng-v1 value=<hex>` is present and an outcome is absent for
an entropy-marked intent, the engine derives it deterministically:

* **RNG**: SplitMix64. State initialized to the 64-bit seed. `next()` adds the
  golden-ratio constant `0x9E3779B97F4A7C15` (mod 2^64), then applies the
  SplitMix64 finalizer (xor-shift 30 × `0xBF58476D1CE4E5B9`, xor-shift 27 ×
  `0x94D049BB133111EB`, xor-shift 31).
* **Draw**: consumes the current `DECK` order from index 0; `qty=n` consumes
  `n` cards in index order.
* **Shuffle**: Fisher–Yates over indices `n-1 … 1`, using `next() mod (i+1)`.
* **Dice**: exactly two `next() mod 4` calls per battle, in order `atk` then
  `def`. `extra_*` and ambush `hits` are never derived; they MUST be explicit.
* `rmn-rng-v1` is the only algorithm in v3.0. New algorithms require a MINOR
  bump and a new name.
* Seed `value` MUST be 16 hex digits; shorter values are zero-extended on the
  left, longer values are `E2302`.

### 13.4 Commit-reveal

1. On a hidden decision, emit `commit who=F domain=D hash=H` where
   `H = SHA-256(nonce_bytes || canonical_json(value))`, `nonce` is 16 recorded
   random bytes, and `canonical_json` is §16.4.
2. The engine stores the commitment in `hidden.pending[D]`. The value is not
   observable.
3. Any intent that reads the hidden value MUST be preceded by
   `reveal-commit who=F domain=D value=v nonce=hex`; the reveal verifies `H`
   and moves the value to `hidden.revealed[D]`. If an effect reads a value that
   is still pending, the log is `E2404`.
4. A `commit` not revealed by log end is `E2404`.
5. The nonce is recorded data and MUST NOT be regenerated during replay; it
   does not affect folding.

### 13.5 No wall-clock

No event, outcome, or derived value may depend on wall-clock, locale, PID,
hash-map iteration order, or floating point. All ordering is explicit (§16.2).

---

## 14. Setup & draft / ADSET grammar [CORE]

### 14.1 Static setup (header)

Map, clearings, deck, factions, seats, first player, seed, landmarks,
hirelings, options (§6).

### 14.2 Dynamic setup (events)

Dynamic setup is round 0, phase `S`, actor `SYS` or the setting-up faction.

| Step | Event |
|---|---|
| Shuffle | `SYS shuffle zone=DECK -> {order=[…]}` |
| Ruins | `SYS assign-ruins -> {ruins=[C2,C3,C7,C11], items=[i.sword#1,i.hammer#1,i.boot#1,i.crossbow#1]}` |
| Deal | `SYS deal who=C cards=(F03+F04+F05)` |
| Faction setup | `place`, `A:mobilize`, `V:choose-character`, `V:move-pawn`, `V:relationship`, `E:appoint-leader`, `L:set-outcast`, `O:set-price`, `D:sway-minister` |
| Draft | `SYS draft-pick who=C pick=marquise` |
| Landmark/hireling | `SYS place-landmark …`, `SYS hire …` |

Setup events use round 0 / phase `S` and MUST precede all round ≥ 1 events
(`E2405`). The complete **normative setup-legal intent set** is:

```
shuffle, deal, assign-ruins, draft-pick, place-landmark, hire, setup,
place, A:mobilize, V:choose-character, V:move-pawn, V:relationship,
E:appoint-leader, L:set-outcast, O:set-price, D:sway-minister
```

Any other intent at `0.S` is `E2405`. The `[S]` marker in §10/§11 marks these
entries; an implementation MAY extend the set only by declaring
`%Option requires=` and a MINOR version bump.

### 14.3 ADSET / draft

* `%Draft pool=… order=… method=…` declares the pool and order.
* If `method=random`, picks MUST be explicit `draft-pick` events (or
  committed).
* `%Map` and `%First` record the map and first-player outcomes.
* `first_player_rotation=seat` rotates `%First` each round by seat; `none`
  keeps it fixed.

### 14.4 Faction setup requirements (checkable)

At end of setup the validator checks:

* Marquise: keep + 1 sawmill + 1 workshop + 1 recruiter + 1 warrior per
  non-keep clearing placed; wood 0.
* Eyrie: roost + 6 warriors; leader chosen.
* Alliance: 3 supporters dealt; no base on map.
* Vagabond: character chosen; 3 items in satchel; pawn in a forest;
  relationships indifferent.

### 14.5 Ruin assignment

`assign-ruins` is an entropy event with an explicit outcome; it participates in
`seq` and hashing.

---

## 15. Validation model, invariants, error taxonomy [CORE]

### 15.1 Layers

| Layer | Codes | Meaning | Recovery |
|---|---|---|---|
| Parse | `E1xxx` | Lexical/syntactic violation; the parser consults only token shapes and the intent/type registry. | Reject line (or log). |
| Validation | `E2xxx` | Well-formed but illegal (schema, semantics, preconditions, invariants). | Reject/flag event. |
| Apply | `E3xxx` | Legal but state inconsistent during fold. | Abort. |

The parser IS dictionary-driven (it looks up intents and value types), so
`E1301` (unknown verb) and `E1307` (bad value for declared type) are parse
errors. Map membership, path order, adjacency, and enum membership are
validation (`E2409`, `E2212`). This resolves the parse/validation boundary:
the parser knows *shapes*, the validator knows *legality*.

### 15.2 Error taxonomy

| Code | Layer | Condition |
|---|---|---|
| `E1001` | Parse | Unsupported MAJOR. |
| `E1002` | Parse | Required feature unsupported. |
| `E1101` | Parse | Missing/wrong `%RMN` or wrong position. |
| `E1102` | Parse | Malformed header directive. |
| `E1103` | Parse | Malformed `%Faction`. |
| `E1201` | Parse | Illegal character. |
| `E1202` | Parse | Unterminated string. |
| `E1203` | Parse | Malformed number. |
| `E1204` | Parse | Malformed location token shape. |
| `E1205` | Parse | Malformed unit/card/item/quest/relic/hireling ref. |
| `E1206` | Parse | Malformed outcome. |
| `E1207` | Parse | Nested list. |
| `E1301` | Parse | Unknown intent verb. |
| `E1302` | Parse | Malformed operand (missing `=`). |
| `E1303` | Parse | Malformed cause. |
| `E1304` | Parse | Trailing tokens. |
| `E1305` | Parse | Duplicate operand key. |
| `E1306` | Parse | Malformed note. |
| `E1307` | Parse | Value does not match declared type. |
| `E2201` | Validation | Zero/negative quantity. |
| `E2202` | Validation | Outcome presence violation. |
| `E2203` | Validation | `cause` invalid. |
| `E2204` | Validation | Seq gap/duplicate. |
| `E2205` | Validation | Non-current actor without `cause`. |
| `E2206` | Validation | Actor not declared in the header. |
| `E2210` | Validation | Operand schema violation (missing/unknown). |
| `E2211` | Validation | Ambiguous pattern where identity required. |
| `E2212` | Validation | Unknown enum value. |
| `E2213` | Validation | Value out of range (e.g. a die outside `0..3`). |
| `E2301` | Validation | Unregistered namespace. |
| `E2302` | Validation | Unknown seed algorithm / bad seed length. |
| `E2303` | Validation | Unknown card/item/quest/relic ID. |
| `E2304` | Validation | Unknown reserved option value. |
| `E2401` | Validation | Supply cap exceeded. |
| `E2402` | Validation | Battle without dice and without seed. |
| `E2403` | Validation | Casualty names illegal piece. |
| `E2404` | Validation | Commit/reveal mismatch or unrevealed. |
| `E2405` | Validation | Setup ordering. |
| `E2406` | Validation | Index not monotonic. |
| `E2407` | Validation | Rule/control precondition failed. |
| `E2408` | Validation | Resource check failed. |
| `E2409` | Validation | Map/adjacency/zone capacity violation. |
| `E2410` | Validation | Hidden value without explicit outcome/commit. |
| `E2411` | Validation | Action economy exceeded. |
| `E2412` | Validation | Casualties do not equal hits. |
| `E2413` | Validation | Pattern resolved hidden without outcome. |
| `E3001` | Apply | Piece not found where event says it is. |
| `E3002` | Apply | Zone overflow / negative count. |
| `E3003` | Apply | Card/item duplicated or missing. |
| `E3004` | Apply | Checkpoint hash mismatch. |

### 15.3 Required invariants

Checked at every checkpoint and at log end:

1. **Piece conservation**: `map + board + supply + exile == cap` per faction
   and piece class.
2. **Supply caps** (`E2401`).
3. **Card conservation**: every manifest card in exactly one zone (`E3003`).
4. **Item conservation**: every catalogue item in exactly one zone (`E3003`).
5. **Seq integrity** (`E2204`); **index monotonicity** (`E2406`).
6. **Cause integrity** (`E2203`, `E2205`).
7. **Rule/control** (`E2407`); **resources** (`E2408`); **map** (`E2409`).
8. **Action economy**: in phase `D`, `actions_used <= 3 + bird_spent`
   (`E2411`), where `actions_used` counts action-costing events this Daylight
   and `bird_spent` counts `spend-bird` events this Daylight.
9. **Casualties == hits** per battle (`E2412`).
10. **Setup completeness** (§14.4, `E2405`).
11. **Entropy completeness**: every entropy intent has outcome or seed;
    hidden/pattern operations explicit (`E2402`, `E2410`, `E2413`).
12. **VP non-negative**.
13. **Coalition consistency**.
14. **Commitment revealed by log end** (`E2404`).
15. **`last_removal` consistency**: every reaction's trigger is present in the
    recorded `last_removal` record.

### 15.4 Preconditions made decidable

To make reaction preconditions decidable from state, the fold records
`last_removal = {seq, by, at, pieces[]}` on every `remove`/`casualties` and
`last_move = {seq, who, to, pieces[]}` on every `move`. Reactions reference the
trigger through `cause`; the validator checks the trigger record. Rule-data
preconditions (turmoil simulation, craft cost, victory thresholds, rule/control
derivation) are loaded from Appendices A–C or supplied by the implementation
and are marked `rule-data` in the dictionaries.

### 15.5 Reporting

A validator MUST report `code`, `layer`, `seq`, `line`, `message`, and
`context` (canonical event JSON). It MAY report multiple errors and MUST NOT
abort on the first validation error unless configured strict.

---

## 16. Replay, checkpoints, hashing, determinism [CORE]

### 16.1 State and fold

```
state_0     = initial_state(header)
state_n     = apply(state_{n-1}, event_n)
final_state = state_N
```

`apply` is pure and total over valid events; it reads entropy only from
`event.outcome` or the seeded RNG state. The engine exposes:

* `delta(event)` — a canonical, **seq-ordered** list of mutations
  `(path, op, old, new)` where `path` is a JSON pointer into the canonical
  state.
* `state_hash(state)` — §16.4.
* `delta_hash(delta)` — `SHA-256` of the canonical JSON of the ordered delta
  list, where deltas are ordered by `(seq, path)`. Implementations that do not
  emit deltas MAY omit `delta_hash`; if emitted, its algorithm is fixed here.

### 16.2 Canonical ordering

1. Events are ordered by ascending `seq`; seq order IS canonical order.
2. **Ordered containers** preserve semantic order and MUST be hashed as
   ordered arrays: `DECK`, `DISCARD`, each `DECREE` column, `SUPPORTERS`,
   `QUEST:DECK`, `QUEST:AVAIL`, `VIZIERS`, `LOSTSOULS`, `HANDS` (hand order),
   `BOARD:<V>:SATCHEL/TRACK/DAMAGED`, and the `RETINUE` columns.
3. **Unordered sets** are sorted by canonical ID:
   * cards by `(suit, serial)` where suit order is `B < F < M < R` (a fixed
     total order; it is not Unicode order);
   * items by `(type, index)`;
   * factions by `seat`, then registry ID;
   * clearings by numeric value;
   * board zones by the declaration order in §4.5.
4. Hash-map iteration MUST NOT affect state, deltas, or hashes.
5. `%_` is an **input-only macro** recognized by the importer/parser and
   expanded to the explicit canonical `unit-group`; it is never emitted
   canonically and never hashed.

### 16.3 Checkpoints

* A checkpoint is `(seq, state_hash, algo?, full_state?, inclusive=true)`.
* `inclusive=true` means the state after folding event `seq`; `inclusive=false`
  means before folding event `seq`.
* The default interval is **advisory** (`%Option checkpoint_every`, default 64);
  implementations MAY emit at any seq.
* A hash-only checkpoint cannot resume a fold; an implementation MUST resume
  from a `full_state` checkpoint or from `state_0`.
* On replay, a computed hash at a checkpoint seq that differs from the recorded
  `state_hash` is `E3004`.

### 16.4 Canonical JSON and hashing

**Canonical JSON (CJ)**: UTF-8; no insignificant whitespace; object keys
sorted by code point; arrays in §16.2 order; integers in decimal with no
leading zeros (the value `10` is `10`); no exponent; no floating point;
strings minimally escaped; booleans `true`/`false`; absent fields omitted (no
`null`).

**State object** (hashed exactly as follows):

```
{
  "header": <normalized header, excluding source_sha256>,
  "turn":   {"round":R,"phase":P,"current":F,"actions_remaining":N,"bird_spent":N},
  "factions": {...},            // vp, counts, board contents, relationships, coalition
  "zones": {
     "deck": [CardID...],       // ordered
     "discard": [CardID...],    // ordered
     "hands": {"C":[CardID...], ...},        // ordered
     "supporters": {"A":[CardID...]},        // ordered
     "decree": {"E":{"recruit":[...],"move":[...],"battle":[...],"build":[...]}},
     "quest_deck": [QuestID...],            // ordered
     "quest_avail": [QuestID...],
     "lost_souls": {...},
     "item_supply": [ItemRef...],
     "items": {ItemRef: {"zone":...,"exhausted":bool,"damaged":bool}},
     "supply": {pieceClass: count},
     "map": {...},              // clearings, pieces, buildings, tokens, ruins, path states
     "boards": {...},
     "relationships": {...},
     "coalitions": {...},
     "landmarks": {...}, "hirelings": {...},
     "last_removal": {"seq":N,"by":F,"at":L,"pieces":[...]},
     "last_move": {"seq":N,"who":F,"to":L,"pieces":[...]},
     "hidden": {"pending":{domain:hash},"revealed":{domain:value}},
     "opaque": {...}            // only if hash_opaque=include
  },
  "rng": {"state":hex,"calls":N} // only if %Seed present
}
```

**Exactly hashed**: the object above. **Excluded**: `checkpoints`, `note`,
comments, `%Option source_sha256`, pending hidden **values** (their hashes are
included), and opaque intents/unknown outcomes unless `hash_opaque=include`.
All reserved `%Option` keys except `source_sha256` are included (inside
`header.options`). `state_hash = SHA-256(utf8(CJ(state)))`, lowercase hex.

### 16.5 Determinism rules

1. No wall-clock, locale, PID, floating point, or non-seed RNG.
2. No dependence on host file order or map iteration order.
3. Explicit outcomes do not advance the RNG (§13.1.3).
4. All state-affecting choices are explicit events or outcome fields.
5. Folding is single-threaded over `seq` or serialized equivalently.

---

## 17. Extensions [EXT]

### 17.1 Principles

* New factions add a `%Faction` declaration and a namespaced dictionary; no
  core grammar change.
* New piece/token types are introduced by the faction state schema and the
  type registries; names MUST NOT collide within a namespace.
* New locations add `BoardZone`/`Location` alternatives (MINOR bump).
* Unknown intents in declared extension namespaces are preserved opaque
  (`opaque:true`), excluded from core state and from the hash unless
  `hash_opaque=include`.
* `%Option requires=` gates replay (`E1002`).

### 17.2 Adding a faction — checklist

1. Add kind/code to §6.2.
2. Add state schema (§7.3).
3. Add board zones to §4.5.
4. Add building/token/relic types to the registries.
5. Add the intent dictionary (operands/effects/entropy).
6. Add setup requirements.
7. Add conformance vectors.
8. Bump MINOR.

### 17.3 Provided extension dictionaries

§11.5–§11.10 give dictionary-level intents for L, O, D, P, H, K, hirelings,
landmarks, persistent-card effects, spies, and Homeland. Their rule tables
(e.g. exact garden costs, minister suits, relic points, plot effects, hireling
abilities, landmark effects, card-effect resolution) are **incomplete** and
MUST be loaded from expansion data or supplied by the implementation. A
core-only implementation MUST parse and reject such intents as `E2301` unless
the namespace is listed in `%Option extensions=`.

---

## 18. Worked example [CORE]

A complete, internally consistent fragment: four base factions on `autumn`
(fixed suits, so `%Clearings` is omitted) with the `standard` deck. All entropy
and choices are explicit. Any prefix `1..n` of this example is the fixture
`P18(n)` used by §20.

```rmn
%RMN 3.0
%Game example-v2
%Map autumn
%Deck standard
%Faction C marquise seat=1 name="Cat"
%Faction E eyrie seat=2 name="Eyrie"
%Faction A alliance seat=3 name="Alliance"
%Faction V vagabond seat=4 name="Vagrant"
%First C
%Option entropy=explicit

# ---- setup (round 0, phase S) ----
1 0.S SYS shuffle zone=DECK -> {order=[F03,F04,F05,R03,R04,R05,M03,M04,M05,B01,B02,B03,F06,R06,M06,B04,F01,F02,F07,F08,F09,F10,F11,F12,F13,F14,R01,R02,R07,R08,R09,R10,R11,R12,R13,R14,M01,M02,M07,M08,M09,M10,M11,M12,M13,M14,B05,B06,B07,B08,B09,B10,B11,B12]}
2 0.S SYS assign-ruins -> {ruins=[C2,C3,C7,C11], items=[i.sword#1,i.hammer#1,i.boot#1,i.crossbow#1]}
3 0.S SYS deal who=C cards=(F03+F04+F05)
4 0.S SYS deal who=E cards=(R03+R04+R05)
5 0.S SYS deal who=A cards=(M03+M04+M05)
6 0.S SYS deal who=V cards=(B01+B02+B03)
7 0.S C place group=(C.b.keep#1+C.b.saw#1+C.b.workshop#1+C.b.recruiter#1) to=C1
8 0.S C place group=2C.w to=C1
9 0.S C place group=2C.w to=[C2,C3]
10 0.S E place group=(E.b.roost#1+6E.w) to=C12
11 0.S E E:appoint-leader leader=E.ldr.despot
12 0.S A A:mobilize cards=(M03+M04+M05)
13 0.S V V:choose-character character=V.character.tinker
14 0.S V place group=(i.boot#2+i.torch#1+i.bag#1) to=BOARD:V:SATCHEL
15 0.S V V:move-pawn to=F6_7_8
16 0.S V V:relationship target=C status=0
17 0.S V V:relationship target=E status=0
18 0.S V V:relationship target=A status=0

# ---- round 1 ----
19 1.B C phase phase=B
20 1.B C turn who=C
21 1.B C C:birdsong-wood at=[C1]
22 1.B E turn who=E
23 1.B E E:decree-add column=RECRUIT cards=R03
24 1.B E E:decree-add column=MOVE cards=R04
25 1.B A turn who=A
26 1.B V turn who=V
27 1.B V V:refresh items=(i.boot#2+i.torch#1+i.bag#1)
28 1.D C turn who=C
29 1.D C C:build who=C building=C.b.saw#2 at=C2 cost=C.t.wood#1
30 1.D C move group=2C.w from=C1 to=C11
31 1.D C craft who=C card=F04 produce=i.coin#1
32 1.D E turn who=E
33 1.D E E:recruit at=[C12]
34 1.D E move group=2E.w from=C12 to=C11
35 1.D E battle attacker=E defender=C at=C11 -> {atk=2,def=1,extra_atk=0,extra_def=0}
36 1.D C casualties at=C11 group=2C.w ~35
37 1.D E casualties at=C11 group=1E.w ~35
38 1.D A turn who=A
39 1.D A A:place-sympathy at=C5 spend=M03
40 1.D A score who=A amount=1
41 1.D V turn who=V
42 1.D V V:move-pawn to=C7
43 1.D V V:exhaust items=i.torch#1
44 1.D V V:explore at=C7 -> {item=i.boot#1}
45 1.D V score who=V amount=1
46 1.D V V:aid target=C card=B01 take=i.coin#1
47 1.D V V:relationship target=C status=1
48 1.E C turn who=C
49 1.E C draw who=C qty=1 from=DECK -> {drawn=[F06]}
50 1.E E turn who=E
51 1.E E draw who=E qty=1 from=DECK -> {drawn=[R06]}
52 1.E A turn who=A
53 1.E A draw who=A qty=1 from=DECK -> {drawn=[M06]}
54 1.E V turn who=V
55 1.E V draw who=V qty=1 from=DECK -> {drawn=[B04]}
```

Consistency checks (all hold):

* **Index** non-decreasing: `0.S`, then `1.B` (19–27), `1.D` (28–47), `1.E`
  (48–55).
* **Seq** contiguous 1–55; causes 36/37 → 35.
* **Card conservation**: dealt F03/F04/F05, R03/R04/R05, M03/M04/M05,
  B01/B02/B03; F04 discarded by `craft`; M03 discarded by
  `A:place-sympathy`; draws F06/R06/M06/B04 unique; no card duplicated.
* **Item conservation**: ruins use `i.sword#1`, `i.hammer#1`, `i.boot#1`,
  `i.crossbow#1`; satchel uses `i.boot#2`, `i.torch#1`, `i.bag#1`; `i.coin#1`
  is crafted and then taken by `V:aid`; all unique.
* **Action economy**: C `build`/`move`/`craft` = 3; E `recruit`/`move`/
  `battle` = 3; A `place-sympathy` = 1; V `move-pawn`/`explore`/`aid` = 3.
* **Casualties == hits**: E deals 2 (`atk 2 + extra_atk 0`), C removes 2;
  C deals 1 (`def 1 + extra_def 0`), E removes 1.
* **Aid precondition**: `i.coin#1` is on `BOARD:C:CRAFTED` when `V:aid` runs.
* **Move legality**: edges `C1–C11` and `C11–C12` exist in Appendix C.

---

## 19. Rootlog → RMN v3.0 import mapping [INFORMATIVE]

### 19.1 Header mapping

| Rootlog | RMN v3.0 |
|---|---|
| `Map: <name>` | `%Map <lowercased>` |
| `Deck: <name>` | `%Deck standard` / `%Deck eap` |
| `Clearings: <suit list>` | `%Clearings C1:R C2:M …` |
| `Pool: <letters>` | `%Draft pool=C,E,A,V` |
| `<F>: <Player>` (seat order) | `%Faction <F> <kind> seat=<n> name="<Player>"` |
| *(implicit first player)* | `%First <first>` (importer infers; `I-01`) |
| `Landmarks: …` | `%Landmark …` + `SYS place-landmark` |
| `Hirelings: …` | `%Hireling …` + `SYS hire` |
| `Winner: <letters>` | `%Winner [ids]` |

Kind mapping: C→marquise, E→eyrie, A→alliance, V→vagabond, G→vagabond (second
instance), L→cult, O→riverfolk, D→duchy, P→corvid, H→hundreds, K→keepers.

### 19.2 Location mapping

| Rootlog | RMN v3.0 |
|---|---|
| `1`..`12` | `C1`..`C12` |
| `0` | `BURROW:D` |
| `a_b_c` (forest) | `F<a>_<b>_<c>` (ascending) |
| `a_b` (path) | `P<a>_<b>` (ascending) |
| `[F]$` | `BOARD:<F>:CRAFTED` |
| `[F]` (hand) | `HAND:<F>` |
| `s`/`t`/`d` | `BOARD:<V>:SATCHEL` / `:TRACK` / `:DAMAGED` |
| `Q` | `QUEST:AVAIL` |
| `$_r/$_m/$_x/$_b` | `BOARD:E:DECREE:RECRUIT/MOVE/BATTLE/BUILD` |
| `$_o/$_ho` | `BOARD:L:OUTCAST` / `BOARD:L:HATED` |
| `$_f` | `BOARD:O:FUNDS` |
| `$_h/$_r/$_m` (prices) | `BOARD:O:PRICE:HAND/RIVERBOATS/MERCENARIES` |
| `$_<F>` (relationship) | `BOARD:<V>:REL:<F>` |
| `$_1/2/3` (retinue) | `RETINUE:K:1/2/3` |
| `*` | `DISCARD` |
| *(omitted card start)* | `DECK` |
| *(omitted item start)* | `BOARD:<V>:<zone>` |

### 19.3 Action mapping (including P1–P8)

| Rootlog | RMN v3.0 |
|---|---|
| `<F>:` turn header | `turn who=<F>` + explicit index |
| `Nw A->B` | `move group=NC.w from=A to=B` |
| `w->A+B+C` | `place group=<F>.w to=[A,B,C]` (multi-destination, **P2**) |
| `Nw->A` | `place group=NC.w to=A` |
| `Nw A->` | `remove group=NC.w at=A` |
| `b_s->N` | `build who=<F> building=<F>.b.saw#n at=N` |
| `t->N` (token) | `place group=<F>.t.<type>#n to=N` |
| `p->N` / `p->a_b` | `V:move-pawn to=…` |
| `%...->s/t/d` | `V:move-item items=… to=BOARD:V:<ZONE>` (**P3**) |
| `%f->e` / `%f->d` | `V:set-item-state items=… exhausted=true` / `damaged=true` (**P3**) |
| `%b->t`, `%c->t` | `V:move-item items=… to=BOARD:V:TRACK` (**P3**) |
| `%sN->$` | `V:explore at=N -> {item=i.sword#n}` |
| `Z%t` / `Z<name>` | `craft who=<F> card=<id> produce=i.tea#n` / `produce=card:<name>` |
| `#->F` / `N#->F` | `draw who=F qty=N from=DECK -> {drawn=[…]}` |
| `#F->` | `discard who=F cards=…` |
| `#->` (deck→discard) | `discard who=<F> cards=… from=DECK` (**P6**) |
| `#F->G` | `give from=F to=HAND:G cards=…` |
| `#O->A$`, `#F->A$` | `give from=O to=BOARD:A:SUPPORTERS cards=…` (**P5**) |
| `#F->$_col` | `E:decree-add column=<COL> cards=…` |
| `#<cardname>->F` | `draw who=F qty=1 from=DISCARD -> {drawn=[…]}` or `give from=… to=HAND:F` (**P8**) |
| `$_->` (Eyrie) | `E:discard-decree` |
| `$_->N` (O) | `O:set-price service=all price=N` |
| `$_h->N` | `O:set-price service=hand price=N` |
| `$_f->N` | `O:set-funds amount=N` |
| `$_o/$_ho->Suit` | `L:set-outcast suit=S` / `L:set-hated-outcast suit=S` |
| `[V]$_F->{h,0,1,2,a}` | `V:relationship target=F status=<s>` |
| `#<leader>->$` | `E:appoint-leader leader=E.ldr.<name>` |
| `#<character>->$` | `V:choose-character character=V.character.<name>` |
| `#<minister>->$` | `D:sway-minister minister=D.m.<name> cards=…` |
| `#<minister>D$->` | `D:price-of-failure minister=D.m.<name>` |
| `F#domC->`, `B#dom->G`, `B#domG->$` | `play-dominance who=<F> card=<id> at?=… -> {target=…}` (**P1**) |
| `X<def><N>` | `battle attacker=<cur> defender=<def> at=N -> {atk,def,extra_atk,extra_def}` |
| `X…B@M@` | `ambush who=… at=N card=<id> -> {hits=…}` |
| `++` / `++N` | `score who=<cur> amount=N` |
| `F++N` | `score who=F amount=N` |
| `--N` | `lose-vp who=<cur> amount=N` |
| `++->F$` | `move-vp-token who=<cur> to_board=F` |
| `tN^t_s` | `P:flip-plot at=N plot_type=snare` |
| `PtN->` | `P:remove-plot at=N` |
| `tN<->tN` | `P:trick at_a=N at_b=M` |
| `?Pt_sN` | `P:expose at=N guess=snare -> {actual=…}` |
| `M#@O->P` | `O:buy buyer=P service=hand cost=N card=<id>` (**P7/O**) |
| `#saboO->A` | `CARD:effect who=O card=<id> effect=…` (**P7**) |
| `(w+Ew+Eb)N->` | `casualties at=N group=(…)` |
| `(a+b)#^` | `reveal who=<F> cards=… to=…` |
| `N#L->` | `L:discard-lost-souls cards=…` |
| `w_w` (H) | `H.warlord` |
| `h_<type>` | `%Hireling` + `HIRE:…` |

### 19.4 Entropy omitted by Rootlog that RMN MUST supply

| Omitted | RMN record | Fatal code if unknown |
|---|---|---|
| Deck order | `shuffle zone=DECK` with `order` | `I-12` |
| Draws | `draw` with `drawn` | `I-10` |
| Setup deals | `deal` with `cards` | `I-10` |
| Ruin contents | `assign-ruins` with `ruins`/`items` | `I-14` |
| Quest draws | `V:take-quest` with `quest` | `I-15` |
| Battle dice | `battle` with `atk`/`def` | `I-11` |
| Ambush hits | `ambush` with `hits` | `I-11` |
| Plot hidden types | `P:place-plot` with `plot` or `commit` | `I-03` |
| Lizard/Duchy reveals | `reveal`/`L:reveal-outcast`/`D:reveal-sway` with `revealed` | `I-13` |
| VB item state | `V:set-item-state` | `I-16` |
| Mob die | `H:roll-mob-die` | `I-17` |
| First player | `%First` | `I-01` |
| Turn/round indices | synthesized from line order | — |

`I-1x` is fatal unless `--allow-lossy`, which inserts `commit` placeholders
(`hash=UNKNOWN`) and marks the log non-replayable until resolved. Other
importer codes: `I-01` inferred first player, `I-02` inferred building type,
`I-20` ad-hoc coalition marker.

### 19.5 Intentional non-equivalences

Rootlog's "effect-as-move" is replaced by typed intents; the importer infers
the action (best-effort). Rootlog comments become `note`. Ad-hoc `#->C+V`
becomes `move-vp-token who=C to_board=V` (`I-20`).

---

## 20. Conformance test vectors [CORE]

Fixtures:

* **F0** — a valid header only:
  `%RMN 3.0`, `%Game fixture`, `%Map autumn`, `%Deck standard`,
  `%Faction C marquise seat=1`, `%Faction E eyrie seat=2`,
  `%Faction A alliance seat=3`, `%Faction V vagabond seat=4`, `%First C`.
* **P18(n)** — F0 plus §18 events `1..n` inclusive. The event under test
  occupies seq `n+1`.

Vector validation disables **only map/adjacency and rule-control checks** (and
craft-cost and victory checks). Actor-kind, piece-class, schema, entropy,
turn-pointer, setup-ordering, and conservation checks remain **enabled**, so
`E2407` (actor-kind/piece-class) is producible in this suite. A validator MAY
report several codes; the `Validation` column names a code that MUST be present
(or `OK` if no code may be present). This makes every vector reproducible from
F0 + the §18 prefix.

| # | Fixture | Event | Parse | Validation |
|---|---|---|---|---|
| 1 | P18(29) | `30 1.D C move group=2C.w from=C1 to=C11` | OK | OK |
| 2 | P18(29) | `30 1.D C move group=2C.w from=C1` | OK | `E2210` (missing `to`) |
| 3 | P18(29) | `30 1.D C move group=0C.w from=C1 to=C11` | OK | `E2201` |
| 4 | F0 | `1 1.D C battle attacker=C defender=E at=C5` | OK | `E2202` (outcome required) |
| 5 | P18(34) | `35 1.D E battle attacker=E defender=C at=C11 -> {atk=2,def=1,extra_atk=0,extra_def=0}` | OK | OK |
| 6 | P18(34) | `35 1.D E battle attacker=E defender=C at=C11 -> {atk=4,def=1,extra_atk=0,extra_def=0}` | OK | `E2213` (die out of range) |
| 7 | F0 | `1 1.D X:move group=2C.w from=C1 to=C11` | OK | `E2301` (unregistered namespace) |
| 8 | F0 | `1 1.D C fly group=2C.w from=C1 to=C11` | `E1301` | — |
| 9 | F0 | `1 1.D C move group=2C.w from=C1 to=C13` | OK | `E2409` (clearing absent) |
| 10 | F0 | `1 1.D C move group=2C.w from=C1 to=P4_2` | OK | `E2409` (`a >= b`) |
| 11 | F0 | `1 1.D C move group=2C.w from=a_b to=C1` | `E1204` | — |
| 12 | P18(29) | `30 1.D C move group=2C.w from=C1 to=C11 ~99` | OK | `E2203` |
| 13 | P18(29) | `31 1.D C move group=2C.w from=C1 to=C11` | OK | `E2204` (seq 30 missing) |
| 14 | F0 | `1 1.D C move group=2C.w from=C1 to=C11 to=C12` | `E1305` | — |
| 15 | P18(29) | `30 1.D C move group=(2C.w+C.b.saw#1) from=C1 to=C11` | OK | `E2407` (building cannot move) |
| 16 | F0 | `1 1.D C move group=(2C.w+(C.b.saw#1)) from=C1 to=C11` | `E1205` | — |
| 17 | P18(48) | `49 1.E C draw who=C qty=1 from=DECK` | OK | `E2202` |
| 18 | P18(48) | `49 1.E C draw who=C qty=1 from=DECK -> {drawn=[F06]}` | OK | OK |
| 19 | P18(48) | `49 1.E C draw who=C qty=1 from=DECK -> {cards=[F06]}` | OK | `E2202` (required `drawn` absent) |
| 20 | P18(35) | `36 1.D A A:outrage from=C card=M04 at=C5` | OK | `E2205` (no cause) |
| 21 | P18(35) | `36 1.D A A:outrage from=C card=M04 at=C5 ~99` | OK | `E2203` |
| 22 | P18(29) | `30 1.D C phase phase=X` | OK | `E2212` |
| 23 | P18(29) | `30 1.D C phase phase=D` | OK | OK |
| 24 | P18(29) | `30 1.D C P:place-plot at=C4` | OK | `E2407` (actor not Corvid) |
| 25 | F0 | `1 1.D P P:place-plot at=C4 -> {plot=snare}` | OK | `E2206` (actor undeclared) |
| 26 | P18(20) | `21 1.B C C:birdsong-wood at=[C1]` | OK | OK |
| 27 | P18(29) | `30 1.D SYS shuffle zone=DECK -> {order=[F01,F02]}` | OK | `E3003` (not a permutation) |
| 28 | P18(20) | `21 1.B SYS deal who=C cards=(F01)` | OK | `E2405` (deal after setup) |
| 29 | P18(29) | `30 1.D C score who=C amount=1` | OK | OK |
| 30 | P18(29) | `30 1.D C score who=C amount=0` | OK | `E2201` |
| 31 | F0 | `1 1.D C move group=2C.w from=C1 to=C11 // note` | OK | (note preserved) |
| 32 | F0 | `1 1.D C move group=2C.w from=C1 to=C11 # x` | `E1304` | — |
| 33 | P18(45) | `46 1.D V V:aid target=C card=B01 take=none` | OK | OK |
| 34 | P18(45) | `46 1.D V V:aid target=C card=B01` | OK | `E2210` (missing `take`) |
| 35 | P18(23) | `24 1.B E E:decree-add column=MOVE cards=R04` | OK | OK |

### 20.1 Round-trip vectors

| # | Action |
|---|---|
| R1 | Parse §18; emit JSON; emit canonical text; byte-compare to canonical form. |
| R2 | Parse §5.6 JSON; emit text; parse text; compare JSON semantically. |
| R3 | Import a Rootlog line using `%_`; assert the macro is expanded to an explicit ascending `(type,index)` unit-group and never re-emitted. |
| R4 | Replay §18 with `%Seed` present but all outcomes explicit; assert `rng.calls == 0` and that the final state **excluding the `rng` object** equals the no-seed replay. |

### 20.2 Hash vectors

| # | Action |
|---|---|
| H1 | Fold F0 plus §18 events 1–55; compute `state_hash` at seq 55; emit a `%Checkpoint seq=55 hash=<computed>`; re-fold from `state_0` and assert equality; assert folding two permutations of the deck order (identical multiset) produce **different** hashes. |
| H2 | Fold the same events with two commuting `score` events swapped in seq; assert the deltas differ and `delta_hash` differs even if `state_hash` is equal. |
| H3 | Add an unknown outcome field `zzz=1`; assert `state_hash` is unchanged when `hash_opaque=exclude` and changes when `include`. |
| H4 | Assert `%Option source_sha256` is excluded from `state_hash`. |

---

## 21. Open questions

1. **Component-data verification.** Appendix A (standard deck) and Appendix B
   (items) are the RMN reference manifests. They MUST be reconciled against the
   physical component list before v1.0; the manifest file's SHA-256 is the
   authoritative fixpoint.
2. **Extension rule tables.** L/O/D/P/H/K/hirelings/landmarks/card-effects/
   spies/Homeland dictionaries are at dictionary level; their rule tables
   (costs, suits, points, effects) are incomplete by design.
3. **Map data for Winter/Lake/Mountain.** Only Autumn/Fall are core; the rest
   are extension data.
4. **Rule/control algorithm.** `rule(faction, clearing)` derivation is rule
   data; the spec defines the predicate name and error code only.
5. **Victory thresholds.** `win` records the result; threshold data is rule
   data.
6. **`O:buy` timing.** Settled as one atomic event with explicit `card`/`at`/
   `via`; verify against expansion timing rules.
7. **`hash_opaque` default.** Currently `exclude`; a future MINOR may change
   it.
8. **Spy-card semantics.** Namespace defined; effects unverified.

---

## Appendix A — Standard deck manifest (RMN reference) [CORE]

> **CORRECTION (v3.0, errata):** the table below is the original placeholder and is
> **superseded** by the verified base-game manifest in `internal/root/data.go`
> (`buildDeck`). The real deck is **54 cards: 14 Fox (F01–F14), 14 Bird (B01–B14),
> 13 Rabbit (R01–R13), 13 Mouse (M01–M13)** with the exact suits, craft costs, and
> VP from the official Crafting Chart. Use `internal/root/data.go` as normative.

Format: `id,suit,name,kind`. Serial assignment is by this order and is stable
per deck ID. `kind` ∈ `normal`,`ambush`,`dominance`,`favor`,`quest`.

| id | suit | name | kind |
|---|---|---|---|
| F01 | F | ambush | ambush |
| F02 | F | dominance | dominance |
| F03 | F | favor | favor |
| F04 | F | betterburrowbank | normal |
| F05 | F | cobbler | normal |
| F06 | F | commandwarren | normal |
| F07 | F | codebreakers | normal |
| F08 | F | royalclaim | normal |
| F09 | F | sappers | normal |
| F10 | F | scoutingparty | normal |
| F11 | F | standanddeliver | normal |
| F12 | F | taxcollector | normal |
| F13 | F | armorers | normal |
| F14 | F | brutaltactics | normal |
| R01 | R | ambush | ambush |
| R02 | R | dominance | dominance |
| R03 | R | favor | favor |
| R04 | R | betterburrowbank | normal |
| R05 | R | cobbler | normal |
| R06 | R | commandwarren | normal |
| R07 | R | codebreakers | normal |
| R08 | R | royalclaim | normal |
| R09 | R | sappers | normal |
| R10 | R | scoutingparty | normal |
| R11 | R | standanddeliver | normal |
| R12 | R | taxcollector | normal |
| R13 | R | armorers | normal |
| R14 | R | brutaltactics | normal |
| M01 | M | ambush | ambush |
| M02 | M | dominance | dominance |
| M03 | M | favor | favor |
| M04 | M | betterburrowbank | normal |
| M05 | M | cobbler | normal |
| M06 | M | commandwarren | normal |
| M07 | M | codebreakers | normal |
| M08 | M | royalclaim | normal |
| M09 | M | sappers | normal |
| M10 | M | scoutingparty | normal |
| M11 | M | standanddeliver | normal |
| M12 | M | taxcollector | normal |
| M13 | M | armorers | normal |
| M14 | M | brutaltactics | normal |
| B01 | B | ambush | ambush |
| B02 | B | ambush | ambush |
| B03 | B | dominance | dominance |
| B04 | B | royalclaim | normal |
| B05 | B | sappers | normal |
| B06 | B | scoutingparty | normal |
| B07 | B | standanddeliver | normal |
| B08 | B | taxcollector | normal |
| B09 | B | armorers | normal |
| B10 | B | brutaltactics | normal |
| B11 | B | betterburrowbank | normal |
| B12 | B | cobbler | normal |

## Appendix B — Item catalogue (RMN reference) [CORE]

IDs are `i.<type>#<1-based index>`. Ruin-eligible items may be placed by
`assign-ruins`.

| type | count | ruin-eligible |
|---|---|---|
| `boot` | 2 | yes |
| `bag` | 2 | yes |
| `coin` | 2 | yes |
| `crossbow` | 2 | yes |
| `sword` | 2 | yes |
| `hammer` | 1 | yes |
| `tea` | 1 | yes |
| `torch` | 1 | yes |
| `club` | 0 core / 1 ext | no |

Total base items: 13. `club` is an extension item.

## Appendix C — Autumn map data (RMN reference) [CORE]

> **CORRECTION (v3.0, errata):** the original table below was a placeholder. The
> verified Autumn data (from the official numbered map card and the
> `haunt-roll-fail` reference engine) is normative in `internal/root/data.go`.

Clearings and fixed suits:

| Clearing | C1 | C2 | C3 | C4 | C5 | C6 | C7 | C8 | C9 | C10 | C11 | C12 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Suit | F | M | R | R | R | F | M | F | M | R | M | F |
| Slots | 1 | 2 | 1 | 1 | 2 | 2 | 2 | 2 | 2 | 2 | 3 | 2 |
| Ruin | – | – | – | – | – | R | – | – | – | R | R | R |

Adjacency (18 undirected edges):

```
C1-C5   C1-C9   C1-C10  C2-C5   C2-C6   C2-C10
C3-C6   C3-C7   C3-C11  C4-C8   C4-C9   C4-C12
C6-C11  C7-C8   C7-C12  C9-C12  C10-C12 C11-C12
```

Opposite-corner pairs (Bird Dominance): **C1↔C3** and **C2↔C4**.

Forests (adjacent clearings):

```
AutumnN:  C1 C2 C5 C10      AutumnNW: C1 C9 C10 C12
Witchwood:C2 C6 C10 C11 C12 AutumnW:  C4 C9 C12
AutumnS:  C3 C7 C11 C12     AutumnE:  C3 C6 C11
AutumnSW: C4 C7 C8 C12
```

Ruin slots: **C6, C10, C11, C12**.

`fall` is core with the same fixed-suit rule and its own adjacency/forest/ruin
data (to be supplied as a companion file). Winter/Lake/Mountain are extension
maps and require `%Clearings`.

## Appendix D — Error-code quick reference

Parse `E1xxx`; validation `E2xxx`; apply `E3xxx`; importer `I-xx`. See §15.2.

## Appendix E — Canonical abbreviations (informative)

Parsers MAY accept the following input-only aliases and MUST expand them before
hashing/emitting; they are never canonical:

| Alias | Canonical |
|---|---|
| `C0` | `BURROW:D` |
| `$E:r` | `BOARD:E:DECREE:RECRUIT` |
| `%_` | all items in the zone, expanded per §16.2 |
| `#` alone | `?` |
| `h_C` | `h.C` |


