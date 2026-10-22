# Back the database up on a schedule

- Date: 2026-10-22
- Area: deployment

## Problem

The runbook in `features/komikan-deployment-runbook.md` has a backup
command, and it was run by hand once. There is no timer and no cron entry,
so the backup exists only if somebody remembers to run it. The old copies
are never pruned either.

Once other people are using the site, the games in it are worth more than a
single copy on the disk: a lost disk is a lost board for everybody who
played.

## Scope

- A systemd timer on the VPS that copies the database into
  `/home/dev/komikan/backups/` on its own, and prunes the old copies so
  the directory does not grow forever.
- The copy has to be safe while the service is running. The database is in
  WAL mode, so the `-wal` file is part of the state: either copy the
  database and its WAL together, or take the copy through the SQLite
  backup API (`sqlite3 ... ".backup ..."`, if the tool is on the host).
  Stopping the service for the length of the copy is the simplest correct
  thing, and it is what the runbook does today.
- The unit files are kept in the repository under `deploy/`, the way
  `deploy/komikan.service` is.
- The runbook says how to see that the backups are running, and how to
  restore one.

## Acceptance criteria

- The timer is enabled and shows up in `systemctl list-timers`.
- A backup file appears in the backups directory without anybody running a
  command.
- A backup restores into a database that opens and still holds the rooms,
  the seats and the positions.
- The old copies are pruned and the directory stays bounded.

## Verification

- `ssh tao 'systemctl list-timers | grep komikan'` shows the timer and the
  next run.
- `ssh tao 'ls -l /home/dev/komikan/backups/'` shows new copies appearing.
- A restore drill, run on the host, with the restored database opened and a
  room checked.

## Notes and open points

- The service is down for the length of the copy. A few megabytes take a
  fraction of a second, which is what the runbook says today; with more
  people the file grows, and the SQLite backup API becomes the better
  answer.
- The backup is a copy of the game, not of the code. The code is in git.
