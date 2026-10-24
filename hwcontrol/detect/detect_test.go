package detect

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestValidFsType(t *testing.T) {
	tests := []struct {
		partEntryType, fsType string
		want                  bool
	}{
		{"0xc", "vfat", true},
		{"0x83", "ext4", true},
		{"0x0", "vfat", false},  // empty partition
		{"0xef", "vfat", false}, // efi system partition
		{"0x82", "swap", false}, // linux swap
		{"", "exfat", true},     // no partition table
		{"", "", false},
	}
	for _, tt := range tests {
		if got := validFsType(tt.partEntryType, tt.fsType); got != tt.want {
			t.Errorf("validFsType(%q, %q) = %v, want %v", tt.partEntryType, tt.fsType, got, tt.want)
		}
	}
}

func skipWithoutSysfs(t *testing.T) {
	t.Helper()
	if _, err := os.Stat("/sys/class/block"); err != nil {
		t.Skip("no sysfs")
	}
}

func TestColdplugOnlyEmitsUSB(t *testing.T) {
	skipWithoutSysfs(t)
	out := make(chan DeviceEvent, 64)
	(&UdevDetector{}).coldplug(context.Background(), out)
	close(out)
	for ev := range out {
		if ev.Type != DevicePresent || ev.Device.Kind() != DeviceUSB {
			t.Errorf("unexpected event %s %s for %s", ev.Type, ev.Device.Kind(), ev.Device.Path())
		}
	}
}

func TestColdplugStopsOnCancel(t *testing.T) {
	skipWithoutSysfs(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		// Unbuffered and never read: only the cancelled context can unblock it
		(&UdevDetector{}).coldplug(ctx, make(chan DeviceEvent))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("coldplug blocked despite the cancelled context")
	}
}
