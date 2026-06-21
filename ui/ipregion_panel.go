package ui

import (
	"context"
	"fmt"
	"strings"

	"adgui/config"
	"adgui/ipregion"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

func (u *UI) ipRegionPanel() *fyne.Container {
	var rows []ipregion.ServiceResult
	var showIPv6 bool
	var scanning bool
	var scanCancel context.CancelFunc

	header := widget.NewLabel(lang.X("ip_region.header", "Check how services on the network see your IP"))
	header.Wrapping = fyne.TextWrapWord

	progressLabel := widget.NewLabel("")
	progressLabel.Alignment = fyne.TextAlignCenter
	progressLabel.Hide()

	progressBar := widget.NewProgressBar()
	progressBar.Min = 0
	progressBar.Max = 1
	progressBar.TextFormatter = func() string { return "" }
	progressBar.Hide()

	progressArea := container.NewStack(progressBar, progressLabel)
	progressArea.Hide()

	vpnLabel := widget.NewLabel("")

	summaryTitle := widget.NewLabel(lang.X("ip_region.summary.title", "Top regions"))
	summaryTitle.TextStyle = fyne.TextStyle{Bold: true}
	summaryTitle.Hide()

	summaryEmpty := widget.NewLabel(lang.X("ip_region.no_results", "No region data collected."))
	summaryEmpty.Hide()

	_, _, sumColIPv6, summaryTableHeader := newIPRegionTableHeader(
		lang.X("ip_region.col.country", "Country"),
		lang.X("ip_region.col.ipv4", "IPv4"),
		lang.X("ip_region.col.ipv6", "IPv6"),
	)
	summaryTableHeader.Hide()

	summaryRowsBox := container.NewVBox()
	summaryRowsBox.Hide()

	_, _, colIPv6, tableHeader := newIPRegionTableHeader(
		lang.X("ip_region.col.service", "Service"),
		lang.X("ip_region.col.ipv4", "IPv4"),
		lang.X("ip_region.col.ipv6", "IPv6"),
	)

	refreshIPv6Columns := func() {
		if showIPv6 {
			sumColIPv6.Show()
			colIPv6.Show()
		} else {
			sumColIPv6.Hide()
			colIPv6.Hide()
		}
	}

	list := widget.NewList(
		func() int { return len(rows) },
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			name.Truncation = fyne.TextTruncateClip
			ipv4 := widget.NewLabel("")
			ipv4.Alignment = fyne.TextAlignCenter
			ipv6 := widget.NewLabel("")
			ipv6.Alignment = fyne.TextAlignCenter
			return container.NewGridWithColumns(3, name, ipv4, ipv6)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(rows) {
				return
			}
			row := rows[id]
			cont := obj.(*fyne.Container)
			nameLbl := cont.Objects[0].(*widget.Label)
			ipv4Lbl := cont.Objects[1].(*widget.Label)
			ipv6Lbl := cont.Objects[2].(*widget.Label)

			nameLbl.SetText(row.Service)
			v4, v6 := formatRegionColumns(row.IPv4, row.IPv6, showIPv6)
			ipv4Lbl.SetText(formatRegionCell(v4, mismatchVPN(u, row.IPv4)))
			if showIPv6 {
				ipv6Lbl.Show()
				ipv6Lbl.SetText(formatRegionCell(v6, mismatchVPN(u, row.IPv6)))
			} else {
				ipv6Lbl.Hide()
			}
		},
	)

	actionBtn := widget.NewButton(lang.X("ip_region.where_am_i", "Where am I?"), nil)

	updateProgress := func(p ipregion.Progress) {
		if p.Total > 0 {
			progressBar.SetValue(float64(p.Completed) / float64(p.Total))
			actionBtn.SetText(lang.X("ip_region.cancel_progress", "Cancel ({{.Done}}/{{.Total}})", map[string]any{
				"Done":  p.Completed,
				"Total": p.Total,
			}))
		}
		service := progressServiceLabel(p.Service)
		progressLabel.SetText(lang.X("ip_region.progress", "Checking {{.Service}} ({{.Done}}/{{.Total}})", map[string]any{
			"Service": service,
			"Done":    p.Completed,
			"Total":   p.Total,
		}))
	}

	hideProgress := func() {
		progressArea.Hide()
		progressBar.Hide()
		progressBar.SetValue(0)
		progressLabel.Hide()
	}

	showProgress := func() {
		progressBar.SetValue(0)
		progressBar.Show()
		progressLabel.Show()
		progressArea.Show()
	}

	showProgressStatus := func(text string) {
		progressLabel.SetText(text)
		progressBar.Hide()
		progressLabel.Show()
		progressArea.Show()
	}

	refreshSummary := func(report *ipregion.Report) {
		summaryRowsBox.Objects = nil
		s := ipregion.BuildSummary(report)
		if len(s.Countries) == 0 {
			summaryTitle.Hide()
			summaryTableHeader.Hide()
			summaryRowsBox.Hide()
			summaryEmpty.Show()
			return
		}
		summaryEmpty.Hide()
		summaryTitle.Show()
		summaryTableHeader.Show()
		for _, stat := range s.Countries {
			country := widget.NewLabel(fmt.Sprintf("%s %s", stat.Code, stat.Name))
			country.Truncation = fyne.TextTruncateClip
			ipv4 := widget.NewLabel(formatPercentCell(stat.IPv4Pct))
			ipv4.Alignment = fyne.TextAlignCenter
			ipv6 := widget.NewLabel(formatPercentCell(stat.IPv6Pct))
			ipv6.Alignment = fyne.TextAlignCenter
			if !showIPv6 {
				ipv6.Hide()
			}
			summaryRowsBox.Add(container.NewGridWithColumns(3, country, ipv4, ipv6))
		}
		summaryRowsBox.Show()
		summaryRowsBox.Refresh()
	}

	applyReport := func(report *ipregion.Report) {
		showIPv6 = report.ExternalIPv6 != ""
		rows = report.Results
		refreshIPv6Columns()
		vpnLabel.SetText(buildVPNCompareLine(u, report))
		refreshSummary(report)
		list.Refresh()
	}

	startScan := func() {
		if scanning {
			return
		}
		scanning = true
		rows = nil
		summaryTitle.Hide()
		summaryTableHeader.Hide()
		summaryRowsBox.Objects = nil
		summaryRowsBox.Hide()
		summaryEmpty.Hide()
		list.Refresh()
		actionBtn.SetText(lang.X("ip_region.cancel", "Cancel"))
		showProgress()

		ctx, cancel := context.WithCancel(context.Background())
		scanCancel = cancel

		go func() {
			keys, err := config.ServiceKeys()
			if err != nil {
				fyne.Do(func() {
					showProgressStatus(err.Error())
					resetScanUI(actionBtn, &scanning)
				})
				return
			}

			checker := ipregion.NewChecker()
			report, runErr := checker.Run(ctx, ipregion.Options{
				ServiceKeys: keys,
				OnProgress: func(p ipregion.Progress) {
					fyne.Do(func() {
						updateProgress(p)
					})
				},
			})

			fyne.Do(func() {
				cancelled := runErr != nil && ctx.Err() != nil
				if cancelled {
					showProgressStatus(lang.X("ip_region.cancelled", "Cancelled"))
				} else if runErr != nil {
					showProgressStatus(runErr.Error())
				} else {
					hideProgress()
				}

				if report != nil && !cancelled {
					applyReport(report)
				}
				resetScanUI(actionBtn, &scanning)
				scanCancel = nil
			})
		}()
	}

	actionBtn.OnTapped = func() {
		if scanning && scanCancel != nil {
			scanCancel()
			scanning = false
			hideProgress()
			actionBtn.SetText(lang.X("ip_region.where_am_i", "Where am I?"))
			return
		}
		startScan()
	}

	refreshIPv6Columns()

	summarySection := container.NewVBox(
		summaryTitle,
		summaryEmpty,
		summaryTableHeader,
		summaryRowsBox,
	)

	servicesTitle := widget.NewLabel(lang.X("ip_region.services.title", "Services"))
	servicesTitle.TextStyle = fyne.TextStyle{Bold: true}

	top := container.NewVBox(
		header,
		actionBtn,
		progressArea,
		vpnLabel,
		summarySection,
		widget.NewSeparator(),
		servicesTitle,
		tableHeader,
	)

	return container.NewBorder(top, nil, nil, nil, list)
}

