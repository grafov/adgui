package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (u *UI) aboutPanel(appVersion string) *fyne.Container {
	adguiLabel := widget.NewLabel("adgui: " + appVersion)
	cliLabel := widget.NewLabel("adguardvpn-cli: loading...")

	go func() {
		version := u.vpnmgr.CLIVersion()
		fyne.Do(func() {
			cliLabel.SetText("adguardvpn-cli: " + version)
		})
	}()

	header := widget.NewLabel("About")
	header.TextStyle.Bold = true

	return container.NewVBox(
		header,
		adguiLabel,
		cliLabel,
	)
}
