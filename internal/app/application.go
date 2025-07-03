package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/gui/widgets"
	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/pipeline"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/rs/zerolog"
)

const (
	AppName    = "Otsu Obliterator"
	AppID      = "com.imageprocessing.otsu-obliterator"
	AppVersion = "1.0.0"
)

type shutdownHandler interface {
	Shutdown()
}

type Application struct {
	fyneApp       fyne.App
	window        fyne.Window
	guiManager    *gui.Manager
	coordinator   pipeline.ProcessingCoordinator
	memoryManager *memory.Manager
	logger        logger.Logger
	shutdownables []shutdownHandler
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	shutdown      chan struct{}
	menuSetup     bool
}

func NewApplication() (*Application, error) {
	app.SetMetadata(fyne.AppMetadata{
		ID:      AppID,
		Name:    AppName,
		Version: AppVersion,
		Build:   1,
	})

	fyneApp := app.NewWithID(AppID)
	window := fyneApp.NewWindow(AppName)

	windowSize := calculateMinimumWindowSize()
	window.Resize(windowSize)
	window.SetFixedSize(false)
	window.SetPadded(false)
	window.CenterOnScreen()
	window.SetMaster()

	ctx, cancel := context.WithCancel(context.Background())
	logLevel := getLogLevel()
	log := logger.NewConsoleLogger(logLevel)

	log.Info("Application", "starting application", map[string]interface{}{
		"version":       AppVersion,
		"window_width":  windowSize.Width,
		"window_height": windowSize.Height,
		"log_level":     logLevel.String(),
	})

	memoryManager := memory.NewManager(log)
	coordinator := pipeline.NewCoordinator(memoryManager, log)

	guiManager, err := gui.NewManager(window, log)
	if err != nil {
		cancel()
		return nil, err
	}

	guiManager.SetProcessingCoordinator(coordinator)

	log.Info("Application", "creating application struct", nil)
	application := &Application{
		fyneApp:       fyneApp,
		window:        window,
		guiManager:    guiManager,
		coordinator:   coordinator,
		memoryManager: memoryManager,
		logger:        log,
		ctx:           ctx,
		cancel:        cancel,
		shutdown:      make(chan struct{}),
		shutdownables: []shutdownHandler{
			memoryManager,
			coordinator,
			guiManager,
		},
	}

	application.setupSignalHandling()
	log.Info("Application", "initialization complete", nil)
	return application, nil
}

func (a *Application) setupMenu() {
	aboutAction := func() {
		a.logger.Info("About", "menu action triggered", nil)
		fyne.Do(func() {
			a.showAbout()
		})
	}

	fileMenu := fyne.NewMenu("File")
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", aboutAction),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, helpMenu)
	a.window.SetMainMenu(mainMenu)

	a.logger.Info("Application", "menu setup completed", map[string]interface{}{
		"menus": []string{"File", "Help"},
	})
}

func (a *Application) showAbout() {
	metadata := a.fyneApp.Metadata()

	a.logger.Info("About", "metadata debug", map[string]interface{}{
		"name":    metadata.Name,
		"version": metadata.Version,
		"build":   metadata.Build,
		"id":      metadata.ID,
	})

	name := metadata.Name
	if name == "" {
		name = AppName
		a.logger.Info("About", "using fallback name", map[string]interface{}{"name": name})
	}

	version := metadata.Version
	if version == "" {
		version = AppVersion
		a.logger.Info("About", "using fallback version", map[string]interface{}{"version": version})
	}

	build := fmt.Sprintf("%d", metadata.Build)
	if metadata.Build == 0 {
		build = "1"
		a.logger.Info("About", "using fallback build", map[string]interface{}{"build": build})
	}

	aboutContent := container.NewVBox(
		widget.NewLabel(name),
		widget.NewLabel(fmt.Sprintf("Version: %s", version)),
		widget.NewLabel(fmt.Sprintf("Build: %s", build)),
		widget.NewLabel(""),
		widget.NewLabel("Author: Ervins Strauhmanis"),
		widget.NewLabel("License: MIT"),
		widget.NewLabel("Year: 2025"),
		widget.NewLabel(""),
		widget.NewLabel("Runtime Info:"),
		widget.NewLabel(fmt.Sprintf("Go: %s", runtime.Version())),
		widget.NewLabel(fmt.Sprintf("Platform: %s/%s", runtime.GOOS, runtime.GOARCH)),
		widget.NewLabel("OpenCV: 4.11.0+"),
	)

	dialog.ShowCustom("About", "Close", aboutContent, a.window)
}

func calculateMinimumWindowSize() fyne.Size {
	imageDisplayWidth := widgets.ImageAreaWidth * 2
	toolbarHeight := float32(50)
	parametersHeight := float32(150)

	minimumWidth := float32(imageDisplayWidth + 100)
	minimumHeight := float32(widgets.ImageAreaHeight + toolbarHeight + parametersHeight + 100)

	return fyne.Size{
		Width:  minimumWidth,
		Height: minimumHeight,
	}
}

func (a *Application) setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		select {
		case sig := <-sigChan:
			a.logger.Info("Application", "shutdown signal received", map[string]interface{}{
				"signal": sig.String(),
			})
			a.initiateShutdown()
		case <-a.ctx.Done():
			return
		}
	}()
}

func (a *Application) Run() error {
	if !a.menuSetup {
		a.logger.Info("Application", "setting up menu in Run", nil)
		a.setupMenu()
		a.menuSetup = true
	}

	a.window.SetCloseIntercept(func() {
		a.logger.Info("Application", "shutdown requested via window close", nil)
		a.initiateShutdown()
		a.window.Close()
	})

	fyne.Do(func() {
		a.guiManager.Show()
		a.logger.Info("Application", "GUI displayed", nil)
	})

	go func() {
		<-a.shutdown
		fyne.Do(func() {
			a.fyneApp.Quit()
		})
	}()

	a.fyneApp.Run()
	a.wg.Wait()
	return nil
}

func (a *Application) ForceMenuSetup() {
	a.logger.Info("Application", "ForceMenuSetup called from main", nil)
	a.setupMenu()
	a.logger.Info("Application", "ForceMenuSetup completed", nil)
}

func (a *Application) initiateShutdown() {
	select {
	case <-a.shutdown:
		return
	default:
		close(a.shutdown)
	}

	a.logger.Info("Application", "shutdown sequence initiated", map[string]interface{}{
		"components": len(a.shutdownables),
	})

	a.cancel()

	for i := len(a.shutdownables) - 1; i >= 0; i-- {
		component := a.shutdownables[i]

		done := make(chan struct{})
		go func() {
			defer close(done)
			component.Shutdown()
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			a.logger.Warning("Application", "component shutdown timeout", map[string]interface{}{
				"component_index": i,
			})
		}
	}

	a.logger.Info("Application", "shutdown sequence completed", nil)
}

func (a *Application) Shutdown(ctx context.Context) error {
	a.initiateShutdown()

	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func getLogLevel() zerolog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		if os.Getenv("OTSU_DEBUG_ALL") == "true" {
			return zerolog.DebugLevel
		}
		return zerolog.InfoLevel
	}
}