func newIPRegionTableHeader(first, ipv4, ipv6 string) (*widget.Label, *widget.Label, *widget.Label, *fyne.Container) {
	colFirst := widget.NewLabel(first)
	colIPv4 := widget.NewLabel(ipv4)
	colIPv6 := widget.NewLabel(ipv6)
	colFirst.TextStyle = fyne.TextStyle{Bold: true}
	colIPv4.TextStyle = fyne.TextStyle{Bold: true}
	colIPv6.TextStyle = fyne.TextStyle{Bold: true}
	colIPv4.Alignment = fyne.TextAlignCenter
	colIPv6.Alignment = fyne.TextAlignCenter
	return colFirst, colIPv4, colIPv6, container.NewGridWithColumns(3, colFirst, colIPv4, colIPv6)
}

func formatPercentCell(pct int) string {
	if pct <= 0 {
		return lang.X("ip_region.same_as_ipv4", "—")
	}
	return fmt.Sprintf("%d%%", pct)
}

func resetScanUI(btn *widget.Button, scanning *bool) {
	*scanning = false
	btn.SetText(lang.X("ip_region.where_am_i", "Where am I?"))
}

func progressServiceLabel(step string) string {
	switch step {
	case "Detecting IPv4":
		return lang.X("ip_region.progress.ipv4", "Detecting IPv4")
	case "Detecting IPv6":
		return lang.X("ip_region.progress.ipv6", "Detecting IPv6")
	case "Detecting ASN":
		return lang.X("ip_region.progress.asn", "Detecting ASN")
	default:
		return step
	}
}

