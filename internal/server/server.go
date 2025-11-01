package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/AvengeMedia/danklinux/internal/log"
	"github.com/AvengeMedia/danklinux/internal/server/bluez"
	"github.com/AvengeMedia/danklinux/internal/server/cups"
	"github.com/AvengeMedia/danklinux/internal/server/dwl"
	"github.com/AvengeMedia/danklinux/internal/server/freedesktop"
	"github.com/AvengeMedia/danklinux/internal/server/loginctl"
	"github.com/AvengeMedia/danklinux/internal/server/models"
	"github.com/AvengeMedia/danklinux/internal/server/network"
	"github.com/AvengeMedia/danklinux/internal/server/wayland"
	"github.com/AvengeMedia/danklinux/internal/server/wlcontext"
)

const APIVersion = 12

type Capabilities struct {
	Capabilities []string `json:"capabilities"`
}

type ServerInfo struct {
	APIVersion   int      `json:"apiVersion"`
	Capabilities []string `json:"capabilities"`
}

type ServiceEvent struct {
	Service string      `json:"service"`
	Data    interface{} `json:"data"`
}

var networkManager *network.Manager
var loginctlManager *loginctl.Manager
var freedesktopManager *freedesktop.Manager
var waylandManager *wayland.Manager
var bluezManager *bluez.Manager
var cupsManager *cups.Manager
var dwlManager *dwl.Manager
var wlContext *wlcontext.SharedContext

func getSocketDir() string {
	if runtime := os.Getenv("XDG_RUNTIME_DIR"); runtime != "" {
		return runtime
	}

	if os.Getuid() == 0 {
		if _, err := os.Stat("/run"); err == nil {
			return "/run/dankdots"
		}
		return "/var/run/dankdots"
	}

	return os.TempDir()
}

func GetSocketPath() string {
	return filepath.Join(getSocketDir(), fmt.Sprintf("danklinux-%d.sock", os.Getpid()))
}

func cleanupStaleSockets() {
	dir := getSocketDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "danklinux-") || !strings.HasSuffix(entry.Name(), ".sock") {
			continue
		}

		pidStr := strings.TrimPrefix(entry.Name(), "danklinux-")
		pidStr = strings.TrimSuffix(pidStr, ".sock")
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			socketPath := filepath.Join(dir, entry.Name())
			os.Remove(socketPath)
			log.Debugf("Removed stale socket: %s", socketPath)
			continue
		}

		err = process.Signal(syscall.Signal(0))
		if err != nil {
			socketPath := filepath.Join(dir, entry.Name())
			os.Remove(socketPath)
			log.Debugf("Removed stale socket: %s", socketPath)
		}
	}
}

func InitializeNetworkManager() error {
	manager, err := network.NewManager()
	if err != nil {
		log.Warnf("Failed to initialize network manager: %v", err)
		return err
	}

	networkManager = manager

	log.Info("Network manager initialized")
	return nil
}

func InitializeLoginctlManager() error {
	manager, err := loginctl.NewManager()
	if err != nil {
		log.Warnf("Failed to initialize loginctl manager: %v", err)
		return err
	}

	loginctlManager = manager

	log.Info("Loginctl manager initialized")
	return nil
}

func InitializeFreedeskManager() error {
	manager, err := freedesktop.NewManager()
	if err != nil {
		log.Warnf("Failed to initialize freedesktop manager: %v", err)
		return err
	}

	freedesktopManager = manager

	log.Info("Freedesktop manager initialized")
	return nil
}

func InitializeWaylandManager() error {
	log.Info("Attempting to initialize Wayland gamma control...")

	if wlContext == nil {
		ctx, err := wlcontext.New()
		if err != nil {
			log.Errorf("Failed to create shared Wayland context: %v", err)
			return err
		}
		wlContext = ctx
	}

	config := wayland.DefaultConfig()
	manager, err := wayland.NewManager(wlContext.Display(), config)
	if err != nil {
		log.Errorf("Failed to initialize wayland manager: %v", err)
		return err
	}

	waylandManager = manager

	log.Info("Wayland gamma control initialized successfully")
	return nil
}

