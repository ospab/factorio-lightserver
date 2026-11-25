# Saves

[Русский](../ru/saves.md) · [Docs](../README.md)

The point of flserver is that you keep one set of saves, used by both the game and the server. This page says exactly what it does with them.

## What happens on a start

1. The save is checked: it must be a whole zip archive, every file in it must read without errors, and it must contain a map. A damaged save stops the start with a message before anything is touched.
2. The free space on both drives is checked (at least 512 MB).
3. A **backup** is made: `data\backups\NAME.YYYYMMDD-HHMMSS.zip`. The newest `backups` copies of each save are kept (10 by default), older ones are deleted. Only files named like this are ever deleted there.
4. The save is copied to `data\world.zip`. The server works on this copy; your original is not open while the server runs.
5. The mods are synced and the game's server is started on `world.zip`.

Each copy is written to a temporary file and moved into place only when complete, so a full disk or a power loss cannot leave half a file under a real name.

## While it runs

The game's server saves into `data\saves\` every `autosave_interval` minutes (5 by default, 3 slots), and into `world.zip` on `/server-save` and on stop. Your saves folder stays untouched.

## When it stops

flserver takes the newest save that was written during this session and that checks as intact (the autosaves and `world.zip` are candidates) and copies it back.

- **Normal case.** The progress replaces the save you started. Its name stays the same, so the game shows it in the usual list with its new date.
- **A save with no new progress.** If the server saved nothing (for example it was stopped within seconds), nothing is copied and the message says so.

### When the original is not overwritten

The progress goes into a new file next to the original, named `NAME-server-YYYYMMDD-HHMMSS.zip`, and the original is left as it is, if any of these is true:

- the original **changed during the session**: its size or modified time is not what it was at the start (the game, a sync tool or you saved over it);
- the original was **deleted**;
- the original is an **autosave** of the game (`_autosave1`, `_autosave2`, ...). The game cycles these and would overwrite them soon anyway. The new file drops the underscore: `autosave1-server-...zip`;
- `sync_back` is `false`.

The message at the end tells which case it was and where the progress is.

## Crash recovery

While a server runs, `data\session.json` exists. If flserver or the game is killed, the computer loses power or you close the window too fast, the file stays. On the next `flserver start` (and also on `flserver stop`) it notices that the session did not end cleanly, takes the newest intact save from it as above and returns it to the game before anything else happens.

How much can be lost: at most what happened since the last autosave, so up to `autosave_interval` minutes (5 by default). A save that was being written when the crash came is detected as damaged and skipped; the previous autosave is used instead.

If returning the save fails (for example the disk is full), the progress stays in `data\` and the next start tries again. Do not delete `data\` while `session.json` is there.

## Opening the result in the game

It is an ordinary save file. Open it from the game's **Load game** list as always. The server's autosaves in `data\saves\` can be loaded too: copy one into the game's saves folder.

## Summary of the files

| File | Written by | When |
|---|---|---|
| your save | flserver | when the server stops, only in the cases above |
| `NAME-server-DATE.zip` | flserver | when the original must not be overwritten |
| `data\backups\*.zip` | flserver | before every start |
| `data\world.zip` | flserver and the game | the working copy |
| `data\saves\*.zip` | the game's server | autosaves |
