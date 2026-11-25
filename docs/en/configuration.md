# Configuration

[Русский](../ru/configuration.md) · [Docs](../README.md)

flserver has two settings files. `flserver.json` is flserver's own and sits next to `flserver.exe`. `data\server-settings.json` is Factorio's server settings.

## flserver.json

Created on the first start with the values found on your machine. Edit it with any editor; it is read on every start.

| Field | Default | Meaning |
|---|---|---|
| `game_dir` | found automatically | The folder of the game, the one that contains `bin\x64\factorio.exe`. |
| `saves_dir` | `%APPDATA%\Factorio\saves` | Where the saves are read from and where progress goes back to. For the zip version of the game, its own `saves` folder. |
| `mods_dir` | `%APPDATA%\Factorio\mods` | Where mods and their settings are copied from. |
| `save` | empty | The save started by a plain `flserver`. Empty means ask. |
| `port` | `34197` | The UDP port of the game server. |
| `bind` | empty | The address to listen on. Empty means all interfaces; `127.0.0.1` makes the server reachable from this computer only. |
| `sync_back` | `true` | `true`: progress goes back into the save you started. `false`: it always goes into a new file next to it. |
| `backups` | `10` | How many backup copies of each save to keep in `data\backups`. At least 1. |
| `idle_stop_minutes` | `0` | Stop the server after this many minutes with no players. `0` means never. |
| `lang` | `auto` | Language of the messages: `en`, `ru`, or `auto` to follow the Windows language. |

Use forward slashes or doubled backslashes in paths: `"D:/Games/Factorio"` or `"D:\\Games\\Factorio"`.

## The data folder

Everything flserver makes while running is in `data\` next to the program, so you can delete it to start from scratch. Your saves and mods are never changed except as described in [Saves](saves.md).

| Path | What |
|---|---|
| `world.zip` | The copy of the save the server works on. |
| `saves\` | The server's own autosaves. They do not clutter the game's folder. |
| `backups\` | A copy of the save made before every start. |
| `mods\` | The copy of your mods. |
| `server-settings.json` | The server settings, see below. Created once, then yours to edit. |
| `server-adminlist.json` | The admins. Copied from the game on the first start, then yours to edit. |
| `config.ini` | Makes the game read its data from `game_dir` and write into `data\`. Rewritten on every start. |
| `session.json` | Marks a running or interrupted session. Removed when the progress is back in the game. |
| `factorio-current.log` | The game's log. The place to look when the server does not start. |

## server-settings.json

This is the file the game documents as `server-settings.example.json` in its `data` folder; every field is described there. The one flserver creates differs from the example in these places:

| Field | Value | Why |
|---|---|---|
| `name` | `Factorio server` | Change it. |
| `visibility` | `public: false`, `lan: true` | Not listed on the public server list, announced on the local network. A public listing needs `username` and `token` from factorio.com. |
| `require_user_verification` | `false` | Anybody can join under any name. See below. |
| `autosave_interval`, `autosave_slots` | `5`, `3` | An autosave every 5 minutes into the server's `saves\` folder. These are what [crash recovery](saves.md#crash-recovery) uses. |
| `auto_pause` | `true` | The map is paused while nobody is connected. |
| `allow_commands` | `admins-only` | Only admins can use `/` commands in the game. |

Changes apply on the next start.

### Passwords and who can join

With `require_user_verification: false` the server does not check names with factorio.com, so anybody who can reach the port can join under any name, an admin's name too. For anything beyond your own network do one of these:

- set `game_password`, a password that players enter to join;
- set `require_user_verification` to `true`, so that only real factorio.com accounts can use a name.

### Admins

The first time, flserver copies `server-adminlist.json` from the game's user folder (the folder above `saves_dir`). Afterwards edit `data\server-adminlist.json`, or use `/promote NAME` in the console. The list is a JSON array of names:

```json
["ospab"]
```

## Mods

On every start flserver makes `data\mods` a copy of `mods_dir`: the mod `.zip` files, `mod-list.json` and `mod-settings.dat`. Mods that were removed or disabled in the game go away from the server too. Unpacked mods (folders) are not copied; flserver prints a warning for each. The expansions (Space Age, Quality, Elevated Rails) are part of the game itself and are enabled by `mod-list.json`.

Players need the same mods as the server, in the same versions. The game downloads missing mods from the server when it can.

## Ports and the firewall

The server needs UDP port `34197` (or your `port`).

- **Same computer or same network.** When Windows asks about `factorio.exe`, allow private networks.
- **Over the internet.** Forward UDP `34197` on your router to the computer and allow `factorio.exe` through the Windows firewall. Set a `game_password` first.
- The server also opens an RCON port, but only on `127.0.0.1`, on a free port, with a new random password on every start. Nothing outside the computer can reach it; flserver uses it for the console and for `stop`.
