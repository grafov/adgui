package commands_test

import (
	"adgui/commands"
	"os"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Location Parsing from Status", func() {
	Context("when parsing location with ANSI codes", func() {
		It("should correctly extract location FRANKFURT from status output", func() {
			// Тестовый вывод статуса с ANSI кодами
			testOutput := "Connected to \x1b[1mFRANKFURT\x1b[0m in \x1b[1mTUN\x1b[0m mode, running on \x1b[1mtun0\x1b[0m\n" +
				"Warning: System DNS could not be configured. DNS queries may bypass the VPN tunnel\n" +
				"You can disconnect by running `/opt/adguardvpn_cli/adguardvpn-cli disconnect`\n"

			// Применяем ту же логику, что и в checkStatus()
			expectedLocation := "FRANKFURT"

			// Применяем ту же логику, что и в основном коде
			location := testOutput
			prefix := "Connected to "
			if idx := strings.Index(location, prefix); idx >= 0 {
				location = location[idx+len(prefix):]
			}
			// Удаляем ANSI коды
			location = strings.ReplaceAll(location, "\x1b[1m", "")
			location = strings.ReplaceAll(location, "\x1b[0m", "")
			// Удаляем суффикс
			if idx := strings.Index(location, " in "); idx >= 0 {
				location = location[:idx]
			}
			// Очищаем от пробелов
			location = strings.TrimSpace(location)

			Expect(location).To(Equal(expectedLocation))
		})

		It("should correctly extract location NEW YORK from status output", func() {
			// Тестовый вывод статуса с другой локацией
			testOutput := "Connected to \x1b[1mNEW YORK\x1b[0m in \x1b[1mTUN\x1b[0m mode, running on \x1b[1mtun0\x1b[0m\n" +
				"Warning: System DNS could not be configured. DNS queries may bypass the VPN tunnel\n"

			expectedLocation := "NEW YORK"

			// Применяем ту же логику, что и в checkStatus()
			location := testOutput
			prefix := "Connected to "
			if idx := strings.Index(location, prefix); idx >= 0 {
				location = location[idx+len(prefix):]
			}
			// Удаляем ANSI коды
			location = strings.ReplaceAll(location, "\x1b[1m", "")
			location = strings.ReplaceAll(location, "\x1b[0m", "")
			// Удаляем суффикс
			if idx := strings.Index(location, " in "); idx >= 0 {
				location = location[:idx]
			}
			// Очищаем от пробелов
			location = strings.TrimSpace(location)

			Expect(location).To(Equal(expectedLocation))
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
})