func InitializeBluezManager() error {
	manager, err := bluez.NewManager()
	if err != nil {
		log.Warnf("Failed to initialize bluez manager: %v", err)
		return err
	}

	bluezManager = manager

	log.Info("Bluez manager initialized")
	return nil
}

func InitializeCupsManager() error {
	manager, err := cups.NewManager()
	if err != nil {
		log.Warnf("Failed to initialize cups manager: %v", err)
		return err
	}

	cupsManager = manager

	log.Info("CUPS manager initialized")
	return nil
}

func InitializeDwlManager() error {
	log.Info("Attempting to initialize DWL IPC...")

	if wlContext == nil {
		ctx, err := wlcontext.New()
		if err != nil {
			log.Errorf("Failed to create shared Wayland context: %v", err)
			return err
		}
		wlContext = ctx
	}

	manager, err := dwl.NewManager(wlContext.Display())
	if err != nil {
		log.Debug("Failed to initialize dwl manager: %v", err)
		return err
	}

	dwlManager = manager

	log.Info("DWL IPC initialized successfully")
	return nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	caps := getCapabilities()
	capsData, _ := json.Marshal(caps)
	conn.Write(capsData)
	conn.Write([]byte("\n"))

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Bytes()

		var req models.Request
		if err := json.Unmarshal(line, &req); err != nil {
			log.Warnf("handleConnection: Failed to unmarshal JSON: %v, line: %s", err, string(line))
			models.RespondError(conn, 0, "invalid json")
			continue
		}

		go RouteRequest(conn, req)
	}
}

func getCapabilities() Capabilities {
	caps := []string{"plugins"}

	if networkManager != nil {
		caps = append(caps, "network")
	}

	if loginctlManager != nil {
		caps = append(caps, "loginctl")
	}

	if freedesktopManager != nil {
		caps = append(caps, "freedesktop")
	}

	if waylandManager != nil {
		caps = append(caps, "gamma")
	}

	if bluezManager != nil {
		caps = append(caps, "bluetooth")
	}

	if cupsManager != nil {
		caps = append(caps, "cups")
	}

	if dwlManager != nil {
		caps = append(caps, "dwl")
	}

	return Capabilities{Capabilities: caps}
}

func getServerInfo() ServerInfo {
	caps := []string{"plugins"}

	if networkManager != nil {
		caps = append(caps, "network")
	}

	if loginctlManager != nil {
		caps = append(caps, "loginctl")
	}

	if freedesktopManager != nil {
		caps = append(caps, "freedesktop")
	}

	if waylandManager != nil {
		caps = append(caps, "gamma")
	}

	if bluezManager != nil {
		caps = append(caps, "bluetooth")
	}

	if dwlManager != nil {
		caps = append(caps, "dwl")
	}

	return ServerInfo{
		APIVersion:   APIVersion,
		Capabilities: caps,
	}
}

