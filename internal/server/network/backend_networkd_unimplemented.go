package network

import "fmt"

func (b *SystemdNetworkdBackend) GetWiFiEnabled() (bool, error) {
	return true, nil
}

func (b *SystemdNetworkdBackend) SetWiFiEnabled(enabled bool) error {
	return fmt.Errorf("WiFi control not supported by networkd backend")
}

func (b *SystemdNetworkdBackend) ScanWiFi() error {
	return fmt.Errorf("WiFi scan not supported by networkd backend")
}

func (b *SystemdNetworkdBackend) GetWiFiNetworkDetails(ssid string) (*NetworkInfoResponse, error) {
	return nil, fmt.Errorf("WiFi details not supported by networkd backend")
}

func (b *SystemdNetworkdBackend) ConnectWiFi(req ConnectionRequest) error {
	return fmt.Errorf("WiFi connect not supported by networkd backend")
}

func (b *SystemdNetworkdBackend) DisconnectWiFi() error {
	return fmt.Errorf("WiFi disconnect not supported by networkd backend")
}

func (b *SystemdNetworkdBackend) ForgetWiFiNetwork(ssid string) error {
	return fmt.Errorf("WiFi forget not supported by networkd backend")
}

func (b *SystemdNetworkdBackend) ListVPNProfiles() ([]VPNProfile, error) {
	return []VPNProfile{}, nil
}

func (b *SystemdNetworkdBackend) ListActiveVPN() ([]VPNActive, error) {
	return []VPNActive{}, nil
}

func (b *SystemdNetworkdBackend) ConnectVPN(uuidOrName string, singleActive bool) error {
	return fmt.Errorf("VPN not supported by networkd backend")
}

func (b *SystemdNetworkdBackend) DisconnectVPN(uuidOrName string) error {
	return fmt.Errorf("VPN not supported by networkd backend")
}

func (b *SystemdNetworkdBackend) DisconnectAllVPN() error {
	return fmt.Errorf("VPN not supported by networkd backend")
}

func (b *SystemdNetworkdBackend) ClearVPNCredentials(uuidOrName string) error {
	return fmt.Errorf("VPN not supported by networkd backend")
}
