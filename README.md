# flserver

[Русский](README.ru.md) · [Documentation](docs/README.md) · [Changelog](CHANGELOG.md) · [Contributing](CONTRIBUTING.md)

![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)
![Platform](https://img.shields.io/badge/Platform-Windows-0078D6.svg)
![Go](https://img.shields.io/badge/Go-1.24-00ADD8.svg)
![Factorio](https://img.shields.io/badge/Factorio-2.0-orange.svg)

**A dedicated Factorio server for Windows that plays your own saves.** Pick a save from the game's folder, and flserver hosts it with the headless mode of the game you already have installed. When you stop the server, the progress goes back into the same save, so you can open it in the game again.

No separate server download, no Linux, no Docker. One small `flserver.exe`.

## What is inside

- **Your saves, your mods.** It reads the saves and mods from the game's own folders, Space Age and the other expansions included. The server always matches the game's version.
- **Light.** No window, graphics or sound: about 370 MB of memory for a Space Age map, against 4 GB for the game itself. With nobody connected the map is paused.
- **Safe for saves.** A backup before every start, a verified copy of the map to work on, and the original is never overwritten if the game changed it in the meantime.
- **Survives crashes.** If the power goes out or the process is killed, the next start brings the last autosave back into the game.
- **Console and remote control.** Type commands in the server window or send them from another one: `flserver status`, `flserver cmd /players`, `flserver stop`.
- **Optional shutdown when empty** after N minutes without players.
- **English and Russian** messages, chosen from the Windows language.

## Quick start

You need Windows 10 or later, Factorio 2.0 installed (GOG, Steam or the zip version), and Go 1.24 or later to build.

```powershell
git clone https://github.com/ospab/factorio-lightserver
cd factorio-lightserver
go build -trimpath -ldflags "-s -w" -o flserver.exe .
.\flserver.exe
```

It lists your saves, newest first. Press Enter for the newest or type a number. When it says the server is up, connect from the game: **Multiplayer → Connect to address**, `localhost`, or the address it prints for your network.

Type `stop` (or press Ctrl+C) to save the map and exit. Close the map first if the game itself is hosting it: both cannot use UDP port 34197.

```powershell
.\flserver.exe start my-factory      # a save by name
.\flserver.exe list                  # the saves
.\flserver.exe status                # from another window
.\flserver.exe stop
```

## Documentation

| | |
|---|---|
| [Getting started](docs/en/getting-started.md) | requirements, build, first start, connecting |
| [Usage](docs/en/usage.md) | every command, the server console, running in the background |
| [Configuration](docs/en/configuration.md) | `flserver.json`, `server-settings.json`, admins, firewall and ports |
| [Saves](docs/en/saves.md) | what happens to a save: backups, returning progress, crash recovery |
| [How it works](docs/en/architecture.md) | the design, and why the game's engine is still needed |
| [Troubleshooting](docs/en/troubleshooting.md) | messages and what to do about them |

## What it is not

It is not a reimplementation of Factorio. The map is simulated by `factorio.exe` in its headless mode, as with the official server, so the memory use is that of the engine and not of a lighter program. flserver is the part around it: choosing the save, keeping it safe, and controlling the process. Only Windows is supported, and it has been tested on Factorio 2.0.72 with Space Age.

## Layout

| Path | What |
|---|---|
| `main.go` | commands, the save list, remote control |
| `server.go` | preparing and running the game, the console, shutdown |
| `saves.go` | save checks, backups, returning progress, mod sync |
| `config.go` | `flserver.json`, finding the game, session state |
| `rcon.go` | the RCON client |
| `windows.go`, `lang.go` | Windows calls, language |
| `docs/` | documentation in English (`en/`) and Russian (`ru/`) |

## License

GNU Affero General Public License v3.0 only, see [LICENSE](LICENSE). Factorio is a trademark of Wube Software; this project contains none of its files and needs your own copy of the game.
