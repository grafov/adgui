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

package sudowrap_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"adgui/commands/sudowrap"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sudowrap", func() {
	var originalPath string

	BeforeEach(func() {
		originalPath = os.Getenv("PATH")
	})

	AfterEach(func() {
		_ = os.Setenv("PATH", originalPath)
	})

	It("does not change parent PATH after Setup", func() {
		env, err := sudowrap.Setup(true)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = env.Close() }()

		Expect(os.Getenv("PATH")).To(Equal(originalPath))
	})

	It("creates private runtime dir with resilient sudo and askpass wrappers", func() {
		env, err := sudowrap.Setup(true)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = env.Close() }()

		dir := env.Dir()
		Expect(dir).NotTo(BeEmpty())

		info, err := os.Stat(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.IsDir()).To(BeTrue())
		Expect(info.Mode().Perm()).To(Equal(os.FileMode(0o700)))

		sudoPath := filepath.Join(dir, "sudo")
		sudoInfo, err := os.Stat(sudoPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(sudoInfo.Mode().Perm() & 0o777).To(Equal(os.FileMode(0o700)))

		sudoContent, err := os.ReadFile(sudoPath)
		Expect(err).NotTo(HaveOccurred())
		sudoScript := string(sudoContent)
		Expect(sudoScript).To(ContainSubstring("-n -v"))
		Expect(sudoScript).To(ContainSubstring(" -A \"$@\""))
		Expect(sudoScript).To(ContainSubstring("[ ! -f \"$PASS\" ]"))
		Expect(sudoScript).To(ContainSubstring("password required but not provided"))

		askpassContent, err := os.ReadFile(filepath.Join(dir, "askpass"))
		Expect(err).NotTo(HaveOccurred())
		askpassScript := string(askpassContent)
		Expect(askpassScript).To(ContainSubstring("[ -f \"$PASS\" ] || exit 1"))
		Expect(askpassScript).To(ContainSubstring("cat \"$PASS\""))
	})

	It("Apply filters terminal env and keeps desktop/XDG vars", func() {
		env, err := sudowrap.Setup(true)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = env.Close() }()

		keys := []string{
			"TERM", "COLORTERM", "KONSOLE_VERSION", "SSH_TTY", "SHELL",
			"SUDO_ASKPASS", "HOME", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS",
			"WAYLAND_DISPLAY", "LC_MESSAGES",
		}
		old := map[string]*string{}
		for _, key := range keys {
			if val, ok := os.LookupEnv(key); ok {
				v := val
				old[key] = &v
			} else {
				old[key] = nil
			}
		}
		defer func() {
			for key, val := range old {
				if val == nil {
					_ = os.Unsetenv(key)
				} else {
					_ = os.Setenv(key, *val)
				}
			}
		}()

		Expect(os.Setenv("TERM", "xterm-256color")).To(Succeed())
		Expect(os.Setenv("COLORTERM", "truecolor")).To(Succeed())
		Expect(os.Setenv("KONSOLE_VERSION", "230804")).To(Succeed())
		Expect(os.Setenv("SSH_TTY", "/dev/pts/9")).To(Succeed())
		Expect(os.Setenv("SHELL", "/bin/bash")).To(Succeed())
		Expect(os.Setenv("SUDO_ASKPASS", "/tmp/old-askpass")).To(Succeed())
		Expect(os.Setenv("HOME", "/home/testuser")).To(Succeed())
		Expect(os.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")).To(Succeed())
		Expect(os.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/tmp/dbus")).To(Succeed())
		Expect(os.Setenv("WAYLAND_DISPLAY", "wayland-0")).To(Succeed())
		Expect(os.Setenv("LC_MESSAGES", "eo")).To(Succeed())

		cmd := exec.Command("sh", "-c", strings.Join([]string{
			`printf 'PATH=%s\n' "$PATH"`,
			`printf 'ASKPASS=%s\n' "$SUDO_ASKPASS"`,
			`printf 'HOME=%s\n' "$HOME"`,
			`printf 'XDG_RUNTIME_DIR=%s\n' "$XDG_RUNTIME_DIR"`,
			`printf 'DBUS=%s\n' "$DBUS_SESSION_BUS_ADDRESS"`,
			`printf 'WAYLAND=%s\n' "$WAYLAND_DISPLAY"`,
			`printf 'LC_MESSAGES=%s\n' "$LC_MESSAGES"`,
			`printf 'TERM=%s\n' "$TERM"`,
			`printf 'COLORTERM=%s\n' "$COLORTERM"`,
			`printf 'KONSOLE=%s\n' "$KONSOLE_VERSION"`,
			`printf 'SSH_TTY=%s\n' "$SSH_TTY"`,
			`printf 'SHELL=%s\n' "$SHELL"`,
		}, "\n"))
		env.Apply(cmd)

		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())
		text := string(output)

		Expect(text).To(ContainSubstring("PATH=" + env.Dir() + string(os.PathListSeparator)))
		Expect(text).To(ContainSubstring("ASKPASS=" + filepath.Join(env.Dir(), "askpass")))
		Expect(text).To(ContainSubstring("HOME=/home/testuser"))
		Expect(text).To(ContainSubstring("XDG_RUNTIME_DIR=/run/user/1000"))
		Expect(text).To(ContainSubstring("DBUS=unix:path=/tmp/dbus"))
		Expect(text).To(ContainSubstring("WAYLAND=wayland-0"))
		Expect(text).To(ContainSubstring("LC_MESSAGES=eo"))

		Expect(text).To(ContainSubstring("TERM=\n"))
		Expect(text).To(ContainSubstring("COLORTERM=\n"))
		Expect(text).To(ContainSubstring("KONSOLE=\n"))
		Expect(text).To(ContainSubstring("SSH_TTY=\n"))
		Expect(text).To(ContainSubstring("SHELL=\n"))
		Expect(text).NotTo(ContainSubstring("/tmp/old-askpass"))
	})

	It("ReadyForAskpass requires both memory password and .pass file", func() {
		env, err := sudowrap.Setup(true)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = env.Close() }()

		Expect(env.ReadyForAskpass()).To(BeFalse())
		Expect(env.PassFileExists()).To(BeFalse())

		Expect(os.WriteFile(filepath.Join(env.Dir(), ".pass"), []byte("secret\n"), 0o600)).To(Succeed())
		Expect(env.PassFileExists()).To(BeTrue())
		Expect(env.ReadyForAskpass()).To(BeFalse()) // memory still empty
	})

	It("Close removes runtime dir and clears password", func() {
		env, err := sudowrap.Setup(true)
		Expect(err).NotTo(HaveOccurred())

		dir := env.Dir()
		Expect(os.WriteFile(filepath.Join(dir, ".pass"), []byte("secret\n"), 0o600)).To(Succeed())

		Expect(env.Close()).To(Succeed())
		_, err = os.Stat(dir)
		Expect(os.IsNotExist(err)).To(BeTrue())
		Expect(env.HasPassword()).To(BeFalse())
		Expect(env.PassFileExists()).To(BeFalse())
	})

	It("disabled Setup returns env without runtime dir", func() {
		env, err := sudowrap.Setup(false)
		Expect(err).NotTo(HaveOccurred())
		Expect(env.Enabled()).To(BeFalse())
		Expect(env.Dir()).To(BeEmpty())
	})
})
