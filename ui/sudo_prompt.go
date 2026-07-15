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

import (
	"errors"

	"adgui/commands"
	"adgui/commands/sudowrap"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

var errSudoPromptCancelled = errors.New("sudo password prompt cancelled")

func (u *UI) installSudoPasswordPrompt() {
	u.vpnmgr.SetPasswordPrompt(u.promptSudoPassword)
}

func (u *UI) promptSudoPassword() ([]byte, error) {
	resultCh := make(chan sudoPromptResult, 1)
	fyne.Do(func() {
		u.showSudoPasswordDialog(resultCh)
	})
	res := <-resultCh
	return res.password, res.err
}

type sudoPromptResult struct {
	password []byte
	err      error
}

func (u *UI) showSudoPasswordDialog(resultCh chan<- sudoPromptResult) {
	window := u.activeWindow()
	usedPromptWindow := window == u.promptWindow

	prompt := widget.NewLabel(lang.X("sudo.prompt.message", "Enter your password to manage VPN connections:"))
	prompt.Wrapping = fyne.TextWrapWord

	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder(lang.X("sudo.prompt.placeholder", "Password"))

	// Prompt on the first line, password field full-width below it.
	content := container.NewVBox(prompt, passEntry)

	d := dialog.NewCustomConfirm(
		lang.X("sudo.prompt.title", "Administrator Authentication"),
		lang.X("sudo.prompt.confirm", "OK"),
		lang.X("sudo.prompt.cancel", "Cancel"),
		content,
		func(confirmed bool) {
			if usedPromptWindow && u.promptWindow != nil {
				u.promptWindow.Hide()
			}
			if !confirmed {
				resultCh <- sudoPromptResult{err: errSudoPromptCancelled}
				return
			}
			password := []byte(passEntry.Text)
			passEntry.SetText("")
			resultCh <- sudoPromptResult{password: password}
		},
		window,
	)
	passEntry.OnSubmitted = func(_ string) {
		d.Confirm()
	}
	d.Resize(fyne.NewSize(420, d.MinSize().Height+8))
	d.Show()
	window.Canvas().Focus(passEntry)
}

// activeWindow picks a visible parent for modal dialogs (Wayland requires a shown window).
func (u *UI) activeWindow() fyne.Window {
	u.locationmx.RLock()
	loc := u.locationWindow
	locShown := u.locationShown
	u.locationmx.RUnlock()
	if loc != nil && locShown {
		return loc
	}

	u.dashboardmx.RLock()
	dash := u.dashboardWindow
	dashShown := u.dashboardShown
	u.dashboardmx.RUnlock()
	if dash != nil && dashShown {
		return dash
	}

	if u.promptWindow == nil {
		u.promptWindow = u.Fyne.NewWindow(lang.X("sudo.prompt.title", "Administrator Authentication"))
	}
	u.promptWindow.Show()
	return u.promptWindow
}

func (u *UI) runPrivileged(action func()) {
	go func() {
		if err := u.vpnmgr.EnsureSudoPassword(); err != nil {
			u.showSudoAuthError(err)
			return
		}
		action()
	}()
}

func (u *UI) showSudoAuthError(err error) {
	if errors.Is(err, errSudoPromptCancelled) || errors.Is(err, commands.ErrSudoPasswordRequired) {
		return
	}

	message := lang.X("sudo.error.generic", "Could not authenticate for privileged VPN operations.")
	if errors.Is(err, sudowrap.ErrInvalidPassword) {
		message = lang.X("sudo.error.invalid", "Incorrect password. VPN operation was not started.")
	} else if errors.Is(err, commands.ErrSudoPasswordPrompt) {
		message = lang.X("sudo.error.prompt", "Sudo password prompt is not available.")
	}

	fyne.Do(func() {
		dialog.ShowError(errors.New(message), u.activeWindow())
	})
}
