package gps

import (
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
	startScriptPath = "/root/P4wnP12026/gpsd/scripts/start.sh"
	stopScriptPath  = "/root/P4wnP12026/gpsd/scripts/stop.sh"
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
		return errors.New("gps startup already running")
	}
	if s.stopping {
		s.mu.Unlock()
		return errors.New("gps stop is running, try again shortly")
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
		return errors.New("gps stop already running")
	}
	if s.starting {
		s.mu.Unlock()
		return errors.New("cannot stop gps while startup is launching")
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
