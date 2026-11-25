# Contributing to flserver

[Русский](CONTRIBUTING.ru.md) · [README](README.md) · [Documentation](docs/README.md)

Thank you for helping. The thing that matters most here is that a save is never lost, so changes to how saves are copied, replaced or recovered get the most careful review.

## Most wanted

1. **Bug reports from real use**, with the other Factorio 2.0.x versions and with large or modded maps. Please include the version (`flserver` prints it at the start), what you did, the message, and `data\factorio-current.log`.
2. **Cases where a save could be lost or overwritten.** Tell it privately if you are not sure it is safe to post (an issue with a description and no save file is enough).
3. **Other ways to find the game** (other launchers, other install folders).
4. **Documentation** fixes and translations that read as if written by a person.

## Rules

- **Saves first.** Nothing may write over a user's save except the one path in `publish`, and that path only for a save that is verified, unchanged since the start and not an autosave. A new case needs a test in `saves_test.go`.
- **Copies go through a temporary file** (`copyFile`) so a failed write leaves the target as it was.
- **Do not trust the game's silence.** A check that could not run is an error with the reason, not a pass.
- **Messages in two languages.** User-visible text goes through `tr("english", "русский")`; English is written first.
- **Documents in two languages**, English first: a change to `docs/en/` comes with the same change in `docs/ru/`, and the same for the READMEs.
- The standard library only; a dependency needs a good reason.

## Development

You need Windows, Go 1.24 or later and an installed Factorio 2.0.

```powershell
gofmt -l .          # prints nothing when formatted
go vet ./...
go test ./...
go build -trimpath -ldflags "-s -w" -o flserver.exe .
```

The tests do not need the game. Anything that starts the server needs it: try your change on a **copy** of a save, a different `port` and a `flserver.json` of its own (put the executable in a scratch folder, as it keeps its files next to itself).

## Sending a change

Keep a change to one thing, describe what it does and how you tried it, and say which saves and which game version it was tried with. By sending a change you agree to license it under the [AGPL-3.0](LICENSE).
