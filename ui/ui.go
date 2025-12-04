package ui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"sync"
	"time"

	"adgui/commands"
	"adgui/locations"
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

		// Dashboard window and widgets for live updates
		dashboardmx          sync.RWMutex
		dashboardWindow      fyne.Window
		dashboardStatusLabel *widget.RichText
		dashboardConnectBtn  *widget.Button

		// Location selector window
		locationmx     sync.RWMutex
		locationWindow fyne.Window

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
		fyne.Do(func() {
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
		u.updateDashboard()
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

func (u *UI) updateDashboardButtons() {
	if u.dashboardConnectBtn == nil {
		return
	}

	connected := u.vpnmgr.IsConnected()
	if connected {
		u.dashboardConnectBtn.SetText("Disconnect")
	} else {
		u.dashboardConnectBtn.SetText("Connect")
	}
}

func (u *UI) updateDashboard() {
	u.dashboardmx.RLock()
	window := u.dashboardWindow
	statusLabel := u.dashboardStatusLabel
	connectBtn := u.dashboardConnectBtn
	u.dashboardmx.RUnlock()

	if window == nil {
		return
	}

	fyne.Do(func() {
		// Re-check if window is still valid by checking the struct field
		// This is safe because we are in the main thread where SetOnClosed callbacks run
		u.dashboardmx.RLock()
		currentWindow := u.dashboardWindow
		u.dashboardmx.RUnlock()

		if currentWindow != window {
			// Window was closed or replaced
			return
		}

		if statusLabel != nil {
			statusLabel.Segments = parseAnsi(u.vpnmgr.Status()).Segments
			statusLabel.Refresh()
		}
		if connectBtn != nil {
			u.updateDashboardButtons()
		}
	})
}

func (u *UI) Dashboard() string {
	u.dashboardmx.Lock()
	defer u.dashboardmx.Unlock()

	// If dashboard window already exists, we can't simply RequestFocus or Show it
	// in Wayland without user interaction token. So just ignore or log.
	if u.dashboardWindow != nil {
		// u.dashboardWindow.Show() // This might crash in Wayland if called without interaction
		return ""
	}

	// Create new dashboard window
	window := u.Fyne.NewWindow("adgui: VPN Dashboard")
	window.Resize(fyne.NewSize(800, 600))
	u.dashboardWindow = window

	// Status section
	statusHeader := widget.NewLabel("Status")
	statusHeader.TextStyle.Bold = true
	statusLbl := parseAnsi(u.vpnmgr.Status())
	u.dashboardStatusLabel = statusLbl

	// Control buttons section
	connectBtn := widget.NewButton("", func() {
		if u.vpnmgr.IsConnected() {
			u.vpnmgr.Disconnect()
		} else {
			u.vpnmgr.ConnectAuto()
		}
	})
	u.dashboardConnectBtn = connectBtn
	u.updateDashboardButtons()

	connectToBtn := widget.NewButton("Connect To...", func() {
		u.LocationSelector()
	})

	buttonContainer := container.NewHBox(connectBtn, connectToBtn)

	// Connections page content
	connectionsContent := container.NewVBox(
		statusHeader,
		statusLbl,
		widget.NewSeparator(),
		buttonContainer,
	)

	license := u.licensePanel()
	tabs := container.NewAppTabs(
		container.NewTabItem("Connections", connectionsContent),
		container.NewTabItem("License", license),
		container.NewTabItem("Excluded", u.exclusionsPanel()),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	window.SetContent(tabs)

	// Close on Esc
	window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if k.Name == fyne.KeyEscape {
			// Close asynchronously to avoid potential event loop conflicts
			fyne.Do(window.Close)
		}
	})

	// Clean up references when window is closed
	window.SetOnClosed(func() {
		u.dashboardmx.Lock()
		defer u.dashboardmx.Unlock()
		u.dashboardWindow = nil
		u.dashboardStatusLabel = nil
		u.dashboardConnectBtn = nil
	})

	window.Show()
	return ""
}

func (u *UI) licensePanel() *fyne.Container {
	return container.New(
		layout.NewVBoxLayout(),
		widget.NewLabel("AdGuard license"),
		parseAnsi(u.vpnmgr.License()),
	)
}

func (u *UI) exclusionsPanel() *fyne.Container {
	exclusions, err := u.vpnmgr.GetSiteExclusions()
	if err != nil {
		fmt.Printf("load exclusions error: %v\n", err)
	}
	filtered := exclusions
	currentQuery := ""

	filterExclusions := func(items []string, query string) []string {
		if query == "" {
			return items
		}
		lowerQuery := strings.ToLower(query)
		var res []string
		for _, item := range items {
			if strings.Contains(strings.ToLower(item), lowerQuery) {
				res = append(res, item)
			}
		}
		return res
	}

	containsIgnoreCase := func(items []string, value string) bool {
		for _, item := range items {
			if strings.EqualFold(item, value) {
				return true
			}
		}
		return false
	}

	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Filter or enter domain...")

	var exclusionsList *widget.List

	refreshFiltered := func() {
		filtered = filterExclusions(exclusions, currentQuery)
		if exclusionsList != nil {
			exclusionsList.Refresh()
		}
	}

	reloadExclusions := func() {
		go func() {
			newExclusions, loadErr := u.vpnmgr.GetSiteExclusions()
			if loadErr != nil {
				fmt.Printf("reload exclusions error: %v\n", loadErr)
				return
			}
			fyne.Do(func() {
				exclusions = newExclusions
				refreshFiltered()
			})
		}()
	}

	exclusionsList = widget.NewList(
		func() int {
			return len(filtered)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("domain")
			removeBtn := widget.NewButton("X", nil)
			return container.NewHBox(label, layout.NewSpacer(), removeBtn)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			cont := obj.(*fyne.Container)
			label := cont.Objects[0].(*widget.Label)
			removeBtn := cont.Objects[2].(*widget.Button)

			domain := filtered[id]
			label.SetText(domain)
			removeBtn.OnTapped = func() {
				go func(target string) {
					if err := u.vpnmgr.RemoveSiteExclusion(target); err != nil {
						fmt.Printf("remove exclusion error: %v\n", err)
						return
					}
					reloadExclusions()
				}(domain)
			}
		},
	)

	filterEntry.OnChanged = func(query string) {
		currentQuery = query
		refreshFiltered()
	}

	appendBtn := widget.NewButton("Append", func() {
		domain := strings.TrimSpace(filterEntry.Text)
		if domain == "" {
			return
		}
		if containsIgnoreCase(exclusions, domain) {
			return
		}
		go func(value string) {
			if err := u.vpnmgr.AddSiteExclusion(value); err != nil {
				fmt.Printf("add exclusion error: %v\n", err)
				return
			}
			reloadExclusions()
		}(domain)
	})

	header := container.NewBorder(nil, nil, nil, appendBtn, filterEntry)
	content := container.NewBorder(header, nil, nil, nil, exclusionsList)
	return content
}

func (u *UI) LocationSelector() {
	u.locationmx.Lock()
	defer u.locationmx.Unlock()

	if u.locationWindow != nil {
		return
	}

	allLocations := u.vpnmgr.ListLocations()
	filteredLocations := allLocations

	fyne.Do(func() {
		window := u.Fyne.NewWindow("adgui: select location")
		window.Resize(fyne.NewSize(640, 720))
		u.locationWindow = window

		window.SetOnClosed(func() {
			u.locationmx.Lock()
			defer u.locationmx.Unlock()
			u.locationWindow = nil
		})

		filterEntry := widget.NewEntry()
		filterEntry.SetPlaceHolder("Filter by city or country...")

		table := widget.NewTable(
			func() (int, int) {
				return len(filteredLocations) + 1, 4 // +1 for header
			},
			func() fyne.CanvasObject {
				return widget.NewLabel("...")
			},
			func(id widget.TableCellID, obj fyne.CanvasObject) {
				label := obj.(*widget.Label)
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

				loc := filteredLocations[id.Row-1]
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

		table.SetColumnWidth(0, 40)
		table.SetColumnWidth(1, 200)
		table.SetColumnWidth(2, 200)
		table.SetColumnWidth(3, 80)

		table.OnSelected = func(id widget.TableCellID) {
			if id.Row == 0 {
				return // Skip header
			}
			selectedLocation := filteredLocations[id.Row-1]
			fmt.Printf("Selected: %+v\n", selectedLocation)
			go u.vpnmgr.ConnectToLocation(selectedLocation.City)
			window.Close()
		}

		filterEntry.OnChanged = func(query string) {
			filteredLocations = locations.FilterLocations(allLocations, query)
			table.Refresh()
		}

		content := container.NewBorder(filterEntry, nil, nil, nil, table)
		window.SetContent(content)

		window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
			if k.Name == fyne.KeyEscape {
				fyne.Do(window.Close)
			}
		})

		window.Show()
	})
}
