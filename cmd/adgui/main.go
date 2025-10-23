package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"adgui/locations"
	"adgui/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// messages captured from adguard-cli stdout
const (
	statusDisconnected = "VPN is disconnected"
	statusConnectedTo  = "Successfully Connected to"
)

var (
	DisconnectedColor = color.NRGBA{R: 128, G: 128, B: 128, A: 255} // Серый
	ConnectedColor    = color.NRGBA{R: 0, G: 255, B: 0, A: 255}     // Зеленый
	WarningColor      = color.NRGBA{R: 255, G: 255, B: 0, A: 255}   // Желтый
)

type VPNManager struct {
	app          fyne.App
	deskApp      desktop.App
	menu         *fyne.Menu
	status       string
	location     string
	isConnected  bool
	mutex        sync.RWMutex
	statusTicker *time.Ticker
	updateChan   chan bool
}

func main() {
	myApp := app.NewWithID("AdGuard VPN Client")
	myApp.SetIcon(theme.DisconnectedIcon)

	if desk, ok := myApp.(desktop.App); ok {
		vpnManager := &VPNManager{
			app:        myApp,
			deskApp:    desk,
			updateChan: make(chan bool, 1),
		}
		vpnManager.createTrayMenu()

		// Запуск фоновой проверки статуса
		go vpnManager.startStatusChecker()
		go func() {
			for range vpnManager.updateChan {
				vpnManager.updateTrayIcon()
				vpnManager.updateMenuItems()
			}
		}()

		myApp.Run()
	} else {
		fmt.Println("System tray not supported")
	}
}

func (v *VPNManager) createTrayMenu() {
	connectAuto := fyne.NewMenuItem("Connect Auto", func() {
		v.connectAuto()
	})

	connectTo := fyne.NewMenuItem("Connect To...", func() {
		v.connectToList()
	})

	disconnect := fyne.NewMenuItem("Disconnect", func() {
		v.disconnect()
	})

	v.menu = fyne.NewMenu("AdGuard VPN Client",
		connectAuto,
		connectTo,
		fyne.NewMenuItemSeparator(),
		disconnect,
	)

	v.deskApp.SetSystemTrayMenu(v.menu)
}

func (v *VPNManager) updateMenuItems() {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Обновляем доступность пунктов меню
	if v.menu != nil {
		items := v.menu.Items
		if len(items) >= 3 {
			items[0].Disabled = v.isConnected  // Connect Auto
			items[1].Disabled = v.isConnected  // Connect To...
			items[3].Disabled = !v.isConnected // Disconnect
			v.menu.Items = items
		}
		if v.isConnected {
			v.menu.Label = "VPN connected"
		} else {
			v.menu.Label = "VPN disconected"
		}
		fyne.Do(func() {
			v.deskApp.SetSystemTrayMenu(v.menu)
		})
	}
}

func (v *VPNManager) updateTrayIcon() {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	fyne.Do(func() {
		if v.isConnected {
			v.deskApp.SetSystemTrayIcon(theme.ConnectedIcon)
		} else {
			v.deskApp.SetSystemTrayIcon(theme.DisconnectedIcon)
		}
	})
	// TODO: Установить tooltip для иконки в трее (если доступно)
}

func (v *VPNManager) executeCommand(args ...string) (string, error) {
	cmdPath := os.Getenv("ADGUARD_CMD")
	if cmdPath == "" {
		cmdPath = "adguardvpn-cli"
	}

	cmd := exec.Command(cmdPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (v *VPNManager) connectAuto() {
	go func() {
		// Получаем список локаций
		output, err := v.executeCommand("list-locations")
		if err != nil {
			fmt.Printf("List locations error: %v\nOutput: %s\n", err, output)
			return
		}

		// Парсим список локаций
		actualLocations := locations.ParseLocations(output)
		if len(actualLocations) == 0 {
			fmt.Println("No locations found for auto-connect")
			return
		}

		// Находим локацию с минимальным пингом
		fastest := locations.FindFastestLocation(actualLocations)
		if fastest == nil {
			fmt.Println("Could not find fastest location")
			return
		}

		// Подключаемся к самому быстрому серверу
		v.connectToLocation(fastest.City)
	}()
}

func (v *VPNManager) connectToList() {
	// Получаем список локаций
	output, err := v.executeCommand("list-locations")
	if err != nil {
		fmt.Printf("List locations error: %v\nOutput: %s\n", err, output)
		return
	}

	// Парсим список локаций
	actualLocations := locations.ParseLocations(output)
	if len(actualLocations) == 0 {
		fmt.Println("No locations found")
		return
	}

	// Создаем окно с выбором локации
	v.showLocationSelector(actualLocations)
}

func (v *VPNManager) showLocationSelector(locations []locations.Location) {
	// Создаем новое окно для выбора локации
	window := v.app.NewWindow("adgui: select location")
	window.Resize(fyne.NewSize(500, 600))

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
		go v.connectToLocation(city)
		window.Close()
	}

	window.SetContent(container.NewStack(table))
	window.Show()
}

func (v *VPNManager) connectToLocation(city string) {
	output, err := v.executeCommand("connect", "-l", city)
	if err != nil {
		fmt.Printf("Connect to location error: %v\nOutput: %s\n", err, output)
		return
	}

	if strings.Contains(output, statusConnectedTo) {
		v.mutex.Lock()
		v.isConnected = true
		v.location = city
		v.mutex.Unlock()

		select {
		case v.updateChan <- true:
		default:
		}
	}
}

func (v *VPNManager) disconnect() {
	go func() {
		output, err := v.executeCommand("disconnect")
		if err != nil {
			fmt.Printf("Disconnect error: %v\nOutput: %s\n", err, output)
			return
		}

		v.mutex.Lock()
		v.isConnected = false
		v.location = ""
		v.status = statusDisconnected
		v.mutex.Unlock()

		select {
		case v.updateChan <- true:
		default:
		}
	}()
}

func (v *VPNManager) startStatusChecker() {
	time.Sleep(2 * time.Second) // Начальная задержка 3 секунды
	v.checkStatus()

	// Regular checks
	v.statusTicker = time.NewTicker(30 * time.Second)
	defer v.statusTicker.Stop()
	for range v.statusTicker.C {
		v.checkStatus()
	}
}

func (v *VPNManager) checkStatus() {
	output, err := v.executeCommand("status")
	if err != nil {
		fmt.Printf("Status check error: %v\n", err)
		return
	}

	v.mutex.Lock()
	v.status = output
	v.mutex.Unlock()

	// Проверяем статус
	if strings.Contains(output, statusDisconnected) {
		v.mutex.Lock()
		v.isConnected = false
		v.location = ""
		fmt.Printf("status check: disconnected\n")
		v.mutex.Unlock()
	} else if strings.Contains(output, "Connected to") {
		// Извлекаем название локации из статуса
		lines := strings.SplitSeq(output, "\n")
		for line := range lines {
			if strings.Contains(line, "Connected to") {
				v.mutex.Lock()
				v.location = strings.TrimSpace(line)
				v.isConnected = true
				fmt.Printf("status check: connected to %s\n", v.location)
				v.mutex.Unlock()
				break
			}
		}
	}

	select {
	case v.updateChan <- true:
	default:
	}
}
