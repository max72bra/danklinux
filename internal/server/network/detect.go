package network

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

type BackendType int

const (
	BackendNone BackendType = iota
	BackendNetworkManager
	BackendIwd
	BackendConnMan
)

func nameHasOwner(bus *dbus.Conn, name string) (bool, error) {
	obj := bus.Object("org.freedesktop.DBus", "/org/freedesktop/DBus")
	var owned bool
	if err := obj.Call("org.freedesktop.DBus.NameHasOwner", 0, name).Store(&owned); err != nil {
		return false, err
	}
	return owned, nil
}

type DetectResult struct {
	Backend      BackendType
	HasNM        bool
	HasIwd       bool
	HasConnMan   bool
	HasWpaSupp   bool
	ChosenReason string
}

func DetectNetworkStack() (*DetectResult, error) {
	bus, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("connect system bus: %w", err)
	}
	defer bus.Close()

	hasNM, _ := nameHasOwner(bus, "org.freedesktop.NetworkManager")
	hasIwd, _ := nameHasOwner(bus, "net.connman.iwd")
	hasConn, _ := nameHasOwner(bus, "net.connman")
	hasWpa, _ := nameHasOwner(bus, "fi.w1.wpa_supplicant1")

	res := &DetectResult{
		HasNM:      hasNM,
		HasIwd:     hasIwd,
		HasConnMan: hasConn,
		HasWpaSupp: hasWpa,
	}

	switch {
	case hasNM:
		res.Backend = BackendNetworkManager
		if hasIwd {
			res.ChosenReason = "NetworkManager present; iwd also running (likely NM's Wi-Fi backend). Using NM API."
		} else {
			res.ChosenReason = "NetworkManager present. Using NM API."
		}
	case hasConn && hasIwd:
		res.Backend = BackendConnMan
		res.ChosenReason = "ConnMan + iwd detected. Use ConnMan API (iwd is its Wi-Fi daemon)."
	case hasIwd:
		res.Backend = BackendIwd
		res.ChosenReason = "iwd detected without NM/ConnMan. Using iwd API."
	default:
		res.Backend = BackendNone
		if hasWpa {
			res.ChosenReason = "No NM/ConnMan/iwd; wpa_supplicant present. Consider a wpa_supplicant path."
		} else {
			res.ChosenReason = "No known network manager bus names found."
		}
	}

	return res, nil
}
