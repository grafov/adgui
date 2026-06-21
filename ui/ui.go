package ui

import (
	"bufio"
	"fmt"
	"image/color"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"adgui/commands"
	"adgui/locations"
	"adgui/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

var (
	DisconnectedColor = color.NRGBA{R: 128, G: 128, B: 128, A: 255} // Серый
	ConnectedColor    = color.NRGBA{R: 0, G: 255, B: 0, A: 255}     // Зеленый
	WarningColor      = color.NRGBA{R: 255, G: 255, B: 0, A: 255}   // Желтый
	StarInactiveColor = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
	StarActiveColor   = color.NRGBA{R: 255, G: 193, B: 7, A: 255}
)

const (
	locationColFlag    = 0
	locationColISO     = 1
	locationColCountry = 2
	locationColCity    = 3
	locationColPing    = 4
	locationColStar    = 5
	locationTableCols  = 6
)

const domainsTabIndex = 2

type (
	// Properties related to UI.
	UI struct {
		Fyne       fyne.App
		desk       desktop.App
		updateReqs chan struct{}
		appVersion string
		// protected by mutex
		traymx          sync.RWMutex
		menu            *fyne.Menu
		domainsMenuItem *fyne.MenuItem
		domainsCount    int

		// Dashboard window and widgets for live updates
		dashboardmx              sync.RWMutex
		dashboardWindow          fyne.Window
		dashboardConnectBtn      *widget.Button
		dashboardTabs            *container.AppTabs
		dashboardConnectionsWids *connectionsPanelWidgets

		// Command queue list reference for live updates
		cmdQueuemx          sync.RWMutex
		cmdQueueRefreshFunc func()

		// Location selector window
		locationmx     sync.RWMutex
		locationWindow fyne.Window

		// Domains tab clipboard polling lifecycle
		pasteWatchStop chan struct{}
		pasteWatchDone chan struct{}
		pasteWatchOnce sync.Once

		// and...
		withLogicIncluded
	}
	// Properties related to application logic.
	withLogicIncluded struct {
		vpnmgr    *commands.VPNManager
		checkReqs chan struct{}
	}
)

func New(vpnmgr *commands.VPNManager, appVersion string) *UI {
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
		appVersion:        appVersion,
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

func (u *UI) startPasteWatcher() {
	if u.pasteWatchStop != nil {
		return
	}
	u.pasteWatchStop = make(chan struct{})
	u.pasteWatchDone = make(chan struct{})
}

func (u *UI) stopPasteWatcher() {
	u.pasteWatchOnce.Do(func() {
		if u.pasteWatchStop != nil {
			close(u.pasteWatchStop)
			<-u.pasteWatchDone
		}
	})
}

func (u *UI) createTrayMenu() {
	status := fyne.NewMenuItem("Adguard VPN", func() {})
	dashboard := fyne.NewMenuItem("Show dashboard", func() {
		u.Dashboard()
	})
	connectAuto := fyne.NewMenuItem("Connect the best", func() {
		go u.vpnmgr.ConnectAuto()
	})
	connectTo := fyne.NewMenuItem("Connect To...", func() {
		u.LocationSelector()
	})
	disconnect := fyne.NewMenuItem("Disconnect", func() {
		go u.vpnmgr.Disconnect()
	})
	quitItem := fyne.NewMenuItem("Quit", func() {
		u.stopPasteWatcher()
		u.Fyne.Quit()
	})
	quitItem.IsQuit = true
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
		fyne.NewMenuItemSeparator(),
		quitItem,
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
			items[6].Disabled = !connected // Disconnect
			items[8].Disabled = false      // Quit
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
	connectBtn := u.dashboardConnectBtn
	connWids := u.dashboardConnectionsWids
	u.dashboardmx.RUnlock()

	if window == nil {
		return
	}

	fyne.Do(func() {
		u.dashboardmx.RLock()
		currentWindow := u.dashboardWindow
		u.dashboardmx.RUnlock()

		if currentWindow != window {
			return
		}

		if connWids != nil {
			u.refreshConnectionsPanel(connWids)
		}
		if connectBtn != nil {
			u.updateDashboardButtons()
		}

		u.cmdQueuemx.RLock()
		refresh := u.cmdQueueRefreshFunc
		u.cmdQueuemx.RUnlock()
		if refresh != nil {
			refresh()
		}
	})
}

func (u *UI) Dashboard() string {
	u.dashboardmx.Lock()
	defer u.dashboardmx.Unlock()

	// Reuse a hidden dashboard instead of Close(): Close() sets the GLFW driver's
	// closing flag while GLFW can still deliver cursor/mouse events, which can panic
	// inside Fyne's processMouseMoved (nil view). Hide() does not set closing.
	if u.dashboardWindow != nil {
		u.dashboardWindow.Show()
		return ""
	}

	// Create new dashboard window
	window := u.Fyne.NewWindow("adgui: VPN Dashboard")
	window.Resize(fyne.NewSize(800, 600))
	u.dashboardWindow = window

	connectionsContent, connWids := u.connectionsPanel()
	u.dashboardConnectionsWids = connWids

	license := u.licensePanel()
	u.startPasteWatcher()
	tabs := container.NewAppTabs(
		container.NewTabItem("Connections", connectionsContent),
		container.NewTabItem("License", license),
		container.NewTabItem("Domains", u.exclusionsPanel(u.pasteWatchStop)),
		container.NewTabItem("Cmd queue", u.cmdQueuePanel()),
		container.NewTabItem("About", u.aboutPanel(u.appVersion)),
	)
	u.dashboardTabs = tabs
	tabs.SetTabLocation(container.TabLocationLeading)
	window.SetContent(tabs)

	// Hide on Esc (see Close vs Hide note above)
	window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if k.Name == fyne.KeyEscape {
			window.Hide()
		}
	})

	window.SetCloseIntercept(func() {
		window.Hide()
	})

	window.Show()
	return ""
}

