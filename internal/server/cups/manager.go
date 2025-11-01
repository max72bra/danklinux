package cups

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AvengeMedia/danklinux/internal/log"
	"github.com/godbus/dbus/v5"
)

const (
	cupsService   = "org.cups.cupsd"
	cupsPath      = "/org/cups/cupsd"
	cupsInterface = "org.cups.cupsd.Notifier"
)

func NewManager() (*Manager, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("system bus connection failed: %w", err)
	}

	m := &Manager{
		state: &CUPSState{
			Printers: make(map[string]*Printer),
		},
		stateMutex:  sync.RWMutex{},
		BaseURL:     "http://localhost:631",
		Client:      &http.Client{Timeout: 30 * time.Second},
		dbusConn:    conn,
		signals:     make(chan *dbus.Signal, 256),
		dirty:       make(chan struct{}, 1),
		stopChan:    make(chan struct{}),
		subscribers: make(map[string]chan CUPSState),
		subMutex:    sync.RWMutex{},
	}

	if err := m.initialize(); err != nil {
		conn.Close()
		return nil, err
	}

	m.notifierWg.Add(1)
	go m.notifier()

	return m, nil
}

func (m *Manager) initialize() error {
	if err := m.updateState(); err != nil {
		return err
	}

	if err := m.startSignalPump(); err != nil {
		m.Close()
		return err
	}
	// dbus on CUPS is not so...
	obj := m.dbusConn.Object(cupsService, cupsPath)
	err := obj.Call("org.freedesktop.DBus.Peer.Ping", 0).Store()
	if err != nil {
		// fallback on stat on cups log and inject fake dbus signals
		log.Warnf("[CUPS] D-Bus interface not available. Fallback to log parsing")
		lm, err := m.NewLogMonitor()
		if err == nil {
			lm.FallbackLogMonitorStart()
		}
		return err
	}

	return nil
}

func (m *Manager) NewLogMonitor() (*LogMonitor, error) {
	ctx, cancel := context.WithCancel(context.Background())

	logPaths := []string{
		"/var/log/cups/access_log",
	}

	lm := &LogMonitor{
		ctx:      ctx,
		cancel:   cancel,
		logPaths: logPaths,
		manager:  m,
	}

	m.lm = *lm

	return lm, nil
}

func (lm *LogMonitor) FallbackLogMonitorStart() error {
	// CUPS log monitor
	for _, logPath := range lm.logPaths {
		if _, err := os.Stat(logPath); err == nil {
			go lm.monitorLogFile(logPath)
		}
	}
	return nil
}

func (lm *LogMonitor) monitorLogFile(logPath string) {
	// tail -F to follow the log
	cmd := exec.CommandContext(lm.ctx, "tail", "-F", "-n", "0", logPath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Errorf("[CUPS] Error stdout pipe for %s: %v", logPath, err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Errorf("[CUPS] Error tail start for %s: %v", logPath, err)
		return
	}

	reader := bufio.NewReader(stdout)

	for {
		select {
		case <-lm.ctx.Done():
			cmd.Process.Kill()
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				log.Errorf("[CUPS] Read error log: %v", err)
				return
			}

			if line != "" {
				lm.processLogLine(strings.TrimSpace(line))
			}
		}
	}
}

func (lm *LogMonitor) processLogLine(line string) {
	entry, err := lm.parseLogLine(line)

	if err != nil {
		log.Error("[CUPS] Log event error", "err", err)
	} else {
		if entry.Status == 200 {
			printerName := entry.GetPrinterName()
			switch entry.IPPOperation {
			case "Create-Job", "Print-Job":
				lm.manager.InjectJobCreated(printerName)

			case "Cancel-Job":
				lm.manager.InjectJobCompleted(printerName)

			case "Pause-Printer", "Resume-Printer", "Enable-Printer", "Disable-Printer":
				lm.manager.InjectPrinterStateChanged(printerName)

			case "FakeAdd-Printer":
				lm.manager.InjectPrinterAdded()

			case "FakeDelete-Printer":
				lm.manager.InjectPrinterDeleted()
			}
		}
	}
}

