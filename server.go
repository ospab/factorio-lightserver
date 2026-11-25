package main

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	minFreeDisk  = 512 << 20
	startTimeout = 5 * time.Minute
	stopTimeout  = 2 * time.Minute
)

// recoverSession deals with the previous session: it is either still running,
// or was cut short and its progress has to go back to the game.
func recoverSession(cfg config) error {
	s, ok, err := cfg.loadSession()
	if err != nil || !ok {
		return err
	}
	if processAlive(s.PID) {
		return errors.New(tr("the server is already running, stop it with: flserver stop", "сервер уже запущен; остановить его: flserver stop"))
	}
	fmt.Println(tr("The last session did not end cleanly, recovering its progress...", "Прошлая сессия завершилась аварийно, восстанавливаю её прогресс..."))
	return finish(cfg, s)
}

// hungAfterClose: the server wrote the map and closed the network, but the
// process is still there. On Windows it usually hangs at this point, and
// killing it loses nothing.
func hungAfterClose(cfg config) bool {
	b, err := os.ReadFile(cfg.data("factorio-current.log"))
	return err == nil && strings.Contains(string(b), closedMark)
}

func killProcess(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		p.Kill()
		p.Release()
	}
}

// finish returns the result of a session to the game. On error session.json
// stays, and the next start tries again.
func finish(cfg config, s session) error {
	dst, err := publish(cfg, s)
	if err != nil {
		return fmt.Errorf(tr("could not return the save to the game: %w\nThe progress is safe in %s; the next start will try again",
			"не удалось вернуть сейв в игру: %w\nПрогресс цел и лежит в %s; попытка повторится при следующем запуске"), err, cfg.data())
	}
	switch dst {
	case "":
		fmt.Println(tr("The session produced no saves.", "Новых сохранений за сессию не было."))
	case s.Origin:
		fmt.Println(tr("Progress saved to", "Прогресс сохранён в"), dst)
	default:
		fmt.Println(tr("The original save was left alone, progress saved to", "Исходный сейв не тронут, прогресс сохранён отдельно:"), dst)
	}
	return os.Remove(cfg.data("session.json"))
}

// prepare checks the machine and lays out the working files in data.
func prepare(cfg config, save saveFile) (session, error) {
	s := session{Origin: save.Path, RconPass: rand.Text()}
	for _, dir := range []string{cfg.data(), filepath.Dir(save.Path)} {
		free, err := freeDisk(dir)
		if err != nil {
			return s, err
		}
		if free < minFreeDisk {
			return s, fmt.Errorf(tr("only %[2]d MB free on %[1]s, at least %[3]d MB needed", "на диске %s свободно всего %d МБ, нужно хотя бы %d МБ"),
				filepath.VolumeName(dir), free>>20, minFreeDisk>>20)
		}
	}

	// Checked beforehand: Factorio's own error about it is easy to miss.
	if udpPortUsed(cfg.Port) {
		return s, fmt.Errorf(tr("UDP port %d is taken (is the game hosting a map?)", "UDP-порт %d занят (возможно, игра уже хостит мир)"), cfg.Port)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return s, err
	}
	s.RconPort = l.Addr().(*net.TCPAddr).Port
	l.Close()

	fi, err := os.Stat(save.Path)
	if err != nil {
		return s, err
	}
	s.OriginSize, s.OriginMod = fi.Size(), fi.ModTime()
	if err := verifySave(save.Path); err != nil {
		return s, err
	}
	if err := backup(save.Path, cfg.data("backups"), save.Name, cfg.Backups); err != nil {
		return s, err
	}
	if err := copyFile(save.Path, cfg.data("world.zip")); err != nil {
		return s, err
	}
	if err := syncMods(cfg.ModsDir, cfg.data("mods")); err != nil {
		return s, fmt.Errorf(tr("copying mods: %w", "копирование модов: %w"), err)
	}

	ini := fmt.Sprintf("[path]\nread-data=%s\nwrite-data=%s\n[general]\nlocale=en\n[other]\ncheck-updates=false\n",
		filepath.ToSlash(filepath.Join(cfg.GameDir, "data")), filepath.ToSlash(cfg.data()))
	if err := os.WriteFile(cfg.data("config.ini"), []byte(ini), 0o644); err != nil {
		return s, err
	}
	settings := cfg.data("server-settings.json")
	if _, err := os.Stat(settings); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(settings, []byte(defaultServerSettings), 0o644); err != nil {
			return s, err
		}
	}
	admins := cfg.data("server-adminlist.json")
	if _, err := os.Stat(admins); errors.Is(err, os.ErrNotExist) {
		src := filepath.Join(filepath.Dir(cfg.SavesDir), "server-adminlist.json")
		if err := copyFile(src, admins); err != nil && !errors.Is(err, os.ErrNotExist) {
			return s, err
		}
	}
	s.Started = time.Now()
	return s, nil
}

