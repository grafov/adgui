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
	"adgui/locations"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Location bookmarks persistence", func() {
	var tempHome string
	var oldHome string

	BeforeEach(func() {
		var err error
		tempHome, err = os.MkdirTemp("", "adgui-bookmarks-home-*")
		Expect(err).NotTo(HaveOccurred())
		oldHome = os.Getenv("HOME")
		Expect(os.Setenv("HOME", tempHome)).To(Succeed())
	})

	AfterEach(func() {
		if oldHome != "" {
			_ = os.Setenv("HOME", oldHome)
		}
		_ = os.RemoveAll(tempHome)
	})

	It("stores bookmarks under ~/.config/adgui/bookmarks", func() {
		path, err := commands.GetLocationBookmarksPath()
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(filepath.Join(tempHome, ".config", "adgui", "bookmarks")))
	})

	It("returns empty slice when file is missing", func() {
		loaded, err := commands.LoadLocationBookmarks()
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded).To(BeEmpty())
	})

	It("loads and saves bookmarks", func() {
		bookmarks := []commands.LocationBookmark{
			{ISO: "DE", Country: "Germany", City: "Frankfurt"},
			{ISO: "LV", Country: "Latvia", City: "Riga"},
		}

		Expect(commands.SaveLocationBookmarks(bookmarks)).To(Succeed())

		loaded, err := commands.LoadLocationBookmarks()
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded).To(HaveLen(2))
		Expect(loaded[0].City).To(Equal("Frankfurt"))
		Expect(loaded[1].City).To(Equal("Riga"))
	})

	It("deduplicates bookmarks by stable key", func() {
		bookmarks := []commands.LocationBookmark{
			{ISO: "DE", Country: "Germany", City: "Frankfurt"},
			{ISO: "de", Country: "germany", City: "frankfurt"},
		}

		Expect(commands.SaveLocationBookmarks(bookmarks)).To(Succeed())

		loaded, err := commands.LoadLocationBookmarks()
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded).To(HaveLen(1))
	})

	It("builds a lookup set from bookmarks", func() {
		bookmarks := []commands.LocationBookmark{
			{ISO: "US", Country: "United States", City: "New York"},
		}
		set := commands.LocationBookmarkSet(bookmarks)
		key := commands.LocationBookmarkKey("US", "United States", "New York")
		Expect(set).To(HaveKey(key))
	})

	It("uses stable keys independent of ping changes", func() {
		key1 := commands.LocationBookmarkKey("DE", "Germany", "Frankfurt")
		key2 := commands.LocationBookmarkKey("DE", "Germany", "Frankfurt")
		Expect(key1).To(Equal(key2))
	})
})

var _ = Describe("PruneAndSaveLocationBookmarks", func() {
	var tempHome string
	var oldHome string

	BeforeEach(func() {
		var err error
		tempHome, err = os.MkdirTemp("", "adgui-bookmarks-prune-*")
		Expect(err).NotTo(HaveOccurred())
		oldHome = os.Getenv("HOME")
		Expect(os.Setenv("HOME", tempHome)).To(Succeed())
	})

	AfterEach(func() {
		if oldHome != "" {
			_ = os.Setenv("HOME", oldHome)
		}
		_ = os.RemoveAll(tempHome)
	})

	It("keeps all bookmarks when every entry matches a location", func() {
		bookmarks := []commands.LocationBookmark{
			{ISO: "DE", Country: "Germany", City: "Frankfurt"},
			{ISO: "LV", Country: "Latvia", City: "Riga"},
		}
		Expect(commands.SaveLocationBookmarks(bookmarks)).To(Succeed())

		locs := []locations.Location{
			{ISO: "DE", Country: "Germany", City: "Frankfurt", Ping: 37},
			{ISO: "LV", Country: "Latvia", City: "Riga", Ping: 29},
			{ISO: "US", Country: "United States", City: "New York", Ping: 121},
		}

		pruned, err := commands.PruneAndSaveLocationBookmarks(bookmarks, locs)
		Expect(err).NotTo(HaveOccurred())
		Expect(pruned).To(HaveLen(2))

		loaded, err := commands.LoadLocationBookmarks()
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded).To(HaveLen(2))
	})

	It("removes stale bookmarks and rewrites the file", func() {
		bookmarks := []commands.LocationBookmark{
			{ISO: "DE", Country: "Germany", City: "Frankfurt"},
			{ISO: "XX", Country: "Removed", City: "Old City"},
		}
		Expect(commands.SaveLocationBookmarks(bookmarks)).To(Succeed())

		locs := []locations.Location{
			{ISO: "DE", Country: "Germany", City: "Frankfurt", Ping: 37},
		}

		pruned, err := commands.PruneAndSaveLocationBookmarks(bookmarks, locs)
		Expect(err).NotTo(HaveOccurred())
		Expect(pruned).To(HaveLen(1))
		Expect(pruned[0].City).To(Equal("Frankfurt"))

		loaded, err := commands.LoadLocationBookmarks()
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded).To(HaveLen(1))
		Expect(loaded[0].City).To(Equal("Frankfurt"))
	})

	It("does not modify bookmarks when the location list is empty", func() {
		bookmarks := []commands.LocationBookmark{
			{ISO: "DE", Country: "Germany", City: "Frankfurt"},
		}
		Expect(commands.SaveLocationBookmarks(bookmarks)).To(Succeed())

		pruned, err := commands.PruneAndSaveLocationBookmarks(bookmarks, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(pruned).To(Equal(bookmarks))

		loaded, err := commands.LoadLocationBookmarks()
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded).To(HaveLen(1))
		Expect(loaded[0].City).To(Equal("Frankfurt"))
	})
})
