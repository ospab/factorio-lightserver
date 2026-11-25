# Changelog

[Русский](CHANGELOG.ru.md)

## 0.1.0

First release.

- Starts a save from the game's saves folder on the headless engine of the installed Factorio and returns the progress into the same save when the server stops.
- Backup of the save before every start; the original is never overwritten if it changed during the session, if it is an autosave, or if `sync_back` is off.
- Recovery of an interrupted session on the next start.
- Console, `status`, `cmd` and `stop` through a local RCON connection.
- Mod sync, finding the game from the GOG and Steam entries, shutdown of an empty server, English and Russian messages.
