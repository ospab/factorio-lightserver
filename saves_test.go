package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeSave makes a minimal save whose map data is content.
func writeSave(t *testing.T, path, content string, mod time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	w, err := z.CreateHeader(&zip.FileHeader{Name: "world/level.dat0", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte(content))
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func readSave(t *testing.T, path string) string {
	t.Helper()
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	r, err := z.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var b bytes.Buffer
	b.ReadFrom(r)
	return b.String()
}

// newSession is a session that started an hour ago from a two hour old save.
func newSession(t *testing.T, saveName string) (config, session) {
	t.Helper()
	dir := t.TempDir()
	cfg := config{root: dir, SavesDir: filepath.Join(dir, "game"), SyncBack: true, Backups: 2}
	now := time.Now()
	origin := filepath.Join(cfg.SavesDir, saveName)
	writeSave(t, origin, "original", now.Add(-2*time.Hour))
	fi, err := os.Stat(origin)
	if err != nil {
		t.Fatal(err)
	}
	writeSave(t, cfg.data("world.zip"), "original", now.Add(-2*time.Hour))
	return cfg, session{Origin: origin, OriginSize: fi.Size(), OriginMod: fi.ModTime(), Started: now.Add(-time.Hour)}
}

func TestPublishOverwritesUnchangedOriginal(t *testing.T) {
	cfg, s := newSession(t, "base.zip")
	writeSave(t, cfg.data("world.zip"), "played", time.Now())
	dst, err := publish(cfg, s)
	if err != nil || dst != s.Origin {
		t.Fatalf("publish = %q, %v", dst, err)
	}
	if got := readSave(t, s.Origin); got != "played" {
		t.Fatalf("original holds %q", got)
	}
}

func TestPublishKeepsOriginal(t *testing.T) {
	for _, c := range []struct {
		name, save string
		change     func(*config, session)
	}{
		{"the game saved over it", "base.zip", func(_ *config, s session) { writeSave(t, s.Origin, "original", time.Now()) }},
		{"it is an autosave", "_autosave1.zip", func(*config, session) {}},
		{"sync_back is off", "base.zip", func(c *config, _ session) { c.SyncBack = false }},
		{"it was deleted", "base.zip", func(_ *config, s session) { os.Remove(s.Origin) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			cfg, s := newSession(t, c.save)
			c.change(&cfg, s)
			writeSave(t, cfg.data("world.zip"), "played", time.Now())
			dst, err := publish(cfg, s)
			if err != nil || dst == s.Origin || !strings.Contains(dst, "-server-") {
				t.Fatalf("publish = %q, %v", dst, err)
			}
			if got := readSave(t, dst); got != "played" {
				t.Fatalf("%s holds %q", dst, got)
			}
			if _, err := os.Stat(s.Origin); err == nil && readSave(t, s.Origin) != "original" {
				t.Fatal("original was overwritten")
			}
		})
	}
}

func TestPublishTakesNewestIntactSave(t *testing.T) {
	cfg, s := newSession(t, "base.zip")
	now := time.Now()
	writeSave(t, cfg.data("world.zip"), "older", now.Add(-30*time.Minute))
	writeSave(t, cfg.data("saves", "_autosave1.zip"), "newer", now.Add(-10*time.Minute))
	writeSave(t, cfg.data("saves", "_autosave2.zip"), "from another session", now.Add(-3*time.Hour))
	damaged := cfg.data("saves", "_autosave3.zip")
	writeSave(t, damaged, "cut short", now)
	b, _ := os.ReadFile(damaged)
	os.WriteFile(damaged, b[:len(b)/2], 0o644)

	if _, err := publish(cfg, s); err != nil {
		t.Fatal(err)
	}
	if got := readSave(t, s.Origin); got != "newer" {
		t.Fatalf("original holds %q", got)
	}
}

func TestPublishNothingNew(t *testing.T) {
	cfg, s := newSession(t, "base.zip")
	if dst, err := publish(cfg, s); dst != "" || err != nil {
		t.Fatalf("publish = %q, %v", dst, err)
	}
}

func TestVerifySave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.zip")
	writeSave(t, path, "map", time.Now())
	if err := verifySave(path); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	b[bytes.Index(b, []byte("map"))] ^= 1 // one flipped bit in the data
	os.WriteFile(path, b, 0o644)
	if verifySave(path) == nil {
		t.Fatal("damaged save accepted")
	}
}

func TestBackupKeepsOnlyItsOwn(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "base.zip")
	writeSave(t, src, "map", time.Now())
	out := filepath.Join(dir, "backups")
	for _, n := range []string{"base.20200101-000000.zip", "base.20210101-000000.zip", "base.old.20200101-000000.zip", "other.20200101-000000.zip"} {
		writeSave(t, filepath.Join(out, n), "map", time.Now())
	}
	if err := backup(src, out, "base", 2); err != nil {
		t.Fatal(err)
	}
	left, _ := os.ReadDir(out)
	var names []string
	for _, e := range left {
		names = append(names, e.Name())
	}
	got := strings.Join(names, " ")
	if len(names) != 4 || strings.Contains(got, "base.20200101") || !strings.Contains(got, "base.20210101") {
		t.Fatalf("left: %s", got)
	}
}

func TestSyncMods(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	writeSave(t, filepath.Join(src, "mod_1.0.0.zip"), "mod", time.Now())
	os.WriteFile(filepath.Join(src, "mod-list.json"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(src, "notes.txt"), nil, 0o644)
	writeSave(t, filepath.Join(dst, "removed_1.0.0.zip"), "mod", time.Now())
	if err := syncMods(src, dst); err != nil {
		t.Fatal(err)
	}
	left, _ := os.ReadDir(dst)
	if len(left) != 2 || left[0].Name() != "mod-list.json" || left[1].Name() != "mod_1.0.0.zip" {
		t.Fatalf("left: %v", left)
	}
}

func TestRconFrames(t *testing.T) {
	var b bytes.Buffer
	if err := rconWrite(&b, 7, rconCommand, "/players"); err != nil {
		t.Fatal(err)
	}
	id, body, err := rconRead(&b)
	if id != 7 || body != "/players" || err != nil {
		t.Fatalf("rconRead = %d, %q, %v", id, body, err)
	}
	if _, _, err := rconRead(bytes.NewReader([]byte{3, 0, 0, 0, 1, 2, 3})); err == nil {
		t.Fatal("short packet accepted")
	}
}

func TestSteamLibraries(t *testing.T) {
	got := steamLibraries("\"libraryfolders\"\n{\n\"0\"\n{\n\"path\"\t\t\"C:\\\\Steam\"\n}\n\"1\" { \"path\" \"D:\\\\Games\\\\Lib\" }\n}")
	if len(got) != 2 || got[0] != `C:\Steam` || got[1] != `D:\Games\Lib` {
		t.Fatalf("%q", got)
	}
}
