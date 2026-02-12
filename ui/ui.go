package ui

import (
	"bufio"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
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
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var (
	DisconnectedColor = color.NRGBA{R: 128, G: 128, B: 128, A: 255} // Серый
	ConnectedColor    = color.NRGBA{R: 0, G: 255, B: 0, A: 255}     // Зеленый
	WarningColor      = color.NRGBA{R: 255, G: 255, B: 0, A: 255}   // Желтый
)

const domainsTabIndex = 2

type (
	// Properties related to UI.
	UI struct {
		Fyne       fyne.App
		desk       desktop.App
		updateReqs chan struct{}
		// protected by mutex
		traymx          sync.RWMutex
		menu            *fyne.Menu
		domainsMenuItem *fyne.MenuItem
		domainsCount    int

		// Dashboard window and widgets for live updates
		dashboardmx          sync.RWMutex
		dashboardWindow      fyne.Window
		dashboardStatusLabel *widget.RichText
		dashboardConnectBtn  *widget.Button
		dashboardTabs        *container.AppTabs

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
		go func() {
			_, exclusions, err := vpnmgr.GetSiteExclusions()
			if err != nil {
				fmt.Printf("load exclusions mode error: %v\n", err)
				return
			}
			ui.setDomainsCount(len(exclusions))
		}()
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
	connectAuto := fyne.NewMenuItem("Connect the best", func() {
		u.vpnmgr.ConnectAuto()
	})
	connectTo := fyne.NewMenuItem("Connect To...", func() {
		u.LocationSelector()
	})
	disconnect := fyne.NewMenuItem("Disconnect", func() {
		u.vpnmgr.Disconnect()
	})
	domains := fyne.NewMenuItem(domainsMenuLabel(u.getDomainsCount()), func() {
		u.showDashboardTab(domainsTabIndex)
	})
	u.domainsMenuItem = domains
	u.menu = fyne.NewMenu("AdGuard VPN Client",
		status,
		dashboard,
		connectAuto,
		connectTo,
		domains,
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
		domainsCount := u.domainsCount
		domainsMenuItem := u.domainsMenuItem
		fyne.Do(func() {
			items := u.menu.Items
			if u.vpnmgr.IsConnected() {
				u.menu.Label = "VPN connected"
				items[0].Icon = theme.MenuConnectedIcon
				modeSuffix := "GEN"
				if u.vpnmgr.SiteExclusionsMode() == commands.SiteExclusionModeSelective {
					modeSuffix = "SEL"
				}
				items[0].Label = fmt.Sprintf("%s mode:%s", strings.ToUpper(u.vpnmgr.Location()), modeSuffix)
			} else {
				u.menu.Label = "VPN disconected"
				items[0].Icon = theme.MenuDisconnectedIcon
				items[0].Label = "OFF"
			}
			connected := u.vpnmgr.IsConnected()
			if domainsMenuItem != nil {
				domainsMenuItem.Label = domainsMenuLabel(domainsCount)
			}
			// false - means available
			items[1].Disabled = false
			items[2].Disabled = connected  // Connect the best
			items[3].Disabled = false      // Connect To...
			items[4].Disabled = false      // Domains
			items[5].Disabled = !connected // Disconnect
			items[6].Disabled = false      // Quit
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

func (u *UI) setDomainsCount(count int) {
	u.traymx.Lock()
	u.domainsCount = count
	u.traymx.Unlock()
	select {
	case u.updateReqs <- struct{}{}:
	default:
	}
}

func (u *UI) getDomainsCount() int {
	u.traymx.RLock()
	count := u.domainsCount
	u.traymx.RUnlock()
	return count
}

func domainsMenuLabel(count int) string {
	if count > 0 {
		return fmt.Sprintf("Domains (%d)", count)
	}
	return "Domains"
}

func (u *UI) showDashboardTab(index int) {
	u.Dashboard()
	u.selectDashboardTab(index)
}

func (u *UI) selectDashboardTab(index int) {
	u.dashboardmx.RLock()
	tabs := u.dashboardTabs
	u.dashboardmx.RUnlock()
	if tabs == nil {
		return
	}
	fyne.Do(func() {
		if index < 0 || index >= len(tabs.Items) {
			return
		}
		tabs.SelectIndex(index)
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
		container.NewTabItem("Domains", u.exclusionsPanel()),
	)
	u.dashboardTabs = tabs
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
		u.dashboardTabs = nil
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
	mode, exclusions, err := u.vpnmgr.GetSiteExclusions()
	if err != nil {
		fmt.Printf("load exclusions error: %v\n", err)
	}
	u.setDomainsCount(len(exclusions))
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
	var modeRadio *widget.RadioGroup
	const (
		optionGeneral   = "The domains in the list excluded"
		optionSelective = "Only domains in the list included"
	)

	refreshFiltered := func() {
		filtered = filterExclusions(exclusions, currentQuery)
		if exclusionsList != nil {
			exclusionsList.Refresh()
		}
	}

	reloadExclusions := func() {
		go func() {
			newMode, newExclusions, loadErr := u.vpnmgr.GetSiteExclusions()
			if loadErr != nil {
				fmt.Printf("reload exclusions error: %v\n", loadErr)
				return
			}
			u.setDomainsCount(len(newExclusions))
			fyne.Do(func() {
				mode = newMode
				exclusions = newExclusions
				if modeRadio != nil {
					if mode == commands.SiteExclusionModeGeneral {
						modeRadio.SetSelected(optionGeneral)
					} else {
						modeRadio.SetSelected(optionSelective)
					}
				}
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
		fyne.Do(func() { filterEntry.SetText("") }) // reset filter text on append
		go func(value string) {
			if err := u.vpnmgr.AddSiteExclusion(value); err != nil {
				fmt.Printf("add exclusion error: %v\n", err)
				return
			}
			reloadExclusions()
		}(domain)
	})

	modeRadio = widget.NewRadioGroup([]string{optionGeneral, optionSelective}, nil)
	if mode == commands.SiteExclusionModeGeneral {
		modeRadio.SetSelected(optionGeneral)
	} else {
		modeRadio.SetSelected(optionSelective)
	}
	modeRadio.OnChanged = func(value string) {
		targetMode := commands.SiteExclusionModeSelective
		if value == optionGeneral {
			targetMode = commands.SiteExclusionModeGeneral
		}
		if targetMode == mode {
			return
		}
		previousMode := mode
		modeRadio.Disable()
		snapshot := append([]string(nil), exclusions...)
		go func() {
			defer fyne.Do(modeRadio.Enable)
			if err := u.vpnmgr.SetSiteExclusionsMode(targetMode, snapshot); err != nil {
				fmt.Printf("set exclusions mode error: %v\n", err)
				fyne.Do(func() {
					if previousMode == commands.SiteExclusionModeGeneral {
						modeRadio.SetSelected(optionGeneral)
					} else {
						modeRadio.SetSelected(optionSelective)
					}
				})
				return
			}
			fyne.Do(func() {
				mode = targetMode
			})
			select {
			case u.updateReqs <- struct{}{}:
			default:
			}
			reloadExclusions()
		}()
	}
	modeControls := container.NewVBox(modeRadio)

	exportBtn := widget.NewButton("Export", func() {
		entry := widget.NewEntry()
		entry.SetPlaceHolder("Filename")

		home, err := os.UserHomeDir()
		if err != nil {
			dialog.ShowError(err, u.dashboardWindow)
			return
		}
		dir := filepath.Join(home, ".local", "share", "adgui", "site-exclusions")

		// Read existing files
		var existingFiles []string
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					existingFiles = append(existingFiles, e.Name())
				}
			}
		}

		var d dialog.Dialog
		var performWrite func(string, bool, string)

		performWrite = func(fPath string, appendMode bool, name string) {
			var f *os.File
			var err error
			if appendMode {
				f, err = os.OpenFile(fPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			} else {
				f, err = os.Create(fPath)
			}
			if err != nil {
				dialog.ShowError(err, u.dashboardWindow)
				return
			}
			defer f.Close()

			content := strings.Join(filtered, "\n")
			if content != "" {
				if appendMode {
					// Add newline before content if file exists and not empty
					stat, _ := f.Stat()
					if stat.Size() > 0 {
						content = "\n" + content
					}
				}
				if _, err := f.WriteString(content); err != nil {
					dialog.ShowError(err, u.dashboardWindow)
					return
				}
			}

			d.Hide()
			mode := "Overwritten"
			if appendMode {
				mode = "Appended to"
			}
			dialog.ShowInformation("Export", mode+" "+name, u.dashboardWindow)
		}

		doExport := func(appendMode bool) {
			name := strings.TrimSpace(entry.Text)
			if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") {
				dialog.ShowError(fmt.Errorf("invalid filename"), u.dashboardWindow)
				return
			}

			if err := os.MkdirAll(dir, 0755); err != nil {
				dialog.ShowError(err, u.dashboardWindow)
				return
			}

			fPath := filepath.Join(dir, name)

			// Check if file exists for overwrite mode
			if !appendMode {
				if _, err := os.Stat(fPath); err == nil {
					// File exists, ask for confirmation
					d.Hide()
					dialog.ShowConfirm("Overwrite", "File '"+name+"' already exists. Overwrite?", func(ok bool) {
						if !ok {
							d.Show()
							return
						}
						// Proceed with overwrite
						performWrite(fPath, appendMode, name)
					}, u.dashboardWindow)
					return
				}
			}

			// Proceed directly if append mode or file doesn't exist
			performWrite(fPath, appendMode, name)
		}

		appendBtn := widget.NewButton("Append", func() {
			doExport(true)
		})
		overwriteBtn := widget.NewButton("Overwrite", func() {
			doExport(false)
		})

		buttons := container.NewHBox(appendBtn, overwriteBtn)

		var fileList *widget.List
		if len(existingFiles) > 0 {
			fileList = widget.NewList(
				func() int { return len(existingFiles) },
				func() fyne.CanvasObject { return widget.NewLabel("template") },
				func(id widget.ListItemID, obj fyne.CanvasObject) {
					obj.(*widget.Label).SetText(existingFiles[id])
				},
			)
			fileList.OnSelected = func(id widget.ListItemID) {
				entry.SetText(existingFiles[id])
				fileList.UnselectAll()
			}
		}

		var content *fyne.Container
		if fileList != nil {
			fileScroll := container.NewScroll(fileList)
			fileScroll.SetMinSize(fyne.NewSize(300, 150))
			content = container.NewVBox(
				widget.NewLabel("Export to ~/.local/share/adgui/site-exclusions/"),
				widget.NewLabel("Existing files:"),
				fileScroll,
				entry,
				buttons,
			)
		} else {
			content = container.NewVBox(
				widget.NewLabel("Export to ~/.local/share/adgui/site-exclusions/"),
				entry,
				buttons,
			)
		}

		d = dialog.NewCustom("Export", "Cancel", content, u.dashboardWindow)
		d.Resize(fyne.NewSize(400, 500))
		d.Show()
	})

	importBtn := widget.NewButton("Import", func() {
		home, err := os.UserHomeDir()
		if err != nil {
			dialog.ShowError(err, u.dashboardWindow)
			return
		}
		dir := filepath.Join(home, ".local", "share", "adgui", "site-exclusions")

		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				dialog.ShowInformation("Import", "No directory "+dir, u.dashboardWindow)
				return
			}
			dialog.ShowError(err, u.dashboardWindow)
			return
		}

		var files []string
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, e.Name())
			}
		}

		if len(files) == 0 {
			dialog.ShowInformation("Import", "No files found in "+dir, u.dashboardWindow)
			return
		}

		list := widget.NewList(
			func() int { return len(files) },
			func() fyne.CanvasObject { return widget.NewLabel("template") },
			func(id widget.ListItemID, obj fyne.CanvasObject) {
				obj.(*widget.Label).SetText(files[id])
			},
		)

		var d dialog.Dialog
		list.OnSelected = func(id widget.ListItemID) {
			d.Hide()
			fname := files[id]
			fPath := filepath.Join(dir, fname)

			f, err := os.Open(fPath)
			if err != nil {
				dialog.ShowError(err, u.dashboardWindow)
				return
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			var toAdd []string
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" && !containsIgnoreCase(exclusions, line) {
					toAdd = append(toAdd, line)
				}
			}

			if len(toAdd) > 0 {
				progress := dialog.NewProgressInfinite("Importing", "Adding "+strconv.Itoa(len(toAdd))+" domains...", u.dashboardWindow)
				progress.Show()
				go func() {
					defer progress.Hide()
					for _, domain := range toAdd {
						u.vpnmgr.AddSiteExclusion(domain)
					}
					reloadExclusions()
				}()
			} else {
				dialog.ShowInformation("Import", "No new unique domains found", u.dashboardWindow)
			}
		}

		scroll := container.NewScroll(list)
		scroll.SetMinSize(fyne.NewSize(300, 400))
		d = dialog.NewCustom("Import from "+dir, "Close", scroll, u.dashboardWindow)
		d.Resize(fyne.NewSize(400, 500))
		d.Show()
	})

	var clearBtn *widget.Button
	clearBtn = widget.NewButton("Clear", func() {
		if len(exclusions) == 0 {
			return
		}

		dialog.ShowConfirm("Clear", "Clear all domains in the list?", func(ok bool) {
			if !ok {
				return
			}

			// Create a copy of the current exclusions to operate on
			snapshot := append([]string(nil), exclusions...)

			// Disable the button during operation to prevent multiple clicks
			fyne.Do(func() {
				clearBtn.Disable()
			})

			// Show progress dialog
			progress := dialog.NewProgressInfinite("Clearing", "Removing "+strconv.Itoa(len(snapshot))+" domains...", u.dashboardWindow)
			progress.Show()

			go func() {
				defer func() {
					// Re-enable button and hide progress
					fyne.Do(func() {
						clearBtn.Enable()
						progress.Hide()
					})
				}()

				for _, domain := range snapshot {
					if err := u.vpnmgr.RemoveSiteExclusion(domain); err != nil {
						fmt.Printf("remove exclusion error: %v\n", err)
					}
				}
				reloadExclusions()
			}()
		}, u.dashboardWindow)
	})

	// Initially disable clear button if list is empty
	if len(exclusions) == 0 {
		clearBtn.Disable()
	}

	updateClearButtonState := func() {
		fyne.Do(func() {
			if len(exclusions) == 0 {
				clearBtn.Disable()
			} else {
				clearBtn.Enable()
			}
		})
	}

	// Override reloadExclusions to update button state after refresh
	reloadExclusions = func() {
		go func() {
			newMode, newExclusions, loadErr := u.vpnmgr.GetSiteExclusions()
			if loadErr != nil {
				fmt.Printf("reload exclusions error: %v\n", loadErr)
				return
			}
			u.setDomainsCount(len(newExclusions))
			fyne.Do(func() {
				mode = newMode
				exclusions = newExclusions
				if modeRadio != nil {
					if mode == commands.SiteExclusionModeGeneral {
						modeRadio.SetSelected(optionGeneral)
					} else {
						modeRadio.SetSelected(optionSelective)
					}
				}
				refreshFiltered()
				updateClearButtonState()
			})
		}()
	}

	header := container.NewBorder(nil, nil, nil, appendBtn, filterEntry)
	bottomButtons := container.NewHBox(importBtn, exportBtn, clearBtn)
	bottomControls := container.NewBorder(nil, nil, modeControls, bottomButtons)

	content := container.NewBorder(header, bottomControls, nil, nil, exclusionsList)
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

	// Состояние сортировки
	sortColumn := locations.SortByPing
	sortAscending := true

	// Функция для получения заголовка с индикатором сортировки
	getHeaderText := func(col int, currentSortCol locations.SortColumn, ascending bool) string {
		headers := []string{"ISO", "Country", "City", "Ping (ms)"}
		colToSort := []locations.SortColumn{locations.SortByISO, locations.SortByCountry, locations.SortByCity, locations.SortByPing}

		text := headers[col]
		if colToSort[col] == currentSortCol {
			if ascending {
				text += " ▲"
			} else {
				text += " ▼"
			}
		}
		return text
	}

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

		var table *widget.Table
		table = widget.NewTable(
			func() (int, int) {
				return len(filteredLocations) + 1, 4 // +1 for header
			},
			func() fyne.CanvasObject {
				return widget.NewLabel("...")
			},
			func(id widget.TableCellID, obj fyne.CanvasObject) {
				label := obj.(*widget.Label)
				if id.Row == 0 {
					label.SetText(getHeaderText(id.Col, sortColumn, sortAscending))
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

		table.SetColumnWidth(0, 60)
		table.SetColumnWidth(1, 200)
		table.SetColumnWidth(2, 200)
		table.SetColumnWidth(3, 100)

		table.OnSelected = func(id widget.TableCellID) {
			if id.Row == 0 {
				// Клик на заголовок - сортировка
				colToSort := []locations.SortColumn{locations.SortByISO, locations.SortByCountry, locations.SortByCity, locations.SortByPing}
				clickedColumn := colToSort[id.Col]

				if clickedColumn == sortColumn {
					// Тот же столбец - переключаем направление
					sortAscending = !sortAscending
				} else {
					// Другой столбец - сортировка по возрастанию
					sortColumn = clickedColumn
					sortAscending = true
				}

				filteredLocations = locations.SortLocations(filteredLocations, sortColumn, sortAscending)
				table.UnselectAll()
				table.Refresh()
				return
			}
			selectedLocation := filteredLocations[id.Row-1]
			fmt.Printf("Selected: %+v\n", selectedLocation)
			go u.vpnmgr.ConnectToLocation(selectedLocation.City)
			window.Close()
		}

		filterEntry.OnChanged = func(query string) {
			filteredLocations = locations.FilterLocations(allLocations, query)
			filteredLocations = locations.SortLocations(filteredLocations, sortColumn, sortAscending)
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
