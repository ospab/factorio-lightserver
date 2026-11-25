// flserver runs a dedicated Factorio server on Windows from the installed game
// and its saves.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const dateLayout = "02.01.2006 15:04"

func usage() string {
	return tr(`flserver - dedicated Factorio server

  flserver                 pick a save from the list and start
  flserver start [SAVE]    start a save (name or path to a .zip; no name: the list)
  flserver start -v SAVE   the same with the game's full log
  flserver list            list the saves
  flserver status          state of the server
  flserver cmd TEXT        send a command to the server from another window
  flserver stop            save the map and stop the server
`, `flserver - выделенный сервер Factorio

  flserver                 выбрать сейв из списка и запустить
  flserver start [СЕЙВ]    запустить сейв (имя или путь к .zip; без имени - список)
  flserver start -v СЕЙВ   то же с полным логом игры
  flserver list            список сохранений
  flserver status          состояние сервера
  flserver cmd ТЕКСТ       отправить команду серверу из другого окна
  flserver stop            сохранить мир и остановить сервер
`)
}

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, tr("Error:", "Ошибка:"), err)
	}
	if ownConsole() {
		fmt.Print(tr("Press Enter to close the window...", "Нажмите Enter, чтобы закрыть окно..."))
		<-stdinLines()
	}
	if err != nil {
		os.Exit(1)
	}
}

// stdinLines is the only reader of stdin: the save list, the server console
// and the final pause share it.
var stdinLines = sync.OnceValue(func() <-chan string {
	lines := make(chan string)
	go func() {
		in := bufio.NewScanner(os.Stdin)
		for in.Scan() {
			lines <- strings.TrimSpace(in.Text())
		}
		close(lines)
	}()
	return lines
})

func run(args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	verb := "start"
	if len(args) > 0 {
		verb, args = args[0], args[1:]
	}
	switch verb {
	case "start":
		verbose := false
		if len(args) > 0 && args[0] == "-v" {
			verbose, args = true, args[1:]
		}
		name := cfg.Save
		if len(args) > 0 {
			name = strings.Join(args, " ")
		}
		if err := recoverSession(cfg); err != nil {
			return err
		}
		save, err := pick(cfg, name)
		if err != nil {
			return err
		}
		return start(cfg, save, verbose)
	case "list":
		saves, err := listSaves(cfg.SavesDir)
		if err != nil {
			return err
		}
		printSaves(saves)
		return nil
	case "status", "stop", "cmd":
		s, ok, err := cfg.loadSession()
		if err != nil {
			return err
		}
		if !ok || !processAlive(s.PID) {
			fmt.Println(tr("The server is not running.", "Сервер не запущен."))
			return nil
		}
		return remote(cfg, s, verb, strings.Join(args, " "))
	case "help", "-h", "--help", "/?":
		fmt.Print(usage())
		return nil
	}
	return fmt.Errorf(tr("unknown command %q\n\n%s", "неизвестная команда %q\n\n%s"), verb, usage())
}

// remote controls a running server from another window.
func remote(cfg config, s session, verb, text string) error {
	switch verb {
	case "cmd":
		if text == "" {
			return errors.New(tr("empty command", "пустая команда"))
		}
		r, err := rcon(s.RconPort, s.RconPass, text)
		if r != "" {
			fmt.Println(r)
		}
		return err
	case "status":
		n, err := playersOnline(s)
		if err != nil {
			return err
		}
		fmt.Printf(tr("Up for %s, save %s, players: %d, %d MB of memory\n", "Работает %s, сейв %s, игроков: %d, память %d МБ\n"),
			time.Since(s.Started).Round(time.Second), trimZip(filepath.Base(s.Origin)), n, workingSet(s.PID)>>20)
		return nil
	}

	if _, err := rcon(s.RconPort, s.RconPass, "/quit"); err != nil && !errors.Is(err, io.EOF) && !hungAfterClose(cfg) {
		return err
	}
	deadline := time.Now().Add(stopTimeout + 10*time.Second)
	for processAlive(s.PID) {
		if hungAfterClose(cfg) {
			time.Sleep(3 * time.Second) // the main window, if there is one, does it itself
			if processAlive(s.PID) {
				killProcess(s.PID)
			}
		}
		if time.Now().After(deadline) {
			return errors.New(tr("the server did not exit in time", "сервер не завершился вовремя"))
		}
		time.Sleep(200 * time.Millisecond)
	}
	// The main window returns the save; without one we do it here.
	for range 25 {
		if _, ok, _ := cfg.loadSession(); !ok {
			fmt.Println(tr("Server stopped.", "Сервер остановлен."))
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return finish(cfg, s)
}

func pick(cfg config, name string) (saveFile, error) {
	if name != "" {
		path := trimZip(name) + ".zip"
		if !filepath.IsAbs(path) {
			path = filepath.Join(cfg.SavesDir, path)
		}
		fi, err := os.Stat(path)
		if err != nil {
			return saveFile{}, fmt.Errorf(tr("save %q not found: %w", "сейв %q не найден: %w"), name, err)
		}
		return saveFile{trimZip(filepath.Base(path)), path, fi.Size(), fi.ModTime()}, nil
	}
	saves, err := listSaves(cfg.SavesDir)
	if err != nil {
		return saveFile{}, err
	}
	if len(saves) == 0 {
		return saveFile{}, fmt.Errorf(tr("no saves in %s", "в %s нет сохранений"), cfg.SavesDir)
	}
	printSaves(saves)
	fmt.Print(tr("Save number [1]: ", "Номер сейва [1]: "))
	line, ok := <-stdinLines()
	if !ok {
		return saveFile{}, errors.New(tr("no save chosen", "сейв не выбран"))
	}
	i := 1
	if line != "" {
		if i, err = strconv.Atoi(line); err != nil || i < 1 || i > len(saves) {
			return saveFile{}, fmt.Errorf(tr("no save with number %q", "нет сейва с номером %q"), line)
		}
	}
	return saves[i-1], nil
}

func printSaves(saves []saveFile) {
	for i, s := range saves {
		fmt.Printf("%3d. %-45s %s  %5.1f %s\n", i+1, s.Name, s.Mod.Format(dateLayout), float64(s.Size)/(1<<20), tr("MB", "МБ"))
	}
}