func start(cfg config, save saveFile, verbose bool) error {
	version, err := cfg.gameVersion()
	if err != nil {
		return err
	}
	fmt.Printf(tr("Factorio %s, save: %s (%s)\n", "Factorio %s, сейв: %s (%s)\n"), version, save.Name, save.Mod.Format(dateLayout))
	s, err := prepare(cfg, save)
	if err != nil {
		return err
	}

	args := []string{
		"--config", cfg.data("config.ini"),
		"--start-server", cfg.data("world.zip"),
		"--mod-directory", cfg.data("mods"),
		"--server-settings", cfg.data("server-settings.json"),
		"--server-adminlist", cfg.data("server-adminlist.json"),
		"--port", strconv.Itoa(cfg.Port),
		"--rcon-bind", fmt.Sprintf("127.0.0.1:%d", s.RconPort),
		"--rcon-password", s.RconPass,
	}
	if cfg.Bind != "" {
		args = append(args, "--bind", cfg.Bind)
	}
	cmd := exec.Command(cfg.exe(), args...)
	// A group of its own and no console: Ctrl+C reaches only us, the game is
	// stopped over RCON.
	const createNoWindow = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow | syscall.CREATE_NEW_PROCESS_GROUP}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	// Closing the window arrives as SIGTERM; Windows allows about 5 seconds for it.
	sig := make(chan os.Signal, 4)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	if err := cmd.Start(); err != nil {
		return err
	}
	s.PID = cmd.Process.Pid
	if err := writeJSON(cfg.data("session.json"), s); err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return err
	}

	ready, closed, exited := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		readOutput(out, verbose, ready, closed)
		exited <- cmd.Wait()
	}()

	lines := stdinLines()
	startDeadline := time.After(startTimeout)
	var stopDeadline, idle <-chan time.Time
	var idleSince time.Time
	running, stopping := false, false
	stop := func() {
		if stopping {
			return
		}
		stopping = true
		if !running {
			fmt.Println(tr("The map is not loaded yet, ending the process.", "Мир ещё не загружен, завершаю процесс."))
			cmd.Process.Kill()
			return
		}
		fmt.Println(tr("Saving the map and stopping the server...", "Сохраняю мир и останавливаю сервер..."))
		stopDeadline = time.After(stopTimeout)
		// There may be no reply to /quit: the connection goes down with the server.
		if _, err := rcon(s.RconPort, s.RconPass, "/quit"); err != nil && !errors.Is(err, io.EOF) {
			fmt.Println("! /quit:", err)
		}
	}

	for {
		select {
		case <-ready:
			ready, startDeadline = nil, nil
			running = true
			fmt.Printf(tr("Server is up in %.1f s, %d MB of memory.\n", "Сервер готов за %.1f с, память %d МБ.\n"),
				time.Since(s.Started).Seconds(), workingSet(s.PID)>>20)
			fmt.Println(tr("Connect to:", "Подключение:"), strings.Join(addresses(cfg), ", "))
			fmt.Println(tr("Commands: stop to save and exit, /players, /server-save, /promote NAME; other text goes to chat.",
				"Команды: stop - сохранить и выйти, /players, /server-save, /promote ИМЯ; обычный текст уходит в чат."))
			if cfg.IdleStopMinutes > 0 {
				idle, idleSince = time.Tick(30*time.Second), time.Now()
			}

		case line, ok := <-lines:
			if !ok {
				lines = nil
				break
			}
			switch strings.ToLower(line) {
			case "":
			case "stop", "quit", "exit", "/quit":
				stop()
			default:
				if !running || stopping {
					fmt.Println(tr("The server is not taking commands now.", "Сервер сейчас не принимает команды."))
					break
				}
				reply, err := rcon(s.RconPort, s.RconPass, line)
				if err != nil {
					fmt.Println("!", err)
				} else if reply != "" {
					fmt.Println(reply)
				}
			}

		case <-idle:
			if stopping {
				break
			}
			if n, err := playersOnline(s); err != nil || n > 0 {
				idleSince = time.Now()
			} else if time.Since(idleSince) >= time.Duration(cfg.IdleStopMinutes)*time.Minute {
				fmt.Printf(tr("No players for %d min.\n", "Игроков нет уже %d мин.\n"), cfg.IdleStopMinutes)
				stop()
			}

		case <-sig:
			stop()

		case <-closed:
			// The map is written and the network is closed. On Windows the
			// process may hang after that; killing it then loses nothing.
			closed = nil
			stopping = true
			stopDeadline = time.After(1500 * time.Millisecond)

		case <-startDeadline:
			startDeadline = nil
			fmt.Println(tr("! The server did not start in time.", "! Сервер не запустился вовремя."))
			cmd.Process.Kill()

		case <-stopDeadline:
			stopDeadline = nil
			cmd.Process.Kill()

		case err := <-exited:
			if !stopping {
				fmt.Printf(tr("! Factorio exited by itself (%v).\n", "! Factorio завершился сам (%v).\n"), err)
			}
			if err := finish(cfg, s); err != nil {
				return err
			}
			if !running {
				return fmt.Errorf(tr("the server did not start, see %s", "сервер не запустился, подробности в %s"), cfg.data("factorio-current.log"))
			}
			return nil
		}
	}
}

