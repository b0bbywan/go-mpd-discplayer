package blockdev

import (
	"os"
	"testing"
)

func TestList(t *testing.T) {
	entries, err := os.ReadDir("/sys/class/block")
	if err != nil || len(entries) == 0 {
		t.Skip("no block device")
	}
	devices, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) == 0 {
		t.Fatalf("no device listed, /sys/class/block has %d", len(entries))
	}
	for _, dev := range devices {
		if dev.Subsystem() != "block" {
			t.Errorf("%s is in subsystem %q", dev.Syspath(), dev.Subsystem())
		}
	}
}
