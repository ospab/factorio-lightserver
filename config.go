package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type config struct {
	GameDir         string `json:"game_dir"`
	SavesDir        string `json:"saves_dir"`
	ModsDir         string `json:"mods_dir"`
	Save            string `json:"save"`
	Port            int    `json:"port"`
	Bind            string `json:"bind"`
	SyncBack        bool   `json:"sync_back"`
	Backups         int    `json:"backups"`
	IdleStopMinutes int    `json:"idle_stop_minutes"`
	Lang            string `json:"lang"`

	root string
}

// session describes a running server, or one that was cut short.
type session struct {
	PID        int       `json:"pid"`
	RconPort   int       `json:"rcon_port"`
	RconPass   string    `json:"rcon_password"`
	Origin     string    `json:"origin"`
	OriginSize int64     `json:"origin_size"`
	OriginMod  time.Time `json:"origin_mtime"`
	Started    time.Time `json:"started"`
}

func (c config) data(parts ...string) string {
	return filepath.Join(append([]string{c.root, "data"}, parts...)...)
}

func (c config) exe() string { return gameExe(c.GameDir) }

func gameExe(dir string) string { return filepath.Join(dir, "bin", "x64", "factorio.exe") }

func loadConfig() (config, error) {
	self, err := os.Executable()
	if err != nil {
		return config{}, err
	}
	c := config{
		Port:     34197,
		SyncBack: true,
		Backups:  10,
		Lang:     "auto",
		root:     filepath.Dir(self),
	}
	path := filepath.Join(c.root, "flserver.json")
	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		c.GameDir = findGame()
		c.SavesDir, c.ModsDir = userDirs(c.GameDir)
		if err := writeJSON(path, c); err != nil {
			return c, err
		}
	case err != nil:
		return c, err
	default:
		if err := json.Unmarshal(b, &c); err != nil {
			return c, fmt.Errorf("flserver.json: %w", err)
		}
	}
	setLang(c.Lang)
	c.Backups = max(c.Backups, 1)
	return c, os.MkdirAll(c.data(), 0o755)
}

// findGame looks for an installed game: GOG and Steam registry entries, Steam
// libraries, then the usual folders.
func findGame() string {
	const factorioGOG = `SOFTWARE\WOW6432Node\GOG.com\Games\1238653230`
	candidates := []string{readRegistry(factorioGOG, "path")}
	if steam := readRegistry(`SOFTWARE\WOW6432Node\Valve\Steam`, "InstallPath"); steam != "" {
		libs := []string{steam}
		if b, err := os.ReadFile(filepath.Join(steam, "steamapps", "libraryfolders.vdf")); err == nil {
			libs = append(libs, steamLibraries(string(b))...)
		}
		for _, l := range libs {
			candidates = append(candidates, filepath.Join(l, "steamapps", "common", "Factorio"))
		}
	}
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if dir := os.Getenv(env); dir != "" {
			candidates = append(candidates, filepath.Join(dir, "Factorio"))
		}
	}
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		if _, err := os.Stat(gameExe(dir)); err == nil {
			return filepath.Clean(dir)
		}
	}
	return ""
}

var vdfPath = regexp.MustCompile(`"path"\s+"([^"]+)"`)

func steamLibraries(vdf string) []string {
	var out []string
	for _, m := range vdfPath.FindAllStringSubmatch(vdf, -1) {
		out = append(out, strings.ReplaceAll(m[1], `\\`, `\`))
	}
	return out
}

// userDirs returns the saves and mods folders. The zip version of the game
// keeps them next to itself, the installed ones in %APPDATA%.
func userDirs(game string) (saves, mods string) {
	base := filepath.Join(os.Getenv("APPDATA"), "Factorio")
	if game != "" {
		_, inGame := os.Stat(filepath.Join(game, "saves"))
		_, inAppData := os.Stat(filepath.Join(base, "saves"))
		if inGame == nil && inAppData != nil {
			base = game
		}
	}
	return filepath.Join(base, "saves"), filepath.Join(base, "mods")
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// loadSession returns ok=false when there is no unfinished session.
func (c config) loadSession() (s session, ok bool, err error) {
	b, err := os.ReadFile(c.data("session.json"))
	if errors.Is(err, os.ErrNotExist) {
		return s, false, nil
	}
	if err == nil {
		err = json.Unmarshal(b, &s)
	}
	if err != nil {
		return s, false, fmt.Errorf("data\\session.json: %w", err)
	}
	return s, true, nil
}

// gameVersion reads the version of the installed game.
func (c config) gameVersion() (string, error) {
	if c.GameDir == "" {
		return "", errors.New(tr("Factorio was not found, set game_dir in flserver.json", "Factorio не найден, укажите game_dir в flserver.json"))
	}
	b, err := os.ReadFile(filepath.Join(c.GameDir, "data", "base", "info.json"))
	if err != nil {
		return "", fmt.Errorf(tr("no game in %s, fix game_dir in flserver.json: %w", "в %s нет игры, поправьте game_dir в flserver.json: %w"), c.GameDir, err)
	}
	var v struct{ Version string }
	if err := json.Unmarshal(b, &v); err != nil {
		return "", err
	}
	return v.Version, nil
}

const defaultServerSettings = `{
  "name": "Factorio server",
  "description": "",
  "tags": [],
  "max_players": 0,
  "visibility": { "public": false, "lan": true },
  "username": "",
  "token": "",
  "game_password": "",
  "require_user_verification": false,
  "max_upload_in_kilobytes_per_second": 0,
  "max_upload_slots": 5,
  "ignore_player_limit_for_returning_players": false,
  "allow_commands": "admins-only",
  "autosave_interval": 5,
  "autosave_slots": 3,
  "afk_autokick_interval": 0,
  "auto_pause": true,
  "only_admins_can_pause_the_game": true,
  "autosave_only_on_server": true
}
`
