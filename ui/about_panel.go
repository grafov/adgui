package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

func (u *UI) aboutPanel(appVersion string) *fyne.Container {
	adguiLabel := widget.NewLabel("adgui: " + appVersion)
	cliLabel := widget.NewLabel(lang.X("about.cli.loading", "adguardvpn-cli: loading..."))

	go func() {
		version := u.vpnmgr.CLIVersion()
		fyne.Do(func() {
			cliLabel.SetText(lang.X("about.cli.version", "adguardvpn-cli: {{.Version}}", map[string]any{"Version": version}))
		})
	}()

	header := widget.NewLabel(lang.X("about.title", "About"))
	header.TextStyle.Bold = true

	return container.NewVBox(
		header,
		adguiLabel,
		cliLabel,
		widget.NewLabel(lang.X("about.ipregion", "IP region checks ported from Davoyan/ipregion (MIT)")),
	)
}
