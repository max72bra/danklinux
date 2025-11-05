package brightness

import (
	"encoding/json"
	"net"

	"github.com/AvengeMedia/danklinux/internal/server/models"
)

func HandleRequest(conn net.Conn, req Request, m *Manager) {
	switch req.Method {
	case "brightness.getState":
		handleGetState(conn, req, m)
	case "brightness.setBrightness":
		handleSetBrightness(conn, req, m)
	case "brightness.increment":
		handleIncrement(conn, req, m)
	case "brightness.decrement":
		handleDecrement(conn, req, m)
	case "brightness.rescan":
		handleRescan(conn, req, m)
	case "brightness.subscribe":
		handleSubscribe(conn, req, m)
	default:
		models.RespondError(conn, req.ID.(int), "unknown method: "+req.Method)
	}
}

func handleGetState(conn net.Conn, req Request, m *Manager) {
	state := m.GetState()
	models.Respond(conn, req.ID.(int), state)
}

func handleSetBrightness(conn net.Conn, req Request, m *Manager) {
	var params SetBrightnessParams

	device, ok := req.Params["device"].(string)
	if !ok {
		models.RespondError(conn, req.ID.(int), "missing or invalid device parameter")
		return
	}
	params.Device = device

	percentFloat, ok := req.Params["percent"].(float64)
	if !ok {
		models.RespondError(conn, req.ID.(int), "missing or invalid percent parameter")
		return
	}
	params.Percent = int(percentFloat)

	if exponential, ok := req.Params["exponential"].(bool); ok {
		params.Exponential = exponential
	}

	if err := m.SetBrightnessWithMode(params.Device, params.Percent, params.Exponential); err != nil {
		models.RespondError(conn, req.ID.(int), err.Error())
		return
	}

	state := m.GetState()
	models.Respond(conn, req.ID.(int), state)
}

func handleIncrement(conn net.Conn, req Request, m *Manager) {
	device, ok := req.Params["device"].(string)
	if !ok {
		models.RespondError(conn, req.ID.(int), "missing or invalid device parameter")
		return
	}

	step := 10
	if stepFloat, ok := req.Params["step"].(float64); ok {
		step = int(stepFloat)
	}

	exponential := false
	if expBool, ok := req.Params["exponential"].(bool); ok {
		exponential = expBool
	}

	if err := m.IncrementBrightnessWithMode(device, step, exponential); err != nil {
		models.RespondError(conn, req.ID.(int), err.Error())
		return
	}

	state := m.GetState()
	models.Respond(conn, req.ID.(int), state)
}

func handleDecrement(conn net.Conn, req Request, m *Manager) {
	device, ok := req.Params["device"].(string)
	if !ok {
		models.RespondError(conn, req.ID.(int), "missing or invalid device parameter")
		return
	}

	step := 10
	if stepFloat, ok := req.Params["step"].(float64); ok {
		step = int(stepFloat)
	}

	exponential := false
	if expBool, ok := req.Params["exponential"].(bool); ok {
		exponential = expBool
	}

	if err := m.IncrementBrightnessWithMode(device, -step, exponential); err != nil {
		models.RespondError(conn, req.ID.(int), err.Error())
		return
	}

	state := m.GetState()
	models.Respond(conn, req.ID.(int), state)
}

func handleRescan(conn net.Conn, req Request, m *Manager) {
	m.Rescan()
	state := m.GetState()
	models.Respond(conn, req.ID.(int), state)
}

func handleSubscribe(conn net.Conn, req Request, m *Manager) {
	clientID := "brightness-subscriber"
	if idStr, ok := req.ID.(string); ok && idStr != "" {
		clientID = idStr
	}

	ch := m.Subscribe(clientID)
	defer m.Unsubscribe(clientID)

	initialState := m.GetState()
	if err := json.NewEncoder(conn).Encode(models.Response[State]{
		ID:     req.ID.(int),
		Result: &initialState,
	}); err != nil {
		return
	}

	for state := range ch {
		if err := json.NewEncoder(conn).Encode(models.Response[State]{
			ID:     req.ID.(int),
			Result: &state,
		}); err != nil {
			return
		}
	}
}
