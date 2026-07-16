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

package main

import (
	"adgui/commands"
	"adgui/config"
	"adgui/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
)

var (
	version   = "v0.0.0-dev"
	gitCommit = "unknown"
)

func main() {
	if err := ui.LoadTranslations(); err != nil {
		fyne.LogError("failed to load translations", err)
	}
	if err := config.EnsureAdguirc(
		lang.X(
			"config.adguirc.header",
			"This config was created by adgui with default values.\n"+
				"Uncomment the keys and set the values you need.\n"+
				"If a variable is not in this file, it is read from the environment.\n"+
				"If it is also missing from the environment, the default value from the code is used.",
		),
		map[string]string{
			"ADGUARD_CMD": lang.X(
				"config.adguirc.ADGUARD_CMD",
				"Path to adguardvpn-cli. Example: /usr/bin/adguardvpn-cli",
			),
			"ADGUARD_KILL_CMD": lang.X(
				"config.adguirc.ADGUARD_KILL_CMD",
				"Optional kill command prefix; PID is appended. Example: /usr/bin/sudo -n kill -TERM. Empty uses SIGTERM/Kill.",
			),
			"ADGUARD_SUDO_WRAP": lang.X(
				"config.adguirc.ADGUARD_SUDO_WRAP",
				"Inject private sudo PATH wrapper. Values: true, false (also 1/0, yes/no, on/off).",
			),
			"ADGUARD_SUDO_ASKPASS": lang.X(
				"config.adguirc.ADGUARD_SUDO_ASKPASS",
				"Show GUI sudo password dialog. Values: true, false (also 1/0, yes/no, on/off).",
			),
		},
	); err != nil {
		fyne.LogError("failed to create config file", err)
	}

	appLogic := commands.New()
	appUI := ui.New(appLogic, version)
	_ = gitCommit
	appUI.Run()
}
