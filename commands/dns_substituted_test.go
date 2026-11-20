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

package commands

import (
	"os"
	"path/filepath"
	"testing"

	"adgui/commands/resolvconf"
)

func TestDNSSubstituted(t *testing.T) {
	dir := t.TempDir()
	mgr := &VPNManager{
		dns: &resolvconf.Manager{StateDir: dir},
	}

	t.Setenv("ADGUARD_MANAGE_DNS", "1")
	if mgr.DNSSubstituted() {
		t.Fatal("DNSSubstituted() = true without a resolv.conf backup")
	}

	// resolv.conf.original is the backup name Pending() treats as ownership.
	backup := filepath.Join(dir, "resolv.conf.original")
	if err := os.WriteFile(backup, []byte("nameserver 1.1.1.1\n"), 0o600); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	if !mgr.DNSSubstituted() {
		t.Fatal("DNSSubstituted() = false with a backup and DNS management enabled")
	}

	t.Setenv("ADGUARD_MANAGE_DNS", "0")
	if mgr.DNSSubstituted() {
		t.Fatal("DNSSubstituted() = true while ADGUARD_MANAGE_DNS is off")
	}
}
