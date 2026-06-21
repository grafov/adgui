package commands_test

import (
	"adgui/commands"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Connection history persistence", func() {
	var tempHome string
	var oldHome string
	var oldXDG string

	BeforeEach(func() {
		var err error
		tempHome, err = os.MkdirTemp("", "adgui-history-home-*")
		Expect(err).NotTo(HaveOccurred())
		oldHome = os.Getenv("HOME")
		oldXDG = os.Getenv("XDG_DATA_HOME")
		Expect(os.Setenv("HOME", tempHome)).To(Succeed())
		Expect(os.Unsetenv("XDG_DATA_HOME")).To(Succeed())
	})

	AfterEach(func() {
		if oldHome != "" {
			_ = os.Setenv("HOME", oldHome)
		}
		if oldXDG != "" {
			_ = os.Setenv("XDG_DATA_HOME", oldXDG)
		} else {
			_ = os.Unsetenv("XDG_DATA_HOME")
		}
		_ = os.RemoveAll(tempHome)
	})

	It("stores history under ~/.local/share/adgui/connections-history", func() {
		path, err := commands.GetConnectionsHistoryPath()
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(filepath.Join(tempHome, ".local", "share", "adgui", "connections-history")))
	})

	It("respects XDG_DATA_HOME when set", func() {
		xdgDir := filepath.Join(tempHome, "custom-data")
		Expect(os.Setenv("XDG_DATA_HOME", xdgDir)).To(Succeed())

		path, err := commands.GetConnectionsHistoryPath()
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(filepath.Join(xdgDir, "adgui", "connections-history")))
	})

	It("loads and saves up to 12 entries", func() {
		now := time.Now().UTC().Truncate(time.Second)
		ended := now.Add(time.Hour)
		entries := make([]commands.ConnectionHistoryEntry, 0, 13)
		for i := range 13 {
			entries = append(entries, commands.ConnectionHistoryEntry{
				City:      "City",
				Country:   "Country",
				Ping:      10 + i,
				StartedAt: now.Add(time.Duration(i) * time.Minute),
				EndedAt:   &ended,
			})
		}

		Expect(commands.SaveConnectionHistory(entries)).To(Succeed())

		loaded, err := commands.LoadConnectionHistory()
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded).To(HaveLen(12))
		Expect(loaded[0].Ping).To(Equal(10))
		Expect(loaded[11].Ping).To(Equal(21))
	})
})

var _ = Describe("ParseLocationFromStatus", func() {
	It("extracts location from ANSI status output", func() {
		output := "Connected to \x1b[1mFRANKFURT\x1b[0m in \x1b[1mTUN\x1b[0m mode, running on \x1b[1mtun0\x1b[0m\n"
		Expect(commands.ParseLocationFromStatus(output)).To(Equal("FRANKFURT"))
	})
})
