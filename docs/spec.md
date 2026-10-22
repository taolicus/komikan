# Komikan — Game Project Specification

Status: implemented. The requirements below are what the live site does,
except where a line names an open task. The implementation notes are in
`features/` (`komikan-web-api.md`, `komikan-deployment-runbook.md`,
`komikan-rules.md`).

## 1. Overview

Komikan is a web-based implementation of the traditional Mapuche abstract strategy board game. The application allows two players to play against each other through a browser, with support for asynchronous turns and multiple independent game rooms.

The application is distributed as a single Go binary that embeds and serves its web interface and static assets.

## 2. Technology Stack

### Backend
- **Language:** Go.
- **Deployment:** A single self-contained binary.
- **HTTP server:** Go standard library `net/http`, unless a router becomes necessary.
- **Game logic:** A server-side implementation of the Komikan rules.
- **Persistence:** A storage layer for rooms, player identities, and game states. The initial implementation may use SQLite.
- **Concurrency:** Go concurrency primitives and synchronization to protect shared game state.

### Frontend
- **Delivery:** Web interface embedded in the Go binary using `embed`.
- **Client:** HTML, CSS, and JavaScript.
- **Communication:** HTTP endpoints for room management and game actions. A room page listens to a Server-Sent Events stream of its room, so a move of the other player arrives without the page asking; a browser without `EventSource` falls back to a slower poll.
- **Rendering:** The board is rendered in the browser, with clear visual indications of pieces, valid moves, the current turn, and the game status.

### Architectural Principles
- The server is authoritative for game state and rule validation.
- The frontend is responsible for presentation and user interaction.
- Game logic must be independent of HTTP handlers and UI implementation.
- The application should be deployable without requiring a separate frontend server or runtime.
- The application must support multiple rooms simultaneously.

## 3. Player Identity

Players receive a browser-based identity represented by a unique, opaque token.

Requirements:
- Generate a cryptographically random token when a browser first accesses the application.
- Persist the token in browser storage so that the identity survives page reloads and browser restarts.
- Associate room seats and player actions with the token.
- Never trust a player identifier supplied independently by the client when validating a move.
- Do not expose the raw token in room URLs.
- Provide a way to recover the player's active rooms from their browser identity.

A browser token identifies a browser instance, not a verified human identity. Clearing browser storage or using another browser may create a different identity.

## 4. Game Rooms

Players can create and join independent game rooms.

### Room Creation
- Any player can create a new room.
- Each room receives a unique identifier.
- Each room has a shareable URL that allows another player to access it.
- The room creator occupies the first seat.
- A newly created room waits for a second player before the game begins.

### Room Joining
- A second player can join through the room URL.
- Each room has exactly two player seats.
- When the second player takes the available seat, both seats become bound to their respective browser token identities.
- Once both seats are occupied, the game starts according to the initial game rules.
- A room must reject additional players after both seats are occupied.
- Reopening the room URL from a browser whose token is already assigned to that room must restore that player's existing seat rather than allocate another one.

### Room Isolation
- Each room maintains an independent board state, turn order, player assignment, and game status.
- Actions in one room must never affect another room.
- Room identifiers must be sufficiently unpredictable to prevent trivial enumeration.

## 5. Game Rules and Board

The implementation must follow the canonical Komikan rules and the board structure defined in the project's dedicated board specification.

Requirements:
- Represent the board as a graph of nodes and legal connections.
- Represent each node's occupancy explicitly.
- Define legal movement and capture rules in the game logic.
- Validate all moves on the server.
- Maintain the current player, piece positions, and game status.
- Detect game-ending conditions according to the rules specification.
- Reject moves that are illegal, out of turn, or submitted by a player who does not own the relevant seat.

The board topology, node identifiers, connections, initial positions, and other board-specific details must be maintained in the separate board specification rather than duplicated here.

## 6. Asynchronous Gameplay

Games are asynchronous. Players do not need to be online simultaneously.

Requirements:
- A player can make a move when it is their turn, regardless of whether the opponent is online.
- A valid move updates the authoritative game state and is persisted.
- After a move, the turn passes to the other player unless the game has ended or the rules specify otherwise.
- The game does not progress without a valid move from the player whose turn it is.
- Players can leave the application and return later without losing their place in the game.
- Returning players can retrieve the latest game state through their browser token identity and room URL.
- The interface must clearly indicate whose turn it is and whether the opponent has moved.
- A player who is looking at the room is told about the opponent's move while the page is open: the server pushes the new state through a Server-Sent Events stream, and the page does not poll for it.
- The server must not make moves automatically on behalf of absent players.

