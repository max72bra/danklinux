package dwl

import (
	"fmt"
	"time"

	wlclient "github.com/yaslama/go-wayland/wayland/client"

	"github.com/AvengeMedia/danklinux/internal/log"
	"github.com/AvengeMedia/danklinux/internal/proto/dwl_ipc"
)

func NewManager(display *wlclient.Display) (*Manager, error) {
	m := &Manager{
		display:        display,
		outputs:        make(map[uint32]*outputState),
		cmdq:           make(chan cmd, 128),
		outputSetupReq: make(chan uint32, 16),
		stopChan:       make(chan struct{}),
		subscribers:    make(map[string]chan State),
		dirty:          make(chan struct{}, 1),
		layouts:        make([]string, 0),
	}

	if err := m.setupRegistry(); err != nil {
		return nil, err
	}

	m.updateState()

	m.notifierWg.Add(1)
	go m.notifier()

	m.wg.Add(1)
	go m.waylandActor()

	return m, nil
}

func (m *Manager) post(fn func()) {
	select {
	case m.cmdq <- cmd{fn: fn}:
	default:
		log.Warn("DWL actor command queue full, dropping command")
	}
}

func (m *Manager) waylandActor() {
	defer m.wg.Done()

	for {
		select {
		case <-m.stopChan:
			return
		case c := <-m.cmdq:
			c.fn()
		case outputID := <-m.outputSetupReq:
			m.outputsMutex.RLock()
			out, exists := m.outputs[outputID]
			m.outputsMutex.RUnlock()

			if !exists {
				log.Warnf("DWL: Output %d no longer exists, skipping setup", outputID)
				continue
			}

			if out.ipcOutput != nil {
				continue
			}

			mgr, ok := m.manager.(*dwl_ipc.ZdwlIpcManagerV2)
			if !ok || mgr == nil {
				log.Errorf("DWL: Manager not available for output %d setup", outputID)
				continue
			}

			log.Infof("DWL: Setting up ipcOutput for dynamically added output %d", outputID)
			if err := m.setupOutput(mgr, out.output); err != nil {
				log.Errorf("DWL: Failed to setup output %d: %v", outputID, err)
			} else {
				m.updateState()
			}
		}
	}
}

func (m *Manager) setupRegistry() error {
	log.Info("DWL: starting registry setup")
	ctx := m.display.Context()

	registry, err := m.display.GetRegistry()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}
	m.registry = registry

	outputs := make([]*wlclient.Output, 0)
	outputRegNames := make(map[uint32]uint32)
	var dwlMgr *dwl_ipc.ZdwlIpcManagerV2

	registry.SetGlobalHandler(func(e wlclient.RegistryGlobalEvent) {
		switch e.Interface {
		case dwl_ipc.ZdwlIpcManagerV2InterfaceName:
			log.Infof("DWL: found %s", dwl_ipc.ZdwlIpcManagerV2InterfaceName)
			manager := dwl_ipc.NewZdwlIpcManagerV2(ctx)
			version := e.Version
			if version > 1 {
				version = 1
			}
			if err := registry.Bind(e.Name, e.Interface, version, manager); err == nil {
				dwlMgr = manager
				log.Info("DWL: manager bound successfully")
			} else {
				log.Errorf("DWL: failed to bind manager: %v", err)
			}
		case "wl_output":
			log.Debugf("DWL: found wl_output (name=%d)", e.Name)
			output := wlclient.NewOutput(ctx)

			outState := &outputState{
				registryName: e.Name,
				output:       output,
				tags:         make([]TagState, 0),
			}

			output.SetNameHandler(func(ev wlclient.OutputNameEvent) {
				log.Debugf("DWL: Output name: %s (registry=%d)", ev.Name, e.Name)
				outState.name = ev.Name
			})

			output.SetDescriptionHandler(func(ev wlclient.OutputDescriptionEvent) {
				log.Debugf("DWL: Output description: %s", ev.Description)
			})

			version := e.Version
			if version > 4 {
				version = 4
			}
			if err := registry.Bind(e.Name, e.Interface, version, output); err == nil {
				outputID := output.ID()
				outState.id = outputID
				log.Infof("DWL: Bound wl_output id=%d registry_name=%d", outputID, e.Name)
				outputs = append(outputs, output)
				outputRegNames[outputID] = e.Name

				m.outputsMutex.Lock()
				m.outputs[outputID] = outState
				m.outputsMutex.Unlock()

				if m.manager != nil {
					select {
					case m.outputSetupReq <- outputID:
						log.Debugf("DWL: Queued setup for output %d", outputID)
					default:
						log.Warnf("DWL: Setup queue full, output %d will not be initialized", outputID)
					}
				}
			} else {
				log.Errorf("DWL: Failed to bind wl_output: %v", err)
			}
		}
	})

	registry.SetGlobalRemoveHandler(func(e wlclient.RegistryGlobalRemoveEvent) {
		m.post(func() {
			m.outputsMutex.Lock()
			var outToRelease *outputState
			for id, out := range m.outputs {
				if out.registryName == e.Name {
					log.Infof("DWL: Output %d removed", id)
					outToRelease = out
					delete(m.outputs, id)
					break
				}
			}
			m.outputsMutex.Unlock()

			if outToRelease != nil {
				if ipcOut, ok := outToRelease.ipcOutput.(*dwl_ipc.ZdwlIpcOutputV2); ok && ipcOut != nil {
					m.wlMutex.Lock()
					ipcOut.Release()
					m.wlMutex.Unlock()
					log.Debugf("DWL: Released ipcOutput for removed output %d", outToRelease.id)
				}
				m.updateState()
			}
		})
	})

	if err := m.display.Roundtrip(); err != nil {
		return fmt.Errorf("first roundtrip failed: %w", err)
	}
	if err := m.display.Roundtrip(); err != nil {
		return fmt.Errorf("second roundtrip failed: %w", err)
	}

	if dwlMgr == nil {
		log.Info("DWL: manager not found in registry")
		return fmt.Errorf("dwl_ipc_manager_v2 not available")
	}

	dwlMgr.SetTagsHandler(func(e dwl_ipc.ZdwlIpcManagerV2TagsEvent) {
		log.Infof("DWL: Tags count: %d", e.Amount)
		m.tagCount = e.Amount
		m.updateState()
	})

	dwlMgr.SetLayoutHandler(func(e dwl_ipc.ZdwlIpcManagerV2LayoutEvent) {
		log.Infof("DWL: Layout: %s", e.Name)
		m.layouts = append(m.layouts, e.Name)
		m.updateState()
	})

	m.manager = dwlMgr

	for _, output := range outputs {
		if err := m.setupOutput(dwlMgr, output); err != nil {
			log.Warnf("DWL: Failed to setup output %d: %v", output.ID(), err)
		}
	}

	if err := m.display.Roundtrip(); err != nil {
		return fmt.Errorf("final roundtrip failed: %w", err)
	}

	log.Info("DWL: registry setup complete")
	return nil
}

