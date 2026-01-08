package hwcontrol

import (
	"context"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
	"log"
)

// Handler définit un handler stateless capable de gérer un type de device.
type Handler interface {
	// Handles retourne true si le handler peut gérer ce type de device
	Handles(kind detect.DeviceKind) bool
	Name() string
	OnAdd(ctx context.Context, dev detect.Device) error
	OnRemove(ctx context.Context, dev detect.Device) error
}

// Exemple de handler de base pour adapter ton EventHandler existant
type BasicHandler struct {
	name string
	kind detect.DeviceKind
	// processFunc est défini par l’utilisateur
	processAdd    func(context.Context, detect.Device) error
	processRemove func(context.Context, detect.Device) error
}

func (h *BasicHandler) Name() string {
	return h.name
}

func (h *BasicHandler) Handles(kind detect.DeviceKind) bool {
	return h.kind == kind
}

func (h *BasicHandler) OnAdd(ctx context.Context, dev detect.Device) error {
	if h.processAdd != nil {
		log.Printf("[%s] OnAdd %s", h.name, dev.Path())
		return h.processAdd(ctx, dev)
	}
	return nil
}

func (h *BasicHandler) OnRemove(ctx context.Context, dev detect.Device) error {
	if h.processRemove != nil {
		log.Printf("[%s] OnRemove %s", h.name, dev.Path())
		return h.processRemove(ctx, dev)
	}
	return nil
}

// Constructeur
func NewBasicHandler(name string, kind detect.DeviceKind,
	processAdd, processRemove func(context.Context, detect.Device) error) *BasicHandler {

	return &BasicHandler{
		name:          name,
		kind:          kind,
		processAdd:    processAdd,
		processRemove: processRemove,
	}
}

