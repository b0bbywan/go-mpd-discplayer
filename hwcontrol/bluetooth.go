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
)

var uuids = map[string]string{
	"GENERIC_AUDIO_UUID":           "00001203-0000-1000-8000-00805f9b34fb",
	"HSP_HS_UUID":                  "00001108-0000-1000-8000-00805f9b34fb",
	"HSP_AG_UUID":                  "00001112-0000-1000-8000-00805f9b34fb",
	"HFP_HS_UUID":                  "0000111e-0000-1000-8000-00805f9b34fb",
	"HFP_AG_UUID":                  "0000111f-0000-1000-8000-00805f9b34fb",
	"ADVANCED_AUDIO_UUID":          "0000110d-0000-1000-8000-00805f9b34fb",
	"A2DP_SOURCE_UUID":             "0000110a-0000-1000-8000-00805f9b34fb",
	"A2DP_SINK_UUID":               "0000110b-0000-1000-8000-00805f9b34fb",
	"AVRCP_REMOTE_UUID":            "0000110e-0000-1000-8000-00805f9b34fb",
	"AVRCP_TARGET_UUID":            "0000110c-0000-1000-8000-00805f9b34fb",
	"PANU_UUID":                    "00001115-0000-1000-8000-00805f9b34fb",
	"NAP_UUID":                     "00001116-0000-1000-8000-00805f9b34fb",
	"GN_UUID":                      "00001117-0000-1000-8000-00805f9b34fb",
	"BNEP_SVC_UUID":                "0000000f-0000-1000-8000-00805f9b34fb",
	"PNPID_UUID":                   "00002a50-0000-1000-8000-00805f9b34fb",
	"DEVICE_INFORMATION_UUID":      "0000180a-0000-1000-8000-00805f9b34fb",
	"GATT_UUID":                    "00001801-0000-1000-8000-00805f9b34fb",
	"IMMEDIATE_ALERT_UUID":         "00001802-0000-1000-8000-00805f9b34fb",
	"LINK_LOSS_UUID":               "00001803-0000-1000-8000-00805f9b34fb",
	"TX_POWER_UUID":                "00001804-0000-1000-8000-00805f9b34fb",
	"SAP_UUID":                     "0000112D-0000-1000-8000-00805f9b34fb",
	"HEART_RATE_UUID":              "0000180d-0000-1000-8000-00805f9b34fb",
	"HEART_RATE_MEASUREMENT_UUID":  "00002a37-0000-1000-8000-00805f9b34fb",
	"BODY_SENSOR_LOCATION_UUID":    "00002a38-0000-1000-8000-00805f9b34fb",
	"HEART_RATE_CONTROL_POINT_UUID":"00002a39-0000-1000-8000-00805f9b34fb",
	"HEALTH_THERMOMETER_UUID":      "00001809-0000-1000-8000-00805f9b34fb",
	"TEMPERATURE_MEASUREMENT_UUID": "00002a1c-0000-1000-8000-00805f9b34fb",
	"TEMPERATURE_TYPE_UUID":        "00002a1d-0000-1000-8000-00805f9b34fb",
	"INTERMEDIATE_TEMPERATURE_UUID":"00002a1e-0000-1000-8000-00805f9b34fb",
	"MEASUREMENT_INTERVAL_UUID":    "00002a21-0000-1000-8000-00805f9b34fb",
	"CYCLING_SC_UUID":              "00001816-0000-1000-8000-00805f9b34fb",
	"CSC_MEASUREMENT_UUID":         "00002a5b-0000-1000-8000-00805f9b34fb",
	"CSC_FEATURE_UUID":             "00002a5c-0000-1000-8000-00805f9b34fb",
	"SENSOR_LOCATION_UUID":         "00002a5d-0000-1000-8000-00805f9b34fb",
	"SC_CONTROL_POINT_UUID":        "00002a55-0000-1000-8000-00805f9b34fb",
	"RFCOMM_UUID_STR":              "00000003-0000-1000-8000-00805f9b34fb",
	"HDP_UUID":                     "00001400-0000-1000-8000-00805f9b34fb",
	"HDP_SOURCE_UUID":              "00001401-0000-1000-8000-00805f9b34fb",
	"HDP_SINK_UUID":                "00001402-0000-1000-8000-00805f9b34fb",
	"HID_UUID":                     "00001124-0000-1000-8000-00805f9b34fb",
	"DUN_GW_UUID":                  "00001103-0000-1000-8000-00805f9b34fb",
	"GAP_UUID":                     "00001800-0000-1000-8000-00805f9b34fb",
	"PNP_UUID":                     "00001200-0000-1000-8000-00805f9b34fb",
	"SPP_UUID":                     "00001101-0000-1000-8000-00805f9b34fb",
	"OBEX_SYNC_UUID":               "00001104-0000-1000-8000-00805f9b34fb",
	"OBEX_OPP_UUID":                "00001105-0000-1000-8000-00805f9b34fb",
	"OBEX_FTP_UUID":                "00001106-0000-1000-8000-00805f9b34fb",
	"OBEX_PCE_UUID":                "0000112e-0000-1000-8000-00805f9b34fb",
	"OBEX_PSE_UUID":                "0000112f-0000-1000-8000-00805f9b34fb",
	"OBEX_PBAP_UUID":               "00001130-0000-1000-8000-00805f9b34fb",
	"OBEX_MAS_UUID":                "00001132-0000-1000-8000-00805f9b34fb",
	"OBEX_MNS_UUID":                "00001133-0000-1000-8000-00805f9b34fb",
	"OBEX_MAP_UUID":                "00001134-0000-1000-8000-00805f9b34fb",
}