func (m *Manager) setupOutput(manager *dwl_ipc.ZdwlIpcManagerV2, output *wlclient.Output) error {
	m.wlMutex.Lock()
	ipcOutput, err := manager.GetOutput(output)
	m.wlMutex.Unlock()
	if err != nil {
		return fmt.Errorf("failed to get dwl output: %w", err)
	}

	m.outputsMutex.Lock()
	outState, exists := m.outputs[output.ID()]
	if !exists {
		m.outputsMutex.Unlock()
		return fmt.Errorf("output state not found for id %d", output.ID())
	}
	outState.ipcOutput = ipcOutput
	m.outputsMutex.Unlock()

	ipcOutput.SetActiveHandler(func(e dwl_ipc.ZdwlIpcOutputV2ActiveEvent) {
		outState.active = e.Active
	})

	ipcOutput.SetTagHandler(func(e dwl_ipc.ZdwlIpcOutputV2TagEvent) {
		updated := false
		for i, tag := range outState.tags {
			if tag.Tag == e.Tag {
				outState.tags[i] = TagState{
					Tag:     e.Tag,
					State:   e.State,
					Clients: e.Clients,
					Focused: e.Focused,
				}
				updated = true
				break
			}
		}

		if !updated {
			outState.tags = append(outState.tags, TagState{
				Tag:     e.Tag,
				State:   e.State,
				Clients: e.Clients,
				Focused: e.Focused,
			})
		}

		m.updateState()
	})

	ipcOutput.SetLayoutHandler(func(e dwl_ipc.ZdwlIpcOutputV2LayoutEvent) {
		outState.layout = e.Layout
	})

	ipcOutput.SetTitleHandler(func(e dwl_ipc.ZdwlIpcOutputV2TitleEvent) {
		outState.title = e.Title
	})

	ipcOutput.SetAppidHandler(func(e dwl_ipc.ZdwlIpcOutputV2AppidEvent) {
		outState.appID = e.Appid
	})

	ipcOutput.SetLayoutSymbolHandler(func(e dwl_ipc.ZdwlIpcOutputV2LayoutSymbolEvent) {
		outState.layoutSymbol = e.Layout
	})

	ipcOutput.SetFrameHandler(func(e dwl_ipc.ZdwlIpcOutputV2FrameEvent) {
		m.updateState()
	})

	return nil
}

