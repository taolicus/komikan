-- Schema for the Komikan store. Applied at startup and tracked with
-- PRAGMA user_version.

create table if not exists players (
    id         text primary key,
    token_hash text not null unique,
    created_at integer not null
);

create table if not exists rooms (
    id         text primary key,
    created_at integer not null,
    updated_at integer not null,
    status     text not null,
    ruleset    text not null default '{}'
);

create table if not exists seats (
    room_id   text not null references rooms(id),
    seat      integer not null,
    player_id text not null references players(id),
    role      text not null default '',
    primary key (room_id, seat)
);

create index if not exists seats_player_idx on seats(player_id);

create table if not exists games (
    room_id    text primary key references rooms(id),
    state      text not null,
    version    integer not null,
    status     text not null,
    winner     text not null default '',
    updated_at integer not null
);

create table if not exists moves (
    room_id    text not null references rooms(id),
    seq        integer not null,
    player_id  text not null,
    payload    text not null,
    created_at integer not null,
    primary key (room_id, seq)
);
