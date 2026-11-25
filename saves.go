package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type saveFile struct {
	Name string // without .zip
	Path string
	Size int64
	Mod  time.Time
}

// listSaves returns the saves in dir, newest first.
func listSaves(dir string) ([]saveFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []saveFile
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".zip") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, saveFile{trimZip(e.Name()), filepath.Join(dir, e.Name()), fi.Size(), fi.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Mod.After(out[j].Mod) })
	return out, nil
}

func trimZip(s string) string {
	if strings.EqualFold(filepath.Ext(s), ".zip") {
		return s[:len(s)-4]
	}
	return s
}

// verifySave checks that the file is an intact zip with a Factorio map in it.
func verifySave(path string) error {
	z, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	defer z.Close()
	hasLevel := false
	for _, f := range z.File {
		if strings.HasPrefix(filepath.Base(f.Name), "level.dat") {
			hasLevel = true
		}
		r, err := f.Open()
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		_, err = io.Copy(io.Discard, r) // reading to the end checks the CRC
		r.Close()
		if err != nil {
			return fmt.Errorf("%s: %s: %w", path, f.Name, err)
		}
	}
	if !hasLevel {
		return fmt.Errorf(tr("%s is not a Factorio save", "%s не похож на сохранение Factorio"), path)
	}
	return nil
}

// copyFile goes through a temporary file, so that a failure (say, a full disk)
// leaves the destination as it was.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.CreateTemp(filepath.Dir(dst), ".flserver-*.tmp")
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err == nil {
		err = out.Sync()
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(out.Name(), dst)
	}
	if err != nil {
		os.Remove(out.Name())
	}
	return err
}

// backup puts a copy of the save into dir and keeps at most keep of them.
func backup(src, dir, name string, keep int) error {
	dst := filepath.Join(dir, name+"."+stamp()+".zip")
	if err := copyFile(src, dst); err != nil {
		return fmt.Errorf(tr("backup: %w", "резервная копия: %w"), err)
	}
	all, err := listSaves(dir)
	if err != nil {
		return nil
	}
	var mine []string
	for _, s := range all {
		rest, ok := strings.CutPrefix(s.Name, name+".")
		if _, err := time.Parse(stampLayout, rest); ok && err == nil {
			mine = append(mine, s.Path)
		}
	}
	sort.Strings(mine)
	for len(mine) > keep {
		os.Remove(mine[0])
		mine = mine[1:]
	}
	return nil
}

const stampLayout = "20060102-150405"

func stamp() string { return time.Now().Format(stampLayout) }

// publish returns the server's progress to the game's saves. If the original
// changed during the session (the game saved over it), it is left alone and the
// progress goes next to it under a new name.
// An empty path with no error means the session produced no saves.
func publish(cfg config, s session) (string, error) {
	candidates, err := listSaves(cfg.data("saves")) // the server's autosaves
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if fi, err := os.Stat(cfg.data("world.zip")); err == nil {
		candidates = append(candidates, saveFile{"world", cfg.data("world.zip"), fi.Size(), fi.ModTime()})
	}
	var best saveFile
	for _, c := range candidates {
		if !c.Mod.After(s.Started) || !c.Mod.After(best.Mod) {
			continue
		}
		if err := verifySave(c.Path); err != nil {
			fmt.Println(tr("! skipping a damaged file:", "! пропускаю повреждённый файл:"), err)
			continue
		}
		best = c
	}
	if best.Path == "" {
		return "", nil
	}

	dst := s.Origin
	fi, err := os.Stat(dst)
	unchanged := err == nil && fi.Size() == s.OriginSize && fi.ModTime().Equal(s.OriginMod)
	isAutosave := strings.HasPrefix(filepath.Base(dst), "_autosave")
	if !cfg.SyncBack || isAutosave || !unchanged {
		name := strings.TrimPrefix(trimZip(filepath.Base(dst)), "_")
		dst = filepath.Join(filepath.Dir(dst), name+"-server-"+stamp()+".zip")
	}
	return dst, copyFile(best.Path, dst)
}

// syncMods makes the server's mod folder a copy of the game's; archives that
// did not change are not copied again.
func syncMods(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	want := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			fmt.Println(tr("! unpacked mod is not copied:", "! распакованный мод не копируется:"), e.Name())
			continue
		}
		isZip := strings.EqualFold(filepath.Ext(e.Name()), ".zip")
		if !isZip && e.Name() != "mod-list.json" && e.Name() != "mod-settings.dat" {
			continue
		}
		want[e.Name()] = true
		a, err := e.Info()
		if err != nil {
			return err
		}
		to := filepath.Join(dst, e.Name())
		if b, err := os.Stat(to); err == nil && isZip && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime()) {
			continue
		}
		if err := copyFile(filepath.Join(src, e.Name()), to); err != nil {
			return err
		}
		os.Chtimes(to, a.ModTime(), a.ModTime())
	}
	have, err := os.ReadDir(dst)
	if err != nil {
		return err
	}
	for _, e := range have {
		if !e.IsDir() && !want[e.Name()] {
			os.Remove(filepath.Join(dst, e.Name()))
		}
	}
	return nil
}
