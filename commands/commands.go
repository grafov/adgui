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

// SiteExclusionMode represents CLI exclusion mode.
type SiteExclusionMode string

const (
	SiteExclusionModeGeneral   SiteExclusionMode = "general"
	SiteExclusionModeSelective SiteExclusionMode = "selective"
)

func (m SiteExclusionMode) String() string {
	return string(m)
}

func parseSiteExclusionMode(line string) SiteExclusionMode {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "selective"):
		return SiteExclusionModeSelective
	case strings.Contains(lower, "general"):
		return SiteExclusionModeGeneral
	default:
		return SiteExclusionModeGeneral
	}
}

type VPNManager struct {
	statusTicker   *time.Ticker
	onStatusChange func()
	checkReqs      chan struct{}

	// all below protected by mutex
	statemx     sync.Mutex
	status      string
	location    string
	isConnected bool
}

func New() *VPNManager {
	mgr := VPNManager{checkReqs: make(chan struct{}, 1)}
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
		return
	}
	v.checkReqs <- struct{}{}
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

// GetSiteExclusions retrieves current exclusion mode and domain list from CLI output.
func (v *VPNManager) GetSiteExclusions() (SiteExclusionMode, []string, error) {
	output, err := v.executeCommand("site-exclusions", "show")
	if err != nil {
		return SiteExclusionModeGeneral, nil, fmt.Errorf("site-exclusions show failed: %w, output: %s", err, output)
	}

	lines := strings.Split(output, "\n")
	mode := SiteExclusionModeGeneral
	var exclusions []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Parse header lines like "Exclusions for GENERAL mode:"
		if strings.Contains(strings.ToLower(trimmed), "exclusions for") {
			mode = parseSiteExclusionMode(trimmed)
			continue
		}
		// Treat any remaining non-empty line as a domain entry.
		exclusions = append(exclusions, trimmed)
	}
	return mode, exclusions, nil
}

// AddSiteExclusion appends a domain to the exclusions list via CLI.
func (v *VPNManager) AddSiteExclusion(domain string) error {
	output, err := v.executeCommand("site-exclusions", "add", domain)
	if err != nil {
		return fmt.Errorf("site-exclusions add failed: %w, output: %s", err, output)
	}
	return nil
}

// RemoveSiteExclusion removes a domain from the exclusions list via CLI.
func (v *VPNManager) RemoveSiteExclusion(domain string) error {
	output, err := v.executeCommand("site-exclusions", "remove", domain)
	if err != nil {
		return fmt.Errorf("site-exclusions remove failed: %w, output: %s", err, output)
	}
	return nil
}

// SetSiteExclusionsMode switches mode and re-applies provided domains.
func (v *VPNManager) SetSiteExclusionsMode(mode SiteExclusionMode, domains []string) error {
	output, err := v.executeCommand("site-exclusions", "mode", mode.String())
	if err != nil {
		return fmt.Errorf("site-exclusions mode %s failed: %w, output: %s", mode, err, output)
	}
	for _, domain := range domains {
		if strings.TrimSpace(domain) == "" {
			continue
		}
		if err := v.AddSiteExclusion(domain); err != nil {
			return fmt.Errorf("re-applying domain %s failed: %w", domain, err)
		}
	}
	return nil
}

func (v *VPNManager) statusCheckLoop() {
	time.Sleep(startDelay)
	v.checkStatus()

	// Regular checks
	v.statusTicker = time.NewTicker(60 * time.Second)
	defer v.statusTicker.Stop()
	for {
		select {
		case <-v.checkReqs:
			time.Sleep(startDelay)
			v.checkStatus()
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
