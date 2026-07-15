// Copyright (C) 2026 Alexander Grafov <grafov@inet.name>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package sudowrap

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

const (
	sudoName    = "sudo"
	askpassName = "askpass"
	passName    = ".pass"
	dirPerm     = 0o700
	passPerm    = 0o600
	execPerm    = 0o700

	defaultSafePath = "/usr/local/bin:/usr/bin:/bin"
)

var keptExactEnvKeys = map[string]struct{}{
	"HOME":                    {},
	"USER":                    {},
	"LOGNAME":                 {},
	"XDG_RUNTIME_DIR":         {},
	"XDG_DATA_HOME":           {},
	"XDG_CONFIG_HOME":         {},
	"XDG_CACHE_HOME":          {},
	"LANG":                    {},
	"DISPLAY":                 {},
	"WAYLAND_DISPLAY":         {},
	"XAUTHORITY":              {},
	"DBUS_SESSION_BUS_ADDRESS": {},
}

var (
	// ErrInvalidPassword is returned when sudo rejects the provided password.
	ErrInvalidPassword = errors.New("invalid sudo password")
)

// Env holds an isolated runtime directory with sudo and askpass wrappers.
// PATH is only injected into child processes via Apply; the parent process PATH is never changed.
type Env struct {
	enabled  bool
	askpass  bool
	dir      string
	pass     []byte
	passMx   sync.Mutex
	realSudo string
}

// Setup creates a private runtime directory with sudo and askpass wrapper scripts.
// When enabled is false, Setup returns a disabled Env without creating files (askpass is ignored).
// When askpass is false, the wrapper only uses sudo -n (passwordless / ticket path).
func Setup(enabled, askpass bool) (*Env, error) {
	realSudo, err := resolveRealSudo()
	if err != nil {
		return nil, err
	}

	env := &Env{
		enabled:  enabled,
		askpass:  enabled && askpass,
		realSudo: realSudo,
	}
	if !enabled {
		return env, nil
	}

	dir, err := createRuntimeDir()
	if err != nil {
		return nil, err
	}
	env.dir = dir

	if err := env.writeScripts(); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	return env, nil
}

func createRuntimeDir() (string, error) {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		return os.MkdirTemp("", "adgui-sudo-")
	}

	pid := os.Getpid()
	dir := filepath.Join(base, "adgui", strconv.Itoa(pid))
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return os.MkdirTemp("", "adgui-sudo-")
	}
	if err := os.Chmod(dir, dirPerm); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func (e *Env) writeScripts() error {
	real := shellQuote(e.realSudo)

	var sudoScript string
	if e.askpass {
		pass := shellQuote(filepath.Join(e.dir, passName))
		// Use askpass when .pass exists; otherwise non-interactive -n on the real
		// command (ticket cache and command-specific NOPASSWD).
		sudoScript = "#!/bin/sh\n" +
			"REAL=" + real + "\n" +
			"PASS=" + pass + "\n" +
			"if [ -f \"$PASS\" ]; then\n" +
			"  exec \"$REAL\" -A \"$@\"\n" +
			"fi\n" +
			"exec \"$REAL\" -n \"$@\"\n"
	} else {
		sudoScript = "#!/bin/sh\n" +
			"REAL=" + real + "\n" +
			"exec \"$REAL\" -n \"$@\"\n"
	}
	if err := writeExecutable(filepath.Join(e.dir, sudoName), sudoScript); err != nil {
		return err
	}

	if !e.askpass {
		return nil
	}

	askpassScript := "#!/bin/sh\n" +
		"PASS=" + shellQuote(filepath.Join(e.dir, passName)) + "\n" +
		"[ -f \"$PASS\" ] || exit 1\n" +
		"cat \"$PASS\"\n"
	return writeExecutable(filepath.Join(e.dir, askpassName), askpassScript)
}

func writeExecutable(path, content string) error {
	if err := os.WriteFile(path, []byte(content), execPerm); err != nil {
		return err
	}
	return os.Chmod(path, execPerm)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func resolveRealSudo() (string, error) {
	candidates := []string{"/usr/bin/sudo", "/bin/sudo"}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	path, err := exec.LookPath("sudo")
	if err != nil {
		return "", fmt.Errorf("sudo not found: %w", err)
	}
	return path, nil
}

// Enabled reports whether the wrapper is active.
func (e *Env) Enabled() bool {
	return e != nil && e.enabled
}

// AskpassEnabled reports whether GUI askpass / .pass path is active.
func (e *Env) AskpassEnabled() bool {
	return e != nil && e.enabled && e.askpass
}

// Dir returns the private runtime directory path.
func (e *Env) Dir() string {
	if e == nil {
		return ""
	}
	return e.dir
}

// HasPassword reports whether a session password is stored in memory.
func (e *Env) HasPassword() bool {
	if e == nil {
		return false
	}
	e.passMx.Lock()
	defer e.passMx.Unlock()
	return len(e.pass) > 0
}

// PassFileExists reports whether the on-disk askpass secret file is present.
func (e *Env) PassFileExists() bool {
	if e == nil || e.dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(e.dir, passName))
	return err == nil
}