func (lm *LogMonitor) parseLogLine(line string) (*CUPSAccessLogEntry, error) {
	// Pattern regex access_log
	// localhost - - [01/Jan/2025:10:30:45 +0100] "POST /printers/PDF HTTP/1.1" 200 123 Print-Job successful-ok
	pattern := `^(\S+)\s+(\S+)\s+(\S+)\s+\[([^\]]+)\]\s+"([A-Z]+)\s+(\S+)\s+([^"]+)"\s+(\d+)\s+(\d+)(?:\s+(\S+))?(?:\s+(\S+))?`

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(line)

	if len(matches) < 10 {
		return nil, fmt.Errorf("formato log non valido")
	}

	// Parse timestamp
	timestamp, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[4])
	if err != nil {
		return nil, fmt.Errorf("errore parsing timestamp: %v", err)
	}

	// Parse status e bytes
	status, _ := strconv.Atoi(matches[8])
	bytes, _ := strconv.Atoi(matches[9])

	entry := &CUPSAccessLogEntry{
		Host:      matches[1],
		Group:     matches[2],
		User:      matches[3],
		Timestamp: timestamp,
		Method:    matches[5],
		Resource:  matches[6],
		Version:   matches[7],
		Status:    status,
		Bytes:     bytes,
	}

	// IPP operation optional
	if len(matches) > 10 && matches[10] != "" {
		entry.IPPOperation = matches[10]
	}
	if len(matches) > 11 && matches[11] != "" {
		entry.IPPStatus = matches[11]
	}

	return entry, nil
}

