package detect

import (
	"github.com/jochenvg/go-udev"
)

type DeviceKind int

const (
	DeviceDisc DeviceKind = iota
	DeviceUSB
)

const (
	EventAdd    = "add"
	EventChange = "change"
	EventRemove = "remove"
)

type Device interface {
	Path() string
	Kind() DeviceKind
	DetectEvent() EventType
	Udev() *udev.Device
}

type EventType int

const (
	InvalidEvent EventType = iota
	DeviceAdded
	DeviceRemoved
)

type DeviceEvent struct {
	Type   EventType
	Device Device
}
