package modeswitch

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

const (
	modeUSBGadget = "usb_gadget"
	modePiDefault = "pi_defaults"
)

type Service struct {
	bootRoot string
	destDir  string
	mu       sync.Mutex
	busy     bool
}

func NewService() *Service {
	return &Service{
		bootRoot: "/root/Install/P4wnP12026/bootdata",
		destDir:  "/boot/firmware",
	}
}

func (s *Service) Trigger(mode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errors.New("mode switch already in progress")
	}
	s.busy = true
	go s.switchAndReboot(mode)
	return nil
}

func (s *Service) switchAndReboot(mode string) {
	defer func() {
		s.mu.Lock()
		s.busy = false
		s.mu.Unlock()
	}()

	srcDir, err := s.sourceFromMode(mode)
	if err != nil {
		return
	}

	if err := validateDir(srcDir); err != nil {
		return
	}
	if err := validateDir(s.destDir); err != nil {
		return
	}
	if err := copyDirContents(srcDir, s.destDir); err != nil {
		return
	}

	_ = exec.Command("sync").Run()
	time.Sleep(5 * time.Second)
	_ = exec.Command("reboot").Run()
}

func (s *Service) sourceFromMode(mode string) (string, error) {
	switch mode {
	case modeUSBGadget, modePiDefault:
		return filepath.Join(s.bootRoot, mode), nil
	default:
		return "", fmt.Errorf("invalid mode %q", mode)
	}
}

func validateDir(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("%s is empty", path)
	}
	return nil
}

func copyDirContents(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == src {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(destPath, info.Mode().Perm())
		}
		return copyFile(path, destPath, info.Mode().Perm())
	})
}

func copyFile(src, dst string, perm fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
