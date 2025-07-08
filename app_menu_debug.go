//go:build debug

package main

import "fyne.io/fyne/v2"

func (a *Application) buildHelpMenu() *fyne.Menu {
	return fyne.NewMenu("Help",
		fyne.NewMenuItem("About", a.showAbout),
		fyne.NewMenuItem("Debug Info", a.showDebugInfo),
	)
}
