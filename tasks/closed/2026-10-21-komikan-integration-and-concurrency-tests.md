# Cross-cutting integration, persistence, and concurrency tests

- Date: 2026-10-21
- Area: testing
- Dependencies: tasks/closed/2026-10-21-komikan-storage-layer.md, tasks/closed/2026-10-21-komikan-move-api.md

## Status

Partly done. `internal/httpapi` already plays a whole game through the API
with two cookie jars standing in for two browsers
(`TestAGameCanBePlayedToTheEnd`), checks the status codes of every kind of
bad request, and proves that a stale version is refused while the game keeps
its state.

Still to do:

- room isolation: activity in one room must not touch another room;
- identity and seat survive a restart of the server, tested through the API
  and not only in the store;
- two simultaneous submissions for the same turn, at the API level, yield
  exactly one success (the store already proves this with eight concurrent
  writers);
- a player can move while the opponent is absent (the server does not care
  about the opponent being online, but it should be proven).

## Outcome

An automated test suite exercises the whole server the way two players and a
restarting machine would, covering the requirements in `docs/spec.md` §11 that
unit tests cannot reach.

## Scope

- Two simulated browsers with separate cookie jars, playing a game across
  separate sessions:
  - create, join, play alternating moves, reach a terminal state;
  - a player can move while the opponent is absent.
- Room isolation: activity in one room does not change another room's state.
- Identity persistence: the same browser token restores the same seat after
  a reload and after a server restart.
- Persistence and recovery: close the store, reopen it against the same
  database file, and confirm rooms, seats, and game state are intact.
- Concurrency: simultaneous move submissions for the same turn yield
  exactly one accepted move; a stale version is rejected.
- Endpoint validation: malformed bodies, unknown rooms, wrong seats, out of
  turn moves, and moves after game end all return the documented status
  codes.
- The suite runs on this device with `go test ./...` and needs no external
  services.

## Acceptance criteria

- `go test ./...` passes on the device and covers each bullet above with at
  least one test.
- Tests are deterministic: no sleeps where a synchronisation point will do.
- Each test is independent and can run alone with `go test -run`.
- Failures name the requirement they break, so a regression is traceable to
  the spec.
- The suite does not require the deployed VPS or any network access.

## Verification

- `go test ./...` on the device.
- Optional on a laptop: `go test -race ./...`, since the race detector may
  not be available for `android/arm64`.

## Notes and open points

- This task covers the cross-cutting behaviour; rules, board, and store unit
  tests live with their own tasks.
- If the identity mechanism is still undecided, keep these tests written
  against a helper that hides how the token is carried, so a switch from
  cookie to header does not rewrite the suite.

## Completion notes

**Status: completed on 2026-10-21.** The suite is in
`internal/httpapi/integration_test.go`, next to the endpoint tests, because
that is where the whole server is reachable from. It runs with `go test ./...`
on the device and needs no network.

What each new test proves, and where the requirement comes from:

| Test | Requirement |
|---|---|
| `TestRoomsDoNotAffectEachOther` | spec §4: a move in one room leaves the other room's board, turn, status and move history alone, and each player only lists their own rooms |
| `TestTheSeatIsRestoredOnReload` | spec §4: reopening the room URL with the same browser token restores the seat and the role |
| `TestTheGameSurvivesARestartOfTheServer` | spec §7 and §11: the seats and the position are intact after the store is closed and reopened against the same database file, a stale version is still refused, and the game carries on |
| `TestAPlayerCanMoveWhileTheOpponentIsAbsent` | spec §6: the opponent need not be online, and the server does not play for them — the version does not move on its own |
| `TestOnlyOneOfTwoSimultaneousMovesIsAccepted` | spec §11: two submissions for the same turn leave exactly one accepted move and one `STALE_VERSION` |
| `TestAMoveAfterTheGameIsOverIsRefused` | spec §11: a move after the end is refused |
| `TestAMoveInAnUnknownRoomIsNotFound` | spec §11: a move in an unknown room is `404` |

How it is done:

- The test client grew a `try` method that reports errors instead of failing
  the test, so it can be called from a goroutine, and a `pointAt` method that
  moves a browser to another server address the way a browser does when the
  service comes back. The cookie jar follows the host, so the identity is
  kept.
- The two concurrent submissions are released by a channel barrier, not by a
  sleep, and the test ran 20 times in a row without flaking.
- The playout loop that plays a game to the end was lifted out of
  `TestAGameCanBePlayedToTheEnd` into a `playOut` helper, so both tests
  that need a finished game share it.

One behaviour changed while writing the tests: a move sent after the game
ended used to be refused with `ROOM_NOT_ACTIVE`, because the room is marked
as finished. It now says `GAME_OVER`, and `ROOM_NOT_ACTIVE` is left for a
room that has not started. Both are `409`, but the code now says which of
the two happened.

Still open, and not part of this task: the race detector cannot run on this
device (`android/arm64`), so `go test -race ./...` has not been run.

