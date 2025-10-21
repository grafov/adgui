package theme

import (
	_ "embed"
)

//go:embed icon-off.png
var iconOff []byte

//go:embed icon-on.png
var iconOn []byte

var (
	DisconnectedIcon = disconnectedIcon{}
	ConnectedIcon    = connectedIcon{}
)

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
