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

package resolvconf_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"adgui/commands/resolvconf"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func directRun(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w (%s)", name, err, bytes.TrimSpace(out))
	}
	return nil
}

var _ = Describe("resolv.conf manager", func() {
	var (
		dir    string
		target string
		mgr    *resolvconf.Manager
	)

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
		target = filepath.Join(dir, "resolv.conf")
		mgr = &resolvconf.Manager{
			Target:   target,
			StateDir: filepath.Join(dir, "state"),
			Run:      directRun,
		}
	})

	writeSrc := func(name, content string) string {
		path := filepath.Join(dir, name)
		Expect(os.WriteFile(path, []byte(content), 0o644)).To(Succeed())
		return path
	}

	It("replaces a regular file and restores the original mode", func() {
		original := "nameserver 192.168.0.1\n"
		Expect(os.WriteFile(target, []byte(original), 0o600)).To(Succeed())
		src := writeSrc("vpn.conf", "nameserver 10.8.0.1\n")

		Expect(mgr.Apply(src)).To(Succeed())
		Expect(mgr.Pending()).To(BeTrue())
		Expect(os.ReadFile(target)).To(Equal([]byte("nameserver 10.8.0.1\n")))
		info, err := os.Lstat(target)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Mode() & os.ModeSymlink).To(BeZero())
		Expect(info.Mode().Perm()).To(Equal(os.FileMode(0o644)))

		Expect(mgr.Restore()).To(Succeed())
		Expect(mgr.Pending()).To(BeFalse())
		Expect(os.ReadFile(target)).To(Equal([]byte(original)))
		info, err = os.Lstat(target)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Mode().Perm()).To(Equal(os.FileMode(0o600)))
	})

	It("keeps the original backup when apply runs again", func() {
		original := "nameserver 192.168.0.1\n"
		Expect(os.WriteFile(target, []byte(original), 0o644)).To(Succeed())
		first := writeSrc("vpn-a.conf", "nameserver 10.8.0.1\n")
		second := writeSrc("vpn-b.conf", "nameserver 10.8.0.2\n")

		Expect(mgr.Apply(first)).To(Succeed())
		Expect(mgr.Apply(first)).To(Succeed())
		Expect(mgr.Apply(second)).To(Succeed())
		Expect(os.ReadFile(target)).To(Equal([]byte("nameserver 10.8.0.2\n")))

		Expect(mgr.Restore()).To(Succeed())
		Expect(os.ReadFile(target)).To(Equal([]byte(original)))
	})

	It("backs up an external rewrite and restores that rewrite", func() {
		Expect(os.WriteFile(target, []byte("nameserver 192.168.0.1\n"), 0o644)).To(Succeed())
		src := writeSrc("vpn.conf", "nameserver 10.8.0.1\n")
		Expect(mgr.Apply(src)).To(Succeed())

		rewritten := "nameserver 9.9.9.9\n"
		Expect(os.WriteFile(target, []byte(rewritten), 0o644)).To(Succeed())
		Expect(mgr.Apply(src)).To(Succeed())
		Expect(os.ReadFile(target)).To(Equal([]byte("nameserver 10.8.0.1\n")))

		Expect(mgr.Restore()).To(Succeed())
		Expect(os.ReadFile(target)).To(Equal([]byte(rewritten)))
	})

	It("restores a symlink without changing the symlink target", func() {
		real := filepath.Join(dir, "upstream.conf")
		upstream := "nameserver 192.168.1.1\n"
		Expect(os.WriteFile(real, []byte(upstream), 0o644)).To(Succeed())
		Expect(os.Symlink(real, target)).To(Succeed())
		src := writeSrc("vpn.conf", "nameserver 10.8.0.1\n")

		Expect(mgr.Apply(src)).To(Succeed())
		Expect(os.ReadFile(real)).To(Equal([]byte(upstream)))
		info, err := os.Lstat(target)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Mode() & os.ModeSymlink).To(BeZero())
		Expect(os.ReadFile(target)).To(Equal([]byte("nameserver 10.8.0.1\n")))

		Expect(mgr.Restore()).To(Succeed())
		link, err := os.Readlink(target)
		Expect(err).NotTo(HaveOccurred())
		Expect(link).To(Equal(real))
		Expect(os.ReadFile(real)).To(Equal([]byte(upstream)))
		Expect(os.ReadFile(target)).To(Equal([]byte(upstream)))
	})

	It("leaves an externally replaced file in place and drops the backup", func() {
		Expect(os.WriteFile(target, []byte("nameserver 192.168.0.1\n"), 0o644)).To(Succeed())
		src := writeSrc("vpn.conf", "nameserver 10.8.0.1\n")
		Expect(mgr.Apply(src)).To(Succeed())

		external := "nameserver 8.8.8.8\n"
		Expect(os.WriteFile(target, []byte(external), 0o644)).To(Succeed())
		Expect(mgr.Restore()).To(Succeed())
		Expect(mgr.Pending()).To(BeFalse())
		Expect(os.ReadFile(target)).To(Equal([]byte(external)))
	})

	It("rejects a source without a nameserver", func() {
		original := "nameserver 192.168.0.1\n"
		Expect(os.WriteFile(target, []byte(original), 0o644)).To(Succeed())
		src := writeSrc("empty.conf", "# no servers\n")

		err := mgr.Apply(src)
		Expect(errors.Is(err, resolvconf.ErrNoNameserver)).To(BeTrue())
		Expect(mgr.Pending()).To(BeFalse())
		Expect(os.ReadFile(target)).To(Equal([]byte(original)))
	})

	It("removes the target on restore when it did not exist before apply", func() {
		src := writeSrc("vpn.conf", "nameserver 10.8.0.1\n")
		Expect(mgr.Apply(src)).To(Succeed())
		Expect(mgr.Pending()).To(BeTrue())
		_, err := os.Lstat(target)
		Expect(err).NotTo(HaveOccurred())

		Expect(mgr.Restore()).To(Succeed())
		_, err = os.Lstat(target)
		Expect(os.IsNotExist(err)).To(BeTrue())
		Expect(mgr.Pending()).To(BeFalse())
	})
})
