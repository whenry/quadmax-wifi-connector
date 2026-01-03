package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/go-toast/toast"

	"github.com/whenry/quadmax-wifi-connector/config"
	"github.com/whenry/quadmax-wifi-connector/icons"
	"github.com/whenry/quadmax-wifi-connector/ui"
	"github.com/whenry/quadmax-wifi-connector/wifi"
)

// ConnectionState represents the current connection state
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateSearching
	StateConnected
)

var (
	cfg           *config.Config
	cfgMutex      sync.RWMutex
	currentState  ConnectionState
	stateMutex    sync.RWMutex
	stopPolling   chan struct{}
	mStatusItem   *systray.MenuItem
)

func main() {
	// Initialize UI before systray
	ui.InitApp()

	// Load configuration
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Printf("Warning: Could not load config: %v\n", err)
	}

	// Run systray (this is blocking on Windows)
	systray.Run(onReady, onExit)
}

func onReady() {
	// Set initial icon
	systray.SetIcon(icons.IconDisconnected)
	systray.SetTitle("Quadmax WiFi")
	systray.SetTooltip("Quadmax WiFi Connector - Not Connected")

	// Create menu items
	mStatusItem = systray.AddMenuItem("Status: Initializing...", "Current connection status")
	mStatusItem.Disable()

	systray.AddSeparator()

	mSettings := systray.AddMenuItem("Settings...", "Open settings window")
	mConnect := systray.AddMenuItem("Connect Now", "Attempt to connect immediately")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Exit", "Quit the application")

	// Start the polling goroutine
	stopPolling = make(chan struct{})
	go pollWiFi()

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-mSettings.ClickedCh:
				cfgMutex.RLock()
				currentCfg := *cfg
				cfgMutex.RUnlock()
				ui.ShowSettings(&currentCfg, func(newCfg *config.Config) {
					cfgMutex.Lock()
					cfg = newCfg
					cfgMutex.Unlock()
				})

			case <-mConnect.ClickedCh:
				go attemptConnection()

			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	// Stop the polling goroutine
	close(stopPolling)
	ui.QuitApp()
}

func pollWiFi() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Initial check
	checkAndConnect()

	for {
		select {
		case <-ticker.C:
			checkAndConnect()
		case <-stopPolling:
			return
		}
	}
}

func checkAndConnect() {
	cfgMutex.RLock()
	adapter := cfg.SelectedAdapter
	targetNetwork := cfg.SelectedNetwork
	cfgMutex.RUnlock()

	// If no network is configured, show disconnected state
	if targetNetwork == "" {
		updateState(StateDisconnected, "No network configured")
		return
	}

	// Check current connection status
	status, err := wifi.GetConnectionStatus(adapter)
	if err != nil {
		updateState(StateDisconnected, "Error checking status")
		return
	}

	// Already connected to target network
	if status.Connected && status.SSID == targetNetwork {
		updateState(StateConnected, fmt.Sprintf("Connected to %s", targetNetwork))
		return
	}

	// Check if target network is available
	available, err := wifi.IsNetworkAvailable(adapter, targetNetwork)
	if err != nil {
		updateState(StateDisconnected, "Error scanning networks")
		return
	}

	if !available {
		updateState(StateDisconnected, fmt.Sprintf("%s not in range", targetNetwork))
		return
	}

	// Network is available but not connected - attempt to connect
	updateState(StateSearching, fmt.Sprintf("Connecting to %s...", targetNetwork))

	err = wifi.Connect(adapter, targetNetwork)
	if err != nil {
		updateState(StateDisconnected, "Connection failed")
		showNotification("Connection Failed", fmt.Sprintf("Could not connect to %s", targetNetwork))
		return
	}

	// Wait a moment and verify connection
	time.Sleep(2 * time.Second)

	status, err = wifi.GetConnectionStatus(adapter)
	if err == nil && status.Connected && status.SSID == targetNetwork {
		updateState(StateConnected, fmt.Sprintf("Connected to %s", targetNetwork))
		showNotification("Connected", fmt.Sprintf("Successfully connected to %s", targetNetwork))
	} else {
		updateState(StateDisconnected, "Connection verification failed")
	}
}

func attemptConnection() {
	cfgMutex.RLock()
	adapter := cfg.SelectedAdapter
	targetNetwork := cfg.SelectedNetwork
	cfgMutex.RUnlock()

	if targetNetwork == "" {
		showNotification("Error", "No target network configured. Open Settings to configure.")
		return
	}

	updateState(StateSearching, fmt.Sprintf("Connecting to %s...", targetNetwork))

	err := wifi.Connect(adapter, targetNetwork)
	if err != nil {
		updateState(StateDisconnected, "Connection failed")
		showNotification("Connection Failed", fmt.Sprintf("Could not connect to %s", targetNetwork))
		return
	}

	// Wait and verify
	time.Sleep(2 * time.Second)

	status, err := wifi.GetConnectionStatus(adapter)
	if err == nil && status.Connected && status.SSID == targetNetwork {
		updateState(StateConnected, fmt.Sprintf("Connected to %s", targetNetwork))
		showNotification("Connected", fmt.Sprintf("Successfully connected to %s", targetNetwork))
	} else {
		updateState(StateDisconnected, "Connection verification failed")
	}
}

func updateState(state ConnectionState, statusText string) {
	stateMutex.Lock()
	previousState := currentState
	currentState = state
	stateMutex.Unlock()

	// Update icon based on state
	switch state {
	case StateConnected:
		systray.SetIcon(icons.IconConnected)
		systray.SetTooltip("Quadmax WiFi - Connected")
	case StateSearching:
		systray.SetIcon(icons.IconSearching)
		systray.SetTooltip("Quadmax WiFi - Connecting...")
	case StateDisconnected:
		systray.SetIcon(icons.IconDisconnected)
		systray.SetTooltip("Quadmax WiFi - Disconnected")
	}

	// Update status menu item
	if mStatusItem != nil {
		mStatusItem.SetTitle("Status: " + statusText)
	}

	// Show notification on state change (connected -> disconnected)
	if previousState == StateConnected && state == StateDisconnected {
		cfgMutex.RLock()
		targetNetwork := cfg.SelectedNetwork
		cfgMutex.RUnlock()
		if targetNetwork != "" {
			showNotification("Disconnected", fmt.Sprintf("Lost connection to %s", targetNetwork))
		}
	}
}

func showNotification(title, message string) {
	notification := toast.Notification{
		AppID:   "Quadmax WiFi Connector",
		Title:   title,
		Message: message,
		Audio:   toast.Default,
	}

	// Best effort - don't block on notification errors
	go func() {
		_ = notification.Push()
	}()
}
