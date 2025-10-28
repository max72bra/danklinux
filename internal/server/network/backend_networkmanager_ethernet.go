package network

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/AvengeMedia/danklinux/internal/log"
	"github.com/Wifx/gonetworkmanager/v2"
)

func (b *NetworkManagerBackend) GetWiredConnections() ([]WiredConnection, error) {
	return b.listEthernetConnections()
}

func (b *NetworkManagerBackend) GetWiredNetworkDetails(uuid string) (*WiredNetworkInfoResponse, error) {
	if b.ethernetDevice == nil {
		return nil, fmt.Errorf("no ethernet device available")
	}

	dev := b.ethernetDevice.(gonetworkmanager.Device)

	iface, _ := dev.GetPropertyInterface()
	driver, _ := dev.GetPropertyDriver()

	hwAddr := "Not available"
	var speed uint32 = 0
	wiredDevice, err := gonetworkmanager.NewDeviceWired(dev.GetPath())
	if err == nil {
		hwAddr, _ = wiredDevice.GetPropertyHwAddress()
		speed, _ = wiredDevice.GetPropertySpeed()
	}
	var ipv4Config WiredIPConfig
	var ipv6Config WiredIPConfig

	activeConn, err := dev.GetPropertyActiveConnection()
	if err == nil && activeConn != nil {
		ip4Config, err := activeConn.GetPropertyIP4Config()
		if err == nil && ip4Config != nil {
			var ips []string
			addresses, err := ip4Config.GetPropertyAddressData()
			if err == nil && len(addresses) > 0 {
				for _, addr := range addresses {
					ips = append(ips, fmt.Sprintf("%s/%s", addr.Address, strconv.Itoa(int(addr.Prefix))))
				}
			}

			gateway, _ := ip4Config.GetPropertyGateway()
			dnsAddrs := ""
			dns, err := ip4Config.GetPropertyNameserverData()
			if err == nil && len(dns) > 0 {
				for _, d := range dns {
					if len(dnsAddrs) > 0 {
						dnsAddrs = strings.Join([]string{dnsAddrs, d.Address}, "; ")
					} else {
						dnsAddrs = d.Address
					}
				}
			}

			ipv4Config = WiredIPConfig{
				IPs:     ips,
				Gateway: gateway,
				DNS:     dnsAddrs,
			}
		}

		ip6Config, err := activeConn.GetPropertyIP6Config()
		if err == nil && ip6Config != nil {
			var ips []string
			addresses, err := ip6Config.GetPropertyAddressData()
			if err == nil && len(addresses) > 0 {
				for _, addr := range addresses {
					ips = append(ips, fmt.Sprintf("%s/%s", addr.Address, strconv.Itoa(int(addr.Prefix))))
				}
			}

			gateway, _ := ip6Config.GetPropertyGateway()
			dnsAddrs := ""
			dns, err := ip6Config.GetPropertyNameservers()
			if err == nil && len(dns) > 0 {
				for _, d := range dns {
					if len(d) == 16 {
						ip := net.IP(d)
						if len(dnsAddrs) > 0 {
							dnsAddrs = strings.Join([]string{dnsAddrs, ip.String()}, "; ")
						} else {
							dnsAddrs = ip.String()
						}
					}
				}
			}

			ipv6Config = WiredIPConfig{
				IPs:     ips,
				Gateway: gateway,
				DNS:     dnsAddrs,
			}
		}
	}

	return &WiredNetworkInfoResponse{
		UUID:   uuid,
		IFace:  iface,
		Driver: driver,
		HwAddr: hwAddr,
		Speed:  strconv.Itoa(int(speed)),
		IPv4:   ipv4Config,
		IPv6:   ipv6Config,
	}, nil
}

