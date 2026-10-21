package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// schemaVersion is the version of the schema this build expects. It is
// stored in PRAGMA user_version.
const schemaVersion = 1

// sqliteStore is the SQLite implementation of Store.
type sqliteStore struct {
	db *sql.DB
}

// NewID returns a random identifier with 128 bits of entropy, as hex.
func NewID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand does not fail in practice; a predictable id would be
		// worse than a crash.
		panic("store: cannot read random bytes: " + err.Error())
	}
	return hex.EncodeToString(buf)
}

// Open opens the database at path, creating and migrating it when needed.
// An empty path opens a private in-memory database.
func Open(path string) (Store, error) {
	dsn := path
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open database: %w", err)
	}

	// SQLite allows one writer at a time. Serialising every statement
	// through a single connection keeps the database happy and makes the
	// version check in ApplyMove meaningful.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(0)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("store: %s: %w", pragma, err)
		}
	}

	s := &sqliteStore{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// migrate applies the schema when the database is new or older.
func (s *sqliteStore) migrate(ctx context.Context) error {
	var version int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("store: read schema version: %w", err)
	}
	if version == schemaVersion {
		return nil
	}
	if version > schemaVersion {
		return fmt.Errorf("store: database schema version %d is newer than %d", version, schemaVersion)
	}

	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("store: apply schema: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version=%d", schemaVersion)); err != nil {
		return fmt.Errorf("store: set schema version: %w", err)
	}
	return nil
}

// Close releases the database.
func (s *sqliteStore) Close() error {
	return s.db.Close()
}

