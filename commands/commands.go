package commands

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"adgui/locations"
	"adgui/theme"
	"adgui/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

const startDelay = 3 * time.Second // Начальная задержка

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
	ui           *ui.UI
	deskApp      desktop.App
	menu         *fyne.Menu
	statusTicker *time.Ticker
	updateReqs   chan struct{}
	checkReqs    chan struct{}

	mutex       sync.RWMutex
	status      string
	location    string
	isConnected bool
}

func New(ui *ui.UI) *VPNManager {
	if desk, ok := ui.DesktopApp(); ok {
		vpnManager := &VPNManager{
			ui:         ui,
			deskApp:    desk,
			updateReqs: make(chan struct{}, 1),
			checkReqs:  make(chan struct{}, 1),
		}
		vpnManager.createTrayMenu()

		// Запуск фоновой проверки статуса
		go vpnManager.statusCheckLoop()
		go vpnManager.updateUI()
		return vpnManager
	}
	fmt.Println("System tray not supported")
	return nil
}

func (v *VPNManager) updateUI() {
	select {
	case v.checkReqs <- struct{}{}:
		time.Sleep(200 * time.Millisecond)
	default:
	}
	for range v.updateReqs {
		v.updateTrayIcon()
		v.updateMenuItems()
	}
}

func (v *VPNManager) createTrayMenu() {
	status := fyne.NewMenuItem("Adguard VPN", func() {})
	dashboard := fyne.NewMenuItem("Show dashboard", func() {
		v.dashboard()
	})
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
		status,
		dashboard,
		connectAuto,
		connectTo,
		fyne.NewMenuItemSeparator(),
		disconnect,
	)
	v.menu.Items[0].Disabled = true // status field

	v.deskApp.SetSystemTrayMenu(v.menu)
}

func (v *VPNManager) updateMenuItems() {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Обновляем доступность пунктов меню
	if v.menu != nil {
		items := v.menu.Items
		if v.isConnected {
			v.menu.Label = "VPN connected"
			items[0].Icon = theme.MenuConnectedIcon
			items[0].Label = strings.ToUpper(v.location)
		} else {
			v.menu.Label = "VPN disconected"
			items[0].Icon = theme.MenuDisconnectedIcon
			items[0].Label = "OFF"
		}
		items[1].Disabled = true           // FIXME Dashboard yet not ready
		items[2].Disabled = v.isConnected  // Connect Auto
		items[3].Disabled = false          // Connect To... available always
		items[4].Disabled = !v.isConnected // Disconnect
		v.menu.Items = items
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
	// Получаем список локаций
	output, err := v.executeCommand("connect")
	if err != nil {
		fmt.Printf("Could not connect: %s: %s\n", err, output)
		// Раз что-то пошло не так, на всякий случай стоит подождать, перед
		// новой попыткой коннекта.
		time.Sleep(1 * time.Second)
	}

	// Идём альтернативным путём.
	// Парсим список локаций и выбираем самую быструю, для коннекта к ней.
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
	v.ui.ShowLocationSelector(actualLocations, v.connectToLocation)
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
		case v.updateReqs <- struct{}{}:
		default:
		}
	}
}

func (v *VPNManager) disconnect() {
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
	case v.updateReqs <- struct{}{}:
	default:
	}
}

func (v *VPNManager) showLicense() {
	output, err := v.executeCommand("license")
	if err != nil {
		fmt.Printf("Show license error: %v\nOutput: %s\n", err, output)
		return
	}
	ui.New().ShowLicense(output)

	select {
	case v.updateReqs <- struct{}{}:
	default:
	}
}

func (v *VPNManager) statusCheckLoop() {
	time.Sleep(startDelay)
	v.checkStatus()

	// Regular checks
	v.statusTicker = time.NewTicker(30 * time.Second)
	defer v.statusTicker.Stop()
	select {
	case <-v.checkReqs:
		v.checkStatus()
	case <-v.statusTicker.C:
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
				// Извлекаем название локации между ANSI кодами
				// Формат: "Connected to [1mLOCATION[0m in ..."
				location := line
				// Удаляем префикс до названия локации
				prefix := "Connected to "
				if idx := strings.Index(location, prefix); idx >= 0 {
					location = location[idx+len(prefix):]
				}
				// Удаляем ANSI коды жирного шрифта
				location = strings.ReplaceAll(location, "[1m", "")
				location = strings.ReplaceAll(location, "[0m", "")
				// Удаляем суффикс после названия локации
				if idx := strings.Index(location, " in "); idx >= 0 {
					location = location[:idx]
				}
				// Очищаем от пробелов
				location = strings.TrimSpace(location)

				v.mutex.Lock()
				v.location = location
				v.isConnected = true
				fmt.Printf("status check: connected to %s\n", v.location)
				v.mutex.Unlock()
				break
			}
		}
	}

	select {
	case v.updateReqs <- struct{}{}:
	default:
	}
}

func (v *VPNManager) dashboard() {
	v.menu.Items[0].Disabled = true
	_ = ui.New().Dashboard() // FIXME
	select {
	case v.updateReqs <- struct{}{}:
	default:
	}
}
