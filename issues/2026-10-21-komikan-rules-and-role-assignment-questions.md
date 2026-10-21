# Rules defaults and seat role assignment need review

- Date: 2026-10-21
- Area: game-rules

## Problem

The rule values in `docs/overview.md` were chosen by an AI and are not yet
endorsed. Separately, `docs/spec.md` never says which of the two seats plays
the Pangui and which plays the twelve Trewa. The room model ("creator takes
the first seat, second player joins and the game starts") has no place to put
that assignment, so the API and UI cannot be specified without it.

## Part 1: rule defaults to confirm

| Setting | Value in the docs | Proposed default | Note |
|---------|--------------------|------------------|------|
| `panguiCapturesToWin` | 6 | 6 | Matches the usual "capture six prey" account. |
| `allowMultipleCaptures` | true | true | Chain of jumps in one turn. |
| `multipleCapturesMandatory` | true | true | Once a capture is made, continue if possible. |
| `trewaCanMoveBackward` | true | true | Trewa may move along any edge, not only toward the Pangui. Some accounts give the prey a forward-only movement; worth a second look. |
| `panguiMovesFirst` | true | true | |
| `puTrewaWinByBlocking` | true | true | Pangui with no legal move loses. |
| `puTrewaWinByOccupyingTriangle` | false | false | Disabled until a layout with a defined "triangle area" exists. |
| Capture preferred over normal move | not stated | true | See the capture geometry issue. |
| No-legal-move-for-Trewa | not stated | draw | See the capture geometry issue. |

## Part 2: role assignment (missing from the specs)

The two seats need to map onto the two roles. Options:

- **(a) Seat 1 = Pangui, seat 2 = Pu Trewa.** The room creator is always the
  Pangui; the joiner is always the Trewa. Simple, but a player who wants to
  be the Trewa has no way to choose.
- **(b) Creator chooses their role at room creation**, the joiner gets the
  other one. Flexible, one extra UI control and one extra persisted field.
- **(c) Roles are random on join.** Avoids a role-preference problem, but
  players cannot decline a role.

Proposed default: **(b)**, with the creator defaulting to Pangui. It costs
little and removes a class of "wrong seat" frustration.

**Answered on 2026-10-21: (c), roles are random on join.** The creator takes
seat 1 without a role; when the second player joins, both seats are given a
role at random. Implemented in `internal/store` (`JoinRoom`) and exposed by
the room API.

## Part 3: scope questions the specs leave open

1. **Reset / rematch inside a room.** `docs/overview.md` asks the UI to
   support "restarting a game with the same or a different ruleset". In a
   two-player room that needs agreement from both players, or it lets one
   player destroy the other's game. Not in the MVP unless you want it.
   Proposed default: **out of MVP scope**; the MVP has one game per room,
   and a finished room stays finished.
2. **Ruleset per room.** Should a room store its ruleset so future variants
   can be played? Proposed default: **yes, store it, but every room starts
   from the same default ruleset** — cheap to add now, avoids a schema change
   later.
3. **Move history in the MVP.** `docs/spec.md` requires move history to be
   persisted. Proposed default: **persist it** (it is needed for the
   authoritative state and for debugging) but **do not display it in the UI**
   in the MVP.
4. **Draw status.** See the capture geometry issue; a `DRAW` game status is
   needed if a draw is reachable.

## Next step

Confirm or change the defaults in Part 1, pick (a)/(b)/(c) in Part 2, and
confirm the Part 3 scope calls. The rules engine, room, and API tasks read
these decisions.

**Update 2026-10-21:** Part 2 is answered (c). Part 1 is still open: the
rules engine task was deferred, so the defaults stand as proposals until
they are confirmed. Part 3 remains open as well.
