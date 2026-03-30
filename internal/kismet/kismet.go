package kismet

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type Service struct {
	mu       sync.Mutex
	starting bool
	stopping bool
}

func NewService() *Service { return &Service{} }

const (
	startScriptPath = "/root/P4wnP12026/kismet/scripts/startup.sh"
	stopScriptPath  = "/root/P4wnP12026/kismet/scripts/teardown.sh"
)

func scriptEnvironment() []string {
	preset := []string{
		"HOME=/root",
		"USER=root",
		"LOGNAME=root",
		"SHELL=/bin/bash",
		"TERM=xterm-256color",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
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

func (s *Service) scriptCommand(scriptPath string) *exec.Cmd {
	cmd := exec.Command("/bin/bash", "-lc", "exec bash "+shellQuote(scriptPath))
	cmd.Dir = filepath.Dir(scriptPath)
	cmd.Env = scriptEnvironment()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func shellQuote(path string) string {
	return `'` + strings.ReplaceAll(path, `'`, `'"'"'`) + `'`
}

func (s *Service) Start() error {
	s.mu.Lock()
	if s.starting {
		s.mu.Unlock()
		return errors.New("startup already running")
	}
	if s.stopping {
		s.mu.Unlock()
		return errors.New("teardown is running, try again shortly")
	}
	s.starting = true
	s.mu.Unlock()

	cmd := s.scriptCommand(startScriptPath)
	if err := cmd.Start(); err != nil {
		s.mu.Lock()
		s.starting = false
		s.mu.Unlock()
		return err
	}

	// We intentionally do NOT tie "running" to this bash process lifetime.
	// startup.sh may daemonize kismet and exit quickly.
	go func() {
		_ = cmd.Wait()
		s.mu.Lock()
		s.starting = false
		s.mu.Unlock()
	}()

	return nil
}

func (s *Service) Stop() error {
	s.mu.Lock()
	if s.stopping {
		s.mu.Unlock()
		return errors.New("teardown already running")
	}
	if s.starting {
		s.mu.Unlock()
		return errors.New("cannot stop while startup is launching")
	}
	s.stopping = true
	s.mu.Unlock()

	cmd := s.scriptCommand(stopScriptPath)
	err := cmd.Run()

	s.mu.Lock()
	s.stopping = false
	s.mu.Unlock()
	return err
}

// Status reports whether kismet is actually running (process-level truth).
func (s *Service) Status() bool {
	// Prefer exact name match; also treat capture helper as "running"
	// because it implies kismet is/was started.
	checks := [][]string{
		{"pgrep", "-x", "kismet"},
		{"pgrep", "-f", "kismet_cap_linux_wifi"},
	}

	for _, args := range checks {
		c := exec.Command(args[0], args[1:]...)
		var out bytes.Buffer
		c.Stdout = &out
		c.Stderr = &out
		if err := c.Run(); err == nil {
			return true
		}
	}
	return false
}
