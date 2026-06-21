package ui

import (
	"strings"

	"adgui/commands"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (u *UI) cmdQueuePanel() *fyne.Container {
	var running []commands.RunningCommand
	var list *widget.List

	refreshQueue := func() {
		running = u.vpnmgr.RunningCommands()
		if list != nil {
			list.Refresh()
		}
	}

	u.cmdQueuemx.Lock()
	u.cmdQueueRefreshFunc = refreshQueue
	u.cmdQueuemx.Unlock()

	u.vpnmgr.SetCommandQueueChangeCallback(func() {
		select {
		case u.updateReqs <- struct{}{}:
		default:
		}
	})

	list = widget.NewList(
		func() int {
			return len(running)
		},
		func() fyne.CanvasObject {
			pidLabel := widget.NewLabel("PID: 1234567")
			cmdLabel := widget.NewLabel("adguardvpn-cli status")
			timeLabel := widget.NewLabel("Started: 00:00:00")
			killBtn := widget.NewButton(lang.X("cmd_queue.kill", "Kill"), nil)

			return container.NewHBox(pidLabel, cmdLabel, timeLabel, layout.NewSpacer(), killBtn)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			cont := obj.(*fyne.Container)
			pidLabel := cont.Objects[0].(*widget.Label)
			cmdLabel := cont.Objects[1].(*widget.Label)
			timeLabel := cont.Objects[2].(*widget.Label)
			killBtn := cont.Objects[4].(*widget.Button)

			if id >= len(running) {
				return
			}
			cmd := running[id]

			pidLabel.SetText(lang.X("cmd_queue.pid", "PID: {{.PID}}", map[string]any{"PID": cmd.PID}))

			fullCmd := cmd.Path + " " + strings.Join(cmd.Args, " ")
			if len(fullCmd) > 50 {
				fullCmd = fullCmd[:47] + "..."
			}
			cmdLabel.SetText(fullCmd)

			timeLabel.SetText(lang.X("cmd_queue.started", "Started: {{.Time}}", map[string]any{
				"Time": cmd.StartedAt.Format("15:04:05"),
			}))

			killBtn.OnTapped = func() {
				dialog.ShowConfirm(
					lang.X("cmd_queue.kill.confirm.title", "Kill Command"),
					lang.X("cmd_queue.kill.confirm.message", "Are you sure you want to kill PID {{.PID}}?", map[string]any{"PID": cmd.PID}),
					func(ok bool) {
						if ok {
							go func(targetID uint64) {
								if err := u.vpnmgr.KillCommand(targetID); err != nil {
									fyne.Do(func() {
										dialog.ShowError(err, u.dashboardWindow)
									})
								}
							}(cmd.ID)
						}
					},
					u.dashboardWindow,
				)
			}
		},
	)

	killAllBtn := widget.NewButton(lang.X("cmd_queue.kill_all", "Kill All"), func() {
		dialog.ShowConfirm(
			lang.X("cmd_queue.kill_all.confirm.title", "Kill All"),
			lang.X("cmd_queue.kill_all.confirm.message", "Are you sure you want to kill all running commands?"),
			func(ok bool) {
				if ok {
					go u.vpnmgr.KillAllCommands()
				}
			},
			u.dashboardWindow,
		)
	})

	refreshQueue()

	bottomControls := container.NewHBox(layout.NewSpacer(), killAllBtn, layout.NewSpacer())
	content := container.NewBorder(nil, bottomControls, nil, nil, list)
	return content
}
