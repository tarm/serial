package serial

import (
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

type serialPort struct {
	f  *os.File
	fd syscall.Handle
	rl sync.Mutex
	wl sync.Mutex
	ro *syscall.Overlapped
	wo *syscall.Overlapped
}

type structDCB struct {
	DCBlength, BaudRate                            uint32
	flags                                          [4]byte
	wReserved, XonLim, XoffLim                     uint16
	ByteSize, Parity, StopBits                     byte
	XonChar, XoffChar, ErrorChar, EofChar, EvtChar byte
	wReserved1                                     uint16
}

type structTimeouts struct {
	ReadIntervalTimeout         uint32
	ReadTotalTimeoutMultiplier  uint32
	ReadTotalTimeoutConstant    uint32
	WriteTotalTimeoutMultiplier uint32
	WriteTotalTimeoutConstant   uint32
}

func OpenPort(name string, baud int) (rwc io.ReadWriteCloser, err os.Error) {
	if len(name) > 0 && name[0] != '\\' {
		name = "\\\\.\\" + name
	}

	const FILE_FLAGS_OVERLAPPED = 0x40000000
	h, e := syscall.CreateFile(syscall.StringToUTF16Ptr(name),
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL|FILE_FLAGS_OVERLAPPED,
		0)
	if e != 0 {
		err = &os.PathError{"open", name, os.Errno(e)}
		return
	}
	f := os.NewFile(h, name)
	defer func() {
		if err != nil {
			f.Close()
		}
	}()

	if err = setCommState(h, baud); err != nil {
		return
	}
	if err = setupComm(h, 64, 64); err != nil {
		return
	}
	if err = setCommTimeouts(h); err != nil {
		return
	}
	if err = setCommMask(h); err != nil {
		return
	}

	ro, err := newOverlapped()
	if err != nil {
		return
	}
	wo, err := newOverlapped()
	if err != nil {
		return
	}
	port := new(serialPort)
	port.f = f
	port.fd = h
	port.ro = ro
	port.wo = wo

	return port, nil
}

func (p *serialPort) Close() os.Error {
	return p.f.Close()
}

func (p *serialPort) Write(buf []byte) (int, os.Error) {
	p.wl.Lock()
	defer p.wl.Unlock()

	if err := resetEvent(p.wo.HEvent); err != nil {
		return 0, err
	}
	var n uint32
	e := syscall.WriteFile(p.fd, buf, &n, p.wo)
	if e != 0 && e != syscall.ERROR_IO_PENDING {
		return int(n), errno(uintptr(e))
	}
	return getOverlappedResult(p.fd, p.wo)
}

func (p *serialPort) Read(buf []byte) (int, os.Error) {
	if p == nil || p.f == nil {
		return 0, fmt.Errorf("Invalid port on read %v %v", p, p.f)
	}

	p.rl.Lock()
	defer p.rl.Unlock()

	if err := resetEvent(p.ro.HEvent); err != nil {
		return 0, err
	}
	var done uint32
	e := syscall.ReadFile(p.fd, buf, &done, p.ro)
	if e != 0 && e != syscall.ERROR_IO_PENDING {
		return int(done), errno(uintptr(e))
	}
	return getOverlappedResult(p.fd, p.ro)
}

var (
	nSetCommState,
	nSetCommTimeouts,
	nSetCommMask,
	nSetupComm,
	nWaitCommEvent,
	nGetOverlappedResult,
	nCreateEvent,
	nResetEvent syscall.Handle
)

func init() {
	k32, err := syscall.LoadLibrary("kernel32.dll")
	if err != 0 {
		panic("LoadLibrary " + syscall.Errstr(err))
	}
	defer syscall.FreeLibrary(k32)

	nSetCommState = getProcAddr(k32, "SetCommState")
	nSetCommTimeouts = getProcAddr(k32, "SetCommTimeouts")
	nSetCommMask = getProcAddr(k32, "SetCommMask")
	nSetupComm = getProcAddr(k32, "SetupComm")
	nWaitCommEvent = getProcAddr(k32, "WaitCommEvent")
	nGetOverlappedResult = getProcAddr(k32, "GetOverlappedResult")
	nCreateEvent = getProcAddr(k32, "CreateEventW")
	nResetEvent = getProcAddr(k32, "ResetEvent")
}

func getProcAddr(lib syscall.Handle, name string) syscall.Handle {
	addr, err := syscall.GetProcAddress(lib, name)
	if err != 0 {
		panic(name + " " + syscall.Errstr(err))
	}
	return addr
}

func errno(e uintptr) os.Error {
	if e != 0 {
		return os.Errno(e)
	}
	return os.Errno(syscall.EINVAL)
}

func setCommState(h syscall.Handle, baud int) os.Error {
	var params structDCB
	params.DCBlength = uint32(unsafe.Sizeof(params))

	params.flags[0] = 0x01  // fBinary
	params.flags[0] |= 0x10 // Assert DSR

	params.BaudRate = uint32(baud)
	params.ByteSize = 8

	r, _, e := syscall.Syscall(uintptr(nSetCommState), 2, uintptr(h), uintptr(unsafe.Pointer(&params)), 0)
	if r == 0 {
		return errno(e)
	}
	return nil
}

func setCommTimeouts(h syscall.Handle) os.Error {
	var timeouts structTimeouts
	timeouts.ReadIntervalTimeout = 1<<32 - 1
	timeouts.ReadTotalTimeoutConstant = 0
	r, _, e := syscall.Syscall(uintptr(nSetCommTimeouts), 2, uintptr(h), uintptr(unsafe.Pointer(&timeouts)), 0)
	if r == 0 {
		return errno(e)
	}
	return nil
}

func setupComm(h syscall.Handle, in, out int) os.Error {
	r, _, e := syscall.Syscall(uintptr(nSetupComm), 3, uintptr(h), uintptr(in), uintptr(out))
	if r == 0 {
		return errno(e)
	}
	return nil
}

func setCommMask(h syscall.Handle) os.Error {
	const EV_RXCHAR = 0x0001
	r, _, e := syscall.Syscall(uintptr(nSetCommMask), 2, uintptr(h), EV_RXCHAR, 0)
	if r == 0 {
		return errno(e)
	}
	return nil
}

func resetEvent(h syscall.Handle) os.Error {
	r, _, e := syscall.Syscall(uintptr(nResetEvent), 1, uintptr(h), 0, 0)
	if r == 0 {
		return errno(e)
	}
	return nil
}

func newOverlapped() (*syscall.Overlapped, os.Error) {
	var overlapped syscall.Overlapped
	r, _, e := syscall.Syscall6(uintptr(nCreateEvent), 4, 0, 1, 0, 0, 0, 0)
	if r == 0 {
		return nil, errno(e)
	}
	overlapped.HEvent = syscall.Handle(r)
	return &overlapped, nil
}

func getOverlappedResult(h syscall.Handle, overlapped *syscall.Overlapped) (int, os.Error) {
	var n int
	r, _, e := syscall.Syscall6(uintptr(nGetOverlappedResult), 4,
		uintptr(h),
		uintptr(unsafe.Pointer(overlapped)),
		uintptr(unsafe.Pointer(&n)), 1, 0, 0)
	if r == 0 {
		return n, errno(e)
	}

	return n, nil
}

func waitCommEvent(h syscall.Handle, events *uint32, overlapped *syscall.Overlapped) os.Error {
	r, _, e := syscall.Syscall(uintptr(nWaitCommEvent), 3, uintptr(h), uintptr(unsafe.Pointer(events)), uintptr(unsafe.Pointer(overlapped)))
	if r == 0 {
		return errno(e)
	}

	return nil
}
