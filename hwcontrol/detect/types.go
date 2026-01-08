package detect

import (
	"github.com/jochenvg/go-udev"
)

type DeviceKind string

type EventType string

const (
	EventAdd    = "add"
	EventChange = "change"
	EventRemove = "remove"

	InvalidEvent  EventType = "invalid"
	DeviceAdded   EventType = EventAdd
	DeviceRemoved EventType = EventRemove

	DeviceDisc DeviceKind = "disc"
	DeviceUSB  DeviceKind = "usb"
)

type Device interface {
	Path() string
	Kind() DeviceKind
	DetectEvent() EventType
	Udev() *udev.Device
}

type DeviceEvent struct {
	Type   EventType
	Device Device
}
