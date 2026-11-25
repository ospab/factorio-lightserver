# Getting started

[Русский](../ru/getting-started.md) · [Docs](../README.md)

## What you need

| | |
|---|---|
| Windows 10 or later | flserver calls Windows directly (process, registry and network tables). Linux and macOS are not supported. |
| Factorio 2.0, installed | GOG, Steam, or the zip version. The server runs `bin\x64\factorio.exe` from it, so the server and your game are always the same version. Tested with 2.0.72 and Space Age. |
| Go 1.24 or later | Only to build `flserver.exe`; it has no dependencies besides the standard library. |
| About 0.5 GB of free disk space | On the drive with flserver and on the drive with your saves. It refuses to start with less, because Factorio cannot write a save to a full disk. |
| About 400 MB of memory | For a Space Age map. Larger factories need more. |

## Build

```powershell
git clone https://github.com/ospab/factorio-lightserver
cd factorio-lightserver
go build -trimpath -ldflags "-s -w" -o flserver.exe .
```

`flserver.exe` can then live anywhere. It keeps its settings and working files next to itself (`flserver.json` and the `data\` folder).

## First start

```powershell
.\flserver.exe
```

On the first run it looks for the game (GOG and Steam registry entries, Steam libraries, `Program Files`) and writes what it found into `flserver.json`. If it finds nothing, it says so: set `game_dir` there to the folder that contains `bin\x64\factorio.exe` and start again.

Then it lists the saves, newest first:

```
  1. podw                                          23.11.2026 20:59    6.7 MB
  2. _autosave3                                    23.11.2026 16:47    6.8 MB
```

Press Enter for the first one or type a number. A few seconds later:

```
Server is up in 2.1 s, 372 MB of memory.
Connect to: localhost:34197, 192.168.1.20:34197
```

Windows may ask whether `factorio.exe` may use the network. For friends on your network allow private networks; for friends over the internet see [Configuration](configuration.md#ports-and-the-firewall).

## Connect

In the game: **Multiplayer → Connect to address**, then `localhost:34197` on the same computer or one of the printed addresses from another one. Everybody needs the same game version and the same mods as the server.

If the game itself is hosting a map, close that first: it holds UDP port 34197, and flserver will say that the port is taken.

## Stop

Type `stop` in the server window, or press Ctrl+C, or close the window. The server saves the map and the progress goes back into the save you chose, see [Saves](saves.md). From another window:

```powershell
.\flserver.exe stop
```

## Next

- [Usage](usage.md): all commands and the console.
- [Configuration](configuration.md): port, passwords, autosave, shutdown when empty.
