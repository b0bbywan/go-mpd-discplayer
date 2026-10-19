package mounts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jochenvg/go-udev"
)

func TestIsRemovableNode(t *testing.T) {
	tests := []struct {
		devnode, mountPoint string
		want                bool
	}{
		{"/dev/sda1", "/media/USB", true},
		{"/dev/sdb", "/run/media/user/KEY", true},
		{"/dev/sda1", "/", false},
		{"/dev/sda2", "/home", false},
		{"/dev/sda3", "/var", false},
		{"/dev/sda4", "/var/lib/docker/overlay", false},
		{"/dev/sda1", "/boot/firmware", false},
		{"/dev/mmcblk0p2", "/media/USB", false},
		{"/dev/nvme0n1p1", "/mnt", false},
		{"tmpfs", "/run", false},
	}
	for _, tt := range tests {
		if got := isRemovableNode(tt.devnode, tt.mountPoint); got != tt.want {
			t.Errorf("isRemovableNode(%q, %q) = %v, want %v", tt.devnode, tt.mountPoint, got, tt.want)
		}
	}
}

func TestGenerateTarget(t *testing.T) {
	library := t.TempDir()
	config := NewMountConfig(library, ".udisks", "symlink")
	want := filepath.Join(library, ".udisks", "KEY")

	if got := generateTarget(config, "/media/user/KEY"); got != want {
		t.Errorf("generateTarget() = %q, want %q", got, want)
	}

	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	got := generateTarget(config, "/media/user/KEY")
	if !strings.HasPrefix(got, want+"-") || len(got) != len(want)+6 {
		t.Errorf("generateTarget() = %q, want %q with a 5 chars suffix", got, want)
	}
}

func TestFindRelPath(t *testing.T) {
	m := &MountManager{config: NewMountConfig("/var/lib/mpd/music", ".udisks", "mpd")}
	got, err := m.FindRelPath("/var/lib/mpd/music/.udisks/KEY")
	if err != nil || got != ".udisks/KEY" {
		t.Errorf("FindRelPath() = %q, %v, want %q", got, err, ".udisks/KEY")
	}
}

func TestCheckSymlink(t *testing.T) {
	dir := t.TempDir()
	mountpoint := filepath.Join(dir, "media", "KEY")
	other := filepath.Join(dir, "media", "OTHER")
	for _, d := range []string{mountpoint, other} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(mountpoint, link); err != nil {
		t.Fatal(err)
	}
	dead := filepath.Join(dir, "dead")
	if err := os.Symlink(filepath.Join(dir, "gone"), dead); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, mountpoint, path string
		wantErr                bool
	}{
		{"valid", mountpoint, link, false},
		{"trailing slash", mountpoint + "/", link, false},
		{"other mountpoint", other, link, true},
		{"not a symlink", mountpoint, mountpoint, true},
		{"dead symlink", filepath.Join(dir, "gone"), dead, true},
		{"missing", mountpoint, filepath.Join(dir, "missing"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSymlink(tt.mountpoint, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkSymlink() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFindMountPointWithRetry(t *testing.T) {
	m := &MountManager{mountPoints: newCache()}

	var device, mountPoint string
	if err := readMountsFile(func(d, mp string) {
		if device == "" && strings.HasPrefix(d, "/") {
			device, mountPoint = d, mp
		}
	}); err != nil || device == "" {
		t.Skip("no device in /proc/mounts")
	}
	got, err := m.findMountPointWithRetry(device, 50*time.Millisecond, 10*time.Millisecond)
	if err != nil || got != mountPoint {
		t.Errorf("findMountPointWithRetry(%q) = %q, %v, want %q", device, got, err, mountPoint)
	}
	if cached, err := m.mountPoints.GetCache(device); err != nil || cached != mountPoint {
		t.Errorf("cache = %q, %v, want %q", cached, err, mountPoint)
	}

	start := time.Now()
	if _, err := m.findMountPointWithRetry("/dev/does-not-exist", 50*time.Millisecond, 10*time.Millisecond); err == nil {
		t.Error("unknown device should time out")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("timeout took %s", elapsed)
	}
}

// anyBlockDevice returns a block device known to udev, to drive the code
// paths that take a *udev.Device.
func anyBlockDevice(t *testing.T) *udev.Device {
	t.Helper()
	entries, err := os.ReadDir("/sys/class/block")
	if err != nil || len(entries) == 0 {
		t.Skip("no block device")
	}
	u := udev.Udev{}
	dev := u.NewDeviceFromSyspath(filepath.Join("/sys/class/block", entries[0].Name()))
	if dev == nil || dev.Devnode() == "" {
		t.Skip("no block device node")
	}
	return dev
}

func TestSymlinkFinderReusesValidSymlink(t *testing.T) {
	dev := anyBlockDevice(t)
	library := t.TempDir()
	mountpoint := t.TempDir()
	s := newSymlinkFinder(library, ".udisks")

	target := filepath.Join(library, ".udisks", "KEY")
	first, err := s.validate(dev, mountpoint, target)
	if err != nil || first != target {
		t.Fatalf("first validate() = %q, %v, want %q", first, err, target)
	}

	// A second mount of the same device must not create another symlink
	again, err := s.validate(dev, mountpoint, target+"-other")
	if err != nil || again != target {
		t.Errorf("second validate() = %q, %v, want %q", again, err, target)
	}
	if _, err := os.Lstat(target + "-other"); !os.IsNotExist(err) {
		t.Errorf("a second symlink was created: %v", err)
	}

	cleared, err := s.clear(dev, mountpoint)
	if err != nil || cleared != target {
		t.Errorf("clear() = %q, %v, want %q", cleared, err, target)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Errorf("symlink still there after clear: %v", err)
	}
}

func TestPresentIdentifiers(t *testing.T) {
	if _, err := os.Stat("/sys/class/block"); err != nil {
		t.Skip("no sysfs")
	}
	ids, err := presentIdentifiers()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if id == "" {
			t.Error("empty identifier")
		}
	}
}
