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
	"fmt"

	"adgui/commands"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type connectionsPanelWidgets struct {
	cityLabel      *canvas.Text
	countryLabel   *canvas.Text
	pingLabel      *canvas.Text
	statusLabel    *canvas.Text
	historyBox     *fyne.Container
	historySection *fyne.Container
}

func (u *UI) connectionsPanel() (*fyne.Container, *connectionsPanelWidgets) {
	widgets := &connectionsPanelWidgets{
		cityLabel:    canvas.NewText("", ConnectedColor),
		countryLabel: canvas.NewText("", ConnectedColor),
		pingLabel:    canvas.NewText("", ConnectedColor),
		statusLabel:  canvas.NewText(lang.X("connections.disconnected", "Disconnected"), DisconnectedStatusColor),
		historyBox:   container.NewVBox(),
	}
	widgets.statusLabel.TextSize = 36
	widgets.statusLabel.Alignment = fyne.TextAlignCenter
	widgets.cityLabel.TextSize = 36
	widgets.cityLabel.Alignment = fyne.TextAlignCenter
	widgets.countryLabel.TextSize = 24
	widgets.countryLabel.Alignment = fyne.TextAlignCenter
	widgets.pingLabel.TextSize = 28
	widgets.pingLabel.Alignment = fyne.TextAlignCenter

	connectBtn := widget.NewButton("", func() {
		u.runPrivileged(func() {
			if u.vpnmgr.IsConnected() {
				u.vpnmgr.Disconnect()
			} else {
				u.vpnmgr.ConnectAuto()
			}
		})
	})
	u.dashboardConnectBtn = connectBtn
	u.updateDashboardButtons()

	connectToBtn := widget.NewButton(lang.X("connections.connect_to", "Connect To..."), func() {
		u.LocationSelector()
	})
	buttonContainer := container.NewHBox(layout.NewSpacer(), connectBtn, connectToBtn, layout.NewSpacer())

	centerContent := container.NewVBox(
		container.NewCenter(widgets.statusLabel),
		container.NewCenter(widgets.cityLabel),
		container.NewCenter(widgets.countryLabel),
		container.NewCenter(widgets.pingLabel),
	)
	centerArea := container.NewCenter(centerContent)

	historyHeader := widget.NewLabel(lang.X("connections.history.header", "Previously connected to:"))
	historyHeader.TextStyle.Bold = true
	historySection := container.NewVBox(
		widget.NewSeparator(),
		historyHeader,
		widgets.historyBox,
	)
	widgets.historySection = historySection
	historySection.Hide()

	content := container.NewBorder(
		buttonContainer,
		historySection,
		nil,
		nil,
		centerArea,
	)

	u.refreshConnectionsPanel(widgets)
	return content, widgets
}

func (u *UI) refreshConnectionsPanel(w *connectionsPanelWidgets) {
	if w == nil {
		return
	}

	if u.vpnmgr.IsConnected() {
		loc, ok := u.vpnmgr.ConnectedLocation()
		if !ok {
			loc.City = u.vpnmgr.Location()
		}
		w.statusLabel.Text = ""
		w.cityLabel.Text = loc.City
		w.countryLabel.Text = loc.Country
		w.pingLabel.Text = formatPing(loc.Ping)
		w.cityLabel.Color = ConnectedColor
		w.countryLabel.Color = ConnectedColor
		w.pingLabel.Color = ConnectedColor
	} else {
		w.statusLabel.Text = lang.X("connections.disconnected", "Disconnected")
		w.statusLabel.Color = DisconnectedStatusColor
		w.cityLabel.Text = ""
		w.countryLabel.Text = ""
		w.pingLabel.Text = ""
		w.cityLabel.Color = DisconnectedColor
		w.countryLabel.Color = DisconnectedColor
		w.pingLabel.Color = DisconnectedColor
	}
	w.cityLabel.Refresh()
	w.countryLabel.Refresh()
	w.pingLabel.Refresh()
	w.statusLabel.Refresh()

	w.historyBox.Objects = nil
	entries := u.vpnmgr.PreviousConnectionHistory()
	if len(entries) == 0 {
		w.historySection.Hide()
	} else {
		w.historySection.Show()
		for _, entry := range entries {
			line := formatHistoryEntry(entry)
			label := widget.NewLabel(line)
			label.Wrapping = fyne.TextWrapWord
			w.historyBox.Add(label)
		}
	}
	w.historyBox.Refresh()
	w.historySection.Refresh()
}

func formatHistoryEntry(entry commands.ConnectionHistoryEntry) string {
	location := entry.City
	if entry.Country != "" {
		location = fmt.Sprintf("%s, %s", entry.City, entry.Country)
	}
	started := entry.StartedAt.Local().Format("2006-01-02 15:04:05")
	ended := "—"
	if entry.EndedAt != nil {
		ended = entry.EndedAt.Local().Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("%s — %s → %s", location, started, ended)
}
