# Usage

[Русский](../ru/usage.md) · [Docs](../README.md)

## Commands

| Command | What it does |
|---|---|
| `flserver` | Same as `flserver start` with no name: shows the list and asks for a save. |
| `flserver start [SAVE]` | Starts a save. `SAVE` is a name from the list (with or without `.zip`) or a full path to a `.zip`. With no name it uses `save` from `flserver.json`, and if that is empty, shows the list. |
| `flserver start -v SAVE` | The same, with the game's whole log in the console instead of only the events. |
| `flserver list` | Lists the saves in `saves_dir`, newest first. |
| `flserver status` | From another window: how long the server has been up, which save, how many players, memory. |
| `flserver cmd TEXT` | From another window: runs a command on the server, for example `flserver cmd /players online`. |
| `flserver stop` | From another window: saves the map and stops the server. |
| `flserver help` | The same list of commands. |

The exit code is 0 on success and 1 on an error. Messages are in English or Russian, see `lang` in [Configuration](configuration.md#flserverjson).

Only one server runs at a time. A second `start` says that the server is already running.

## Choosing a save

The list shows the file name without `.zip`, the modified time and the size. The game's autosaves (`_autosave1` and so on) are in it too. You can start one of them, but flserver never writes progress back over an autosave: see [Saves](saves.md#when-the-original-is-not-overwritten).

```powershell
.\flserver.exe start podw
.\flserver.exe start "C:\Backups\old factory.zip"
```

## The server console

While the server runs, its window is a console:

| You type | What happens |
|---|---|
| `stop`, `quit`, `exit` | Saves the map and stops. Ctrl+C and closing the window do the same. |
| `/players`, `/players online`, `/time`, `/evolution` | Factorio commands, the answer is printed. |
| `/server-save` | Saves the map now, in addition to the autosaves. |
| `/promote NAME`, `/demote NAME` | Makes a player an admin or takes it away. |
| `/kick NAME`, `/ban NAME` | Moves a player out. |
| other text | Appears in the game's chat as a message from `<server>`. |

The console shows joins, leaves, chat, kicks, bans, errors and the end of each save. `-v` shows everything.

## From another window

`status`, `cmd` and `stop` find the running server through `data\session.json`, where flserver keeps the address and password of the server's local RCON port. So they work from any PowerShell window on the same computer, as the same Windows user:

```powershell
.\flserver.exe status
.\flserver.exe cmd /players online
.\flserver.exe cmd "Server restarts in 5 minutes"
.\flserver.exe stop
```

`stop` waits until the server has exited and the progress is back in the save. If the window that started the server is gone, `stop` returns the progress itself.

## Closing the window

Closing the console window is treated as `stop`. Windows gives a program only a few seconds to react to that, so for a big map the save may not finish. Nothing is lost then: the next start finds the last autosave and returns it to the game (see [crash recovery](saves.md#crash-recovery)). Prefer `stop`.

## Double click

Starting `flserver.exe` from Explorer opens a window with the list. When the server stops or an error happens, the window waits for Enter so that you can read the last lines.

## Running without a window

With a save name given, no input is needed, so a minimised window or Task Scheduler works:

```powershell
Start-Process .\flserver.exe -ArgumentList 'start','podw' -WindowStyle Minimized
```

Stop it later with `.\flserver.exe stop`. For a scheduled task set "Start in" to the folder with `flserver.exe`, and add `idle_stop_minutes` to `flserver.json` if the server should shut itself down when empty.
