# Komikan on the VPS: runbook

What runs where, and what to do when something needs looking after. The
general deployment pattern of the host is in
`features/tao-vps-go-service-deployment.md`; this file is the Komikan
specific part.

## The pieces

| What | Where |
|---|---|
| binary | `/home/dev/komikan/komikan`, root-owned, replaced by rename |
| service | `komikan.service`, runs as `dev`, listens on `127.0.0.1:18082` |
| unit file | `deploy/komikan.service` in this repository, copied to `/etc/systemd/system/komikan.service` |
| database | `/home/dev/komikan/data/komikan.db`, with its `-wal` and `-shm` files |
| backups | `/home/dev/komikan/backups/` |
| public URL | `https://komikan.tequionet.org` (nginx proxies `/` to `127.0.0.1:18082`) |
| pages | `/` is the home page, `/rooms/` is the rooms and the game |

The unit sets `ADDR` and `DB_PATH` explicitly, and `UMask=0027` so the
database and its write-ahead log are readable by the service user and by
nobody else.

## Deploying

```sh
sh deploy.sh
```

From the phone. It cross-builds the binary for linux/amd64, copies it to the
VPS by rename, restarts the service, and health-checks the public URL. The
commit that is deployed is stamped into the binary and shows up in the
service log.

## Checking that it is alive

```sh
curl -fsS https://komikan.tequionet.org/healthz   # -> ok
ssh tao 'systemctl status komikan'
ssh tao 'journalctl -u komikan -n 50 --no-pager'
```

The log says which build is running:

```
msg="komikan: listening" addr=127.0.0.1:18082 version=9fb8391
```

## Restarting

```sh
ssh tao 'systemctl restart komikan'
```

The rooms, the seats and the games are in the database, so a restart does not
lose a game. The browser keeps its seat because the identity is the session
cookie, which points at a row in the database.

The service is `enabled`, so it comes back on its own after a reboot.

## The database

One SQLite file in WAL mode. Nothing else holds state, so the file is the
whole game.

Back it up by stopping the service, copying the file, and starting it again:

```sh
ssh tao 'systemctl stop komikan && \
  cp -a /home/dev/komikan/data/komikan.db \
        /home/dev/komikan/backups/komikan-$(date +%Y%m%d-%H%M%S).db && \
  systemctl start komikan'
```

The service is down for the length of the copy, which is a fraction of a
second for a few megabytes. If the `sqlite3` command line tool is installed
on the host, `sqlite3 /home/dev/komikan/data/komikan.db ".backup '/tmp/k.db'"`
takes the same copy without stopping anything.

To restore, stop the service, put the copy back, and start it again:

```sh
ssh tao 'systemctl stop komikan && \
  cp -a /home/dev/komikan/backups/komikan-20261021-122336.db \
        /home/dev/komikan/data/komikan.db && \
  rm -f /home/dev/komikan/data/komikan.db-wal /home/dev/komikan/data/komikan.db-shm && \
  systemctl start komikan'
```

The write-ahead log belongs to the copy that was taken, so it is removed
with the restore; the database replays what is missing from its own file.

## When something is wrong

- **The site does not answer.** `systemctl status komikan` and
  `journalctl -u komikan`. The binary is replaced by rename, so a restart
  never runs a half-written file.
- **The rooms are empty.** The service is looking at the wrong file: check
  `DB_PATH` in the unit against the path in the log line
  `msg="komikan: database ready"`.
- **A game is stuck.** The state is in the `games` row of the room and can be
  read with any SQLite client; the API shows the same thing to the players.
