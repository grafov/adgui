package main

import (
	"adgui/commands"
	"adgui/ui"
)

func main() {
	appUI := ui.New()
	commands.New(appUI)
	appUI.Run()
}
