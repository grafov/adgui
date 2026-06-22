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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Exclusions Persistence and Helpers", func() {
	Context("when normalizing domains", func() {
		It("should trim spaces and remove empty domains", func() {
			input := []string{"  example.com  ", "", "   ", "google.com"}
			expected := []string{"example.com", "google.com"}
			Expect(commands.NormalizeDomains(input)).To(Equal(expected))
		})

		It("should deduplicate domains case-insensitively and preserve the first casing", func() {
			input := []string{"Example.Com", "example.com", "GOOGLE.COM", "google.com", "Example.com"}
			expected := []string{"Example.Com", "GOOGLE.COM"}
			Expect(commands.NormalizeDomains(input)).To(Equal(expected))
		})
	})

	Context("when saving and loading exclusions", func() {
		var tempHome string
		var oldHome string

		BeforeEach(func() {
			var err error
			tempHome, err = os.MkdirTemp("", "adgui-test-home-*")
			Expect(err).NotTo(HaveOccurred())

			oldHome = os.Getenv("HOME")
			err = os.Setenv("HOME", tempHome)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if oldHome != "" {
				_ = os.Setenv("HOME", oldHome)
			} else {
				_ = os.Unsetenv("HOME")
			}
			_ = os.RemoveAll(tempHome)
		})

		It("should return empty list and no error if file does not exist", func() {
			domains, err := commands.LoadExclusionsForMode(commands.SiteExclusionModeGeneral)
			Expect(err).NotTo(HaveOccurred())
			Expect(domains).To(BeEmpty())
		})

		It("should save and load general mode exclusions", func() {
			input := []string{"example.com", "github.com", "  ", "Example.com"}
			err := commands.SaveExclusionsForMode(commands.SiteExclusionModeGeneral, input)
			Expect(err).NotTo(HaveOccurred())

			expectedDir := filepath.Join(tempHome, ".config", "adgui", "site-exclusions")
			Expect(expectedDir).To(BeADirectory())

			expectedFile := filepath.Join(expectedDir, "general.txt")
			Expect(expectedFile).To(BeAnExistingFile())

			content, err := os.ReadFile(expectedFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("example.com\ngithub.com\n"))

			loaded, err := commands.LoadExclusionsForMode(commands.SiteExclusionModeGeneral)
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded).To(Equal([]string{"example.com", "github.com"}))
		})

		It("should save and load selective mode exclusions", func() {
			input := []string{"youtube.com", "reddit.com"}
			err := commands.SaveExclusionsForMode(commands.SiteExclusionModeSelective, input)
			Expect(err).NotTo(HaveOccurred())

			expectedFile := filepath.Join(tempHome, ".config", "adgui", "site-exclusions", "selective.txt")
			Expect(expectedFile).To(BeAnExistingFile())

			loaded, err := commands.LoadExclusionsForMode(commands.SiteExclusionModeSelective)
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded).To(Equal(input))
		})

		It("should migrate legacy general mode exclusions on first load", func() {
			legacyDir := filepath.Join(tempHome, ".local", "share", "adgui", "site-exclusions")
			err := os.MkdirAll(legacyDir, 0o755)
			Expect(err).NotTo(HaveOccurred())

			legacyFile := filepath.Join(legacyDir, "general.txt")
			err = os.WriteFile(legacyFile, []byte("legacy.com\nExample.Com\n"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			loaded, err := commands.LoadExclusionsForMode(commands.SiteExclusionModeGeneral)
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded).To(Equal([]string{"legacy.com", "Example.Com"}))

			newFile := filepath.Join(tempHome, ".config", "adgui", "site-exclusions", "general.txt")
			Expect(newFile).To(BeAnExistingFile())
			Expect(legacyFile).To(BeAnExistingFile())

			content, err := os.ReadFile(newFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("legacy.com\nExample.Com\n"))
		})
	})
})