func getUuidKey(value string) string {
	for k, v := range uuids {
		if v == value {
			return k
		}
	}
	return "UNKNOWN_SERVICE_UUID"
}

type Agent struct{}

func (a *Agent) Release() *dbus.Error {
	fmt.Println("Release")
	os.Exit(0)
	return nil
}

func (a *Agent) AuthorizeService(device string, uuid string) *dbus.Error {
	if uuid == uuids["ADVANCED_AUDIO_UUID"] || uuid == uuids["AVRCP_REMOTE_UUID"] || uuid == uuids["GENERIC_AUDIO_UUID"] {
		log.Printf("AuthorizeService (%s, %s)\n", device, uuid)
		return nil
	}
	log.Printf("Service rejected (%s, %s:%s)\n", device, uuid, getUuidKey(uuid))
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
/*	props := map[string]interface{}{
		"Name":                 "CustomA2DPSink",
		"Role":                 "server",
		"RequireAuthentication": true,
		"RequireAuthorization":  true,
	}
*/
/*	if err = obj.Call(busPropertyName, 0,
		adapterInterface,
		"sspmode",
		dbus.MakeVariant(false),
	).Store(); err != nil {
		return nil, fmt.Errorf("Failed to set SSP property: %w", err)
	}
*/
	if err = obj.Call(busPropertyName, 0,
		adapterInterface,
		"Class",
		dbus.MakeVariant(uint32(0x240428)),
	).Store(); err != nil {
		log.Printf("Failed to set Class property: %v", err)
	} else {
		log.Println("Class property set successfully!")
	}

	if err := obj.Call(busPropertyName, 0,
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
/*	if err = manager.Call("org.bluez.ProfileManager1.RegisterProfile", 0,
		dbus.ObjectPath("/custom/profile"),
		"0000110b-0000-1000-8000-00805f9b34fb",
		props,
	).Store(); err != nil {
		return nil, fmt.Errorf("Failed to register profile: %v", err)
	}
	log.Printf("profile registered")
*/	if err = manager.Call("org.bluez.AgentManager1.RegisterAgent", 0,
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
