package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"adgui/config"
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

// RunningCommand represents information about a currently executing CLI command.
// It includes the unique identifier, process ID, command executable path, arguments,
// and the time when the execution started.
type RunningCommand struct {
	ID        uint64
	PID       int
	Path      string
	Args      []string
	StartedAt time.Time
}

type VPNManager struct {
	statusTicker   *time.Ticker
	onStatusChange func()
	checkReqs      chan struct{}

	// all below protected by mutex
	statemx            sync.Mutex
	status             string
	location           string
	isConnected        bool
	siteExclusionsMode SiteExclusionMode

	// command queue tracking
	queueMx       sync.Mutex
	runningCmds   map[uint64]*exec.Cmd
	cmdInfos      map[uint64]RunningCommand
	nextCmdID     uint64
	onQueueChange func()
}

func New() *VPNManager {
	mgr := VPNManager{
		checkReqs:   make(chan struct{}, 1),
		runningCmds: make(map[uint64]*exec.Cmd),
		cmdInfos:    make(map[uint64]RunningCommand),
	}
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

func (v *VPNManager) SiteExclusionsMode() SiteExclusionMode {
	v.statemx.Lock()
	defer v.statemx.Unlock()
	if v.siteExclusionsMode == "" {
		return SiteExclusionModeGeneral
	}
	return v.siteExclusionsMode
}

func (v *VPNManager) SetStatusChangeCallback(callback func()) {
	v.statemx.Lock()
	defer v.statemx.Unlock()
	v.onStatusChange = callback
}

func (v *VPNManager) executeCommand(args ...string) (string, error) {
	cmdPath := resolveCommandPath()
	cmd := exec.Command(cmdPath, args...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		return "", err
	}

	_, done := v.registerCommand(cmdPath, args, cmd)
	defer done()

	err := cmd.Wait()
	return buf.String(), err
}

func (v *VPNManager) registerCommand(path string, args []string, cmd *exec.Cmd) (uint64, func()) {
	v.queueMx.Lock()
	v.nextCmdID++
	id := v.nextCmdID

	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	v.runningCmds[id] = cmd
	v.cmdInfos[id] = RunningCommand{
		ID:        id,
		PID:       pid,
		Path:      path,
		Args:      args,
		StartedAt: time.Now(),
	}
	callback := v.onQueueChange
	v.queueMx.Unlock()

	if callback != nil {
		callback()
	}

	return id, func() {
		v.queueMx.Lock()
		delete(v.runningCmds, id)
		delete(v.cmdInfos, id)
		callback := v.onQueueChange
		v.queueMx.Unlock()

		if callback != nil {
			callback()
		}
	}
}

// RunningCommands returns a list of all currently running CLI commands.
// The returned slice is a snapshot of the current state and is safe for concurrent read.
func (v *VPNManager) RunningCommands() []RunningCommand {
	v.queueMx.Lock()
	defer v.queueMx.Unlock()

	cmds := make([]RunningCommand, 0, len(v.cmdInfos))
	for _, info := range v.cmdInfos {
		cmds = append(cmds, info)
	}
	return cmds
}

// SetCommandQueueChangeCallback sets a function to be called when the command queue changes.
func (v *VPNManager) SetCommandQueueChangeCallback(callback func()) {
	v.queueMx.Lock()
	defer v.queueMx.Unlock()
	v.onQueueChange = callback
}

// KillCommand attempts to terminate a specific running command by its ID.
// It sends SIGTERM first, and falls back to SIGKILL if the process does not terminate.
func (v *VPNManager) KillCommand(id uint64) error {
	v.queueMx.Lock()
	cmd, ok := v.runningCmds[id]
	info, hasInfo := v.cmdInfos[id]
	v.queueMx.Unlock()

	if !ok || cmd == nil || cmd.Process == nil {
		return fmt.Errorf("command not found or not running")
	}

	pid := cmd.Process.Pid
	if hasInfo && info.PID != 0 {
		pid = info.PID
	}

	killCmdStr, err := config.AdguardKillCmd()
	if err != nil {
		fmt.Printf("Config read error for kill cmd: %v\n", err)
	}
	if killCmdStr == "" {
		killCmdStr = os.Getenv("ADGUARD_KILL_CMD")
	}

	if killCmdStr != "" {
		fields := strings.Fields(killCmdStr)
		if len(fields) > 0 {
			fields = append(fields, fmt.Sprintf("%d", pid))
			killCmd := exec.Command(fields[0], fields[1:]...)
			output, err := killCmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to run kill command %v: %w (output: %s)", fields, err, string(output))
			}
			return nil
		}
	}

	err = cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return cmd.Process.Kill()
	}

	go func(p *os.Process) {
		time.Sleep(500 * time.Millisecond)
		if err := p.Signal(syscall.Signal(0)); err == nil {
			_ = p.Kill()
		}
	}(cmd.Process)

	return nil
}

// KillAllCommands terminates all currently running CLI commands.
func (v *VPNManager) KillAllCommands() {
	v.queueMx.Lock()
	ids := make([]uint64, 0, len(v.runningCmds))
	for id := range v.runningCmds {
		ids = append(ids, id)
	}
	v.queueMx.Unlock()

	for _, id := range ids {
		_ = v.KillCommand(id)
	}
}

func resolveCommandPath() string {
	cmdPath, err := config.AdguardCmd()
	if err != nil {
		fmt.Printf("Config read error: %v\n", err)
	}
	if cmdPath == "" {
		cmdPath = os.Getenv("ADGUARD_CMD")
	}
	if cmdPath == "" {
		cmdPath = "adguardvpn-cli"
	}
	return cmdPath
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
	v.statemx.Lock()
	v.siteExclusionsMode = mode
	v.statemx.Unlock()
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

// SetSiteExclusionsMode switches the site exclusions mode in AdGuard VPN.
// Detailed explanation: It saves the domains of the previous mode to its corresponding file,
// executes the CLI mode switch command, loads the domains for the new target mode from its file,
// and applies them to the CLI.
func (v *VPNManager) SetSiteExclusionsMode(mode SiteExclusionMode, domains []string) error {
	prevMode := v.SiteExclusionsMode()

	if err := SaveExclusionsForMode(prevMode, domains); err != nil {
		return fmt.Errorf("failed to save exclusions for previous mode %s: %w", prevMode, err)
	}

	output, err := v.executeCommand("site-exclusions", "mode", mode.String())
	if err != nil {
		return fmt.Errorf("site-exclusions mode %s failed: %w, output: %s", mode, err, output)
	}

	newDomains, err := LoadExclusionsForMode(mode)
	if err != nil {
		return fmt.Errorf("failed to load exclusions for target mode %s: %w", mode, err)
	}

	for _, domain := range newDomains {
		if strings.TrimSpace(domain) == "" {
			continue
		}
		if err := v.AddSiteExclusion(domain); err != nil {
			return fmt.Errorf("re-applying domain %s failed: %w", domain, err)
		}
	}

	v.statemx.Lock()
	v.siteExclusionsMode = mode
	v.statemx.Unlock()
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
				location = strings.ReplaceAll(location, "\x1b[1m", "")
				location = strings.ReplaceAll(location, "\x1b[0m", "")
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
