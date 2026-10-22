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

**Answered on 2026-10-22.** The values are the ones in
`features/komikan-rules.md`; this table records only what changed against the
proposals.

| Setting | Proposed | Answer | Note |
|---|---|---|---|
| `panguiCapturesToWin` | 6 | **8** | Raised to give the dogs room. A design choice, not a measured balance result; see the playout note in the rules feature document. |
| `allowMultipleCaptures` | true | true | Chain of jumps in one turn. |
| `multipleCapturesMandatory` | true | **false** | A chain may be stopped at any point. The puma keeps the turn after a jump, but it may end it instead of jumping again. |
| `trewaCanMoveBackward` | true | true | Trewa may move along any edge, not only toward the Pangui. |
| `panguiMovesFirst` | true | true | |
| `puTrewaWinByBlocking` | true | true | Pangui with no legal move loses. |
| `puTrewaWinByOccupyingTriangle` | false | false | Disabled until a layout with a defined "triangle area" exists. |
| Capture preferred over normal move | false | **false** | The puma is not made to eat. |
| No-legal-move-for-Trewa | draw | **draw** | `DRAW` is a game status. |

The changed values have tests: `TestPanguiMayDeclineACapture`,
`TestPanguiMayEndACaptureChain`, and the threshold in
`internal/httpapi/rooms_test.go`.

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

**Answered on 2026-10-22.** Of the four, only live updates were picked to be
worked on now; that work is done (`/api/rooms/{roomID}/events`, see
`features/komikan-web-api.md`).

1. **Reset / rematch inside a room.** Out of scope: one game per room, and a
   finished room stays finished.
2. **Ruleset per room.** Stored with the room (the schema has the column) but
   every room starts from the same default ruleset. The engine still plays
   the server's ruleset, not the stored copy, so a rule change applies to
   games in progress.
3. **Move history in the MVP.** Persisted and used for the authoritative
   state, not displayed in the UI.
4. **Draw status.** `DRAW` exists and is reachable.

## Next step

Nothing here is waiting on an answer. Reopen only if a rule needs to change
after play-testing.

**Update 2026-10-21:** Part 2 is answered (c). Part 1 is still open: the
rules engine task was deferred, so the defaults stand as proposals until
they are confirmed. Part 3 remains open as well.

**Update 2026-10-21 (later the same day):** the rules engine has been
written and shipped with the proposed defaults, so the game can be played
while the answers are still open. Every value in Part 1 is a field of
`rules.Ruleset`, and `rules.DefaultRuleset()` holds the proposals: changing
an answer means changing one struct, not the engine. The rules are written
up in `features/komikan-rules.md`.

**Update 2026-10-22:** every question is answered. The puma is not forced to
capture (`CapturePreferred` off), a chain of jumps may be stopped
(`MultipleCapturesMandatory` off), and the threshold is 8. The page offers a
"Terminar la captura" button while a chain is running. Playouts show that
raising the threshold lengthened the games (about 153 to 220 random moves)
and, in the only measurement available, did not produce a dog win: random
dogs never coordinate, so the balance has to be judged in real play. The
issue is closed.
