package bootdata

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const maxCheckOutputBytes = 256 * 1024

const (
	usbGadgetScript = "/root/P4wnP12026/bootdata/usb_gadget/usb_gadget.sh"
	defaultsScript  = "/root/P4wnP12026/bootdata/defaults/defaults.sh"
	checkScript     = "/root/P4wnP12026/bootdata/check/check.sh"
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

func shellQuote(path string) string {
	return `'` + strings.ReplaceAll(path, `'`, `'"'"'`) + `'`
}

func scriptCommand(scriptPath string) *exec.Cmd {
	cmd := exec.Command("/bin/bash", "-lc", "exec bash "+shellQuote(scriptPath))
	cmd.Dir = filepath.Dir(scriptPath)
	cmd.Env = scriptEnvironment()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// RunUSBGadget launches the usb_gadget boot script in the background.
func RunUSBGadget() error {
	return startScript(usbGadgetScript)
}

// RunDefaults launches the defaults boot script in the background.
func RunDefaults() error {
	return startScript(defaultsScript)
}

// RunCheckOutput runs the check script, captures combined stdout/stderr, and
// honors ctx (e.g. 10s timeout). Non-zero exit returns err but output may still be set.
func RunCheckOutput(ctx context.Context) (output string, err error) {
	cmd := exec.CommandContext(ctx, "/bin/bash", "-lc", "exec bash "+shellQuote(checkScript))
	cmd.Dir = filepath.Dir(checkScript)
	cmd.Env = scriptEnvironment()
	b, err := cmd.CombinedOutput()
	s := string(b)
	if len(s) > maxCheckOutputBytes {
		s = s[:maxCheckOutputBytes] + "\n... [truncated]"
	}
	return s, err
}

func startScript(scriptPath string) error {
	cmd := scriptCommand(scriptPath)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