func (u *UI) licensePanel() *fyne.Container {
	licenseLabel := parseAnsi("Loading license...")
	go func() {
		text := u.vpnmgr.License()
		fyne.Do(func() {
			parsed := parseAnsi(text)
			licenseLabel.Segments = parsed.Segments
			licenseLabel.Refresh()
		})
	}()

	return container.New(
		layout.NewVBoxLayout(),
		widget.NewLabel("AdGuard license"),
		licenseLabel,
	)
}

func (u *UI) exclusionsPanel(stopCh <-chan struct{}) *fyne.Container {
	var mode = commands.SiteExclusionModeGeneral
	var exclusions []string
	var filtered []string
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
	clipboard := u.Fyne.Clipboard()

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

	var reloadExclusions func()
	var reloadExclusionsAndSave func()

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
					reloadExclusionsAndSave()
				}(domain)
			}
		},
	)

	filterEntry.OnChanged = func(query string) {
		currentQuery = query
		refreshFiltered()
	}

	appendCurrent := func() {
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
			reloadExclusionsAndSave()
		}(domain)
	}

	filterEntry.OnSubmitted = func(_ string) {
		appendCurrent()
	}

	appendBtn := widget.NewButton("Append", func() {
		appendCurrent()
	})
	var pasteBtn *widget.Button

	parseDomainFromClipboard := func(content string) string {
		content = strings.TrimSpace(content)
		if content == "" {
			return ""
		}
		fields := strings.Fields(content)
		if len(fields) == 0 {
			return ""
		}
		token := fields[0]
		if !strings.Contains(token, "://") {
			token = "http://" + token
		}
		parsed, parseErr := url.Parse(token)
		if parseErr != nil {
			return ""
		}
		host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
		host = strings.TrimPrefix(host, "www.")
		if !strings.Contains(host, ".") {
			return ""
		}
		return host
	}

	updatePasteButtonState := func() {
		if pasteBtn == nil {
			return
		}
		fyne.Do(func() {
			content := ""
			if clipboard != nil {
				content = strings.TrimSpace(clipboard.Content())
			}
			if content == "" {
				pasteBtn.Disable()
			} else {
				pasteBtn.Enable()
			}
		})
	}

	pasteFromClipboard := func() {
		fyne.Do(func() {
			content := ""
			if clipboard != nil {
				content = strings.TrimSpace(clipboard.Content())
			}
			domain := parseDomainFromClipboard(content)
			if content == "" {
				pasteBtn.Disable()
			} else {
				pasteBtn.Enable()
			}
			if domain == "" {
				return
			}
			filterEntry.SetText(domain)
			go func(target string) {
				entries := []string{"www." + target, "*." + target}
				for _, entry := range entries {
					if containsIgnoreCase(exclusions, entry) {
						continue
					}
					if err := u.vpnmgr.AddSiteExclusion(entry); err != nil {
						fmt.Printf("add exclusion error: %v\n", err)
						return
					}
				}
				reloadExclusionsAndSave()
			}(domain)
		})
	}

	pasteBtn = widget.NewButton("Paste", func() {
		pasteFromClipboard()
	})

	updatePasteButtonState()
	if stopCh != nil {
		go func() {
			ticker := time.NewTicker(750 * time.Millisecond)
			defer ticker.Stop()
			defer close(u.pasteWatchDone)
			for {
				select {
				case <-ticker.C:
					updatePasteButtonState()
				case <-stopCh:
					return
				}
			}
		}()
	}

	if u.dashboardWindow != nil {
		u.dashboardWindow.Canvas().AddShortcut(
			&desktop.CustomShortcut{KeyName: fyne.KeyV, Modifier: fyne.KeyModifierAlt | fyne.KeyModifierControl},
			func(shortcut fyne.Shortcut) {
				if u.dashboardTabs == nil || u.dashboardTabs.SelectedIndex() != domainsTabIndex {
					return
				}
				pasteFromClipboard()
			},
		)
	}

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
		defaultName := mode.String() + ".adgui"
		saveDlg := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, u.dashboardWindow)
				return
			}
			if writer == nil {
				return
			}

			uri := writer.URI()
			if uri.Extension() == "" && uri.Scheme() == "file" {
				_ = writer.Close()
				uri = storage.NewFileURI(uri.Path() + ".adgui")
				writer, err = storage.Writer(uri)
				if err != nil {
					dialog.ShowError(err, u.dashboardWindow)
					return
				}
			}

			exportedName := uri.Name()
			defer func() {
				_ = writer.Close()
			}()

			content := strings.Join(filtered, "\n")
			if content != "" {
				if _, err := writer.Write([]byte(content)); err != nil {
					dialog.ShowError(err, u.dashboardWindow)
					return
				}
			}

			dialog.ShowInformation("Export", "Exported "+exportedName, u.dashboardWindow)
		}, u.dashboardWindow)
		saveDlg.SetFileName(defaultName)
		saveDlg.SetFilter(storage.NewExtensionFileFilter([]string{".adgui"}))
		if dir, err := commands.GetExclusionsDirPath(); err == nil {
			if listable, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
				saveDlg.SetLocation(listable)
			}
		}
		saveDlg.Show()
	})

	importBtn := widget.NewButton("Import", func() {
		openDlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, u.dashboardWindow)
				return
			}
			if reader == nil {
				return
			}
			defer func() {
				_ = reader.Close()
			}()

			scanner := bufio.NewScanner(reader)
			var toAdd []string
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" && !containsIgnoreCase(exclusions, line) {
					toAdd = append(toAdd, line)
				}
			}
			if err := scanner.Err(); err != nil {
				dialog.ShowError(err, u.dashboardWindow)
				return
			}

			if len(toAdd) == 0 {
				dialog.ShowInformation("Import", "No new unique domains found", u.dashboardWindow)
				return
			}

			hideProgress := showInfiniteProgressDialog("Importing", "Adding "+strconv.Itoa(len(toAdd))+" domains...", u.dashboardWindow)
			go func(domains []string) {
				defer hideProgress()
				var importErr error
				for _, domain := range domains {
					if err := u.vpnmgr.AddSiteExclusion(domain); err != nil {
						importErr = err
						break
					}
				}
				if importErr != nil {
					fyne.Do(func() { dialog.ShowError(importErr, u.dashboardWindow) })
				}
				reloadExclusionsAndSave()
			}(toAdd)
		}, u.dashboardWindow)
		openDlg.Show()
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

			hideProgress := showInfiniteProgressDialog("Clearing", "Removing "+strconv.Itoa(len(snapshot))+" domains...", u.dashboardWindow)
			go func() {
				defer func() {
					hideProgress()
					fyne.Do(func() {
						clearBtn.Enable()
					})
				}()

				for _, domain := range snapshot {
					if err := u.vpnmgr.RemoveSiteExclusion(domain); err != nil {
						fmt.Printf("remove exclusion error: %v\n", err)
					}
				}
				reloadExclusionsAndSave()
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

	reloadExclusionsAndSave = func() {
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

				if err := commands.SaveExclusionsForMode(mode, exclusions); err != nil {
					fmt.Printf("failed to auto-save exclusions for mode %s: %v\n", mode, err)
				}
			})
		}()
	}

	header := container.NewBorder(nil, nil, nil, container.NewHBox(appendBtn, pasteBtn), filterEntry)
	bottomButtons := container.NewHBox(importBtn, exportBtn, clearBtn)
	bottomControls := container.NewBorder(nil, nil, modeControls, bottomButtons)

	content := container.NewBorder(header, bottomControls, nil, nil, exclusionsList)
	reloadExclusions()
	return content
}

