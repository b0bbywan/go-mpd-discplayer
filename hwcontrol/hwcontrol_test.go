package hwcontrol

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"unsafe"
)

// sgIoHdr is handed to the kernel as is, so it must match struct sg_io_hdr
// on every target, 32-bit ARM included.
func TestSgIoHdrSize(t *testing.T) {
	want := uintptr(88)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		want = 64
	}
	if got := unsafe.Sizeof(sgIoHdr{}); got != want {
		t.Errorf("sizeof(sgIoHdr) = %d, want %d", got, want)
	}
}

// notADrive returns a regular file, on which every drive ioctl fails.
func notADrive(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sr0")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEjectDisc(t *testing.T) {
	if err := EjectDisc(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("ejecting a missing device should fail")
	}
	// Both the CD-ROM ioctl and the SCSI fallback fail
	if err := EjectDisc(notADrive(t)); err == nil {
		t.Error("ejecting a regular file should fail")
	}
}

func TestOpenDriveFallsBackToReadOnly(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can open read-only files for writing")
	}
	path := notADrive(t)
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	file, err := openDrive(path)
	if err != nil {
		t.Fatalf("openDrive() error = %v", err)
	}
	_ = file.Close()
}

func TestSetDiscSpeed(t *testing.T) {
	if err := SetDiscSpeed(filepath.Join(t.TempDir(), "missing"), 12); err == nil {
		t.Error("setting the speed of a missing device should fail")
	}
	if err := SetDiscSpeed(notADrive(t), 12); err == nil {
		t.Error("setting the speed of a regular file should fail")
	}
}

func TestWithDrive(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	called := false
	if err := withDrive(missing, openReadOnly, func(int) error { called = true; return nil }); err == nil || called {
		t.Errorf("withDrive() on a missing device = %v, fn called = %v", err, called)
	}

	errFn := errors.New("ioctl failed")
	var gotFd int
	err := withDrive(notADrive(t), openReadOnly, func(fd int) error {
		gotFd = fd
		return errFn
	})
	if !errors.Is(err, errFn) {
		t.Errorf("withDrive() error = %v, want the error of fn", err)
	}
	if gotFd <= 0 {
		t.Errorf("fn got fd %d", gotFd)
	}
}
