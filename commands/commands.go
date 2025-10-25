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
	status       string
	location     string
	isConnected  bool
	mutex        sync.RWMutex
	statusTicker *time.Ticker
	updateChan   chan bool
}

func New(ui *ui.UI) *VPNManager {
	if desk, ok := ui.DesktopApp(); ok {
		vpnManager := &VPNManager{
			ui:         ui,
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
		return vpnManager
	}
	fmt.Println("System tray not supported")
	return nil
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
			items[1].Disabled = false          // Connect To... available always
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
		case v.updateChan <- true:
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
	case v.updateChan <- true:
	default:
	}
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
