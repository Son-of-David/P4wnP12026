package ptyshell

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

type Service struct {
	upgrader websocket.Upgrader
}

func NewService() *Service {
	return &Service{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
}

// ptyEnv forces a capable TERM so tmux, bash, and colorized tools work over the websocket PTY.
func ptyEnv() []string {
	preset := []string{
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"HOME=/root",
		"USER=root",
		"LOGNAME=root",
		"SHELL=/bin/bash",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"PS1=P4wnP12026$ ",
		"HISTFILE=/dev/null",
	}
	seen := make(map[string]struct{}, 16)
	for _, e := range preset {
		k, _, _ := strings.Cut(e, "=")
		seen[k] = struct{}{}
	}
	out := append([]string(nil), preset...)
	for _, e := range os.Environ() {
		k, _, ok := strings.Cut(e, "=")
		if !ok || k == "" {
			continue
		}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, e)
	}
	return out
}

func (s *Service) HandleShell(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	cmd := exec.Command("/bin/bash", "--noprofile", "--norc", "-i")
	cmd.Dir = "/usr/share/S.o.D/"
	cmd.Env = ptyEnv()
	s.runPTY(conn, cmd)
}

func (s *Service) HandleAirgeddon(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Keep this terminal interactive without auto-starting tmux/airgeddon.
	// Airgeddon lifecycle is controlled by explicit Start/Stop buttons in the UI.
	cmd := exec.Command("/bin/bash", "--noprofile", "--norc", "-i")
	cmd.Dir = "/root/P4wnP12026/airgeddon/scripts/"
	cmd.Env = ptyEnv()
	s.runPTY(conn, cmd)
}

type resizeMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func (s *Service) runPTY(conn *websocket.Conn, cmd *exec.Cmd) {
	f, err := pty.Start(cmd)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("failed to start pty\r\n"))
		return
	}
	defer f.Close()

	_ = pty.Setsize(f, &pty.Winsize{Rows: 40, Cols: 120})

	done := make(chan struct{})

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := f.Read(buf)
			if err != nil {
				break
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				break
			}
		}
		close(done)
	}()

	for {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if msgType == websocket.TextMessage {
			var rm resizeMessage
			if json.Unmarshal(payload, &rm) == nil && rm.Type == "resize" {
				_ = pty.Setsize(f, &pty.Winsize{Rows: rm.Rows, Cols: rm.Cols})
				continue
			}
		}
		if _, err := f.Write(payload); err != nil {
			return
		}
		select {
		case <-done:
			return
		default:
		}
	}
}

func EnsureSODDirExists() error {
	return os.MkdirAll("/usr/share/S.o.D/", 0o755)
}
