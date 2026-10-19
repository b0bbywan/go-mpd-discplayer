package hwcontrol

import (
	"fmt"
	"log"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	CDROMEJECT = 0x5309 // ioctl command for ejecting the disc
	SG_IO      = 0x2285 // ioctl command for sending a SCSI command

	sgInterfaceID          = 'S'
	sgDxferNone            = -1
	sgTimeoutMs            = 10000
	scsiAllowMediumRemoval = 0x1e
	scsiStartStopUnit      = 0x1b
)

// sgIoHdr mirrors struct sg_io_hdr from <scsi/sg.h>.
type sgIoHdr struct {
	InterfaceID    int32
	DxferDirection int32
	CmdLen         uint8
	MxSbLen        uint8
	IovecCount     uint16
	DxferLen       uint32
	Dxferp         uintptr
	Cmdp           uintptr
	Sbp            uintptr
	Timeout        uint32
	Flags          uint32
	PackID         int32
	UsrPtr         uintptr
	Status         uint8
	MaskedStatus   uint8
	MsgStatus      uint8
	SbLenWr        uint8
	HostStatus     uint16
	DriverStatus   uint16
	Resid          int32
	Duration       uint32
	Info           uint32
}

// EjectDisc ejects the disc like eject(1): the CD-ROM ioctl first, then raw
// SCSI commands, which still work while another process (MPD, gvfs) holds the
// drive open or the tray is locked.
func EjectDisc(device string) error {
	return withDrive(device, openDrive, func(fd int) error {
		err := cdromEject(fd)
		if err == nil {
			return nil
		}
		log.Printf("info: CD-ROM eject failed, trying SCSI commands: %v", err)
		return scsiEject(fd)
	})
}

func cdromEject(fd int) error {
	ret, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), CDROMEJECT, 0)
	if errno != 0 {
		return errno
	}
	// The kernel passes a failed tray unlock through as a positive result
	if ret != 0 {
		return fmt.Errorf("drive returned %#x", ret)
	}
	return nil
}

func scsiEject(fd int) error {
	// Refused while the tray is locked, START STOP UNIT still ejects
	if err := scsiCommand(fd, []byte{scsiAllowMediumRemoval, 0, 0, 0, 0, 0}); err != nil {
		log.Printf("info: allow medium removal: %v", err)
	}
	if err := scsiCommand(fd, []byte{scsiStartStopUnit, 0, 0, 0, 1, 0}); err != nil {
		log.Printf("info: start unit: %v", err)
	}
	if err := scsiCommand(fd, []byte{scsiStartStopUnit, 0, 0, 0, 2, 0}); err != nil {
		return fmt.Errorf("failed to eject: %w", err)
	}
	return nil
}

func scsiCommand(fd int, cmd []byte) error {
	sense := make([]byte, 32)
	hdr := sgIoHdr{
		InterfaceID:    sgInterfaceID,
		DxferDirection: sgDxferNone,
		CmdLen:         uint8(len(cmd)),
		MxSbLen:        uint8(len(sense)),
		Cmdp:           uintptr(unsafe.Pointer(&cmd[0])),
		Sbp:            uintptr(unsafe.Pointer(&sense[0])),
		Timeout:        sgTimeoutMs,
	}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), SG_IO, uintptr(unsafe.Pointer(&hdr)))
	runtime.KeepAlive(cmd)
	runtime.KeepAlive(sense)
	if errno != 0 {
		return errno
	}
	if hdr.Status != 0 || hdr.HostStatus != 0 || hdr.DriverStatus != 0 {
		return fmt.Errorf("SCSI command %#x failed: status %#x, host %#x, driver %#x, sense % x",
			cmd[0], hdr.Status, hdr.HostStatus, hdr.DriverStatus, sense[:hdr.SbLenWr])
	}
	return nil
}