// Lines of the game's log that the server's state is read from.
const (
	readyMark  = "Starting RCON interface"
	closedMark = "to(Closed)"
)

// eventLine picks what is worth showing in the console.
var eventLine = regexp.MustCompile(`\[(JOIN|LEAVE|CHAT|KICK|BAN|UNBAN|PROMOTE|DEMOTE|COMMAND)\]|\bError\b|Saving finished`)

func readOutput(r io.Reader, verbose bool, ready, closed chan struct{}) {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line != "" && (verbose || eventLine.MatchString(line)) {
			fmt.Println(line)
		}
		switch {
		case ready != nil && strings.Contains(line, readyMark):
			close(ready)
			ready = nil
		case closed != nil && strings.Contains(line, closedMark):
			close(closed)
			closed = nil
		}
		if err != nil {
			return
		}
	}
}

var parens = regexp.MustCompile(`\((\d+)\)`)

func playersOnline(s session) (int, error) {
	r, err := rcon(s.RconPort, s.RconPass, "/players online")
	if err != nil {
		return 0, err
	}
	m := parens.FindStringSubmatch(r)
	if m == nil {
		return 0, fmt.Errorf("/players online: %q", r)
	}
	return strconv.Atoi(m[1])
}

func addresses(cfg config) []string {
	port := strconv.Itoa(cfg.Port)
	if cfg.Bind != "" && cfg.Bind != "0.0.0.0" {
		return []string{net.JoinHostPort(cfg.Bind, port)}
	}
	out := []string{"localhost:" + port}
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil && n.IP.IsPrivate() {
			out = append(out, net.JoinHostPort(n.IP.String(), port))
		}
	}
	return out
}
