// Copyright (C) 2026 Alexander Grafov <grafov@inet.name>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package commands

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"adgui/commands/sudowrap"
	"adgui/config"
	"adgui/locations"
)

const startDelay = 3 * time.Second // Начальная задержка

var (
	// ErrSudoPasswordRequired is returned when sudo auth was cancelled or empty.
	ErrSudoPasswordRequired = errors.New("sudo password required")
	// ErrSudoPasswordPrompt is returned when the UI password prompt is not configured.
	ErrSudoPasswordPrompt = errors.New("sudo password prompt not configured")
)

// PasswordPrompt requests the user's sudo password from the UI layer.
type PasswordPrompt func() ([]byte, error)

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

	// all below protected by statemx
	statemx            sync.Mutex
	status             string
	location           string
	connectedLocation  locations.Location
	isConnected        bool
	siteExclusionsMode SiteExclusionMode

	// connection history (historyMx)
	historyMx          sync.Mutex
	history            []ConnectionHistoryEntry
	activeConnection   *ConnectionHistoryEntry
	locationsCache     []locations.Location
	locationsCacheTime time.Time

	// command queue tracking
	queueMx       sync.Mutex
	runningCmds   map[uint64]*exec.Cmd
	cmdInfos      map[uint64]RunningCommand
	nextCmdID     uint64
	onQueueChange func()

	sudoEnv       *sudowrap.Env
	passwordPrompt PasswordPrompt
	promptMx       sync.Mutex
}

func New() *VPNManager {
	mgr := VPNManager{
		checkReqs:   make(chan struct{}, 1),
		runningCmds: make(map[uint64]*exec.Cmd),
		cmdInfos:    make(map[uint64]RunningCommand),
	}
	if history, err := LoadConnectionHistory(); err != nil {
		fmt.Printf("load connections history error: %v\n", err)
	} else {
		mgr.history = history
	}

	enabled, err := config.AdguardSudoWrapEnabled()
	if err != nil {
		fmt.Printf("config read error for sudo wrap: %v\n", err)
		enabled = true
	}
	sudoEnv, err := sudowrap.Setup(enabled)
	if err != nil {
		fmt.Printf("sudo wrap setup error: %v\n", err)
	} else {
		mgr.sudoEnv = sudoEnv
	}

	go mgr.statusCheckLoop()
	return &mgr
}

// SetPasswordPrompt configures the UI callback used to collect the sudo password.
func (v *VPNManager) SetPasswordPrompt(prompt PasswordPrompt) {
	v.promptMx.Lock()
	defer v.promptMx.Unlock()
	v.passwordPrompt = prompt
}

// EnsureSudoPassword validates or collects sudo credentials for privileged CLI operations.
// A valid sudo ticket is enough (wrapper uses sudo -n path). Otherwise both the in-memory
// password and the on-disk .pass file are required for the askpass path.
func (v *VPNManager) EnsureSudoPassword() error {
	if v.sudoEnv == nil || !v.sudoEnv.Enabled() {
		return nil
	}
	if sudowrap.ValidTicket("") {
		return nil
	}
	if v.sudoEnv.ReadyForAskpass() {
		return nil
	}

	v.promptMx.Lock()
	prompt := v.passwordPrompt
	v.promptMx.Unlock()
	if prompt == nil {
		return ErrSudoPasswordPrompt
	}

	password, err := prompt()
	if err != nil {
		return err
	}
	defer sudowrapZero(password)

	if len(password) == 0 {
		return ErrSudoPasswordRequired
	}
	if err := v.sudoEnv.SetPassword(password); err != nil {
		return err
	}
	return nil
}

// Close wipes sudo session secrets and removes the private wrapper directory.
func (v *VPNManager) Close() error {
	if v.sudoEnv == nil {
		return nil
	}
	return v.sudoEnv.Close()
}

func sudowrapZero(password []byte) {
	for i := range password {
		password[i] = 0
	}
}

func (v *VPNManager) Location() string {
	v.statemx.Lock()
	defer v.statemx.Unlock()
	return v.location
}

func (v *VPNManager) ConnectedLocation() (locations.Location, bool) {
	v.statemx.Lock()
	defer v.statemx.Unlock()
	if !v.isConnected {
		return locations.Location{}, false
	}
	return v.connectedLocation, true
}

func (v *VPNManager) ConnectionHistory() []ConnectionHistoryEntry {
	v.historyMx.Lock()
	defer v.historyMx.Unlock()

	result := make([]ConnectionHistoryEntry, 0, len(v.history)+1)
	if v.activeConnection != nil {
		result = append(result, *v.activeConnection)
	}
	result = append(result, v.history...)
	if len(result) > maxHistoryEntries {
		result = result[:maxHistoryEntries]
	}
	return result
}

