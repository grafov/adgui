package ui

import (
	"fmt"
	"strconv"

	"adgui/locations"
	"adgui/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type UI struct {
	Fyne fyne.App
}

func New() *UI {
	myApp := app.NewWithID("AdGuard VPN Client")
	myApp.SetIcon(theme.DisconnectedIcon)
	return &UI{Fyne: myApp}
}

func (u *UI) Run() {
	u.Fyne.Run()
}

func (u *UI) DesktopApp() (desktop.App, bool) {
	desk, ok := u.Fyne.(desktop.App)
	return desk, ok
}

func (u *UI) Dashboard() string {
	// Создаем новое окно для выбора локации
	window := u.Fyne.NewWindow("adgui")
	window.Resize(fyne.NewSize(800, 600))

	//container := widget.Container()

	label := widget.NewTextGridFromString("TODO")
	window.SetContent(label)
	window.Show()
	return "" // TODO
}

func (u *UI) ShowLicense(text string) {
	// Создаем новое окно для выбора локации
	window := u.Fyne.NewWindow("adgui: select location")
	window.Resize(fyne.NewSize(500, 600))

	label := widget.NewTextGridFromString(text)
	window.SetContent(label)
	window.Show()
}

func (u *UI) ShowLocationSelector(locations []locations.Location, connectCity func(string)) {
	// Создаем новое окно для выбора локации
	window := u.Fyne.NewWindow("adgui: select location")
	window.Resize(fyne.NewSize(500, 600))

	table := widget.NewTable(
		// Return number of rows and columns
		func() (int, int) {
			return len(locations), 4
		},
		// Create a template widget for cells
		func() fyne.CanvasObject {
			return widget.NewLabel("...")
		},
		// Create a template widget for cells
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			// Update the content of the cell based on its ID
			label := obj.(*widget.Label)
			// Формируем заголовок
			if id.Row == 0 {
				switch id.Col {
				case 0:
					label.SetText("ISO")
				case 1:
					label.SetText("Country")
				case 2:
					label.SetText("City")
				case 3:
					label.SetText("Ping (ms)")
				}
				label.TextStyle.Bold = true
				return
			}

			loc := locations[id.Row]
			switch id.Col {
			case 0:
				label.SetText(loc.ISO)
			case 1:
				label.SetText(loc.Country)
			case 2:
				label.SetText(loc.City)
			case 3:
				label.SetText(strconv.Itoa(loc.Ping))
			}
			label.TextStyle.Bold = false
		},
	)

	// Set column widths (optional)
	table.SetColumnWidth(0, 30)
	table.SetColumnWidth(1, 300)
	table.SetColumnWidth(2, 200)
	table.SetColumnWidth(3, 30)

	// Обработчик выбора локации
	table.OnSelected = func(id widget.TableCellID) {
		fmt.Printf("Selected: %+v\n", locations[id.Row])
		city := locations[id.Row].City
		go connectCity(city)
		window.Close()
	}

	window.SetContent(container.NewStack(table))
	window.Show()
}
