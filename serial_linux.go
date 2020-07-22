// +build linux

package serial

import (
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

func openPort(name string, baud int, databits byte, parity Parity, stopbits StopBits, readTimeout time.Duration) (p *Port, err error) {
	f, err := os.OpenFile(name, unix.O_RDWR|unix.O_NOCTTY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil && f != nil {
			f.Close()
		}
	}()

	// Base settings
	var cflagToUse uint32 = unix.CREAD | unix.CLOCAL | unix.BOTHER | unix.HUPCL
	switch databits {
	case 5:
		cflagToUse |= unix.CS5
	case 6:
		cflagToUse |= unix.CS6
	case 7:
		cflagToUse |= unix.CS7
	case 8:
		cflagToUse |= unix.CS8
	default:
		return nil, ErrBadSize
	}
	// Stop bits settings
	switch stopbits {
	case Stop1:
		// default is 1 stop bit
	case Stop2:
		cflagToUse |= unix.CSTOPB
	default:
		// Don't know how to set 1.5
		return nil, ErrBadStopBits
	}
	// Parity settings
	switch parity {
	case ParityNone:
		// default is no parity
	case ParityOdd:
		cflagToUse |= unix.PARENB
		cflagToUse |= unix.PARODD
	case ParityEven:
		cflagToUse |= unix.PARENB
	default:
		return nil, ErrBadParity
	}
	fd := f.Fd()
	vmin, vtime := posixTimeoutValues(readTimeout)
	speed := uint32(baud)
	t := unix.Termios{
		Iflag:  unix.IGNPAR,
		Cflag:  cflagToUse,
		Ispeed: speed,
		Ospeed: speed,
	}
	t.Cc[unix.VMIN] = vmin
	t.Cc[unix.VTIME] = vtime

	/*
		Raise DTR (open modem) on open. DTR will be set low
		again when the connection is closed due to HUPCL.
		This should guarantee a correct "restart" action
		on some devices.
	*/
	dtrFlag := unix.TIOCM_DTR
	unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.TIOCMBIS), // set/raise DTR pin
		uintptr(unsafe.Pointer(&dtrFlag)))

	if _, _, errno := unix.Syscall6(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.TCSETS2),
		uintptr(unsafe.Pointer(&t)),
		0,
		0,
		0,
	); errno != 0 {
		return nil, errno
	}

	if err = unix.SetNonblock(int(fd), false); err != nil {
		return
	}

	return &Port{f: f}, nil
}

type Port struct {
	// We intentionly do not use an "embedded" struct so that we
	// don't export File
	f *os.File
}

func (p *Port) Read(b []byte) (n int, err error) {
	return p.f.Read(b)
}

func (p *Port) Write(b []byte) (n int, err error) {
	return p.f.Write(b)
}

// Discards data written to the port but not transmitted,
// or data received but not read
func (p *Port) Flush() error {
	const TCFLSH = 0x540B
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(p.f.Fd()),
		uintptr(TCFLSH),
		uintptr(unix.TCIOFLUSH),
	)

	if errno == 0 {
		return nil
	}
	return errno
}

func (p *Port) Close() (err error) {
	return p.f.Close()
}