func (b *NetworkManagerBackend) ConnectEthernet() error {
	if b.ethernetDevice == nil {
		return fmt.Errorf("no ethernet device available")
	}

	nm := b.nmConn.(gonetworkmanager.NetworkManager)
	dev := b.ethernetDevice.(gonetworkmanager.Device)

	settingsMgr, err := gonetworkmanager.NewSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	connections, err := settingsMgr.ListConnections()
	if err != nil {
		return fmt.Errorf("failed to get connections: %w", err)
	}

	for _, conn := range connections {
		connSettings, err := conn.GetSettings()
		if err != nil {
			continue
		}

		if connMeta, ok := connSettings["connection"]; ok {
			if connType, ok := connMeta["type"].(string); ok && connType == "802-3-ethernet" {
				_, err := nm.ActivateConnection(conn, dev, nil)
				if err != nil {
					return fmt.Errorf("failed to activate ethernet: %w", err)
				}

				b.updateEthernetState()
				b.listEthernetConnections()
				b.updatePrimaryConnection()

				if b.onStateChange != nil {
					b.onStateChange()
				}

				return nil
			}
		}
	}

	settings := make(map[string]map[string]interface{})
	settings["connection"] = map[string]interface{}{
		"id":   "Wired connection",
		"type": "802-3-ethernet",
	}

	_, err = nm.AddAndActivateConnection(settings, dev)
	if err != nil {
		return fmt.Errorf("failed to create and activate ethernet: %w", err)
	}

	b.updateEthernetState()
	b.listEthernetConnections()
	b.updatePrimaryConnection()

	if b.onStateChange != nil {
		b.onStateChange()
	}

	return nil
}

func (b *NetworkManagerBackend) DisconnectEthernet() error {
	if b.ethernetDevice == nil {
		return fmt.Errorf("no ethernet device available")
	}

	dev := b.ethernetDevice.(gonetworkmanager.Device)

	err := dev.Disconnect()
	if err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	b.updateEthernetState()
	b.listEthernetConnections()
	b.updatePrimaryConnection()

	if b.onStateChange != nil {
		b.onStateChange()
	}

	return nil
}

func (b *NetworkManagerBackend) ActivateWiredConnection(uuid string) error {
	if b.ethernetDevice == nil {
		return fmt.Errorf("no ethernet device available")
	}

	nm := b.nmConn.(gonetworkmanager.NetworkManager)
	dev := b.ethernetDevice.(gonetworkmanager.Device)

	settingsMgr, err := gonetworkmanager.NewSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	connections, err := settingsMgr.ListConnections()
	if err != nil {
		return fmt.Errorf("failed to get connections: %w", err)
	}

	var targetConnection gonetworkmanager.Connection
	for _, conn := range connections {
		settings, err := conn.GetSettings()
		if err != nil {
			continue
		}

		if connectionSettings, ok := settings["connection"]; ok {
			if connUUID, ok := connectionSettings["uuid"].(string); ok && connUUID == uuid {
				targetConnection = conn
				break
			}
		}
	}

	if targetConnection == nil {
		return fmt.Errorf("connection with UUID %s not found", uuid)
	}

	_, err = nm.ActivateConnection(targetConnection, dev, nil)
	if err != nil {
		return fmt.Errorf("error activation connection: %w", err)
	}

	b.updateEthernetState()
	b.listEthernetConnections()
	b.updatePrimaryConnection()

	if b.onStateChange != nil {
		b.onStateChange()
	}

	return nil
}

func (b *NetworkManagerBackend) listEthernetConnections() ([]WiredConnection, error) {
	if b.ethernetDevice == nil {
		return nil, fmt.Errorf("no ethernet device available")
	}

	s := b.settings
	if s == nil {
		s, err := gonetworkmanager.NewSettings()
		if err != nil {
			return nil, fmt.Errorf("failed to get settings: %w", err)
		}
		b.settings = s
	}

	settingsMgr := s.(gonetworkmanager.Settings)
	connections, err := settingsMgr.ListConnections()
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}

	wiredConfigs := make([]WiredConnection, 0)
	activeUUIDs, err := b.getActiveConnections()

	if err != nil {
		return nil, fmt.Errorf("failed to get active wired connections: %w", err)
	}

	currentUuid := ""
	for _, connection := range connections {
		path := connection.GetPath()
		settings, err := connection.GetSettings()
		if err != nil {
			log.Errorf("unable to get settings for %s: %v", path, err)
			continue
		}

		connectionSettings := settings["connection"]
		connType, _ := connectionSettings["type"].(string)
		connID, _ := connectionSettings["id"].(string)
		connUUID, _ := connectionSettings["uuid"].(string)

		if connType == "802-3-ethernet" {
			wiredConfigs = append(wiredConfigs, WiredConnection{
				Path:     path,
				ID:       connID,
				UUID:     connUUID,
				Type:     connType,
				IsActive: activeUUIDs[connUUID],
			})
			if activeUUIDs[connUUID] {
				currentUuid = connUUID
			}
		}
	}

	b.stateMutex.Lock()
	b.state.EthernetConnectionUuid = currentUuid
	b.state.WiredConnections = wiredConfigs
	b.stateMutex.Unlock()

	return wiredConfigs, nil
}