A player's absence does not cause a timeout, forfeiture, or automatic move unless such behavior is explicitly introduced as a separate game rule.

## 7. Game State and Persistence

The application must persist enough information to resume rooms and games after page reloads or server restarts.

Minimum persisted data:

### Player Identity
- Internal player identifier.
- Browser token identity, or a secure representation of it.
- Creation timestamp.

### Room
- Unique room identifier.
- Room status.
- References to the two player seats.
- Creation and last-updated timestamps.

### Game
- Board state.
- Current player to move.
- Move history.
- Game status, including waiting, active, and completed.
- Winner or draw result, when applicable.
- State version or revision number.

The implementation must define transaction boundaries so that a move and its associated turn change are persisted atomically.

Concurrent requests must not allow duplicate moves, conflicting state updates, or two moves to be accepted for the same turn.

## 8. User Interface

The web interface must provide:

- A home page listing the player's rooms.
- A way to create a new room.
- A shareable room URL with a copy-link action.
- A waiting-room view when only one player has taken a seat.
- A board view when both players are seated.
- Clear player-seat and turn indicators.
- Visual feedback for selecting pieces and making moves.
- Clear feedback for invalid moves and unavailable actions.
- Game-over information, including the winner or draw result.
- A way to return to the room list without losing game progress.

The interface must remain usable on desktop and mobile browsers.

## 9. HTTP API

The backend exposes a small API for browser interactions:

- `POST /api/rooms` — create a room.
- `GET /api/rooms/{roomID}` — retrieve room and game state.
- `GET /api/rooms/{roomID}/events` — stream the room whenever it changes (Server-Sent Events; one JSON document per event, the same shape as the `GET`).
- `GET /api/rooms/{roomID}/board` — the board topology on its own.
- `POST /api/rooms/{roomID}/join` — take an available seat or restore an existing seat.
- `POST /api/rooms/{roomID}/moves` — submit a move.
- `GET /api/rooms` — list rooms associated with the authenticated browser identity.
- `GET /healthz` — liveness, used by the deployment.

Identity validation, room isolation, and server-side game validation are
mandatory. The request and response shapes, the error codes, and the identity
cookies are documented in `features/komikan-web-api.md`, which is kept in step
with the code.

Room pages must use a distinct shareable URL, such as `/rooms/{roomID}`.

## 10. Security and Reliability

- Use cryptographically secure browser tokens.
- Use HTTPS when deployed beyond a trusted local environment.
- Validate every request and all move data on the server.
- Apply appropriate protections against cross-site request forgery when using cookie-based authentication.
- Protect against unauthorized room access and seat impersonation.
- Avoid exposing browser tokens in URLs, logs, or error messages.
- Handle simultaneous requests safely.
- Set reasonable request size limits and validate room identifiers. (The size limit is not enforced yet; it is tracked in `tasks/open/2026-10-22-komikan-public-endpoint-hardening.md`.)
- Ensure that invalid requests do not partially mutate game state.
- Provide clear errors without leaking sensitive implementation details.

## 11. Testing

The project must include automated tests for:

- Board topology and initial state.
- Legal and illegal moves.
- Capture rules and end-game conditions.
- Turn order and player authorization.
- Room creation and seat assignment.
- Rejoining with an existing browser token.
- Room isolation.
- Asynchronous play across separate sessions.
- Persistence and recovery after restarting the server.
- Concurrent move submissions and stale game-state updates.
- Live updates: a move and a join reach the other player's stream.
- HTTP endpoint validation.

Game rules should be tested independently of the HTTP server and browser interface.

## 12. Initial Scope and Future Extensions

The initial implementation should prioritize a reliable, complete two-player game with persistent asynchronous rooms.

Potential future extensions include:
- In-room move history and replay.
- Optional player display names.
- Notifications when an opponent makes a move while the page is closed.
- Room expiration and archival policies.
- A local development mode with an in-memory database.
- Administrative diagnostics and structured logging.
- Additional language support, including Mapuzugun terminology.

Future features must not compromise the authoritative server-side game logic or the ability to resume asynchronous games.

## 13. Definition of Done

The initial project is complete when:

1. The application builds into a single Go binary containing the web interface.
2. A browser receives a persistent token identity.
3. A player can create a room and share its unique URL.
4. A second player can join, and both seats become bound to their browser identities.
5. Only the two assigned players can make moves in the room.
6. The server enforces the Komikan rules and turn order.
7. Players can take turns asynchronously, including when the opponent is offline.
8. Room and game state survive page reloads and server restarts.
9. Multiple games can proceed independently.
10. Automated tests cover the critical game, identity, persistence, and concurrency requirements.
