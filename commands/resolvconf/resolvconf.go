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

// Package resolvconf swaps the system resolver file with the AdGuard VPN
// resolv.conf and restores the previous file or symlink afterwards.
package resolvconf

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultTarget is the system resolver file replaced while the VPN is up.
	DefaultTarget = "/etc/resolv.conf"

	fileOriginal   = "resolv.conf.original"
	fileMode       = "resolv.conf.original.mode"
	fileLink       = "resolv.conf.original.link"
	fileMissing    = "resolv.conf.original.missing"
	fileApplied    = "resolv.conf.applied"
	fileAppliedNew = "resolv.conf.applied.new"

	appliedMode = "0644"
	statePerm   = 0o600
	dirPerm     = 0o700
)

// ErrNoNameserver is returned when the VPN resolv.conf has no nameserver entry.
// Apply refuses such a file so a bad CLI output cannot wipe system DNS.
var ErrNoNameserver = errors.New("resolv.conf source contains no nameserver")

// Runner executes one program without a shell.
// name is the executable (install, mv, ln, or rm) and args are its arguments.
// Production runs them through sudo. Tests run them directly.
type Runner func(name string, args ...string) error

// Manager replaces Target with a VPN resolv.conf and restores the previous
// file or symlink from StateDir.
//
// StateDir holds the backup (resolv.conf.original and its mode, or
// resolv.conf.original.link, or resolv.conf.original.missing) and the content
// last written (resolv.conf.applied). Apply records a new backup when Target
// is not that applied content, so a resolver rewritten while the VPN is up is
// what Restore puts back. Restore changes Target only while it still matches
// the applied content; otherwise the backup is dropped and Target is kept.
type Manager struct {
	// Target is the system resolver file, normally /etc/resolv.conf.
	Target string
	// StateDir is a user-writable directory that survives process restarts.
	StateDir string
	// Run executes install, mv, ln, and rm without a shell.
	Run Runner
}

type snapshot struct {
	missing bool
	symlink bool
	link    string
	content []byte
	mode    os.FileMode
}

// Pending reports whether a backup is waiting to be restored.
// A pending backup survives process restarts because it lives in StateDir.
func (m *Manager) Pending() bool {
	if m == nil || m.StateDir == "" {
		return false
	}
	for _, name := range []string{fileOriginal, fileLink, fileMissing} {
		if _, err := os.Stat(m.path(name)); err == nil {
			return true
		}
	}
	return false
}

// Apply copies src over Target.
// src must contain at least one nameserver. When Target already has that
// content, Apply only refreshes the bookkeeping and leaves the file in place.
// The previous Target is saved when it differs from the content adgui last
// wrote. The write itself is install into Target.adgui-new followed by mv -f,
// which replaces a symlink without modifying the file it pointed at.
func (m *Manager) Apply(src string) error {
	if err := m.validate(); err != nil {
		return err
	}
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read dns source %s: %w", src, err)
	}
	if !hasNameserver(content) {
		return fmt.Errorf("%w: %s", ErrNoNameserver, src)
	}
	cur, err := readSnapshot(m.Target)
	if err != nil {
		return err
	}
	if !cur.missing && !cur.symlink && bytes.Equal(cur.content, content) {
		if m.Pending() || m.hasAppliedRecord() {
			if err := m.commitApplied(content); err != nil {
				return err
			}
		}
		return nil
	}
	if !m.isOurs(cur) {
		if err := m.saveOriginal(cur); err != nil {
			return err
		}
	}
	return m.installContent(content, appliedMode)
}

// Restore puts the backed-up file or symlink back when Target still contains
// the resolv.conf adgui installed. If something else replaced Target, the
// backup is discarded and Target is left unchanged. A missing backup is a
// no-op. Restore still runs when DNS management was turned off after the
// backup was taken, so a previous session can be unwound.
func (m *Manager) Restore() error {
	if m == nil || !m.Pending() {
		return nil
	}
	if err := m.validate(); err != nil {
		return err
	}
	cur, err := readSnapshot(m.Target)
	if err != nil {
		return err
	}
	if !m.isOurs(cur) {
		return m.clearState()
	}
	if err := m.restoreOriginal(); err != nil {
		return err
	}
	return m.clearState()
}

func (m *Manager) validate() error {
	if m == nil {
		return errors.New("dns manager is not initialized")
	}
	if m.Run == nil {
		return errors.New("dns manager has no command runner")
	}
	if m.Target == "" || m.StateDir == "" {
		return errors.New("dns manager target or state directory is empty")
	}
	return nil
}

func (m *Manager) path(name string) string {
	return filepath.Join(m.StateDir, name)
}

func (m *Manager) hasAppliedRecord() bool {
	for _, name := range []string{fileApplied, fileAppliedNew} {
		if _, err := os.Stat(m.path(name)); err == nil {
			return true
		}
	}
	return false
}

func (m *Manager) appliedContents() [][]byte {
	var out [][]byte
	for _, name := range []string{fileApplied, fileAppliedNew} {
		data, err := os.ReadFile(m.path(name))
		if err != nil {
			continue
		}
		out = append(out, data)
	}
	return out
}

