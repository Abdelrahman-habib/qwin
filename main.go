package main

import (
	"embed"
	"log"

	"qwin/internal/app"
	"qwin/internal/infrastructure/logging"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed assets/appicon.png
var icon []byte

// AppEnvironment is set at build time through ldflags
var AppEnvironment string

func main() {

	// Set default environment if not provided by ldflags (e.g., for 'wails dev')
	if AppEnvironment == "" {
		AppEnvironment = "development"
	}
	log.Printf("Application starting in '%s' mode", AppEnvironment)

	// Create an instance of the app structure
	application, err := app.NewApp(AppEnvironment)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Configure log level based on environment
	var logLevel logger.LogLevel
	if AppEnvironment == "production" {
		logLevel = logger.INFO
	} else {
		logLevel = logger.DEBUG
	}

	// Create Wails logger adapter using app's structured logger
	wailsLogger := logging.NewWailsLoggerAdapter(application.GetLogger())

	// Create application with options
	err = wails.Run(&options.App{
		Title:             "Qwin",
		Width:             1200,
		Height:            800,
		MinWidth:          1000,
		MinHeight:         600,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         true,
		StartHidden:       false,
		HideWindowOnClose: false,
		AlwaysOnTop:       false,
		BackgroundColour:  &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Menu:             nil,
		Logger:           wailsLogger,
		LogLevel:         logLevel,
		OnStartup:        application.Startup,
		OnDomReady:       application.DomReady,
		OnBeforeClose:    application.BeforeClose,
		OnShutdown:       application.Shutdown,
		WindowStartState: options.Normal,
		Bind: []interface{}{
			application,
		},
		// Windows platform specific options
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			DisableWindowIcon:    true,
			WebviewUserDataPath:  "",
			ZoomFactor:           1.0,
			BackdropType:         windows.Mica,
		},
		// Mac platform specific options
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Qwin",
				Message: "",
				Icon:    icon,
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