func (u *UI) LocationSelector() {
	u.locationmx.Lock()
	defer u.locationmx.Unlock()

	if u.locationWindow != nil {
		u.locationWindow.Show()
		return
	}

	var allLocations []locations.Location
	var filteredLocations []locations.Location
	bookmarks, err := commands.LoadLocationBookmarks()
	if err != nil {
		fmt.Printf("failed to load location bookmarks: %v\n", err)
		bookmarks = nil
	}

	sortColumn := locations.SortByPing
	sortAscending := true
	bookmarksFirst := false

	sortByColumn := []locations.SortColumn{
		locations.SortByISO,
		locations.SortByISO,
		locations.SortByCountry,
		locations.SortByCity,
		locations.SortByPing,
	}

	getHeaderText := func(col int, currentSortCol locations.SortColumn, ascending bool, favoritesFirst bool) string {
		switch col {
		case locationColFlag:
			return ""
		case locationColStar:
			text := "★"
			if favoritesFirst {
				text += " ▲"
			}
			return text
		}

		headers := []string{"", "ISO", "Country", "City", "Ping (ms)", "★"}
		text := headers[col]
		if sortByColumn[col] == currentSortCol {
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
		window.Resize(fyne.NewSize(700, 720))
		u.locationWindow = window

		window.SetCloseIntercept(func() {
			window.Hide()
		})

		filterEntry := widget.NewEntry()
		filterEntry.SetPlaceHolder("Filter by city or country...")
		currentFilter := ""

		applyBookmarkFlags := func(locs []locations.Location) []locations.Location {
			set := commands.LocationBookmarkSet(bookmarks)
			return locations.ApplyBookmarkFlags(locs, func(loc locations.Location) bool {
				_, ok := set[commands.LocationBookmarkKey(loc.ISO, loc.Country, loc.City)]
				return ok
			})
		}

		var table *widget.Table
		refreshTable := func() {
			filteredLocations = locations.FilterLocations(allLocations, currentFilter)
			filteredLocations = applyBookmarkFlags(filteredLocations)
			filteredLocations = locations.SortLocationsWithBookmarks(
				filteredLocations,
				sortColumn,
				sortAscending,
				bookmarksFirst,
			)
			if table != nil {
				table.Refresh()
			}
		}

		toggleBookmark := func(loc locations.Location) {
			key := commands.LocationBookmarkKey(loc.ISO, loc.Country, loc.City)
			found := -1
			for i, bookmark := range bookmarks {
				if commands.LocationBookmarkKey(bookmark.ISO, bookmark.Country, bookmark.City) == key {
					found = i
					break
				}
			}
			if found >= 0 {
				bookmarks = append(bookmarks[:found], bookmarks[found+1:]...)
			} else {
				bookmarks = append(bookmarks, commands.LocationBookmark{
					ISO:     loc.ISO,
					Country: loc.Country,
					City:    loc.City,
				})
			}
			if saveErr := commands.SaveLocationBookmarks(bookmarks); saveErr != nil {
				fmt.Printf("failed to save location bookmarks: %v\n", saveErr)
			}
			allLocations = applyBookmarkFlags(allLocations)
		}

		table = widget.NewTable(
			func() (int, int) {
				return len(filteredLocations) + 1, locationTableCols
			},
			func() fyne.CanvasObject {
				flagImg := canvas.NewImageFromResource(nil)
				flagImg.FillMode = canvas.ImageFillContain
				flagImg.SetMinSize(fyne.NewSize(28, 18))
				label := widget.NewLabel("")
				star := canvas.NewText("☆", StarInactiveColor)
				star.TextSize = 16
				star.Alignment = fyne.TextAlignCenter
				return container.NewStack(flagImg, label, star)
			},
			func(id widget.TableCellID, obj fyne.CanvasObject) {
				box := obj.(*fyne.Container)
				flagImg := box.Objects[0].(*canvas.Image)
				label := box.Objects[1].(*widget.Label)
				star := box.Objects[2].(*canvas.Text)

				flagImg.Hide()
				flagImg.Resource = nil
				flagImg.Refresh()
				label.Hide()
				label.SetText("")
				star.Hide()
				star.Text = ""
				label.TextStyle.Bold = false

				if id.Row == 0 {
					label.Show()
					label.TextStyle.Bold = true
					label.SetText(getHeaderText(id.Col, sortColumn, sortAscending, bookmarksFirst))
					label.Refresh()
					return
				}

				loc := filteredLocations[id.Row-1]
				switch id.Col {
				case locationColFlag:
					if res := theme.FlagResource(loc.ISO); res != nil {
						flagImg.Resource = res
						flagImg.Show()
						flagImg.Refresh()
					} else {
						label.Show()
						label.SetText(loc.ISO)
					}
				case locationColISO:
					label.Show()
					label.SetText(loc.ISO)
				case locationColCountry:
					label.Show()
					label.SetText(loc.Country)
				case locationColCity:
					label.Show()
					label.SetText(loc.City)
				case locationColPing:
					label.Show()
					label.SetText(strconv.Itoa(loc.Ping))
				case locationColStar:
					star.Show()
					if loc.Bookmarked {
						star.Text = "★"
						star.Color = StarActiveColor
					} else {
						star.Text = "☆"
						star.Color = StarInactiveColor
					}
					star.Refresh()
				}
				box.Refresh()
			},
		)

		table.SetColumnWidth(locationColFlag, 36)
		table.SetColumnWidth(locationColISO, 60)
		table.SetColumnWidth(locationColCountry, 180)
		table.SetColumnWidth(locationColCity, 180)
		table.SetColumnWidth(locationColPing, 90)
		table.SetColumnWidth(locationColStar, 40)

		table.OnSelected = func(id widget.TableCellID) {
			if id.Row == 0 {
				switch id.Col {
				case locationColFlag:
					table.UnselectAll()
					return
				case locationColStar:
					bookmarksFirst = !bookmarksFirst
					refreshTable()
					table.UnselectAll()
					return
				}

				clickedColumn := sortByColumn[id.Col]
				if clickedColumn == sortColumn {
					sortAscending = !sortAscending
				} else {
					sortColumn = clickedColumn
					sortAscending = true
				}

				refreshTable()
				table.UnselectAll()
				return
			}

			if id.Col == locationColStar {
				toggleBookmark(filteredLocations[id.Row-1])
				refreshTable()
				table.UnselectAll()
				return
			}

			selectedLocation := filteredLocations[id.Row-1]
			fmt.Printf("Selected: %+v\n", selectedLocation)
			go u.vpnmgr.ConnectToLocation(selectedLocation)
			window.Hide()
		}

		filterEntry.OnChanged = func(query string) {
			currentFilter = query
			refreshTable()
		}

		content := container.NewBorder(filterEntry, nil, nil, nil, table)
		window.SetContent(content)

		window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
			if k.Name == fyne.KeyEscape {
				window.Hide()
			}
		})

		window.Show()

		go func() {
			locs := u.vpnmgr.ListLocations()
			fyne.Do(func() {
				if len(locs) > 0 {
					pruned, pruneErr := commands.PruneAndSaveLocationBookmarks(bookmarks, locs)
					if pruneErr != nil {
						fmt.Printf("failed to prune location bookmarks: %v\n", pruneErr)
					} else {
						bookmarks = pruned
					}
				}
				allLocations = applyBookmarkFlags(locs)
				refreshTable()
			})
		}()
	})
}

func showInfiniteProgressDialog(title, message string, window fyne.Window) func() {
	bar := widget.NewProgressBarInfinite()
	content := container.NewVBox(widget.NewLabel(message), bar)
	d := dialog.NewCustomWithoutButtons(title, content, window)
	d.Show()
	return func() {
		fyne.Do(func() {
			bar.Stop()
			d.Hide()
		})
	}
}
