package main

import (
	"adgui/theme"
	_ "embed"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

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

// Location представляет информацию о локации VPN
type Location struct {
	ISO     string
	Country string
	City    string
}

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
		output, err := v.executeCommand("connect")
		if err != nil {
			fmt.Printf("Connect auto error: %v\nOutput: %s\n", err, output)
			return
		}

		if strings.Contains(output, statusConnectedTo) {
			v.mutex.Lock()
			v.isConnected = true
			// Извлекаем название локации из вывода
			lines := strings.SplitSeq(output, "\n")
			for line := range lines {
				if strings.Contains(line, statusConnectedTo) {
					parts := strings.Split(line, statusConnectedTo)
					if len(parts) > 1 {
						v.location = strings.TrimSpace(parts[1])
					}
					break
				}
			}
			v.mutex.Unlock()

			select {
			case v.updateChan <- true:
			default:
			}
		}
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
	locations := v.parseLocations(output)
	if len(locations) == 0 {
		fmt.Println("No locations found")
		return
	}

	// Создаем окно с выбором локации
	v.showLocationSelector(locations)
}

// parseLocations парсит вывод команды list-locations
func (v *VPNManager) parseLocations(output string) []Location {
	lines := strings.Split(output, "\n")
	var locations []Location

	for i, line := range lines {
		// Пропускаем заголовки (первые 2 строки)
		if i < 2 {
			continue
		}

		// Убираем ANSI escape codes и лишние пробелы
		line = strings.ReplaceAll(line, "\x1b[1m", "")
		line = strings.ReplaceAll(line, "\x1b[0m", "")
		line = strings.TrimRight(line, " ")

		if strings.TrimSpace(line) == "" {
			continue
		}

		// Парсим строку с фиксированной шириной колонок
		// Формат: ISO(6) COUNTRY(21) CITY(31) PING ESTIMATE
		//         0-5   6-26        27-57    58+

		if len(line) < 27 {
			continue
		}

		iso := strings.TrimSpace(line[0:6])
		country := strings.TrimSpace(line[6:27])
		city := strings.TrimSpace(line[27:58])

		// Пропускаем заголовок
		if iso == "ISO" || city == "CITY" {
			continue
		}

		if iso != "" && country != "" && city != "" {
			locations = append(locations, Location{
				ISO:     iso,
				Country: country,
				City:    city,
			})
		}
	}

	return locations
}

func (v *VPNManager) showLocationSelector(locations []Location) {
	// Создаем новое окно для выбора локации
	window := v.app.NewWindow("Select Location")
	window.Resize(fyne.NewSize(400, 500))

	// Создаем список локаций в формате "City, Country"
	locationStrings := make([]string, len(locations))
	for i, loc := range locations {
		locationStrings[i] = fmt.Sprintf("%s, %s", loc.City, loc.Country)
	}

	// Создаем список локаций
	list := widget.NewList(
		func() int {
			return len(locationStrings)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(locationStrings[id])
		},
	)

	// Обработчик выбора локации
	list.OnSelected = func(id widget.ListItemID) {
		// Используем только название города для подключения
		city := locations[id].City
		go v.connectToLocation(city)
		window.Close()
	}

	container := container.NewBorder(nil, nil, nil, nil, list)
	window.SetContent(container)
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
	v.statusTicker = time.NewTicker(10 * time.Second)
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
		fmt.Printf("status check: disconnected")
		v.mutex.Unlock()
	} else if strings.Contains(output, "Connected to") {
		// Извлекаем название локации из статуса
		lines := strings.SplitSeq(output, "\n")
		for line := range lines {
			if strings.Contains(line, "Connected to") {
				v.mutex.Lock()
				v.location = strings.TrimSpace(line)
				v.isConnected = true
				fmt.Printf("status check: connected to %s", v.location)
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
