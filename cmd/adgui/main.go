package main

import (
	"adgui/commands"
	"adgui/ui"
)

func main() {
	appLogic := commands.New()
	appUI := ui.New(appLogic)
	appUI.Run()
}
