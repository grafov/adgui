package commands_test

import (
	"adgui/commands"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Location bookmarks persistence", func() {
	var tempHome string
	var oldHome string
	var oldXDG string

	BeforeEach(func() {
		var err error
		tempHome, err = os.MkdirTemp("", "adgui-bookmarks-home-*")
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

	It("stores bookmarks under ~/.local/share/adgui/location-bookmarks", func() {
		path, err := commands.GetLocationBookmarksPath()
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(filepath.Join(tempHome, ".local", "share", "adgui", "location-bookmarks")))
	})

	It("respects XDG_DATA_HOME when set", func() {
		xdgDir := filepath.Join(tempHome, "custom-data")
		Expect(os.Setenv("XDG_DATA_HOME", xdgDir)).To(Succeed())

		path, err := commands.GetLocationBookmarksPath()
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(filepath.Join(xdgDir, "adgui", "location-bookmarks")))
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
