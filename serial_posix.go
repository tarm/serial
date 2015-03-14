// +build !windows,cgo

package serial

// #include <termios.h>
// #include <unistd.h>
import "C"

// TODO: Maybe change to using syscall package + ioctl instead of cgo

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
	//"unsafe"
)

func openPort(name string, baud int, readTimeout time.Duration) (p *Port, err error) {
	f, err := os.OpenFile(name, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		return
	}

	fd := C.int(f.Fd())
	if C.isatty(fd) != 1 {
		f.Close()
		return nil, errors.New("File is not a tty")
	}

	var st C.struct_termios
	_, err = C.tcgetattr(fd, &st)
	if err != nil {
		f.Close()
		return nil, err
	}
	var speed C.speed_t
	switch baud {
	case 115200:
		speed = C.B115200
	case 57600:
		speed = C.B57600
	case 38400:
		speed = C.B38400
	case 19200:
		speed = C.B19200
	case 9600:
		speed = C.B9600
	case 4800:
		speed = C.B4800
	case 2400:
		speed = C.B2400
	default:
		f.Close()
		return nil, fmt.Errorf("Unknown baud rate %v", baud)
	}

	_, err = C.cfsetispeed(&st, speed)
	if err != nil {
		f.Close()
		return nil, err
	}
	_, err = C.cfsetospeed(&st, speed)
	if err != nil {
		f.Close()
		return nil, err
	}

	// Turn off break interrupts, CR->NL, Parity checks, strip, and IXON
	st.c_iflag &= ^C.tcflag_t(C.BRKINT | C.ICRNL | C.INPCK | C.ISTRIP | C.IXOFF | C.IXON | C.PARMRK)

	// Select local mode, turn off parity, set to 8 bits
	st.c_cflag &= ^C.tcflag_t(C.CSIZE | C.PARENB)
	st.c_cflag |= (C.CLOCAL | C.CREAD | C.CS8)

	// Select raw mode
	st.c_lflag &= ^C.tcflag_t(C.ICANON | C.ECHO | C.ECHOE | C.ISIG)
	st.c_oflag &= ^C.tcflag_t(C.OPOST)

	// set blocking / non-blocking read
	/*
	*	http://man7.org/linux/man-pages/man3/termios.3.html
	* - Supports blocking read and read with timeout operations
	 */
	vmin, vtime := posixTimeoutValues(readTimeout)
	st.c_cc[C.VMIN] = C.cc_t(vmin)
	st.c_cc[C.VTIME] = C.cc_t(vtime)

	_, err = C.tcsetattr(fd, C.TCSANOW, &st)
	if err != nil {
		f.Close()
		return nil, err
	}

	//fmt.Println("Tweaking", name)
	r1, _, e := syscall.Syscall(syscall.SYS_FCNTL,
		uintptr(f.Fd()),
		uintptr(syscall.F_SETFL),
		uintptr(0))
	if e != 0 || r1 != 0 {
		s := fmt.Sprint("Clearing NONBLOCK syscall error:", e, r1)
		f.Close()
		return nil, errors.New(s)
	}

	/*
				r1, _, e = syscall.Syscall(syscall.SYS_IOCTL,
			                uintptr(f.Fd()),
			                uintptr(0x80045402), // IOSSIOSPEED
			                uintptr(unsafe.Pointer(&baud)));
			        if e != 0 || r1 != 0 {
			                s := fmt.Sprint("Baudrate syscall error:", e, r1)
					f.Close()
		                        return nil, os.NewError(s)
				}
	*/

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

func (p *Port) Flush() error {
	_, err := C.tcflush(C.int(p.f.Fd()), C.TCIOFLUSH)
	return err
}

func (p *Port) Close() (err error) {
	return p.f.Close()
}
