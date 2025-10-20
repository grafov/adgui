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
	isChecking   bool
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
		if len(items) >= 4 {
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
	} else if strings.Contains(v.status, "VPN is disconnected") {
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

		if strings.Contains(output, "Successfully Connected to") {
			v.mutex.Lock()
			v.isConnected = true
			// Извлекаем название локации из вывода
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				if strings.Contains(line, "Successfully Connected to") {
					parts := strings.Split(line, "Successfully Connected to")
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

func (v *VPNManager) parseLocations(output string) []string {
	lines := strings.Split(output, "\n")
	var locations []string

	for _, line := range lines {
		// Парсим строку с локацией, предполагая что CITY это последняя колонка
		fields := strings.Fields(line)
		if len(fields) > 0 {
			// Берем последнее поле как название города
			city := fields[len(fields)-1]
			if city != "" && city != "CITY" { // Исключаем заголовок
				locations = append(locations, city)
			}
		}
	}

	return locations
}

func (v *VPNManager) showLocationSelector(locations []string) {
	// Создаем новое окно для выбора локации
	window := v.app.NewWindow("Select Location")
	window.Resize(fyne.NewSize(300, 400))

	// Создаем список локаций
	list := widget.NewList(
		func() int {
			return len(locations)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(locations[id])
		},
	)

	// Обработчик выбора локации
	list.OnSelected = func(id widget.ListItemID) {
		city := locations[id]
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

	if strings.Contains(output, "Successfully Connected to") {
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
		v.status = "VPN is disconnected"
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
	if strings.Contains(output, "VPN is disconnected") {
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
