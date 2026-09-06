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

package ui

import "fyne.io/fyne/v2"

func dismissWindow(w fyne.Window, onGone func()) {
	if w == nil {
		return
	}
	if reuseHiddenWindows {
		w.Hide()
		if onGone != nil {
			onGone()
		}
		return
	}
	w.Close()
}

func windowHasOverlays(w fyne.Window) bool {
	if w == nil {
		return false
	}
	return len(w.Canvas().Overlays().List()) > 0
}

func (u *UI) releaseDashboardWindow(w fyne.Window) {
	u.dashboardmx.Lock()
	if u.dashboardWindow != w {
		u.dashboardmx.Unlock()
		return
	}
	u.dashboardWindow = nil
	u.dashboardShown = false
	u.dashboardTabs = nil
	u.dashboardConnectBtn = nil
	u.dashboardConnectionsWids = nil
	u.dashboardmx.Unlock()

	u.cmdQueuemx.Lock()
	u.cmdQueueRefreshFunc = nil
	u.cmdQueuemx.Unlock()

	u.ipRegionmx.Lock()
	u.ipRegionRefreshFunc = nil
	u.ipRegionmx.Unlock()
}

func (u *UI) releaseLocationWindow(w fyne.Window) {
	u.locationmx.Lock()
	defer u.locationmx.Unlock()
	if u.locationWindow != w {
		return
	}
	u.locationWindow = nil
	u.locationShown = false
}

func (u *UI) revealDashboard() bool {
	u.dashboardmx.Lock()
	defer u.dashboardmx.Unlock()
	if u.dashboardWindow == nil {
		return false
	}
	u.dashboardWindow.Show()
	u.dashboardShown = true
	return true
}
