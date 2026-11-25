# Troubleshooting

[Русский](../ru/troubleshooting.md) · [Docs](../README.md)

When the server does not start, the answer is usually in `data\factorio-current.log`, the game's own log, or in `flserver start -v SAVE`, which prints it.

## Messages

**`Factorio was not found, set game_dir in flserver.json`**
The game was not found automatically. Put the folder that contains `bin\x64\factorio.exe` into `game_dir`, for example `"D:/Games/Factorio"`.

**`UDP port 34197 is taken (is the game hosting a map?)`**
Something else listens on the port. Most often it is Factorio itself hosting a map, or a server you started earlier. Close the map in the game, run `flserver status`, or set another `port`.

**`only N MB free on C:, at least 512 MB needed`**
The drive with `data\` or with your saves is almost full. Factorio cannot write a save to a full disk, and flserver refuses to start a session that could not be saved. Free some space. (`data\backups` is the first thing to clean.)

**`... is not a Factorio save`, `zip: not a valid zip file`, or a file name with a checksum error**
The save is damaged or is not a save. Start another one; a backup of the same save is in `data\backups`.

**`the server is already running, stop it with: flserver stop`**
A server started by this copy of flserver is still alive. If you cannot find its window, run `flserver stop`.

**`The last session did not end cleanly, recovering its progress...`**
Not an error. The previous run was interrupted and flserver is returning its progress to the game, see [Saves](saves.md#crash-recovery).

**`could not return the save to the game: ...`**
The copy back failed, most often a full disk or a save open in another program. The progress is safe in `data\`. Fix the cause and start flserver again; it retries first thing.

**`unpacked mod is not copied: NAME`**
Mods that are folders in the mods directory are skipped. Zip the folder (it must be named `NAME_VERSION.zip` and have a single `NAME_VERSION` folder inside) or the server will not have that mod and a save that needs it will not load.

**`the server did not start, see ...\factorio-current.log`**
The game exited during loading. Look for lines with `Error` in the log: a mod that fails to load, a mod missing from `data\mods`, or a save from a newer game version are the usual reasons.

## Players cannot connect

- **The same version.** The game of each player must be the same version as the one flserver runs (it prints it at the start), with the same mods.
- **Firewall.** Allow `factorio.exe` through the Windows firewall for the network type you use.
- **Over the internet.** UDP `34197` must be forwarded on the router to this computer, and your provider must give you a public address (not CGNAT). Check from outside the network.
- **Wrong address.** `localhost` works only on the same computer. Use one of the addresses the server prints.
- **Password.** If `game_password` is set, players are asked for it.

## Other questions

**`factorio.exe` is in the Task Manager.**
That is the server: flserver runs the game's engine. It should be 340 to 400 MB for a map like the test one and use no graphics memory.

**The server saved, but the progress is not in my save.**
Read the last message. If the original changed or is an autosave, the progress is in `NAME-server-DATE.zip` next to it, see [Saves](saves.md#when-the-original-is-not-overwritten).

**I want a new map, not a save.**
Create it in the game and save it, then start it with flserver. Creating maps is not part of flserver.

**The window closes at once after an error.**
Run flserver from a PowerShell window instead of double clicking, or read the message after pressing Enter: when started from Explorer it waits.

**Where are the logs?**
`data\factorio-current.log` is the game's log of the last start, `factorio-previous.log` the one before it.
