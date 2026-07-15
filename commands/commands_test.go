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

package commands_test

import (
	"adgui/commands"
	"os"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Location Parsing from Status", func() {
	Context("when parsing location with ANSI codes", func() {
		It("should correctly extract location FRANKFURT from status output", func() {
			testOutput := "Connected to \x1b[1mFRANKFURT\x1b[0m in \x1b[1mTUN\x1b[0m mode, running on \x1b[1mtun0\x1b[0m\n" +
				"Warning: System DNS could not be configured. DNS queries may bypass the VPN tunnel\n" +
				"You can disconnect by running `/opt/adguardvpn_cli/adguardvpn-cli disconnect`\n"

			Expect(commands.ParseLocationFromStatus(testOutput)).To(Equal("FRANKFURT"))
		})

		It("should correctly extract location NEW YORK from status output", func() {
			testOutput := "Connected to \x1b[1mNEW YORK\x1b[0m in \x1b[1mTUN\x1b[0m mode, running on \x1b[1mtun0\x1b[0m\n" +
				"Warning: System DNS could not be configured. DNS queries may bypass the VPN tunnel\n"

			Expect(commands.ParseLocationFromStatus(testOutput)).To(Equal("NEW YORK"))
		})
	})
})

var _ = Describe("Command Queue Tracking and Killing", func() {
	var (
		tempScriptPath string
		oldAdguardCmd  string
	)

	BeforeEach(func() {
		// Create a temporary executable shell script that sleeps
		f, err := os.CreateTemp("", "fake-adguard-*.sh")
		Expect(err).NotTo(HaveOccurred())
		_, err = f.WriteString("#!/bin/sh\nexec sleep 10\n")
		Expect(err).NotTo(HaveOccurred())
		err = f.Close()
		Expect(err).NotTo(HaveOccurred())

		err = os.Chmod(f.Name(), 0755)
		Expect(err).NotTo(HaveOccurred())

		tempScriptPath = f.Name()

		// Save old ADGUARD_CMD env and set to our temp script
		oldAdguardCmd = os.Getenv("ADGUARD_CMD")
		err = os.Setenv("ADGUARD_CMD", tempScriptPath)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Clean up env and temporary script
		if oldAdguardCmd != "" {
			_ = os.Setenv("ADGUARD_CMD", oldAdguardCmd)
		} else {
			_ = os.Unsetenv("ADGUARD_CMD")
		}
		_ = os.Remove(tempScriptPath)
	})

	Context("when executing long-running CLI command", func() {
		It("should register command in queue, notify callbacks, and remove on completion after kill", func() {
			mgr := commands.New()

			var callbackCount int32
			mgr.SetCommandQueueChangeCallback(func() {
				atomic.AddInt32(&callbackCount, 1)
			})

			// Initially queue should be empty
			Expect(mgr.RunningCommands()).To(BeEmpty())

			// Run License() in a goroutine because our fake script sleeps
			errChan := make(chan error, 1)
			go func() {
				_ = mgr.License()
				errChan <- nil
			}()

			// Give process a small amount of time to start and register
			Eventually(func() []commands.RunningCommand {
				return mgr.RunningCommands()
			}, 2*time.Second, 10*time.Millisecond).Should(HaveLen(1))

			running := mgr.RunningCommands()
			Expect(running).To(HaveLen(1))
			cmd := running[0]
			Expect(cmd.Path).To(Equal(tempScriptPath))
			Expect(cmd.Args).To(ConsistOf("license"))
			Expect(cmd.PID).To(BeNumerically(">", 0))

			// Verify callback was triggered at least once
			Expect(atomic.LoadInt32(&callbackCount)).To(BeNumerically(">", 0))

			// Kill the command
			err := mgr.KillCommand(cmd.ID)
			Expect(err).NotTo(HaveOccurred())

			// Wait for process to terminate and clean up from registry
			Eventually(func() []commands.RunningCommand {
				return mgr.RunningCommands()
			}, 2*time.Second, 10*time.Millisecond).Should(BeEmpty())

			// Make sure License goroutine has exited
			Eventually(errChan, 2*time.Second).Should(Receive())
		})

		It("should terminate all commands when KillAllCommands is called", func() {
			mgr := commands.New()

			// Run License() in a goroutine
			go func() { _ = mgr.License() }()

			Eventually(func() []commands.RunningCommand {
				return mgr.RunningCommands()
			}, 2*time.Second, 10*time.Millisecond).Should(HaveLen(1))

			mgr.KillAllCommands()

			Eventually(func() []commands.RunningCommand {
				return mgr.RunningCommands()
			}, 2*time.Second, 10*time.Millisecond).Should(BeEmpty())
		})
	})

	Context("when sudo wrapper is enabled", func() {
		var (
			tempScriptPath string
			oldAdguardCmd  string
			oldSudoWrap    string
		)

		BeforeEach(func() {
			f, err := os.CreateTemp("", "fake-adguard-wrap-*.sh")
			Expect(err).NotTo(HaveOccurred())
			script := "#!/bin/sh\n" +
				"printf 'PATH:%s\\n' \"$PATH\"\n" +
				"printf 'ASKPASS:%s\\n' \"$SUDO_ASKPASS\"\n" +
				"printf 'TERM:%s\\n' \"$TERM\"\n" +
				"command -v sudo\n"
			_, err = f.WriteString(script)
			Expect(err).NotTo(HaveOccurred())
			err = f.Close()
			Expect(err).NotTo(HaveOccurred())
			err = os.Chmod(f.Name(), 0o755)
			Expect(err).NotTo(HaveOccurred())
			tempScriptPath = f.Name()

			oldAdguardCmd = os.Getenv("ADGUARD_CMD")
			oldSudoWrap = os.Getenv("ADGUARD_SUDO_WRAP")
			Expect(os.Setenv("ADGUARD_CMD", tempScriptPath)).To(Succeed())
			Expect(os.Setenv("ADGUARD_SUDO_WRAP", "1")).To(Succeed())
		})

		AfterEach(func() {
			if oldAdguardCmd != "" {
				_ = os.Setenv("ADGUARD_CMD", oldAdguardCmd)
			} else {
				_ = os.Unsetenv("ADGUARD_CMD")
			}
			if oldSudoWrap != "" {
				_ = os.Setenv("ADGUARD_SUDO_WRAP", oldSudoWrap)
			} else {
				_ = os.Unsetenv("ADGUARD_SUDO_WRAP")
			}
			_ = os.Remove(tempScriptPath)
		})

		It("should inject private sudo wrapper only into child CLI environment", func() {
			originalPath := os.Getenv("PATH")
			_ = os.Setenv("TERM", "xterm-test")
			mgr := commands.New()
			defer func() { _ = mgr.Close() }()

			output := mgr.License()
			Expect(os.Getenv("PATH")).To(Equal(originalPath))
			Expect(output).To(ContainSubstring("ASKPASS:"))
			Expect(output).To(Or(
				ContainSubstring("/adgui/"),
				ContainSubstring("adgui-sudo"),
			))
			Expect(output).To(ContainSubstring("sudo"))
			Expect(output).To(ContainSubstring("TERM:\n"))
		})
	})
})
