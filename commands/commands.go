package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"adgui/locations"
)

const startDelay = 3 * time.Second // Начальная задержка

// messages captured from adguard-cli stdout
const (
	statusDisconnected = "VPN is disconnected"
	statusConnectedTo  = "Successfully Connected to"
)

type VPNManager struct {
	statusTicker   *time.Ticker
	onStatusChange func()

	// all below protected by mutex
	statemx     sync.Mutex
	status      string
	location    string
	isConnected bool
}

func New() *VPNManager {
	mgr := VPNManager{}
	go mgr.statusCheckLoop()
	return &mgr
}

func (v *VPNManager) Location() string {
	v.statemx.Lock()
	defer v.statemx.Unlock()
	return v.location
}

func (v *VPNManager) Status() string {
	v.statemx.Lock()
	defer v.statemx.Unlock()
	return v.status
}

func (v *VPNManager) IsConnected() bool {
	v.statemx.Lock()
	defer v.statemx.Unlock()
	return v.isConnected
}

func (v *VPNManager) SetStatusChangeCallback(callback func()) {
	v.statemx.Lock()
	defer v.statemx.Unlock()
	v.onStatusChange = callback
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

func (v *VPNManager) ConnectAuto() {
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
	v.ConnectToLocation(fastest.City)
}

func (v *VPNManager) ListLocations() []locations.Location {
	// Получаем список локаций
	output, err := v.executeCommand("list-locations")
	if err != nil {
		fmt.Printf("List locations error: %v\nOutput: %s\n", err, output)
		return nil
	}

	// Парсим список локаций
	actualLocations := locations.ParseLocations(output)
	if len(actualLocations) == 0 {
		fmt.Println("No locations found")
		return nil
	}
	return actualLocations
}

func (v *VPNManager) ConnectToLocation(city string) {
	output, err := v.executeCommand("connect", "-l", city)
	if err != nil {
		fmt.Printf("Connect to location error: %v\nOutput: %s\n", err, output)
		return
	}

	if strings.Contains(output, statusConnectedTo) {
		v.statemx.Lock()
		v.isConnected = true
		v.location = city
		callback := v.onStatusChange
		v.statemx.Unlock()
		if callback != nil {
			callback()
		}
	}
}

func (v *VPNManager) Disconnect() {
	output, err := v.executeCommand("disconnect")
	if err != nil {
		fmt.Printf("Disconnect error: %v\nOutput: %s\n", err, output)
		return
	}

	v.statemx.Lock()
	v.isConnected = false
	v.location = ""
	v.status = statusDisconnected
	callback := v.onStatusChange
	v.statemx.Unlock()
	if callback != nil {
		callback()
	}
}

func (v *VPNManager) License() string {
	output, err := v.executeCommand("license")
	if err != nil {
		fmt.Printf("Show license error: %v\nOutput: %s\n", err, output)
		return ""
	}
	return output
}

func (v *VPNManager) statusCheckLoop() {
	time.Sleep(startDelay)
	v.checkStatus()

	// Regular checks
	v.statusTicker = time.NewTicker(30 * time.Second)
	defer v.statusTicker.Stop()
	for {
		select {
		// case <-v.checkReqs:
		// 	v.checkStatus()
		case <-v.statusTicker.C:
			v.checkStatus()
		}
	}
}

func (v *VPNManager) checkStatus() {
	output, err := v.executeCommand("status")
	if err != nil {
		fmt.Printf("Status check error: %v\n", err)
		return
	}

	v.statemx.Lock()
	v.status = output
	v.statemx.Unlock()

	// Проверяем статус
	if strings.Contains(output, statusDisconnected) {
		v.statemx.Lock()
		v.isConnected = false
		v.location = ""
		callback := v.onStatusChange
		v.statemx.Unlock()
		fmt.Printf("status check: disconnected\n")
		if callback != nil {
			callback()
		}
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
				location = strings.ReplaceAll(location, `[1m`, ``)
				location = strings.ReplaceAll(location, `[0m`, ``)
				// Удаляем суффикс после названия локации
				if idx := strings.Index(location, " in "); idx >= 0 {
					location = location[:idx]
				}
				// Очищаем от пробелов
				location = strings.TrimSpace(location)

				v.statemx.Lock()
				v.location = location
				v.isConnected = true
				callback := v.onStatusChange
				v.statemx.Unlock()
				fmt.Printf("status check: connected to %s\n", location)
				if callback != nil {
					callback()
				}
				break
			}
		}
	}
}
