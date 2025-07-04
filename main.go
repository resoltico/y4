package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

const (
	AppName    = "Otsu Obliterator"
	AppID      = "com.imageprocessing.otsu-obliterator"
	AppVersion = "1.0.0"
)

func main() {
	app.SetMetadata(fyne.AppMetadata{
		ID:      AppID,
		Name:    AppName,
		Version: AppVersion,
		Build:   1,
	})

	fyneApp := app.NewWithID(AppID)
	window := fyneApp.NewWindow(AppName)

	ctx, cancel := context.WithCancel(context.Background())

	application := NewApplication(fyneApp, window, ctx, cancel)

	setupSignalHandling(cancel)

	application.ShowAndRun()
}

func setupSignalHandling(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		log.Println("Signal received, shutting down...")
		cancel()
	}()
}