func (m *Manager) isOurs(cur snapshot) bool {
	if cur.missing || cur.symlink {
		return false
	}
	for _, applied := range m.appliedContents() {
		if bytes.Equal(cur.content, applied) {
			return true
		}
	}
	return false
}

func (m *Manager) saveOriginal(cur snapshot) error {
	if err := os.MkdirAll(m.StateDir, dirPerm); err != nil {
		return fmt.Errorf("create dns state dir: %w", err)
	}
	switch {
	case cur.missing:
		if err := writeState(m.path(fileMissing), nil); err != nil {
			return err
		}
		return removeState(m.path(fileOriginal), m.path(fileMode), m.path(fileLink))
	case cur.symlink:
		if err := writeState(m.path(fileLink), []byte(cur.link)); err != nil {
			return err
		}
		return removeState(m.path(fileOriginal), m.path(fileMode), m.path(fileMissing))
	default:
		mode := cur.mode.Perm()
		if mode == 0 {
			mode = 0o644
		}
		if err := writeState(m.path(fileOriginal), cur.content); err != nil {
			return err
		}
		if err := writeState(m.path(fileMode), []byte(fmt.Sprintf("%#o\n", mode))); err != nil {
			return err
		}
		return removeState(m.path(fileLink), m.path(fileMissing))
	}
}

func (m *Manager) installContent(content []byte, mode string) error {
	if err := os.MkdirAll(m.StateDir, dirPerm); err != nil {
		return fmt.Errorf("create dns state dir: %w", err)
	}
	pending := m.path(fileAppliedNew)
	if err := writeState(pending, content); err != nil {
		return err
	}
	staging := m.Target + ".adgui-new"
	if err := m.Run("install", "-m", mode, "--", pending, staging); err != nil {
		return fmt.Errorf("install resolv.conf: %w", err)
	}
	if err := m.Run("mv", "-f", "--", staging, m.Target); err != nil {
		return fmt.Errorf("replace resolv.conf: %w", err)
	}
	return m.commitApplied(content)
}

func (m *Manager) commitApplied(content []byte) error {
	if err := os.MkdirAll(m.StateDir, dirPerm); err != nil {
		return fmt.Errorf("create dns state dir: %w", err)
	}
	if err := writeState(m.path(fileApplied), content); err != nil {
		return err
	}
	return removeState(m.path(fileAppliedNew))
}

func (m *Manager) restoreOriginal() error {
	switch {
	case stateExists(m.path(fileMissing)):
		if err := m.Run("rm", "-f", "--", m.Target); err != nil {
			return fmt.Errorf("remove resolv.conf: %w", err)
		}
		return nil
	case stateExists(m.path(fileLink)):
		link, err := os.ReadFile(m.path(fileLink))
		if err != nil {
			return fmt.Errorf("read symlink backup: %w", err)
		}
		if len(link) == 0 {
			return errors.New("symlink backup is empty")
		}
		if err := m.Run("ln", "-sfn", "--", string(link), m.Target); err != nil {
			return fmt.Errorf("restore resolv.conf symlink: %w", err)
		}
		return nil
	default:
		mode := appliedMode
		if raw, err := os.ReadFile(m.path(fileMode)); err == nil {
			if parsed := strings.TrimSpace(string(raw)); validMode(parsed) {
				mode = parsed
			}
		}
		staging := m.Target + ".adgui-new"
		if err := m.Run("install", "-m", mode, "--", m.path(fileOriginal), staging); err != nil {
			return fmt.Errorf("install original resolv.conf: %w", err)
		}
		if err := m.Run("mv", "-f", "--", staging, m.Target); err != nil {
			return fmt.Errorf("restore resolv.conf: %w", err)
		}
		return nil
	}
}

func (m *Manager) clearState() error {
	return removeState(
		m.path(fileOriginal),
		m.path(fileMode),
		m.path(fileLink),
		m.path(fileMissing),
		m.path(fileApplied),
		m.path(fileAppliedNew),
	)
}

func readSnapshot(path string) (snapshot, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return snapshot{missing: true}, nil
		}
		return snapshot{}, fmt.Errorf("stat %s: %w", path, err)
	}
	if info.IsDir() {
		return snapshot{}, fmt.Errorf("%s is a directory", path)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, err := os.Readlink(path)
		if err != nil {
			return snapshot{}, fmt.Errorf("readlink %s: %w", path, err)
		}
		return snapshot{symlink: true, link: link}, nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return snapshot{}, fmt.Errorf("read %s: %w", path, err)
	}
	return snapshot{content: content, mode: info.Mode().Perm()}, nil
}

func hasNameserver(content []byte) bool {
	for line := range strings.SplitSeq(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.EqualFold(fields[0], "nameserver") {
			return true
		}
	}
	return false
}

func validMode(mode string) bool {
	if len(mode) < 3 || len(mode) > 4 {
		return false
	}
	for _, r := range mode {
		if r < '0' || r > '7' {
			return false
		}
	}
	return true
}

func writeState(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, statePerm); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Chmod(tmp, statePerm); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s: %w", path, err)
	}
	return nil
}

func removeState(paths ...string) error {
	var first error
	for _, path := range paths {
		err := os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) && first == nil {
			first = fmt.Errorf("remove %s: %w", path, err)
		}
	}
	return first
}

func stateExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
