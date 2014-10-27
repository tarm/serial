// +build darwin

package serial

import (
	"io"
	"os"
	"syscall"
	"unsafe"
)

const (
	IOSSIOSPEED = 0x80045402 // _IOW('T', 2, speed_t)
)

func openPort(name string, baud int) (rwc io.ReadWriteCloser, err error) {
	f, err := os.OpenFile(name, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil && f != nil {
			f.Close()
		}
	}()

	fd := f.Fd()

	t := &syscall.Termios{}

	// Fetch old flags
	if _, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TIOCGETA),
		uintptr(unsafe.Pointer(t)),
		0,
		0,
		0,
	); errno != 0 {
		return nil, errno
	}

	// These are mostly torn out of the pyserial implementation
	t.Cflag &^= (syscall.CSIZE | syscall.CSTOPB | syscall.PARENB | syscall.PARODD | syscall.IXON | syscall.IXOFF)
	t.Cflag |= (syscall.CLOCAL | syscall.CREAD | syscall.CS8)
	t.Lflag &^= (syscall.ICANON | syscall.ECHO | syscall.ECHOE | syscall.ECHOK | syscall.ECHONL | syscall.ISIG | syscall.IEXTEN)
	t.Oflag &^= (syscall.OPOST)
	t.Iflag &^= (syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IGNBRK)

	t.Cc[syscall.VMIN] = 1
	t.Cc[syscall.VTIME] = 30

	// Apply flags
	if _, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TIOCSETA),
		uintptr(unsafe.Pointer(t)),
		0,
		0,
		0,
	); errno != 0 {
		return nil, errno
	}

	// Set baudrate
	if _, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		IOSSIOSPEED,
		uintptr(unsafe.Pointer(&baud)),
		0,
		0,
		0,
	); errno != 0 {
		return nil, errno
	}

	if err = syscall.SetNonblock(int(fd), false); err != nil {
		return
	}

	return f, nil
}