// CreatePlayer records a new browser identity.
func (s *sqliteStore) CreatePlayer(ctx context.Context, tokenHash string) (Player, error) {
	if tokenHash == "" {
		return Player{}, fmt.Errorf("%w: empty token hash", ErrInvalid)
	}
	p := Player{
		ID:        NewID(),
		TokenHash: tokenHash,
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.db.ExecContext(ctx,
		"insert into players (id, token_hash, created_at) values (?, ?, ?)",
		p.ID, p.TokenHash, p.CreatedAt.UnixMilli())
	if err != nil {
		return Player{}, fmt.Errorf("store: create player: %w", err)
	}
	return p, nil
}

// PlayerByTokenHash finds the identity behind a token hash.
func (s *sqliteStore) PlayerByTokenHash(ctx context.Context, tokenHash string) (Player, error) {
	var (
		p         Player
		createdAt int64
	)
	err := s.db.QueryRowContext(ctx,
		"select id, token_hash, created_at from players where token_hash = ?", tokenHash).
		Scan(&p.ID, &p.TokenHash, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Player{}, fmt.Errorf("%w: player", ErrNotFound)
	}
	if err != nil {
		return Player{}, fmt.Errorf("store: load player: %w", err)
	}
	p.CreatedAt = time.UnixMilli(createdAt).UTC()
	return p, nil
}

// CreateRoom creates a room with one seat for the creator.
func (s *sqliteStore) CreateRoom(ctx context.Context, roomID, creatorID string, ruleset State, initial State) (Room, error) {
	if roomID == "" || creatorID == "" {
		return Room{}, fmt.Errorf("%w: room needs an id and a creator", ErrInvalid)
	}
	if len(ruleset) == 0 {
		ruleset = State("{}")
	}
	if len(initial) == 0 {
		initial = State("{}")
	}

	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Room{}, fmt.Errorf("store: begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"insert into rooms (id, created_at, updated_at, status, ruleset) values (?, ?, ?, ?, ?)",
		roomID, now.UnixMilli(), now.UnixMilli(), string(RoomWaiting), string(ruleset)); err != nil {
		return Room{}, fmt.Errorf("store: create room: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"insert into seats (room_id, seat, player_id, role) values (?, 1, ?, '')",
		roomID, creatorID); err != nil {
		return Room{}, fmt.Errorf("store: seat creator: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"insert into games (room_id, state, version, status, winner, updated_at) values (?, ?, 0, ?, '', ?)",
		roomID, string(initial), "IN_PROGRESS", now.UnixMilli()); err != nil {
		return Room{}, fmt.Errorf("store: create game: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Room{}, fmt.Errorf("store: commit room: %w", err)
	}
	return s.GetRoom(ctx, roomID)
}

// GetRoom returns a room with its seats.
func (s *sqliteStore) GetRoom(ctx context.Context, roomID string) (Room, error) {
	var (
		room      Room
		createdAt int64
		updatedAt int64
		ruleset   string
	)
	err := s.db.QueryRowContext(ctx,
		"select id, created_at, updated_at, status, ruleset from rooms where id = ?", roomID).
		Scan(&room.ID, &createdAt, &updatedAt, &room.Status, &ruleset)
	if errors.Is(err, sql.ErrNoRows) {
		return Room{}, fmt.Errorf("%w: room %s", ErrNotFound, roomID)
	}
	if err != nil {
		return Room{}, fmt.Errorf("store: load room: %w", err)
	}
	room.CreatedAt = time.UnixMilli(createdAt).UTC()
	room.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	room.Ruleset = State(ruleset)

	seats, err := s.seats(ctx, roomID)
	if err != nil {
		return Room{}, err
	}
	room.Seats = seats
	return room, nil
}

// seats returns the seats of a room in seat order.
func (s *sqliteStore) seats(ctx context.Context, roomID string) ([]Seat, error) {
	rows, err := s.db.QueryContext(ctx,
		"select room_id, seat, player_id, role from seats where room_id = ? order by seat", roomID)
	if err != nil {
		return nil, fmt.Errorf("store: load seats: %w", err)
	}
	defer rows.Close()

	var seats []Seat
	for rows.Next() {
		var (
			seat Seat
			role string
		)
		if err := rows.Scan(&seat.RoomID, &seat.Seat, &seat.PlayerID, &role); err != nil {
			return nil, fmt.Errorf("store: scan seat: %w", err)
		}
		seat.Role = Role(role)
		seats = append(seats, seat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: load seats: %w", err)
	}
	return seats, nil
}

// RoomsForPlayer lists the rooms a player is seated in.
func (s *sqliteStore) RoomsForPlayer(ctx context.Context, playerID string) ([]Room, error) {
	rows, err := s.db.QueryContext(ctx,
		`select r.id from rooms r join seats s on s.room_id = r.id where s.player_id = ? order by r.created_at`, playerID)
	if err != nil {
		return nil, fmt.Errorf("store: list rooms: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: scan room id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list rooms: %w", err)
	}

	rooms := make([]Room, 0, len(ids))
	for _, id := range ids {
		room, err := s.GetRoom(ctx, id)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

// JoinRoom seats a player in the free seat, or returns their existing seat.
// Roles are assigned at random once both seats are taken.
func (s *sqliteStore) JoinRoom(ctx context.Context, roomID, playerID string) (Room, error) {
	if playerID == "" {
		return Room{}, fmt.Errorf("%w: join needs a player", ErrInvalid)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Room{}, fmt.Errorf("store: begin transaction: %w", err)
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRowContext(ctx, "select status from rooms where id = ?", roomID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return Room{}, fmt.Errorf("%w: room %s", ErrNotFound, roomID)
	}
	if err != nil {
		return Room{}, fmt.Errorf("store: load room: %w", err)
	}
	if status == string(RoomFinished) {
		return Room{}, fmt.Errorf("%w: room %s is finished", ErrConflict, roomID)
	}

	rows, err := tx.QueryContext(ctx,
		"select seat, player_id, role from seats where room_id = ? order by seat", roomID)
	if err != nil {
		return Room{}, fmt.Errorf("store: load seats: %w", err)
	}
	var seats []Seat
	for rows.Next() {
		var (
			seat Seat
			role string
		)
		if err := rows.Scan(&seat.Seat, &seat.PlayerID, &role); err != nil {
			rows.Close()
			return Room{}, fmt.Errorf("store: scan seat: %w", err)
		}
		seat.RoomID = roomID
		seat.Role = Role(role)
		seats = append(seats, seat)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Room{}, fmt.Errorf("store: load seats: %w", err)
	}

	// Already seated: the player keeps their seat.
	for _, seat := range seats {
		if seat.PlayerID == playerID {
			if err := tx.Commit(); err != nil {
				return Room{}, fmt.Errorf("store: commit join: %w", err)
			}
			return s.GetRoom(ctx, roomID)
		}
	}

	if len(seats) >= 2 {
		return Room{}, fmt.Errorf("%w: room %s is full", ErrConflict, roomID)
	}

	free := 1
	if len(seats) == 1 {
		free = 2
	}
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx,
		"insert into seats (room_id, seat, player_id, role) values (?, ?, ?, '')",
		roomID, free, playerID); err != nil {
		return Room{}, fmt.Errorf("store: take seat: %w", err)
	}

	// The room is full now: assign the roles at random.
	if free == 2 {
		roles, err := randomRoles()
		if err != nil {
			return Room{}, err
		}
		if _, err := tx.ExecContext(ctx,
			"update seats set role = ? where room_id = ? and seat = 1", roles[0], roomID); err != nil {
			return Room{}, fmt.Errorf("store: assign role: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			"update seats set role = ? where room_id = ? and seat = 2", roles[1], roomID); err != nil {
			return Room{}, fmt.Errorf("store: assign role: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			"update rooms set status = ?, updated_at = ? where id = ?",
			string(RoomActive), now.UnixMilli(), roomID); err != nil {
			return Room{}, fmt.Errorf("store: activate room: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			"update games set status = ?, updated_at = ? where room_id = ?",
			"IN_PROGRESS", now.UnixMilli(), roomID); err != nil {
			return Room{}, fmt.Errorf("store: activate game: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Room{}, fmt.Errorf("store: commit join: %w", err)
	}
	return s.GetRoom(ctx, roomID)
}

// randomRoles assigns the two roles at random.
func randomRoles() ([2]Role, error) {
	var b [1]byte
	if _, err := rand.Read(b[:]); err != nil {
		return [2]Role{}, fmt.Errorf("store: cannot read random bytes: %w", err)
	}
	if b[0]&1 == 0 {
		return [2]Role{RolePangui, RoleTrewa}, nil
	}
	return [2]Role{RoleTrewa, RolePangui}, nil
}

// Game returns the game state of a room.
func (s *sqliteStore) Game(ctx context.Context, roomID string) (Game, error) {
	var (
		g         Game
		state     string
		updatedAt int64
	)
	err := s.db.QueryRowContext(ctx,
		"select room_id, state, version, status, winner, updated_at from games where room_id = ?", roomID).
		Scan(&g.RoomID, &state, &g.Version, &g.Status, &g.Winner, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Game{}, fmt.Errorf("%w: game for room %s", ErrNotFound, roomID)
	}
	if err != nil {
		return Game{}, fmt.Errorf("store: load game: %w", err)
	}
	g.State = State(state)
	g.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return g, nil
}

// ApplyMove stores a move and the resulting state atomically. It fails with
// ErrConflict when the expected version is stale, so two concurrent
// submissions for the same turn cannot both succeed.
func (s *sqliteStore) ApplyMove(ctx context.Context, roomID string, expectedVersion int, playerID string, payload State, result GameResult) (Game, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Game{}, fmt.Errorf("store: begin transaction: %w", err)
	}
	defer tx.Rollback()

	var version int
	err = tx.QueryRowContext(ctx, "select version from games where room_id = ?", roomID).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return Game{}, fmt.Errorf("%w: game for room %s", ErrNotFound, roomID)
	}
	if err != nil {
		return Game{}, fmt.Errorf("store: load game: %w", err)
	}
	if version != expectedVersion {
		return Game{}, fmt.Errorf("%w: room %s is at version %d, not %d", ErrConflict, roomID, version, expectedVersion)
	}

	now := time.Now().UTC()
	res, err := tx.ExecContext(ctx,
		"update games set state = ?, version = version + 1, status = ?, winner = ?, updated_at = ? where room_id = ? and version = ?",
		string(result.State), result.Status, result.Winner, now.UnixMilli(), roomID, expectedVersion)
	if err != nil {
		return Game{}, fmt.Errorf("store: update game: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return Game{}, fmt.Errorf("store: update game: %w", err)
	}
	if affected == 0 {
		return Game{}, fmt.Errorf("%w: room %s changed under us", ErrConflict, roomID)
	}

	if _, err := tx.ExecContext(ctx,
		"insert into moves (room_id, seq, player_id, payload, created_at) values (?, ?, ?, ?, ?)",
		roomID, expectedVersion+1, playerID, string(payload), now.UnixMilli()); err != nil {
		return Game{}, fmt.Errorf("store: record move: %w", err)
	}

	if result.Finished {
		if _, err := tx.ExecContext(ctx,
			"update rooms set status = ?, updated_at = ? where id = ?",
			string(RoomFinished), now.UnixMilli(), roomID); err != nil {
			return Game{}, fmt.Errorf("store: finish room: %w", err)
		}
	} else {
		if _, err := tx.ExecContext(ctx,
			"update rooms set updated_at = ? where id = ?", now.UnixMilli(), roomID); err != nil {
			return Game{}, fmt.Errorf("store: touch room: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Game{}, fmt.Errorf("store: commit move: %w", err)
	}
	return s.Game(ctx, roomID)
}

// Moves lists the moves of a room in order.
func (s *sqliteStore) Moves(ctx context.Context, roomID string) ([]Move, error) {
	rows, err := s.db.QueryContext(ctx,
		"select room_id, seq, player_id, payload, created_at from moves where room_id = ? order by seq", roomID)
	if err != nil {
		return nil, fmt.Errorf("store: list moves: %w", err)
	}
	defer rows.Close()

	var moves []Move
	for rows.Next() {
		var (
			m         Move
			payload   string
			createdAt int64
		)
		if err := rows.Scan(&m.RoomID, &m.Seq, &m.PlayerID, &payload, &createdAt); err != nil {
			return nil, fmt.Errorf("store: scan move: %w", err)
		}
		m.Payload = State(payload)
		m.CreatedAt = time.UnixMilli(createdAt).UTC()
		moves = append(moves, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list moves: %w", err)
	}
	return moves, nil
}