func handleSubscribe(conn net.Conn, req models.Request) {
	clientID := fmt.Sprintf("meta-client-%p", conn)

	var services []string
	if servicesParam, ok := req.Params["services"].([]interface{}); ok {
		for _, s := range servicesParam {
			if str, ok := s.(string); ok {
				services = append(services, str)
			}
		}
	}

	if len(services) == 0 {
		services = []string{"all"}
	}

	subscribeAll := false
	for _, s := range services {
		if s == "all" {
			subscribeAll = true
			break
		}
	}

	var wg sync.WaitGroup
	eventChan := make(chan ServiceEvent, 256)
	stopChan := make(chan struct{})

	shouldSubscribe := func(service string) bool {
		if subscribeAll {
			return true
		}
		for _, s := range services {
			if s == service {
				return true
			}
		}
		return false
	}

	if shouldSubscribe("network") && networkManager != nil {
		wg.Add(1)
		netChan := networkManager.Subscribe(clientID + "-network")
		go func() {
			defer wg.Done()
			defer networkManager.Unsubscribe(clientID + "-network")

			initialState := networkManager.GetState()
			select {
			case eventChan <- ServiceEvent{Service: "network", Data: initialState}:
			case <-stopChan:
				return
			}

			for {
				select {
				case state, ok := <-netChan:
					if !ok {
						return
					}
					select {
					case eventChan <- ServiceEvent{Service: "network", Data: state}:
					case <-stopChan:
						return
					}
				case <-stopChan:
					return
				}
			}
		}()
	}

	if shouldSubscribe("network.credentials") && networkManager != nil {
		wg.Add(1)
		credChan := networkManager.SubscribeCredentials(clientID + "-credentials")
		go func() {
			defer wg.Done()
			defer networkManager.UnsubscribeCredentials(clientID + "-credentials")

			for {
				select {
				case prompt, ok := <-credChan:
					if !ok {
						return
					}
					select {
					case eventChan <- ServiceEvent{Service: "network.credentials", Data: prompt}:
					case <-stopChan:
						return
					}
				case <-stopChan:
					return
				}
			}
		}()
	}

	if shouldSubscribe("loginctl") && loginctlManager != nil {
		wg.Add(1)
		loginChan := loginctlManager.Subscribe(clientID + "-loginctl")
		go func() {
			defer wg.Done()
			defer loginctlManager.Unsubscribe(clientID + "-loginctl")

			initialState := loginctlManager.GetState()
			select {
			case eventChan <- ServiceEvent{Service: "loginctl", Data: initialState}:
			case <-stopChan:
				return
			}

			for {
				select {
				case state, ok := <-loginChan:
					if !ok {
						return
					}
					select {
					case eventChan <- ServiceEvent{Service: "loginctl", Data: state}:
					case <-stopChan:
						return
					}
				case <-stopChan:
					return
				}
			}
		}()
	}

	if shouldSubscribe("gamma") && waylandManager != nil {
		wg.Add(1)
		waylandChan := waylandManager.Subscribe(clientID + "-gamma")
		go func() {
			defer wg.Done()
			defer waylandManager.Unsubscribe(clientID + "-gamma")

			initialState := waylandManager.GetState()
			select {
			case eventChan <- ServiceEvent{Service: "gamma", Data: initialState}:
			case <-stopChan:
				return
			}

			for {
				select {
				case state, ok := <-waylandChan:
					if !ok {
						return
					}
					select {
					case eventChan <- ServiceEvent{Service: "gamma", Data: state}:
					case <-stopChan:
						return
					}
				case <-stopChan:
					return
				}
			}
		}()
	}

	if shouldSubscribe("bluetooth") && bluezManager != nil {
		wg.Add(1)
		bluezChan := bluezManager.Subscribe(clientID + "-bluetooth")
		go func() {
			defer wg.Done()
			defer bluezManager.Unsubscribe(clientID + "-bluetooth")

			initialState := bluezManager.GetState()
			select {
			case eventChan <- ServiceEvent{Service: "bluetooth", Data: initialState}:
			case <-stopChan:
				return
			}

			for {
				select {
				case state, ok := <-bluezChan:
					if !ok {
						return
					}
					select {
					case eventChan <- ServiceEvent{Service: "bluetooth", Data: state}:
					case <-stopChan:
						return
					}
				case <-stopChan:
					return
				}
			}
		}()
	}

	if shouldSubscribe("bluetooth.pairing") && bluezManager != nil {
		wg.Add(1)
		pairingChan := bluezManager.SubscribePairing(clientID + "-pairing")
		go func() {
			defer wg.Done()
			defer bluezManager.UnsubscribePairing(clientID + "-pairing")

			for {
				select {
				case prompt, ok := <-pairingChan:
					if !ok {
						return
					}
					select {
					case eventChan <- ServiceEvent{Service: "bluetooth.pairing", Data: prompt}:
					case <-stopChan:
						return
					}
				case <-stopChan:
					return
				}
			}
		}()
	}

	if shouldSubscribe("cups") && cupsManager != nil {
		wg.Add(1)
		cupsChan := cupsManager.Subscribe(clientID + "-cups")
		go func() {
			defer wg.Done()
			defer cupsManager.Unsubscribe(clientID + "-cups")

			initialState := cupsManager.GetState()
			select {
			case eventChan <- ServiceEvent{Service: "cups", Data: initialState}:
			case <-stopChan:
				return
			}

			for {
				select {
				case state, ok := <-cupsChan:
					if !ok {
						return
					}
					select {
					case eventChan <- ServiceEvent{Service: "cups", Data: state}:
					case <-stopChan:
						return
					}
				case <-stopChan:
					return
				}
			}
		}()
	}
	
	if shouldSubscribe("dwl") && dwlManager != nil {
		wg.Add(1)
		dwlChan := dwlManager.Subscribe(clientID + "-dwl")
		go func() {
			defer wg.Done()
			defer dwlManager.Unsubscribe(clientID + "-dwl")

			initialState := dwlManager.GetState()
			select {
			case eventChan <- ServiceEvent{Service: "dwl", Data: initialState}:
			case <-stopChan:
				return
			}

			for {
				select {
				case state, ok := <-dwlChan:
					if !ok {
						return
					}
					select {
					case eventChan <- ServiceEvent{Service: "dwl", Data: state}:
					case <-stopChan:
						return
					}
				case <-stopChan:
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(eventChan)
	}()

	info := getServerInfo()
	if err := json.NewEncoder(conn).Encode(models.Response[ServiceEvent]{
		ID:     req.ID,
		Result: &ServiceEvent{Service: "server", Data: info},
	}); err != nil {
		close(stopChan)
		return
	}

	for event := range eventChan {
		if err := json.NewEncoder(conn).Encode(models.Response[ServiceEvent]{
			ID:     req.ID,
			Result: &event,
		}); err != nil {
			close(stopChan)
			return
		}
	}
}

func cleanupManagers() {
	if networkManager != nil {
		networkManager.Close()
	}
	if loginctlManager != nil {
		loginctlManager.Close()
	}
	if freedesktopManager != nil {
		freedesktopManager.Close()
	}
	if waylandManager != nil {
		waylandManager.Close()
	}
	if bluezManager != nil {
		bluezManager.Close()
	}
	if cupsManager != nil {
		cupsManager.Close()
	}
	if dwlManager != nil {
		dwlManager.Close()
	}
	if wlContext != nil {
		wlContext.Close()
	}
}

func Start(printDocs bool) error {
	cleanupStaleSockets()

	socketPath := GetSocketPath()
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer listener.Close()
	defer cleanupManagers()

	go func() {
		if err := InitializeNetworkManager(); err != nil {
			log.Warnf("Network manager unavailable: %v", err)
		}
	}()

	go func() {
		if err := InitializeLoginctlManager(); err != nil {
			log.Warnf("Loginctl manager unavailable: %v", err)
		}
	}()

	go func() {
		if err := InitializeFreedeskManager(); err != nil {
			log.Warnf("Freedesktop manager unavailable: %v", err)
		}
	}()

	if err := InitializeWaylandManager(); err != nil {
		log.Warnf("Wayland manager unavailable: %v", err)
	}

	go func() {
		if err := InitializeBluezManager(); err != nil {
			log.Warnf("Bluez manager unavailable: %v", err)
		}
	}()

	go func() {
		if err := InitializeCupsManager(); err != nil {
			log.Warnf("CUPS manager unavailable: %v", err)
		}
	}()
	
	if err := InitializeDwlManager(); err != nil {
		log.Debugf("DWL manager unavailable: %v", err)
	}

	if wlContext != nil {
		wlContext.Start()
		log.Info("Wayland event dispatcher started")
	}

	log.Infof("DMS API Server listening on: %s", socketPath)
	log.Info("Protocol: JSON over Unix socket")
	log.Info("Request format: {\"id\": <any>, \"method\": \"...\", \"params\": {...}}")
	log.Info("Response format: {\"id\": <any>, \"result\": {...}} or {\"id\": <any>, \"error\": \"...\"}")
	if printDocs {
		log.Info("Available methods:")
		log.Info("  ping          - Test connection")
		log.Info("  getServerInfo - Get server info (API version and capabilities)")
		log.Info("  subscribe     - Subscribe to multiple services (params: services [default: all])")
		log.Info("Plugins:")
		log.Info(" plugins.list                - List all plugins")
		log.Info(" plugins.listInstalled       - List installed plugins")
		log.Info(" plugins.install             - Install plugin (params: name)")
		log.Info(" plugins.uninstall           - Uninstall plugin (params: name)")
		log.Info(" plugins.update              - Update plugin (params: name)")
		log.Info(" plugins.search              - Search plugins (params: query, category?, compositor?, capability?)")
		log.Info("Network:")
		log.Info(" network.getState            - Get current network state")
		log.Info(" network.wifi.scan           - Scan for WiFi networks")
		log.Info(" network.wifi.networks       - Get WiFi network list")
		log.Info(" network.wifi.connect        - Connect to WiFi (params: ssid, password?, username?)")
		log.Info(" network.wifi.disconnect     - Disconnect WiFi")
		log.Info(" network.wifi.forget         - Forget network (params: ssid)")
		log.Info(" network.wifi.toggle         - Toggle WiFi radio")
		log.Info(" network.wifi.enable         - Enable WiFi")
		log.Info(" network.wifi.disable        - Disable WiFi")
		log.Info(" network.ethernet.connect    - Connect Ethernet")
		log.Info(" network.ethernet.connect.config - Connect Ethernet to a specific configuration")
		log.Info(" network.ethernet.disconnect - Disconnect Ethernet")
		log.Info(" network.vpn.profiles        - List VPN profiles")
		log.Info(" network.vpn.active          - List active VPN connections")
		log.Info(" network.vpn.connect         - Connect VPN (params: uuidOrName|name|uuid, singleActive?)")
		log.Info(" network.vpn.disconnect      - Disconnect VPN (params: uuidOrName|name|uuid)")
		log.Info(" network.vpn.disconnectAll   - Disconnect all VPNs")
		log.Info(" network.vpn.clearCredentials - Clear saved VPN credentials (params: uuidOrName|name|uuid)")
		log.Info(" network.preference.set      - Set preference (params: preference [auto|wifi|ethernet])")
		log.Info(" network.info                - Get network info (params: ssid)")
		log.Info(" network.credentials.submit  - Submit credentials for prompt (params: token, secrets, save?)")
		log.Info(" network.credentials.cancel  - Cancel credential prompt (params: token)")
		log.Info(" network.subscribe           - Subscribe to network state changes (streaming)")
		log.Info("Loginctl:")
		log.Info(" loginctl.getState           - Get current session state")
		log.Info(" loginctl.lock               - Lock session")
		log.Info(" loginctl.unlock             - Unlock session")
		log.Info(" loginctl.activate           - Activate session")
		log.Info(" loginctl.setIdleHint        - Set idle hint (params: idle)")
		log.Info(" loginctl.setLockBeforeSuspend - Set lock before suspend (params: enabled)")
		log.Info(" loginctl.setSleepInhibitorEnabled - Enable/disable sleep inhibitor (params: enabled)")
		log.Info(" loginctl.lockerReady        - Signal locker UI is ready (releases sleep inhibitor)")
		log.Info(" loginctl.terminate          - Terminate session")
		log.Info(" loginctl.subscribe          - Subscribe to session state changes (streaming)")
		log.Info("Freedesktop:")
		log.Info(" freedesktop.getState                  - Get accounts & settings state")
		log.Info(" freedesktop.accounts.setIconFile      - Set profile icon (params: path)")
		log.Info(" freedesktop.accounts.setRealName      - Set real name (params: name)")
		log.Info(" freedesktop.accounts.setEmail         - Set email (params: email)")
		log.Info(" freedesktop.accounts.setLanguage      - Set language (params: language)")
		log.Info(" freedesktop.accounts.setLocation      - Set location (params: location)")
		log.Info(" freedesktop.accounts.getUserIconFile  - Get user icon (params: username)")
		log.Info(" freedesktop.settings.getColorScheme   - Get color scheme")
		log.Info(" freedesktop.settings.setIconTheme     - Set icon theme (params: iconTheme)")
		log.Info("Wayland:")
		log.Info(" wayland.gamma.getState                - Get current gamma control state")
		log.Info(" wayland.gamma.setTemperature          - Set temperature range (params: low, high)")
		log.Info(" wayland.gamma.setLocation             - Set location (params: latitude, longitude)")
		log.Info(" wayland.gamma.setManualTimes          - Set manual times (params: sunrise, sunset)")
		log.Info(" wayland.gamma.setGamma                - Set gamma value (params: gamma)")
		log.Info(" wayland.gamma.setEnabled              - Enable/disable gamma control (params: enabled)")
		log.Info(" wayland.gamma.subscribe               - Subscribe to gamma state changes (streaming)")
		log.Info("Bluetooth:")
		log.Info(" bluetooth.getState                    - Get current bluetooth state")
		log.Info(" bluetooth.startDiscovery              - Start device discovery")
		log.Info(" bluetooth.stopDiscovery               - Stop device discovery")
		log.Info(" bluetooth.setPowered                  - Set adapter power state (params: powered)")
		log.Info(" bluetooth.pair                        - Pair with device (params: device)")
		log.Info(" bluetooth.connect                     - Connect to device (params: device)")
		log.Info(" bluetooth.disconnect                  - Disconnect from device (params: device)")
		log.Info(" bluetooth.remove                      - Remove/unpair device (params: device)")
		log.Info(" bluetooth.trust                       - Trust device (params: device)")
		log.Info(" bluetooth.untrust                     - Untrust device (params: device)")
		log.Info(" bluetooth.pairing.submit              - Submit pairing response (params: token, secrets?, accept?)")
		log.Info(" bluetooth.pairing.cancel              - Cancel pairing prompt (params: token)")
		log.Info(" bluetooth.subscribe                   - Subscribe to bluetooth state changes (streaming)")
		log.Info("CUPS:")
		log.Info(" cups.getPrinters                      - Get printers list")
		log.Info(" cups.getJobs                          - Get non-completed jobs list (params: printerName)")
		log.Info(" cups.pausePrinter                     - Pause printer (params: printerName)")
		log.Info(" cups.resumePrinter                    - Resume printer (params: printerName)")
		log.Info(" cups.cancelJob                        - Cancel job (params: printerName, jobID)")
		log.Info(" cups.purgeJobs                        - Cancel all jobs (params: printerName)")
		log.Info("DWL:")
		log.Info(" dwl.getState                          - Get current dwl state (tags, windows, layouts)")
		log.Info(" dwl.setTags                           - Set active tags (params: output, tagmask, toggleTagset)")
		log.Info(" dwl.setClientTags                     - Set focused client tags (params: output, andTags, xorTags)")
		log.Info(" dwl.setLayout                         - Set layout (params: output, index)")
		log.Info(" dwl.subscribe                         - Subscribe to dwl state changes (streaming)")
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go handleConnection(conn)
	}
}
