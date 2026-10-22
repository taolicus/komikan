# The docs are provisional and need review before scope is frozen

- Date: 2026-10-21
- Area: product

## Problem

`docs/overview.md`, `docs/spec.md`, and `docs/board.md` were generated with AI
and have not been reviewed in detail. They are detailed enough to look
authoritative, which is a risk: implementation will otherwise follow wording
that nobody has checked. Two of the specs also contradict or omit things the
MVP depends on (see below), and the whole `docs/` directory is still untracked
in git, so it is not even in the repository history yet.

## Known contradictions and gaps found while reading them

1. **Role assignment is missing.** `docs/spec.md` describes two seats and a
   creator/joiner flow but never says which seat plays the Pangui. Tracked in
   `2026-10-21-komikan-rules-and-role-assignment-questions.md`.
2. **The board sketch is ambiguous** and probably not the traditional shape.
   **Resolved 2026-10-21:** the project pins the 38-point layout read off the
   sketch; see `tasks/closed/2026-10-21-komikan-board-topology-uncertainty.md`
   and `features/komikan-board-layout.md`. `docs/board.md` now carries a
   pointer to the authoritative definition.
3. **Capture geometry is prose, not a rule.** Tracked in
   `2026-10-21-komikan-capture-geometry-uncertainty.md`.
4. **"Restart with the same or a different ruleset"** (`docs/overview.md` §13)
   conflicts with the two-player room model, where a reset needs both
   players' agreement.
5. **Draws are referenced but never defined.** `docs/spec.md` mentions a
   "winner or draw result, when applicable" without ever saying when a draw
   occurs.
6. **`docs/spec.md` §9 lists a suggested API**; the endpoint shapes are
   "suggested" and will change. Treat them as a starting point, not a contract.
7. **The docs and the deployed site disagree.** The VPS currently serves the
   coming-soon page, which is fine, but the docs describe a finished product
   and should not be read as a description of what is deployed.

## What I would like from you

1. **Read `docs/board.md` first** — it is the one document whose content
   cannot be guessed from the other two.
2. **Annotate or correct the docs in place.** Direct edits to `docs/*.md` are
   welcome; they are the source of truth once reviewed.
3. **Add a status marker.** Proposed: a line at the top of each doc reading
   `Status: provisional — pending review`. This keeps the docs useful while
   making it obvious that they are not a settled spec.
4. **Decide whether `docs/` should be committed** as it stands. It is
   currently untracked; committing it would put the unreviewed text into
   history, and leaving it untracked means the issues and tasks I wrote refer
   to files that are not in the repository.

## Proposed MVP scope for sign-off

**In scope**

- One ruleset (configurable in code, not user-selectable in the MVP).
- Two-player rooms with browser-token identity, asynchronous play.
- Persistent rooms and games across restarts.
- Board rendering, move submission, server-side validation, win detection.
- Automated tests for rules, identity, rooms, persistence, concurrency.

**Out of scope for the MVP** (each is already listed as a future extension in
`docs/spec.md`, except where noted)

- Move history UI and replay (history is persisted, not displayed).
- Player names.
- Push/SSE notifications.
- Room expiry or archival.
- In-memory database mode for local development.
- Multiple concurrent rulesets chosen by the user.
- Reset/rematch inside a room (see gap 4 above).
- Optional real-time updates.

## Next step

Review the three documents, then confirm the scope list above. The task
sequence in `tasks/open/` implements the "in scope" list and stops there.

## Resolution (2026-10-22)

The documents were checked line by line against the code and corrected where
they had drifted:

- `docs/overview.md`: status line added; the capture-continuation rule now
  says a chain may be stopped; the threshold is 8; a draw section defines the
  `DRAW` status; the game state list and the ruleset defaults match
  `rules.DefaultRuleset()`; "restart with the same or a different ruleset" is
  marked out of scope.
- `docs/spec.md`: status line added; the communication line says the room is
  pushed over Server-Sent Events; the API section lists the endpoints that
  exist, including `/events` and `/board`, and points at
  `features/komikan-web-api.md` for the shapes; the future-extension list no
  longer offers real-time updates as a future item; the missing request size
  limit is called out as an open task instead of implying it exists.
- `docs/board.md` already pointed at `features/komikan-board-layout.md`.

The scope list is signed off, with one change: **live updates are in scope
and are done** (`tasks/closed/2026-10-22-komikan-live-room-updates.md`). Push
notifications while the page is closed stay out.

The owner also confirmed on 2026-10-22 that the phone layout is fine, having
looked at it. Gap 7 is stale: the site is live at
<https://komikan.tequionet.org>, not the coming-soon page.

This issue is closed. `docs/` describes the implemented system; `features/`
remains the authoritative implementation record.

## Scope list as signed off

**In scope**

- One ruleset (configurable in code, not user-selectable in the MVP).
- Two-player rooms with browser-token identity, asynchronous play.
- Persistent rooms and games across restarts.
- Board rendering, move submission, server-side validation, win detection.
- Live room updates over Server-Sent Events (added 2026-10-22).
- Automated tests for rules, identity, rooms, persistence, concurrency.

**Out of scope for the MVP**

- Move history UI and replay (history is persisted, not displayed).
- Player names.
- Notifications while the page is closed.
- Room expiry or archival.
- In-memory database mode for local development.
- Multiple concurrent rulesets chosen by the user.
- Reset/rematch inside a room.