func (m *Manager) updateState() {
	m.outputsMutex.RLock()
	outputs := make(map[string]*OutputState)
	activeOutput := ""

	for _, out := range m.outputs {
		name := out.name
		if name == "" {
			name = fmt.Sprintf("output-%d", out.id)
		}

		tagsCopy := make([]TagState, len(out.tags))
		copy(tagsCopy, out.tags)

		outputs[name] = &OutputState{
			Name:         name,
			Active:       out.active,
			Tags:         tagsCopy,
			Layout:       out.layout,
			LayoutSymbol: out.layoutSymbol,
			Title:        out.title,
			AppID:        out.appID,
		}

		if out.active != 0 {
			activeOutput = name
		}
	}
	m.outputsMutex.RUnlock()

	newState := State{
		Outputs:      outputs,
		TagCount:     m.tagCount,
		Layouts:      m.layouts,
		ActiveOutput: activeOutput,
	}

	m.stateMutex.Lock()
	m.state = &newState
	m.stateMutex.Unlock()

	m.notifySubscribers()
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
				subCount := len(m.subscribers)
				m.subMutex.RUnlock()

				if subCount == 0 {
					pending = false
					return
				}

				currentState := m.GetState()

				if m.lastNotified != nil && !stateChanged(m.lastNotified, &currentState) {
					pending = false
					return
				}

				m.subMutex.RLock()
				for _, ch := range m.subscribers {
					select {
					case ch <- currentState:
					default:
						log.Warn("DWL: subscriber channel full, dropping update")
					}
				}
				m.subMutex.RUnlock()

				stateCopy := currentState
				m.lastNotified = &stateCopy
				pending = false
			})
		}
	}
}

func (m *Manager) ensureOutputSetup(out *outputState) error {
	if out.ipcOutput != nil {
		return nil
	}

	return fmt.Errorf("output not yet initialized - setup in progress, retry in a moment")
}

func (m *Manager) SetTags(outputName string, tagmask uint32, toggleTagset uint32) error {
	m.outputsMutex.RLock()

	availableOutputs := make([]string, 0, len(m.outputs))
	var targetOut *outputState
	for _, out := range m.outputs {
		name := out.name
		if name == "" {
			name = fmt.Sprintf("output-%d", out.id)
		}
		availableOutputs = append(availableOutputs, name)
		if name == outputName {
			targetOut = out
			break
		}
	}
	m.outputsMutex.RUnlock()

	if targetOut == nil {
		return fmt.Errorf("output not found: %s (available: %v)", outputName, availableOutputs)
	}

	if err := m.ensureOutputSetup(targetOut); err != nil {
		return fmt.Errorf("failed to setup output %s: %w", outputName, err)
	}

	ipcOut, ok := targetOut.ipcOutput.(*dwl_ipc.ZdwlIpcOutputV2)
	if !ok {
		return fmt.Errorf("output %s has invalid ipcOutput type", outputName)
	}

	m.wlMutex.Lock()
	err := ipcOut.SetTags(tagmask, toggleTagset)
	m.wlMutex.Unlock()
	return err
}

func (m *Manager) SetClientTags(outputName string, andTags uint32, xorTags uint32) error {
	m.outputsMutex.RLock()

	var targetOut *outputState
	for _, out := range m.outputs {
		name := out.name
		if name == "" {
			name = fmt.Sprintf("output-%d", out.id)
		}
		if name == outputName {
			targetOut = out
			break
		}
	}
	m.outputsMutex.RUnlock()

	if targetOut == nil {
		return fmt.Errorf("output not found: %s", outputName)
	}

	if err := m.ensureOutputSetup(targetOut); err != nil {
		return fmt.Errorf("failed to setup output %s: %w", outputName, err)
	}

	ipcOut, ok := targetOut.ipcOutput.(*dwl_ipc.ZdwlIpcOutputV2)
	if !ok {
		return fmt.Errorf("output %s has invalid ipcOutput type", outputName)
	}

	m.wlMutex.Lock()
	err := ipcOut.SetClientTags(andTags, xorTags)
	m.wlMutex.Unlock()
	return err
}

func (m *Manager) SetLayout(outputName string, index uint32) error {
	m.outputsMutex.RLock()

	var targetOut *outputState
	for _, out := range m.outputs {
		name := out.name
		if name == "" {
			name = fmt.Sprintf("output-%d", out.id)
		}
		if name == outputName {
			targetOut = out
			break
		}
	}
	m.outputsMutex.RUnlock()

	if targetOut == nil {
		return fmt.Errorf("output not found: %s", outputName)
	}

	if err := m.ensureOutputSetup(targetOut); err != nil {
		return fmt.Errorf("failed to setup output %s: %w", outputName, err)
	}

	ipcOut, ok := targetOut.ipcOutput.(*dwl_ipc.ZdwlIpcOutputV2)
	if !ok {
		return fmt.Errorf("output %s has invalid ipcOutput type", outputName)
	}

	m.wlMutex.Lock()
	err := ipcOut.SetLayout(index)
	m.wlMutex.Unlock()
	return err
}

func (m *Manager) Close() {
	close(m.stopChan)
	m.wg.Wait()
	m.notifierWg.Wait()

	m.subMutex.Lock()
	for _, ch := range m.subscribers {
		close(ch)
	}
	m.subscribers = make(map[string]chan State)
	m.subMutex.Unlock()

	m.outputsMutex.Lock()
	for _, out := range m.outputs {
		if ipcOut, ok := out.ipcOutput.(*dwl_ipc.ZdwlIpcOutputV2); ok {
			ipcOut.Release()
		}
	}
	m.outputs = make(map[uint32]*outputState)
	m.outputsMutex.Unlock()

	if mgr, ok := m.manager.(*dwl_ipc.ZdwlIpcManagerV2); ok {
		mgr.Release()
	}
}
