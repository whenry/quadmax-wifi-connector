package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/whenry/quadmax-wifi-connector/config"
	"github.com/whenry/quadmax-wifi-connector/wifi"
)

var (
	fyneApp    fyne.App
	mainWindow fyne.Window
)

// Custom theme for a more polished look
type quadmaxTheme struct {
	fyne.Theme
}

func newQuadmaxTheme() *quadmaxTheme {
	return &quadmaxTheme{Theme: theme.DefaultTheme()}
}

func (t *quadmaxTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0x00, G: 0x7A, B: 0xCC, A: 0xFF} // Nice blue
	case theme.ColorNameBackground:
		if variant == theme.VariantDark {
			return color.NRGBA{R: 0x1E, G: 0x1E, B: 0x2E, A: 0xFF}
		}
		return color.NRGBA{R: 0xF8, G: 0xF9, B: 0xFA, A: 0xFF}
	}
	return t.Theme.Color(name, variant)
}

// InitApp initializes the Fyne application (call once at startup)
func InitApp() {
	fyneApp = app.New()
	fyneApp.Settings().SetTheme(newQuadmaxTheme())
}

// createHeader creates a styled header section
func createHeader() fyne.CanvasObject {
	title := canvas.NewText("Quadmax WiFi Connector", color.White)
	title.TextSize = 20
	title.TextStyle = fyne.TextStyle{Bold: true}

	subtitle := canvas.NewText("Automatic WiFi Connection Manager", color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	subtitle.TextSize = 12

	// Header background
	headerBg := canvas.NewRectangle(color.NRGBA{R: 0x00, G: 0x7A, B: 0xCC, A: 0xFF})
	headerBg.SetMinSize(fyne.NewSize(0, 80))

	headerContent := container.NewVBox(
		container.NewCenter(title),
		container.NewCenter(subtitle),
	)

	return container.NewStack(
		headerBg,
		container.NewPadded(headerContent),
	)
}

// createCard creates a card-like container with a title
func createCard(title string, content fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	cardContent := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		content,
	)

	return container.NewPadded(cardContent)
}

// ShowSettings displays the settings window
func ShowSettings(cfg *config.Config, onSave func(*config.Config)) {
	if mainWindow != nil {
		mainWindow.Show()
		mainWindow.RequestFocus()
		return
	}

	mainWindow = fyneApp.NewWindow("Quadmax WiFi Connector")
	mainWindow.Resize(fyne.NewSize(450, 420))
	mainWindow.CenterOnScreen()

	// Get available adapters
	adapters, err := wifi.GetAdapters()
	adapterNames := []string{}
	if err == nil {
		for _, a := range adapters {
			adapterNames = append(adapterNames, a.Name)
		}
	}

	// Get saved WiFi profiles
	profiles, err := wifi.GetSavedProfiles()
	if err != nil {
		profiles = []string{}
	}

	// Adapter section
	adapterSelect := widget.NewSelect(adapterNames, nil)
	adapterSelect.PlaceHolder = "Select a network adapter..."
	if cfg.SelectedAdapter != "" {
		adapterSelect.SetSelected(cfg.SelectedAdapter)
	} else if len(adapterNames) > 0 {
		adapterSelect.SetSelected(adapterNames[0])
	}

	refreshAdaptersBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		adapters, err := wifi.GetAdapters()
		if err == nil {
			names := []string{}
			for _, a := range adapters {
				names = append(names, a.Name)
			}
			adapterSelect.Options = names
			adapterSelect.Refresh()
		}
	})

	adapterRow := container.NewBorder(nil, nil, nil, refreshAdaptersBtn, adapterSelect)
	adapterHelp := widget.NewLabelWithStyle("Choose the wireless adapter to use for connecting", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	adapterHelp.Wrapping = fyne.TextWrapWord
	adapterSection := container.NewVBox(adapterRow, adapterHelp)

	// Network section
	networkSelect := widget.NewSelect(profiles, nil)
	networkSelect.PlaceHolder = "Select a saved network profile..."
	if cfg.SelectedNetwork != "" {
		networkSelect.SetSelected(cfg.SelectedNetwork)
	}

	refreshNetworksBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		profiles, err := wifi.GetSavedProfiles()
		if err == nil {
			networkSelect.Options = profiles
			networkSelect.Refresh()
		}
	})

	networkRow := container.NewBorder(nil, nil, nil, refreshNetworksBtn, networkSelect)
	networkHelp := widget.NewLabelWithStyle("Select your Quadmax launch monitor network", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	networkHelp.Wrapping = fyne.TextWrapWord
	networkSection := container.NewVBox(networkRow, networkHelp)

	// Status indicator
	statusIcon := canvas.NewCircle(color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	statusIcon.Resize(fyne.NewSize(12, 12))
	statusLabel := widget.NewLabel("Checking connection status...")

	statusRow := container.NewHBox(statusIcon, statusLabel)

	// Update status based on current connection
	go func() {
		if cfg.SelectedAdapter != "" {
			status, err := wifi.GetConnectionStatus(cfg.SelectedAdapter)
			if err == nil && status.Connected && status.SSID == cfg.SelectedNetwork {
				statusIcon.FillColor = color.NRGBA{R: 0x00, G: 0xC8, B: 0x00, A: 0xFF}
				statusLabel.SetText("Connected to " + status.SSID)
			} else if err == nil && status.Connected {
				statusIcon.FillColor = color.NRGBA{R: 0xFF, G: 0xC8, B: 0x00, A: 0xFF}
				statusLabel.SetText("Connected to " + status.SSID + " (not target)")
			} else {
				statusIcon.FillColor = color.NRGBA{R: 0xE0, G: 0x00, B: 0x00, A: 0xFF}
				statusLabel.SetText("Not connected to target network")
			}
			statusIcon.Refresh()
		} else {
			statusLabel.SetText("No adapter selected")
		}
	}()

	// Message label for feedback
	messageLabel := widget.NewLabel("")
	messageLabel.Alignment = fyne.TextAlignCenter

	// Action buttons
	saveBtn := widget.NewButtonWithIcon("Save Settings", theme.DocumentSaveIcon(), func() {
		cfg.SelectedAdapter = adapterSelect.Selected
		cfg.SelectedNetwork = networkSelect.Selected

		if err := cfg.Save(); err != nil {
			messageLabel.SetText("Error: " + err.Error())
			return
		}

		messageLabel.SetText("Settings saved successfully!")
		if onSave != nil {
			onSave(cfg)
		}
	})
	saveBtn.Importance = widget.HighImportance

	closeBtn := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
		mainWindow.Hide()
	})

	buttonRow := container.NewHBox(
		layout.NewSpacer(),
		closeBtn,
		saveBtn,
	)

	// Build the main content
	content := container.NewVBox(
		createHeader(),
		container.NewPadded(
			container.NewVBox(
				createCard("Network Adapter", adapterSection),
				createCard("Target Network", networkSection),
				createCard("Status", statusRow),
				widget.NewSeparator(),
				messageLabel,
				buttonRow,
			),
		),
	)

	mainWindow.SetContent(content)
	mainWindow.SetCloseIntercept(func() {
		mainWindow.Hide()
	})

	mainWindow.Show()
}

// UpdateStatus updates the connection status in the settings window if open
func UpdateStatus(connected bool, ssid string) {
	// This could be expanded to update the UI in real-time
}

// RunApp runs the Fyne event loop (blocking)
func RunApp() {
	if fyneApp != nil {
		fyneApp.Run()
	}
}

// QuitApp quits the Fyne application
func QuitApp() {
	if fyneApp != nil {
		fyneApp.Quit()
	}
}
