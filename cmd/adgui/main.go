package main

import (
	"adgui/commands"
	"adgui/ui"
)

var (
	version   = "v0.0.0-dev"
	gitCommit = "unknown"
)

func main() {
	appLogic := commands.New()
	appUI := ui.New(appLogic, version)
	_ = gitCommit
	appUI.Run()
}
