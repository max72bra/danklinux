package network

import (
	"fmt"
	"sync"
)

type HybridIwdNetworkdBackend struct {
	wifi          *IWDBackend
	l3            *SystemdNetworkdBackend
	onStateChange func()
	stateMutex    sync.RWMutex
}

func NewHybridIwdNetworkdBackend(w *IWDBackend, n *SystemdNetworkdBackend) (*HybridIwdNetworkdBackend, error) {
	return &HybridIwdNetworkdBackend{
		wifi: w,
		l3:   n,
	}, nil
}

func (b *HybridIwdNetworkdBackend) Initialize() error {
	if err := b.wifi.Initialize(); err != nil {
		return fmt.Errorf("iwd init: %w", err)
	}
	if err := b.l3.Initialize(); err != nil {
		return fmt.Errorf("networkd init: %w", err)
	}
	return nil
}

func (b *HybridIwdNetworkdBackend) Close() {
	b.wifi.Close()
	b.l3.Close()
}

func (b *HybridIwdNetworkdBackend) StartMonitoring(onStateChange func()) error {
	b.onStateChange = onStateChange

	mergedCallback := func() {
		ws, _ := b.wifi.GetCurrentState()
		ls, _ := b.l3.GetCurrentState()

		if ws != nil && ls != nil && ws.WiFiDevice != "" && ls.WiFiIP != "" {
			b.wifi.MarkIPConfigSeen()
		}

		if b.onStateChange != nil {
			b.onStateChange()
		}
	}

	if err := b.wifi.StartMonitoring(mergedCallback); err != nil {
		return fmt.Errorf("wifi monitoring: %w", err)
	}
	if err := b.l3.StartMonitoring(mergedCallback); err != nil {
		return fmt.Errorf("l3 monitoring: %w", err)
	}

	return nil
}

func (b *HybridIwdNetworkdBackend) StopMonitoring() {
	b.wifi.StopMonitoring()
	b.l3.StopMonitoring()
}

func (b *HybridIwdNetworkdBackend) GetCurrentState() (*BackendState, error) {
	ws, err := b.wifi.GetCurrentState()
	if err != nil {
		return nil, err
	}
	ls, err := b.l3.GetCurrentState()
	if err != nil {
		return nil, err
	}

	merged := *ws
	merged.Backend = "iwd+networkd"

	merged.WiFiIP = ls.WiFiIP
	merged.EthernetConnected = ls.EthernetConnected
	merged.EthernetIP = ls.EthernetIP
	merged.EthernetDevice = ls.EthernetDevice
	merged.EthernetConnectionUuid = ls.EthernetConnectionUuid
	merged.WiredConnections = ls.WiredConnections

	if ls.EthernetConnected && ls.EthernetIP != "" {
		merged.NetworkStatus = StatusEthernet
	} else if ws.WiFiConnected && ls.WiFiIP != "" {
		merged.NetworkStatus = StatusWiFi
	} else {
		merged.NetworkStatus = StatusDisconnected
	}

	return &merged, nil
}

func (b *HybridIwdNetworkdBackend) GetWiFiEnabled() (bool, error) {
	return b.wifi.GetWiFiEnabled()
}

func (b *HybridIwdNetworkdBackend) SetWiFiEnabled(enabled bool) error {
	return b.wifi.SetWiFiEnabled(enabled)
}

func (b *HybridIwdNetworkdBackend) ScanWiFi() error {
	return b.wifi.ScanWiFi()
}

func (b *HybridIwdNetworkdBackend) GetWiFiNetworkDetails(ssid string) (*NetworkInfoResponse, error) {
	return b.wifi.GetWiFiNetworkDetails(ssid)
}

func (b *HybridIwdNetworkdBackend) ConnectWiFi(req ConnectionRequest) error {
	if err := b.wifi.ConnectWiFi(req); err != nil {
		return err
	}

	ws, err := b.wifi.GetCurrentState()
	if err == nil && ws.WiFiDevice != "" {
		_ = b.l3.EnsureDhcpUp(ws.WiFiDevice)
	}

	return nil
}

func (b *HybridIwdNetworkdBackend) DisconnectWiFi() error {
	return b.wifi.DisconnectWiFi()
}

func (b *HybridIwdNetworkdBackend) ForgetWiFiNetwork(ssid string) error {
	return b.wifi.ForgetWiFiNetwork(ssid)
}

func (b *HybridIwdNetworkdBackend) GetWiredConnections() ([]WiredConnection, error) {
	return b.l3.GetWiredConnections()
}

func (b *HybridIwdNetworkdBackend) GetWiredNetworkDetails(uuid string) (*WiredNetworkInfoResponse, error) {
	return b.l3.GetWiredNetworkDetails(uuid)
}

func (b *HybridIwdNetworkdBackend) ConnectEthernet() error {
	return b.l3.ConnectEthernet()
}

func (b *HybridIwdNetworkdBackend) DisconnectEthernet() error {
	return b.l3.DisconnectEthernet()
}

func (b *HybridIwdNetworkdBackend) ActivateWiredConnection(uuid string) error {
	return b.l3.ActivateWiredConnection(uuid)
}

func (b *HybridIwdNetworkdBackend) ListVPNProfiles() ([]VPNProfile, error) {
	return []VPNProfile{}, nil
}

func (b *HybridIwdNetworkdBackend) ListActiveVPN() ([]VPNActive, error) {
	return []VPNActive{}, nil
}

func (b *HybridIwdNetworkdBackend) ConnectVPN(uuidOrName string, singleActive bool) error {
	return fmt.Errorf("VPN not supported in hybrid mode")
}

func (b *HybridIwdNetworkdBackend) DisconnectVPN(uuidOrName string) error {
	return fmt.Errorf("VPN not supported in hybrid mode")
}

func (b *HybridIwdNetworkdBackend) DisconnectAllVPN() error {
	return fmt.Errorf("VPN not supported in hybrid mode")
}

func (b *HybridIwdNetworkdBackend) ClearVPNCredentials(uuidOrName string) error {
	return fmt.Errorf("VPN not supported in hybrid mode")
}

func (b *HybridIwdNetworkdBackend) GetPromptBroker() PromptBroker {
	return b.wifi.GetPromptBroker()
}

func (b *HybridIwdNetworkdBackend) SetPromptBroker(broker PromptBroker) error {
	return b.wifi.SetPromptBroker(broker)
}

func (b *HybridIwdNetworkdBackend) SubmitCredentials(token string, secrets map[string]string, save bool) error {
	return b.wifi.SubmitCredentials(token, secrets, save)
}

func (b *HybridIwdNetworkdBackend) CancelCredentials(token string) error {
	return b.wifi.CancelCredentials(token)
}