// formatRegionColumns avoids showing the same country code twice when IPv4 and IPv6 agree.
func formatRegionColumns(ipv4, ipv6 string, showIPv6 bool) (string, string) {
	ipv4 = normalizeRegionCode(ipv4)
	ipv6 = normalizeRegionCode(ipv6)
	if !showIPv6 {
		if ipv4 != ipregion.NotAvailable {
			return ipv4, ""
		}
		return ipv6, ""
	}
	if ipv4 == ipv6 {
		return ipv4, lang.X("ip_region.same_as_ipv4", "—")
	}
	if ipv6 == ipregion.NotAvailable {
		return ipv4, ipregion.NotAvailable
	}
	if ipv4 == ipregion.NotAvailable {
		return ipregion.NotAvailable, ipv6
	}
	return ipv4, ipv6
}

func normalizeRegionCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ipregion.NotAvailable
	}
	return code
}

func formatRegionCell(code string, mismatch bool) string {
	code = normalizeRegionCode(code)
	if code == lang.X("ip_region.same_as_ipv4", "—") {
		return code
	}
	if mismatch && code != ipregion.NotAvailable {
		return "! " + code
	}
	return code
}

func mismatchVPN(u *UI, code string) bool {
	code = normalizeRegionCode(code)
	if code == ipregion.NotAvailable || code == lang.X("ip_region.same_as_ipv4", "—") {
		return false
	}
	loc, ok := u.vpnmgr.ConnectedLocation()
	if !ok || loc.ISO == "" {
		return false
	}
	return !strings.EqualFold(code, loc.ISO)
}

func buildVPNCompareLine(u *UI, report *ipregion.Report) string {
	loc, connected := u.vpnmgr.ConnectedLocation()
	consensus := ipregion.FormatSummaryLine(report)
	if !connected {
		if consensus == "" {
			return ""
		}
		return lang.X("ip_region.consensus_only", "Consensus: {{.Summary}}", map[string]any{"Summary": consensus})
	}
	if consensus == "" {
		return lang.X("ip_region.vpn_only", "VPN location: {{.Country}} ({{.ISO}})", map[string]any{
			"Country": loc.Country,
			"ISO":     loc.ISO,
		})
	}
	return lang.X("ip_region.vpn_vs_consensus", "VPN: {{.Country}} ({{.ISO}}) — consensus: {{.Summary}}", map[string]any{
		"Country": loc.Country,
		"ISO":     loc.ISO,
		"Summary": consensus,
	})
}