func (e *CUPSAccessLogEntry) GetPrinterName() string {
	parts := strings.Split(e.Resource, "/")
	for i, part := range parts {
		if part == "printers" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func (e *CUPSAccessLogEntry) IsSuccessful() bool {
	return e.Status >= 200 && e.Status < 300
}

func (e *CUPSAccessLogEntry) EventType() string {
	if e.IPPOperation != "" {
		return e.IPPOperation
	}
	// Fallback
	if strings.Contains(e.Resource, "/jobs") {
		return "Job-Operation"
	}
	if strings.Contains(e.Resource, "/printers") {
		return "Printer-Operation"
	}
	return "Unknown"
}

func (lm *LogMonitor) Close() {
	lm.cancel()
}

func (m *Manager) updateState() error {
	printers, err := m.GetPrinters()
	if err != nil {
		return err
	}

	printerMap := make(map[string]*Printer, len(printers))
	for _, printer := range printers {
		jobs, err := m.GetJobs(printer.Name, "not-completed")
		if err != nil {
			return err
		}

		printer.Jobs = jobs

		printerMap[printer.Name] = &printer
	}

	m.stateMutex.Lock()
	m.state.Printers = printerMap
	m.stateMutex.Unlock()

	return nil
}

func (m *Manager) startSignalPump() error {
	m.dbusConn.Signal(m.signals)

	if err := m.dbusConn.AddMatchSignal(
		dbus.WithMatchInterface(cupsInterface),
		dbus.WithMatchMember("PrinterAdded"),
	); err != nil {
		return err
	}

	if err := m.dbusConn.AddMatchSignal(
		dbus.WithMatchInterface(cupsInterface),
		dbus.WithMatchMember("PrinterDeleted"),
	); err != nil {
		return err
	}

	if err := m.dbusConn.AddMatchSignal(
		dbus.WithMatchInterface(cupsInterface),
		dbus.WithMatchMember("PrinterStateChanged"),
	); err != nil {
		return err
	}

	if err := m.dbusConn.AddMatchSignal(
		dbus.WithMatchInterface(cupsInterface),
		dbus.WithMatchMember("JobCreated"),
	); err != nil {
		return err
	}

	if err := m.dbusConn.AddMatchSignal(
		dbus.WithMatchInterface(cupsInterface),
		dbus.WithMatchMember("JobCompleted"),
	); err != nil {
		return err
	}

	m.sigWG.Add(1)
	go func() {
		defer m.sigWG.Done()
		for {
			select {
			case <-m.stopChan:
				return
			case sig, ok := <-m.signals:
				if !ok {
					return
				}
				if sig == nil {
					continue
				}
				m.handleSignal(sig)
			}
		}
	}()

	return nil
}

func (m *Manager) handleSignal(sig *dbus.Signal) {
	switch sig.Name {
	case cupsInterface + ".PrinterAdded", cupsInterface + ".PrinterDeleted",
		cupsInterface + ".PrinterStateChanged":
		m.updateState()
		m.notifySubscribers()
	case cupsInterface + ".JobCreated", cupsInterface + ".JobCompleted":
		if len(sig.Body) >= 1 {
			if printerName, ok := sig.Body[0].(string); ok {
				m.stateMutex.Lock()

				printers := m.state.Printers
				printer, exists := printers[printerName]
				if exists {
					jobs, err := m.GetJobs(printerName, "not-completed")
					if err == nil {
						printer.Jobs = jobs
					}
				}
				m.state.Printers = printers

				m.stateMutex.Unlock()
				m.notifySubscribers()
			}
		}
	}
}

func (m *Manager) InjectSignal(name string, body ...interface{}) bool {
	sig := &dbus.Signal{
		Name: name,
		Body: body,
	}

	select {
	case m.signals <- sig:
		return true
	case <-time.After(2 * time.Second):
		return false // Timeout
	}
}

func (m *Manager) InjectPrinterAdded() bool {
	return m.InjectSignal(cupsInterface + ".PrinterAdded")
}

func (m *Manager) InjectPrinterDeleted() bool {
	return m.InjectSignal(cupsInterface + ".PrinterDeleted")
}

func (m *Manager) InjectPrinterStateChanged(printerName string) bool {
	return m.InjectSignal(cupsInterface+".PrinterStateChanged", printerName)
}

func (m *Manager) InjectJobCreated(printerName string) bool {
	return m.InjectSignal(cupsInterface+".JobCreated", printerName)
}

func (m *Manager) InjectJobCompleted(printerName string) bool {
	return m.InjectSignal(cupsInterface+".JobCompleted", printerName)
}

func (m *Manager) notifier() {
	defer m.notifierWg.Done()
	const minGap = 100 * time.Millisecond
	var timer *time.Timer
	var pending bool
	for {
		select {
		case <-m.stopChan:
			return
		case <-m.dirty:
			if pending {
				continue
			}
			pending = true
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(minGap, func() {
				m.subMutex.RLock()
				if len(m.subscribers) == 0 {
					m.subMutex.RUnlock()
					pending = false
					return
				}

				currentState := m.snapshotState()

				if m.lastNotifiedState != nil && !stateChanged(m.lastNotifiedState, &currentState) {
					m.subMutex.RUnlock()
					pending = false
					return
				}

				for _, ch := range m.subscribers {
					select {
					case ch <- currentState:
					default:
					}
				}
				m.subMutex.RUnlock()

				stateCopy := currentState
				m.lastNotifiedState = &stateCopy
				pending = false
			})
		}
	}
}

func (m *Manager) notifySubscribers() {
	select {
	case m.dirty <- struct{}{}:
	default:
	}
}

func (m *Manager) GetState() CUPSState {
	return m.snapshotState()
}

func (m *Manager) snapshotState() CUPSState {
	m.stateMutex.RLock()
	defer m.stateMutex.RUnlock()

	s := CUPSState{
		Printers: make(map[string]*Printer, len(m.state.Printers)),
	}
	for name, printer := range m.state.Printers {
		printerCopy := *printer
		s.Printers[name] = &printerCopy
	}
	return s
}

func (m *Manager) Subscribe(id string) chan CUPSState {
	ch := make(chan CUPSState, 64)
	m.subMutex.Lock()
	m.subscribers[id] = ch
	m.subMutex.Unlock()
	return ch
}

func (m *Manager) Unsubscribe(id string) {
	m.subMutex.Lock()
	if ch, ok := m.subscribers[id]; ok {
		close(ch)
		delete(m.subscribers, id)
	}
	m.subMutex.Unlock()
}

func (m *Manager) Close() {
	close(m.stopChan)
	m.notifierWg.Wait()

	m.sigWG.Wait()

	if m.signals != nil {
		m.dbusConn.RemoveSignal(m.signals)
		close(m.signals)
	}

	m.subMutex.Lock()
	for _, ch := range m.subscribers {
		close(ch)
	}
	m.subscribers = make(map[string]chan CUPSState)
	m.subMutex.Unlock()

	if m.dbusConn != nil {
		m.dbusConn.Close()
	}
}

func stateChanged(old, new *CUPSState) bool {
	return !reflect.DeepEqual(old, new)
}