// ReadyForAskpass reports whether memory and on-disk password are both available.
func (e *Env) ReadyForAskpass() bool {
	return e.HasPassword() && e.PassFileExists()
}

// NeedsPrompt reports whether elevation likely requires a password prompt.
func (e *Env) NeedsPrompt() bool {
	if e == nil || !e.enabled || !e.askpass {
		return false
	}
	return !ValidTicket(e.realSudo) && !e.ReadyForAskpass()
}

// ValidTicket reports whether sudo credentials are currently cached.
func ValidTicket(realSudo string) bool {
	if realSudo == "" {
		var err error
		realSudo, err = resolveRealSudo()
		if err != nil {
			return false
		}
	}
	cmd := exec.Command(realSudo, "-n", "-v")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// Apply injects a filtered child environment with wrapper PATH and SUDO_ASKPASS.
// Parent process environment is never modified.
func (e *Env) Apply(cmd *exec.Cmd) {
	if e == nil || !e.enabled || e.dir == "" || cmd == nil {
		return
	}
	cmd.Env = buildChildEnv(e.dir, e.askpass)
}

// buildChildEnv builds a whitelist-based environment for CLI and sudo children.
func buildChildEnv(wrapperDir string, askpass bool) []string {
	out := make([]string, 0, 24)
	for _, item := range os.Environ() {
		key, _, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if !keepEnvKey(key) {
			continue
		}
		out = append(out, item)
	}

	pathValue := wrapperDir + string(os.PathListSeparator) + safePath(os.Getenv("PATH"))
	out = setEnvVar(out, "PATH", pathValue)
	if askpass {
		out = setEnvVar(out, "SUDO_ASKPASS", filepath.Join(wrapperDir, askpassName))
	}
	return out
}

func keepEnvKey(key string) bool {
	if _, ok := keptExactEnvKeys[key]; ok {
		return true
	}
	if strings.HasPrefix(key, "LC_") {
		return true
	}
	return false
}

func safePath(parentPath string) string {
	parts := strings.Split(parentPath, string(os.PathListSeparator))
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		cleaned = append(cleaned, part)
	}
	if len(cleaned) == 0 {
		return defaultSafePath
	}
	return strings.Join(cleaned, string(os.PathListSeparator))
}

func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	replaced := false
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			out = append(out, prefix+value)
			replaced = true
			continue
		}
		out = append(out, item)
	}
	if !replaced {
		out = append(out, prefix+value)
	}
	return out
}

// SetPassword stores the password for askpass and warms the sudo ticket.
func (e *Env) SetPassword(password []byte) error {
	if e == nil || !e.enabled || !e.askpass {
		return nil
	}
	if len(password) == 0 {
		return ErrInvalidPassword
	}

	copyPass := append([]byte(nil), password...)
	if err := e.writePasswordFile(copyPass); err != nil {
		zeroBytes(copyPass)
		return err
	}

	if err := e.warmTicket(); err != nil {
		e.ClearPassword()
		zeroBytes(copyPass)
		return err
	}

	e.passMx.Lock()
	zeroBytes(e.pass)
	e.pass = copyPass
	e.passMx.Unlock()
	return nil
}

func (e *Env) writePasswordFile(password []byte) error {
	passPath := filepath.Join(e.dir, passName)
	if err := os.WriteFile(passPath, append(password, '\n'), passPerm); err != nil {
		return err
	}
	return os.Chmod(passPath, passPerm)
}

// warmTicket validates credentials via sudo -A (askpass), never via interactive TTY or -S stdin.
func (e *Env) warmTicket() error {
	cmd := exec.Command(e.realSudo, "-A", "-v")
	cmd.Env = buildChildEnv(e.dir, true)
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		combined := stderr.String() + stdout.String()
		lower := strings.ToLower(combined)
		if strings.Contains(lower, "password") ||
			strings.Contains(lower, "incorrect") ||
			strings.Contains(lower, "sorry") ||
			strings.Contains(lower, "authentication") {
			return ErrInvalidPassword
		}
		return fmt.Errorf("sudo ticket warmup failed: %w (%s)", err, strings.TrimSpace(combined))
	}
	return nil
}

// ClearPassword wipes the in-memory and on-disk password secret.
func (e *Env) ClearPassword() {
	if e == nil {
		return
	}
	e.passMx.Lock()
	zeroBytes(e.pass)
	e.pass = nil
	e.passMx.Unlock()

	if e.dir == "" {
		return
	}
	passPath := filepath.Join(e.dir, passName)
	data, err := os.ReadFile(passPath)
	if err == nil {
		zeroBytes(data)
	}
	_ = os.Remove(passPath)
}

// Close clears secrets and removes the runtime directory.
func (e *Env) Close() error {
	if e == nil {
		return nil
	}
	e.ClearPassword()
	if e.dir == "" {
		return nil
	}
	err := os.RemoveAll(e.dir)
	e.dir = ""
	return err
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
