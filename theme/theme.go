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
