package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	rconCommand = 2
	rconAuth    = 3
)

// rcon runs one command over a short connection of its own.
func rcon(port int, password, cmd string) (string, error) {
	c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 3*time.Second)
	if err != nil {
		return "", err
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(15 * time.Second))

	if err := rconWrite(c, 1, rconAuth, password); err != nil {
		return "", err
	}
	id, _, err := rconRead(c)
	if err != nil {
		return "", err
	}
	if id != 1 {
		return "", errors.New(tr("RCON: wrong password", "RCON: неверный пароль"))
	}
	if err := rconWrite(c, 2, rconCommand, cmd); err != nil {
		return "", err
	}
	_, body, err := rconRead(c)
	return strings.TrimRight(body, "\r\n"), err
}

func rconWrite(w io.Writer, id, kind int32, body string) error {
	b := binary.LittleEndian.AppendUint32(nil, uint32(len(body)+10))
	b = binary.LittleEndian.AppendUint32(b, uint32(id))
	b = binary.LittleEndian.AppendUint32(b, uint32(kind))
	b = append(b, body...)
	_, err := w.Write(append(b, 0, 0))
	return err
}

func rconRead(r io.Reader) (int32, string, error) {
	var n uint32
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return 0, "", err
	}
	if n < 10 || n > 1<<24 {
		return 0, "", errors.New(tr("RCON: malformed packet", "RCON: некорректный пакет"))
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, "", err
	}
	return int32(binary.LittleEndian.Uint32(b)), string(b[8 : n-2]), nil
}
