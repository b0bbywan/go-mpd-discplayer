package hwcontrol

import (
	"fmt"
	"log"
	"os"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

const (
	busName          = "org.bluez"
	agentPath        = "/speaker/agent"
	agentInterface   = "org.bluez.Agent1"
	adapterPath      = "/org/bluez/hci0"
	adapterInterface = "org.bluez.Adapter1"
	busPropertyName  = "org.freedesktop.DBus.Properties.Set"

	a2dp           = "0000110d-0000-1000-8000-00805f9b34fb"
	avrcp          = "0000110e-0000-1000-8000-00805f9b34fb"
)

type Agent struct{}

func (a *Agent) Release() *dbus.Error {
	fmt.Println("Release")
	os.Exit(0)
	return nil
}

func (a *Agent) AuthorizeService(device string, uuid string) *dbus.Error {
	if uuid == a2dp || uuid == avrcp {
		log.Printf("AuthorizeService (%s, %s)\n", device, uuid)
		return nil
	}
	log.Printf("Service rejected (%s, %s)\n", device, uuid)
	return dbus.NewError("org.bluez.Error.Rejected", nil)
}

func (a *Agent) Cancel() *dbus.Error {
	log.Println("Cancel")
	return nil
}

func NewBluetoothAgent() (*Agent, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to system bus: %w", err)
	}

	agent := &Agent{}
	conn.Export(agent, agentPath, agentInterface)

	node := &introspect.Node{
		Name: "/speaker/agent",
		Interfaces: []introspect.Interface{
			{
				Name:    agentInterface,
				Methods: introspect.Methods(agent),
			},
		},
	}
	conn.Export(introspect.NewIntrospectable(node), agentPath, "org.freedesktop.DBus.Introspectable")

	obj := conn.Object(busName, adapterPath)
	props := map[string]interface{}{
		"Name":                 "CustomA2DPSink",
		"Role":                 "server",
		"RequireAuthentication": true,
		"RequireAuthorization":  true,
	}
	if err = obj.Call(busPropertyName, 0,
		adapterInterface,
		"DiscoverableTimeout",
		dbus.MakeVariant(uint32(0)),
	).Store(); err != nil {
		return nil, fmt.Errorf("Failed to set DiscoverableTimeout: %w", err)
	}
	if err = obj.Call(busPropertyName, 0,
		adapterInterface,
		"Discoverable",
		dbus.MakeVariant(true),
	).Store(); err != nil {
		return nil, fmt.Errorf("Failed to set Discoverable: %w", err)
	}
	log.Println("RPi speaker discoverable")

	manager := conn.Object(busName, "/org/bluez")
	if err = manager.Call("org.bluez.ProfileManager1.RegisterProfile", 0,
		dbus.ObjectPath("/custom/profile"),
		"0000110b-0000-1000-8000-00805f9b34fb",
		props,
	).Store(); err != nil {
		return nil, fmt.Errorf("Failed to register profile: %v", err)
	}
	log.Printf("profile registered")
	if err = manager.Call("org.bluez.AgentManager1.RegisterAgent", 0,
		dbus.ObjectPath(agentPath),
		"NoInputNoOutput",
	).Store(); err != nil {
		return nil, fmt.Errorf("Failed to register agent: %w", err)
	}
	log.Println("Agent registered")

	if err = manager.Call("org.bluez.AgentManager1.RequestDefaultAgent", 0,
		dbus.ObjectPath(agentPath),
	).Store(); err != nil {
		return nil, fmt.Errorf("Failed to request default agent: %w", err)
	}

	log.Println("Agent set as default. Waiting for events...")
	return agent, nil
}
