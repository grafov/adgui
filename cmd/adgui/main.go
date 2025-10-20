package main

import (
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
}

func main() {
	myApp := app.NewWithID("AdGuard VPN Client")
	myApp.SetIcon(getDefaultIcon())

	if desk, ok := myApp.(desktop.App); ok {
		vpnManager := &VPNManager{
			app:     myApp,
			deskApp: desk,
		}
		vpnManager.createTrayMenu()
		vpnManager.updateTrayIcon()

		// Запуск фоновой проверки статуса
		go vpnManager.startStatusChecker()

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

	v.menu = fyne.NewMenu("AdGuard VPN",
		connectAuto,
		connectTo,
		fyne.NewMenuItemSeparator(),
		disconnect,
	)

	v.deskApp.SetSystemTrayMenu(v.menu)
	v.updateMenuItems()
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
		}
	}
}

func (v *VPNManager) updateTrayIcon() {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	var trayColor color.Color
	// var tooltip string // TODO: Использовать для установки tooltip

	if v.isConnected {
		trayColor = ConnectedColor
		// tooltip = v.location
	} else if strings.Contains(v.status, statusDisconnected) {
		trayColor = WarningColor
		// tooltip = "VPN disconnected"
	} else {
		trayColor = DisconnectedColor
		// tooltip = "VPN disconnected"
	}

	v.deskApp.SetSystemTrayIcon(getColoredIcon(trayColor))
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
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				if strings.Contains(line, statusConnectedTo) {
					parts := strings.Split(line, statusConnectedTo)
					if len(parts) > 1 {
						v.location = strings.TrimSpace(parts[1])
					}
					break
				}
			}
			v.mutex.Unlock()

			v.updateMenuItems()
			v.updateTrayIcon()
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

		v.updateMenuItems()
		v.updateTrayIcon()
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

		v.updateMenuItems()
		v.updateTrayIcon()
	}()
}

func (v *VPNManager) startStatusChecker() {
	time.Sleep(3 * time.Second) // Начальная задержка 3 секунды

	v.statusTicker = time.NewTicker(15 * time.Second)
	defer v.statusTicker.Stop()

	for range v.statusTicker.C {
		v.checkStatus()
	}
}

func (v *VPNManager) checkStatus() {
	if !v.isConnected {
		return
	}

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
		v.mutex.Unlock()
	} else if strings.Contains(output, "Connected to") {
		// Извлекаем название локации из статуса
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Connected to") {
				v.mutex.Lock()
				v.location = strings.TrimSpace(line)
				v.mutex.Unlock()
				break
			}
		}
	}

	v.updateMenuItems()
	v.updateTrayIcon()
}

// Вспомогательные функции для создания иконок
func getDefaultIcon() fyne.Resource {
	// Используем встроенную иконку из темы
	return fyne.CurrentApp().Settings().Theme().Icon("settings")
}

func getColoredIcon(c color.Color) fyne.Resource {
	// Используем ту же иконку
	return getDefaultIcon()
}
