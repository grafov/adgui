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

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

func (u *UI) aboutPanel(appVersion string) *fyne.Container {
	adguiLabel := widget.NewLabel("adgui: " + appVersion + " GPL v3")
	cliLabel := widget.NewLabel(lang.X("about.cli.loading", "adguardvpn-cli: loading..."))

	go func() {
		version := u.vpnmgr.CLIVersion()
		fyne.Do(func() {
			cliLabel.SetText(lang.X("about.cli.version", "adguardvpn-cli: {{.Version}}", map[string]any{"Version": version}))
		})
	}()

	header := widget.NewLabel(lang.X("about.title", "About"))
	header.TextStyle.Bold = true

	ipLabel := widget.NewLabel(lang.X("about.ipregion", "IP region checks ported from Davoyan/ipregion (MIT)"))
	urlLabel := widget.NewLabel("https://github.com/grafov/adgui")

	return container.NewVBox(
		header,
		adguiLabel,
		cliLabel,
		ipLabel,
		urlLabel,
	)
}
