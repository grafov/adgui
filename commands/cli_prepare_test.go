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
	"os/exec"
	"testing"
)

func TestPrepareCLICommand(t *testing.T) {
	cmd := exec.Command("true")
	prepareCLICommand(cmd)

	if cmd.Stdin != nil {
		t.Fatal("expected Stdin to be nil")
	}
	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be set")
	}
	if !cmd.SysProcAttr.Setsid {
		t.Fatal("expected Setsid to be true")
	}
}

func TestPrepareCLICommandNilSafe(t *testing.T) {
	prepareCLICommand(nil)
}