// PreviousConnectionHistory returns completed sessions only (excludes the active connection).
func (v *VPNManager) PreviousConnectionHistory() []ConnectionHistoryEntry {
	v.historyMx.Lock()
	defer v.historyMx.Unlock()

	result := make([]ConnectionHistoryEntry, len(v.history))
	copy(result, v.history)
	if len(result) > maxHistoryEntries {
		result = result[:maxHistoryEntries]
	}
	return result
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
	v.applySudoWrap(cmd)
	prepareCLICommand(cmd)

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

// prepareCLICommand detaches the child from the parent's controlling terminal so
// sudo inside adguardvpn-cli cannot prompt on the Konsole TTY that started adgui.
func prepareCLICommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
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

func (v *VPNManager) applySudoWrap(cmd *exec.Cmd) {
	if v.sudoEnv != nil {
		v.sudoEnv.Apply(cmd)
	}
}

func (v *VPNManager) ConnectAuto() {
	if err := v.EnsureSudoPassword(); err != nil {
		fmt.Printf("sudo auth error: %v\n", err)
		return
	}
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

func (v *VPNManager) ConnectToLocation(loc locations.Location) {
	if err := v.EnsureSudoPassword(); err != nil {
		fmt.Printf("sudo auth error: %v\n", err)
		return
	}
	output, err := v.executeCommand("connect", "-l", loc.City)
	if err != nil {
		fmt.Printf("Connect to location error: %v\nOutput: %s\n", err, output)
		return
	}

	if strings.Contains(output, statusConnectedTo) {
		v.applyConnected(loc)
	}
}

func (v *VPNManager) applyConnected(loc locations.Location) {
	v.statemx.Lock()
	wasConnected := v.isConnected
	prevLoc := v.connectedLocation
	v.isConnected = true
	v.location = loc.City
	v.connectedLocation = loc
	callback := v.onStatusChange
	v.statemx.Unlock()

	v.updateConnectionHistory(wasConnected, prevLoc, loc)

	if callback != nil {
		callback()
	}
}

func (v *VPNManager) applyDisconnected() {
	v.statemx.Lock()
	wasConnected := v.isConnected
	v.isConnected = false
	v.location = ""
	v.connectedLocation = locations.Location{}
	callback := v.onStatusChange
	v.statemx.Unlock()

	if wasConnected {
		v.finalizeActiveConnection()
	}

	if callback != nil {
		callback()
	}
}

func locationsEqual(a, b locations.Location) bool {
	return strings.EqualFold(a.City, b.City) && strings.EqualFold(a.Country, b.Country)
}

func (v *VPNManager) updateConnectionHistory(wasConnected bool, prevLoc, newLoc locations.Location) {
	v.historyMx.Lock()
	defer v.historyMx.Unlock()

	if wasConnected && !locationsEqual(prevLoc, newLoc) {
		v.finalizeActiveConnectionLocked()
		v.startActiveConnectionLocked(newLoc)
		return
	}
	if !wasConnected {
		v.startActiveConnectionLocked(newLoc)
	}
}

func (v *VPNManager) startActiveConnectionLocked(loc locations.Location) {
	now := time.Now()
	entry := ConnectionHistoryEntry{
		City:      loc.City,
		Country:   loc.Country,
		Ping:      loc.Ping,
		StartedAt: now,
	}
	v.activeConnection = &entry
}

func (v *VPNManager) finalizeActiveConnection() {
	v.historyMx.Lock()
	defer v.historyMx.Unlock()
	v.finalizeActiveConnectionLocked()
}

func (v *VPNManager) finalizeActiveConnectionLocked() {
	if v.activeConnection == nil {
		return
	}
	now := time.Now()
	v.activeConnection.EndedAt = &now
	v.history = prependHistoryEntry(v.history, *v.activeConnection)
	if len(v.history) > maxHistoryEntries {
		v.history = v.history[:maxHistoryEntries]
	}
	v.activeConnection = nil
	if err := SaveConnectionHistory(v.history); err != nil {
		fmt.Printf("save connections history error: %v\n", err)
	}
}

func prependHistoryEntry(entries []ConnectionHistoryEntry, entry ConnectionHistoryEntry) []ConnectionHistoryEntry {
	result := make([]ConnectionHistoryEntry, 0, len(entries)+1)
	result = append(result, entry)
	result = append(result, entries...)
	return result
}

func (v *VPNManager) resolveLocation(cityName string) locations.Location {
	if cityName == "" {
		return locations.Location{}
	}

	v.statemx.Lock()
	cached := v.locationsCache
	cacheTime := v.locationsCacheTime
	v.statemx.Unlock()

	if len(cached) == 0 || time.Since(cacheTime) > 5*time.Minute {
		cached = v.ListLocations()
		v.statemx.Lock()
		v.locationsCache = cached
		v.locationsCacheTime = time.Now()
		v.statemx.Unlock()
	}

	if found := locations.FindByCity(cached, cityName); found != nil {
		return *found
	}
	return locations.Location{City: cityName, Ping: -1}
}

var ansiStripRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func (v *VPNManager) CLIVersion() string {
	cmdPath := resolveCommandPath()
	cmd := exec.Command(cmdPath, "--version")
	v.applySudoWrap(cmd)
	prepareCLICommand(cmd)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		fmt.Printf("CLI version error: %v\nOutput: %s\n", err, buf.String())
		return "unavailable"
	}
	output := ansiStripRegex.ReplaceAllString(buf.String(), "")
	output = strings.TrimSpace(output)
	if output == "" {
		return "unavailable"
	}
	return output
}

func (v *VPNManager) Disconnect() {
	if err := v.EnsureSudoPassword(); err != nil {
		fmt.Printf("sudo auth error: %v\n", err)
		return
	}
	output, err := v.executeCommand("disconnect")
	if err != nil {
		fmt.Printf("Disconnect error: %v\nOutput: %s\n", err, output)
		return
	}

	v.statemx.Lock()
	v.status = statusDisconnected
	v.statemx.Unlock()
	v.applyDisconnected()
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
		fmt.Printf("status check: disconnected\n")
		v.applyDisconnected()
	} else if strings.Contains(output, "Connected to") {
		locationName := ParseLocationFromStatus(output)
		loc := v.resolveLocation(locationName)
		fmt.Printf("status check: connected to %s\n", locationName)
		v.applyConnected(loc)
	}
}
