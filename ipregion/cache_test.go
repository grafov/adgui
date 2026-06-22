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

package ipregion_test

import (
	"os"
	"path/filepath"
	"time"

	"adgui/ipregion"
	"adgui/locations"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Region IP cache", func() {
	var tempHome string
	var oldHome string

	BeforeEach(func() {
		var err error
		tempHome, err = os.MkdirTemp("", "adgui-cache-test-*")
		Expect(err).NotTo(HaveOccurred())

		oldHome = os.Getenv("HOME")
		Expect(os.Setenv("HOME", tempHome)).To(Succeed())
		Expect(os.Unsetenv("XDG_CACHE_HOME")).To(Succeed())
	})

	AfterEach(func() {
		if oldHome != "" {
			_ = os.Setenv("HOME", oldHome)
		} else {
			_ = os.Unsetenv("HOME")
		}
		_ = os.RemoveAll(tempHome)
	})

	It("should build cache keys for connected and vpn-off states", func() {
		loc := locations.Location{ISO: "DE", City: "Frankfurt"}
		Expect(ipregion.CacheKeyForState(loc, true)).To(Equal("region-ip.de.frankfurt"))
		Expect(ipregion.CacheKeyForState(loc, false)).To(Equal("region-ip.vpn-off"))
	})

	It("should sanitize city names with spaces", func() {
		loc := locations.Location{ISO: "US", City: "New York"}
		Expect(ipregion.CacheKeyForState(loc, true)).To(Equal("region-ip.us.new-york"))
	})

	It("should save and load cached reports with checked time", func() {
		loc := locations.Location{ISO: "LV", City: "Riga"}
		key := ipregion.CacheKeyForState(loc, true)
		checkedAt := time.Date(2026, 6, 22, 15, 30, 0, 0, time.UTC)
		report := &ipregion.Report{
			ExternalIPv4: "1.2.3.4",
			Results: []ipregion.ServiceResult{
				{Service: "test", IPv4: "LV", IPv6: "LV"},
			},
		}

		err := ipregion.SaveCacheForState(loc, true, report, checkedAt)
		Expect(err).NotTo(HaveOccurred())

		expectedPath := filepath.Join(tempHome, ".cache", "adgui", key)
		Expect(expectedPath).To(BeAnExistingFile())

		loaded, err := ipregion.LoadCacheForState(loc, true)
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded).NotTo(BeNil())
		Expect(loaded.CheckedAt.Equal(checkedAt)).To(BeTrue())
		Expect(loaded.ISO).To(Equal("LV"))
		Expect(loaded.Location).To(Equal("Riga"))
		Expect(loaded.VPNOff).To(BeFalse())
		Expect(loaded.Report.ExternalIPv4).To(Equal("1.2.3.4"))
		Expect(loaded.Report.Results).To(HaveLen(1))
	})

	It("should save vpn-off cache separately", func() {
		checkedAt := time.Now().UTC().Truncate(time.Second)
		report := &ipregion.Report{ExternalIPv4: "9.9.9.9"}

		err := ipregion.SaveCacheForState(locations.Location{}, false, report, checkedAt)
		Expect(err).NotTo(HaveOccurred())

		loaded, err := ipregion.LoadCacheForState(locations.Location{}, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded).NotTo(BeNil())
		Expect(loaded.VPNOff).To(BeTrue())
		Expect(loaded.Report.ExternalIPv4).To(Equal("9.9.9.9"))
	})

	It("should return nil when cache file does not exist", func() {
		loc := locations.Location{ISO: "PL", City: "Warsaw"}
		loaded, err := ipregion.LoadCacheForState(loc, true)
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded).To(BeNil())
	})

	It("should clear only region-ip cache files", func() {
		loc := locations.Location{ISO: "DE", City: "Berlin"}
		report := &ipregion.Report{ExternalIPv4: "1.1.1.1"}
		Expect(ipregion.SaveCacheForState(loc, true, report, time.Now())).To(Succeed())
		Expect(ipregion.SaveCacheForState(locations.Location{}, false, report, time.Now())).To(Succeed())

		cacheDir, err := ipregion.GetCacheDir()
		Expect(err).NotTo(HaveOccurred())
		otherFile := filepath.Join(cacheDir, "other-data.txt")
		Expect(os.WriteFile(otherFile, []byte("keep"), 0o644)).To(Succeed())

		Expect(ipregion.ClearCache()).To(Succeed())
		Expect(otherFile).To(BeAnExistingFile())
		Expect(filepath.Join(cacheDir, "region-ip.de.berlin")).NotTo(BeAnExistingFile())
		Expect(filepath.Join(cacheDir, "region-ip.vpn-off")).NotTo(BeAnExistingFile())
	})
})
