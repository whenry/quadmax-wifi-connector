package wifi

import (
	"bufio"
	"os/exec"
	"strings"
)

// Adapter represents a wireless network adapter
type Adapter struct {
	Name  string
	State string
}

// Network represents a visible WiFi network
type Network struct {
	SSID string
}

// ConnectionStatus represents the current WiFi connection state
type ConnectionStatus struct {
	Connected    bool
	SSID         string
	AdapterName  string
	SignalStrength string
}

// GetAdapters returns a list of wireless network adapters
func GetAdapters() ([]Adapter, error) {
	cmd := exec.Command("netsh", "wlan", "show", "interfaces")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var adapters []Adapter
	var currentAdapter Adapter

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "Name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				if currentAdapter.Name != "" {
					adapters = append(adapters, currentAdapter)
				}
				currentAdapter = Adapter{Name: strings.TrimSpace(parts[1])}
			}
		} else if strings.HasPrefix(line, "State") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentAdapter.State = strings.TrimSpace(parts[1])
			}
		}
	}

	// Add last adapter
	if currentAdapter.Name != "" {
		adapters = append(adapters, currentAdapter)
	}

	return adapters, nil
}

// ScanNetworks scans for available WiFi networks on a specific adapter
func ScanNetworks(adapterName string) ([]Network, error) {
	var cmd *exec.Cmd
	if adapterName != "" {
		cmd = exec.Command("netsh", "wlan", "show", "networks", "interface="+adapterName)
	} else {
		cmd = exec.Command("netsh", "wlan", "show", "networks")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var networks []Network
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "SSID") && !strings.HasPrefix(line, "SSID ") {
			// Handle "SSID 1 : NetworkName" format
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				ssid := strings.TrimSpace(parts[1])
				if ssid != "" {
					networks = append(networks, Network{SSID: ssid})
				}
			}
		}
	}

	return networks, nil
}

// GetSavedProfiles returns a list of saved WiFi profiles
func GetSavedProfiles() ([]string, error) {
	cmd := exec.Command("netsh", "wlan", "show", "profiles")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var profiles []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "All User Profile") || strings.Contains(line, "Current User Profile") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				profile := strings.TrimSpace(parts[1])
				if profile != "" {
					profiles = append(profiles, profile)
				}
			}
		}
	}

	return profiles, nil
}

// GetConnectionStatus returns the current WiFi connection status for an adapter
func GetConnectionStatus(adapterName string) (*ConnectionStatus, error) {
	cmd := exec.Command("netsh", "wlan", "show", "interfaces")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	status := &ConnectionStatus{}
	var currentAdapterName string
	var inTargetAdapter bool

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "Name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentAdapterName = strings.TrimSpace(parts[1])
				inTargetAdapter = (adapterName == "" || currentAdapterName == adapterName)
			}
		}

		if !inTargetAdapter {
			continue
		}

		if strings.HasPrefix(line, "State") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				state := strings.TrimSpace(parts[1])
				status.Connected = (state == "connected")
				status.AdapterName = currentAdapterName
			}
		} else if strings.HasPrefix(line, "SSID") && !strings.HasPrefix(line, "SSID ") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				status.SSID = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Signal") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				status.SignalStrength = strings.TrimSpace(parts[1])
			}
		}

		// If we found connection info for our target adapter, return
		if inTargetAdapter && status.AdapterName != "" && adapterName != "" {
			break
		}
	}

	return status, nil
}

// Connect connects to a WiFi network using an existing Windows profile
func Connect(adapterName, ssid string) error {
	var cmd *exec.Cmd
	if adapterName != "" {
		cmd = exec.Command("netsh", "wlan", "connect", "name="+ssid, "interface="+adapterName)
	} else {
		cmd = exec.Command("netsh", "wlan", "connect", "name="+ssid)
	}

	return cmd.Run()
}

// IsNetworkAvailable checks if a specific SSID is in range
func IsNetworkAvailable(adapterName, targetSSID string) (bool, error) {
	networks, err := ScanNetworks(adapterName)
	if err != nil {
		return false, err
	}

	for _, network := range networks {
		if network.SSID == targetSSID {
			return true, nil
		}
	}

	return false, nil
}
