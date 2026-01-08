package hwcontrol

import (
	"context"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
)

// Handler définit un handler stateless capable de gérer un type de device.
type Handler interface {
	// Handles retourne true si le handler peut gérer ce type de device
	Handles(kind detect.DeviceKind) bool
	OnAdd(ctx context.Context, dev detect.Device) error
	OnRemove(ctx context.Context, dev detect.Device) error
}

// Exemple de handler de base pour adapter ton EventHandler existant
type BasicHandler struct {
	kind detect.DeviceKind
	// processFunc est défini par l’utilisateur
	processAdd    func(context.Context, detect.Device) error
	processRemove func(context.Context, detect.Device) error
}

func (h *BasicHandler) Handles(kind detect.DeviceKind) bool {
	return h.kind == kind
}

func (h *BasicHandler) OnAdd(ctx context.Context, dev detect.Device) error {
	if h.processAdd != nil {
		return h.processAdd(ctx, dev)
	}
	return nil
}

func (h *BasicHandler) OnRemove(ctx context.Context, dev detect.Device) error {
	if h.processRemove != nil {
		return h.processRemove(ctx, dev)
	}
	return nil
}

// Constructeur
func NewBasicHandler(kind detect.DeviceKind,
	processAdd, processRemove func(context.Context, detect.Device) error) *BasicHandler {

	return &BasicHandler{
		kind:          kind,
		processAdd:    processAdd,
		processRemove: processRemove,
	}
}

