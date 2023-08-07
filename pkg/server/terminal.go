package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

type windowSize struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
	X    uint16
	Y    uint16
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Accepting all requests
	},
}

func (a *API) terminal(w http.ResponseWriter, r *http.Request) {
	workdir := "/"
	if r.URL.Query().Has("workdir") {
		workdir = r.URL.Query().Get("workdir")
	}

	user := "root"
	if r.URL.Query().Has("workdir") {
		user = r.URL.Query().Get("user")
	}

	target := "local"
	if r.URL.Query().Has("workdir") {
		target = r.URL.Query().Get("target")
	}

	shell := "/bin/sh"
	if r.URL.Query().Has("workdir") {
		shell = r.URL.Query().Get("shell")
	}

	// Upgrade to websockets
	connection, _ := upgrader.Upgrade(w, r, nil)

	var cmd *exec.Cmd
	if target == "local" {
		defaultShell := "bash"
		if runtime.GOOS == "windows" {
			defaultShell = "powershell.exe"
		}

		a.log.Debug("Connecting to local terminal", "shell", defaultShell)
		cmd = exec.Command(defaultShell)
		cmd.Dir = workdir
	} else {
		a.log.Debug("Connecting to remote Docker container", "workdir", workdir, "user", user, "target", target, "shell", shell)
		cmd = exec.Command("docker", "exec", "-ti", "-w", workdir, "-u", user, target, shell)
	}

	cmd.Env = append(os.Environ(), "TERM=xterm")

	tty, err := pty.Start(cmd)
	if err != nil {
		connection.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		a.log.Error("Unable to start pty/cmd", "error", err)
		return
	}

	defer func() {
		cmd.Process.Kill()
		cmd.Process.Wait()
		tty.Close()
		connection.Close()
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.log.Info("Websocket crashed", "error", r)
			}
		}()

		for {
			buf := make([]byte, 1024)
			read, err := tty.Read(buf)
			if err != nil {
				_ = connection.WriteMessage(websocket.TextMessage, []byte(err.Error()))

				a.log.Error("Unable to read from pty/cmd", "error", err)
				return
			}
			_ = connection.WriteMessage(websocket.BinaryMessage, buf[:read])
		}
	}()

	for {
		_, reader, err := connection.NextReader()
		if err != nil {
			a.log.Error("Unable to grab next reader", "error", err)
			return
		}

		dataTypeBuf := make([]byte, 1)
		read, err := reader.Read(dataTypeBuf)
		if err != nil {
			a.log.Error("Unable to read message type from reader", "error", err)
			_ = connection.WriteMessage(websocket.TextMessage, []byte("Unable to read message type from reader"))
			return
		}

		if read != 1 {
			a.log.Error("Unexpected number of bytes read")
			return
		}

		switch dataTypeBuf[0] {
		case 0:
			copied, err := io.Copy(tty, reader)
			if err != nil {
				a.log.Error("Error after copying data", "bytes", copied, "error", err)
			}
		case 1:
			decoder := json.NewDecoder(reader)
			resizeMessage := windowSize{}
			err := decoder.Decode(&resizeMessage)
			if err != nil {
				_ = connection.WriteMessage(websocket.TextMessage, []byte("Error decoding resize message: "+err.Error()))
				continue
			}

			a.log.Debug("Resizing terminal")
			pty.Setsize(
				tty,
				&pty.Winsize{
					Cols: resizeMessage.Cols,
					Rows: resizeMessage.Rows,
					X:    resizeMessage.X,
					Y:    resizeMessage.Y,
				})

			// #nosec G103
			//_, _, errno := syscall.Syscall(
			//	syscall.SYS_IOCTL,
			//	tty.Fd(),
			//	syscall.TIOCSWINSZ,
			//	uintptr(unsafe.Pointer(&resizeMessage)),
			//)
			//if errno != 0 {
			//	a.log.Error("Unable to resize terminal")
			//}
		default:
			a.log.Error("Unknown data", "type", dataTypeBuf[0])
		}
	}
}
