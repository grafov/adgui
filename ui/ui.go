package ui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"sync"
	"time"

	"adgui/commands"
	"adgui/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var (
	DisconnectedColor = color.NRGBA{R: 128, G: 128, B: 128, A: 255} // Серый
	ConnectedColor    = color.NRGBA{R: 0, G: 255, B: 0, A: 255}     // Зеленый
	WarningColor      = color.NRGBA{R: 255, G: 255, B: 0, A: 255}   // Желтый
)

type (
	// Properties related to UI.
	UI struct {
		Fyne       fyne.App
		desk       desktop.App
		updateReqs chan struct{}
		// protected by mutex
		traymx sync.RWMutex
		menu   *fyne.Menu

		// and...
		withLogicIncluded
	}
	// Properties related to application logic.
	withLogicIncluded struct {
		vpnmgr    *commands.VPNManager
		checkReqs chan struct{}
	}
)

func New(vpnmgr *commands.VPNManager) *UI {
	myApp := app.NewWithID("AdGuard VPN Client")
	myApp.SetIcon(theme.DisconnectedIcon)
	logic := withLogicIncluded{
		vpnmgr:    vpnmgr,
		checkReqs: make(chan struct{}, 1),
	}
	desk, ok := myApp.(desktop.App)
	ui := UI{
		Fyne:              myApp,
		desk:              desk,
		updateReqs:        make(chan struct{}, 1),
		withLogicIncluded: logic,
	}
	if ok {
		ui.createTrayMenu()
		// Register callback to notify UI about status changes
		vpnmgr.SetStatusChangeCallback(func() {
			select {
			case ui.updateReqs <- struct{}{}:
			default:
			}
		})
	} else {
		fmt.Println("System tray not supported")
	}
	return &ui
}

func (u *UI) Run() {
	u.Fyne.Run()
}

func (u *UI) createTrayMenu() {
	status := fyne.NewMenuItem("Adguard VPN", func() {})
	dashboard := fyne.NewMenuItem("Show dashboard", func() {
		u.Dashboard()
	})
	connectAuto := fyne.NewMenuItem("Connect Auto", func() {
		u.vpnmgr.ConnectAuto()
	})
	connectTo := fyne.NewMenuItem("Connect To...", func() {
		u.LocationSelector()
	})
	disconnect := fyne.NewMenuItem("Disconnect", func() {
		u.vpnmgr.Disconnect()
	})
	u.menu = fyne.NewMenu("AdGuard VPN Client",
		status,
		dashboard,
		connectAuto,
		connectTo,
		fyne.NewMenuItemSeparator(),
		disconnect,
	)
	u.menu.Items[0].Disabled = true // status field

	u.desk.SetSystemTrayMenu(u.menu)
	go u.updateUI()
}

func (u *UI) updateMenuItems() {
	u.traymx.Lock()
	defer u.traymx.Unlock()

	// Обновляем доступность пунктов меню
	if u.menu != nil {
		items := u.menu.Items
		if u.vpnmgr.IsConnected() {
			u.menu.Label = "VPN connected"
			items[0].Icon = theme.MenuConnectedIcon
			items[0].Label = strings.ToUpper(u.vpnmgr.Location())
		} else {
			u.menu.Label = "VPN disconected"
			items[0].Icon = theme.MenuDisconnectedIcon
			items[0].Label = "OFF"
		}
		connected := u.vpnmgr.IsConnected()
		items[1].Disabled = false
		items[2].Disabled = connected  // Connect Auto
		items[3].Disabled = false      // Connect To... available always
		items[4].Disabled = !connected // Disconnect
		u.menu.Items = items
		fyne.Do(func() {
			u.desk.SetSystemTrayMenu(u.menu)
		})
	}
}

func (u *UI) updateUI() {
	select {
	case u.checkReqs <- struct{}{}:
		time.Sleep(200 * time.Millisecond)
	default:
	}
	for range u.updateReqs {
		u.updateTrayIcon()
		u.updateMenuItems()
	}
}

func (u *UI) updateTrayIcon() {
	u.traymx.RLock()
	defer u.traymx.RUnlock()

	fyne.Do(func() {
		if u.vpnmgr.IsConnected() {
			u.desk.SetSystemTrayIcon(theme.ConnectedIcon)
		} else {
			u.desk.SetSystemTrayIcon(theme.DisconnectedIcon)
		}
	})
}

func (u *UI) Dashboard() string {
	// Создаем новое окно для выбора локации
	window := u.Fyne.NewWindow("adgui")
	window.Resize(fyne.NewSize(800, 600))

	// Connections page
	statusLbl := widget.NewLabel(u.vpnmgr.Status())
	turnOn := widget.NewButton("Connect", func() {})
	connectTo := widget.NewButton("Connect To...", func() {})
	close := widget.NewButton("X", func() { window.Close() })
	grid := container.New(layout.NewFormLayout(), statusLbl, turnOn, connectTo, close)

	license := u.licensePanel()
	tabs := container.NewAppTabs(
		container.NewTabItem("Connections", grid),
		container.NewTabItem("License", license),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	window.SetContent(tabs)
	window.Show()
	return ""
}

func (u *UI) licensePanel() *fyne.Container {
	return container.New(
		layout.NewVBoxLayout(),
		widget.NewLabel("AdGuard license"),
		widget.NewTextGridFromString(u.vpnmgr.License()),
	)
}

func (u *UI) LocationSelector() {
	// Создаем новое окно для выбора локации
	window := u.Fyne.NewWindow("adgui: select location")
	window.Resize(fyne.NewSize(500, 600))

	locations := u.vpnmgr.ListLocations()

	table := widget.NewTable(
		// Return number of rows and columns
		func() (int, int) {
			return len(locations), 4
		},
		// Create a template widget for cells
		func() fyne.CanvasObject {
			return widget.NewLabel("...")
		},
		// Create a template widget for cells
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			// Update the content of the cell based on its ID
			label := obj.(*widget.Label)
			// Формируем заголовок
			if id.Row == 0 {
				switch id.Col {
				case 0:
					label.SetText("ISO")
				case 1:
					label.SetText("Country")
				case 2:
					label.SetText("City")
				case 3:
					label.SetText("Ping (ms)")
				}
				label.TextStyle.Bold = true
				return
			}

			loc := locations[id.Row]
			switch id.Col {
			case 0:
				label.SetText(loc.ISO)
			case 1:
				label.SetText(loc.Country)
			case 2:
				label.SetText(loc.City)
			case 3:
				label.SetText(strconv.Itoa(loc.Ping))
			}
			label.TextStyle.Bold = false
		},
	)

	// Set column widths (optional)
	table.SetColumnWidth(0, 30)
	table.SetColumnWidth(1, 300)
	table.SetColumnWidth(2, 200)
	table.SetColumnWidth(3, 30)

	// Обработчик выбора локации
	table.OnSelected = func(id widget.TableCellID) {
		fmt.Printf("Selected: %+v\n", locations[id.Row])
		city := locations[id.Row].City
		go u.vpnmgr.ConnectToLocation(city)
		window.Close()
	}

	window.SetContent(container.NewStack(table))
	window.Show()
}
