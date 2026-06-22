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

package theme

import (
	_ "embed"
)

//go:embed icon-off.png
var iconOff []byte

//go:embed icon-on.png
var iconOn []byte

//go:embed menu-icon-on.png
var menuIconOn []byte

//go:embed menu-icon-off.png
var menuIconOff []byte

var (
	DisconnectedIcon     = disconnectedIcon{}
	ConnectedIcon        = connectedIcon{}
	MenuDisconnectedIcon = menuDisconnectedIcon{}
	MenuConnectedIcon    = menuConnectedIcon{}
)

type menuDisconnectedIcon struct{}

func (menuDisconnectedIcon) Name() string {
	return "vpn-disconnected"
}

func (menuDisconnectedIcon) Content() []byte {
	return menuIconOff
}

type menuConnectedIcon struct{}

func (menuConnectedIcon) Name() string {
	return "vpn-connected"
}

func (menuConnectedIcon) Content() []byte {
	return menuIconOn
}

type disconnectedIcon struct{}

func (disconnectedIcon) Name() string {
	return "disconnected"
}

func (disconnectedIcon) Content() []byte {
	return iconOff
}

type connectedIcon struct{}

func (connectedIcon) Name() string {
	return "connected"
}

func (connectedIcon) Content() []byte {
	return iconOn
}

// func main() {
// 	myApp := app.NewWithID("Test")
// 	myApp.SetIcon(ConnectedIcon)
// 	deskApp := myApp.(desktop.App)
// 	deskApp.SetSystemTrayIcon(ConnectedIcon)
// 	deskApp.SetSystemTrayMenu(fyne.NewMenu("test"))
// 	go func() {
// 		time.Sleep(3 * time.Second)
// 		fyne.Do(func() {
// 			deskApp.SetSystemTrayIcon(DisconnectedIcon)
// 			myApp.SetIcon(DisconnectedIcon)
// 		})
// 	}()
// 	myApp.Run()
// }
