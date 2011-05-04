package serial

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

type serialPort struct {
	f  *os.File
	fd int32
	rl sync.Mutex
	wl sync.Mutex
	ro *syscall.Overlapped
	wo *syscall.Overlapped
	eo *syscall.Overlapped
}

const EV_RXCHAR = 0x0001

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

var nSetCommState uint32
var nSetCommTimeouts uint32
var nSetCommMask uint32
var nSetupComm uint32
var nWaitCommEvent uint32
var nGetOverlappedResult uint32
var nCreateEvent uint32
var nResetEvent uint32

func here() {
	_, file, line, _ := runtime.Caller(1)
	fmt.Println("Got to", file, ":", line)
}

func getProcAddr(lib uint32, name string) uint32 {
	addr, err := syscall.GetProcAddress(lib, name)
	if err != 0 {
		panic(name + " " + syscall.Errstr(err))
	}
	return addr
}

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

func setCommState(h int32, baud int) os.Error {
	var params structDCB
	params.DCBlength = uint32(unsafe.Sizeof(params))

	params.flags[0] = 0x01  // fBinary
	params.flags[0] |= 0x10 // Assert DSR

	params.BaudRate = uint32(baud)
	params.ByteSize = 8

	//fmt.Printf("%#v\n", params)
	r, _, e := syscall.Syscall(uintptr(nSetCommState), 2, uintptr(h), uintptr(unsafe.Pointer(&params)), 0)
	if r == 0 {
		return os.Errno(e)
	}
	return nil
}

func setCommTimeouts(h int32) os.Error {
	var timeouts structTimeouts
	timeouts.ReadIntervalTimeout = 1<<32 - 1
	timeouts.ReadTotalTimeoutConstant = 0
	r, _, e := syscall.Syscall(uintptr(nSetCommTimeouts), 2, uintptr(h), uintptr(unsafe.Pointer(&timeouts)), 0)
	if r == 0 {
		return os.Errno(e)
	}
	return nil
}

func setupComm(h int32, in, out int) os.Error {
	r, _, e := syscall.Syscall(uintptr(nSetupComm), 3, uintptr(h), uintptr(in), uintptr(out))
	if r == 0 {
		return os.Errno(e)
	}
	return nil
}

func setCommMask(h int32) os.Error {
	r, _, e := syscall.Syscall(uintptr(nSetCommMask), 2, uintptr(h), EV_RXCHAR, 0)
	if r == 0 {
		return os.Errno(e)
	}
	return nil
}

func resetEvent(h int32) os.Error {
	r, _, e := syscall.Syscall(uintptr(nResetEvent), 1, uintptr(h), 0, 0)
	if r == 0 {
		if e !=0 {
			return os.Errno(e)
		} else {
			return os.Errno(syscall.EINVAL)
		}
	}
	return nil
}

const FILE_FLAGS_OVERLAPPED = 0x40000000

func OpenPort(name string, baud int) (io.ReadWriteCloser, os.Error) {
	if len(name) > 0 && name[0] != '\\' {
		name = "\\\\.\\" + name
	}

	h, e := syscall.CreateFile(syscall.StringToUTF16Ptr(name),
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL|FILE_FLAGS_OVERLAPPED,
		0)

	if e != 0 {
		return nil, &os.PathError{"open", name, os.Errno(e)}
	}
	f := os.NewFile(int(h), name)

	if err := setCommState(h, baud); err != nil {
		f.Close()
		return nil, err
	}

	if err := setupComm(h, 64, 64); err != nil {
		f.Close()
		return nil, err
	}

	if err := setCommTimeouts(h); err != nil {
		f.Close()
		return nil, err
	}
	if err := setCommMask(h); err != nil {
		f.Close()
		return nil, err
	}

	ro, err := newOverlapped()
	if err != nil {
		f.Close()
		return nil, err
	}
	wo, err := newOverlapped()
	if err != nil {
		f.Close()
		return nil, err
	}
	eo, err := newOverlapped()
	if err != nil {
		f.Close()
		return nil, err
	}
	port := new(serialPort)
	port.f  = f
	port.fd = h
	port.ro = ro
	port.wo = wo
	port.eo = eo

	return port, nil
}

func (p *serialPort) Close() os.Error {
	return p.f.Close()
}

func newOverlapped() (*syscall.Overlapped, os.Error) {
	var overlapped syscall.Overlapped
	r, _, e := syscall.Syscall6(uintptr(nCreateEvent), 4, 0, 1, 0, 0, 0, 0)
	if r == 0 {
		if e !=0 {
			return nil, os.Errno(e)
		} else {
			return nil, os.Errno(syscall.EINVAL)
		}
	}
	overlapped.HEvent = int32(r)
	return &overlapped, nil
}

func getOverlappedResult(h int32, overlapped *syscall.Overlapped) (int, os.Error) {
	var n int
	r, _, e := syscall.Syscall6(uintptr(nGetOverlappedResult), 4,
		uintptr(h),
		uintptr(unsafe.Pointer(overlapped)),
		uintptr(unsafe.Pointer(&n)), 1, 0, 0)
	if r == 0 {
		if e !=0 {
			return n, os.Errno(e)
		} else {
			return n, os.Errno(syscall.EINVAL)
		}
	}

	return n, nil
}

func (p *serialPort) Write(buf []byte) (int, os.Error) {
	p.wl.Lock()
	defer p.wl.Unlock()

	err := resetEvent(p.wo.HEvent)
	if err != nil {
		return 0, err
	}

	var n uint32
	e := syscall.WriteFile(p.fd, buf, &n, p.wo)
	if e != 0 && e != syscall.ERROR_IO_PENDING {
		return int(n), os.Errno(e)
	}
	return getOverlappedResult(p.fd, p.wo)
}

func waitCommEvent(h int32, overlapped *syscall.Overlapped) os.Error {
	var events uint32
	r, _, e := syscall.Syscall(uintptr(nWaitCommEvent), 3, uintptr(h), uintptr(unsafe.Pointer(&events)), uintptr(unsafe.Pointer(overlapped)))
	if r == 0 {
		if e !=0 {
			return os.Errno(e)
		} else {
			return os.NewSyscallError("WaitCommEvent", int(e))
		}
	}
	if events != EV_RXCHAR {
		return os.NewError("Bad event in wait comm event")
	}

	return nil
}

func (p *serialPort) Read(buf []byte) (int, os.Error) {
	p.rl.Lock()
	defer p.rl.Unlock()

	var events uint32

	if p == nil || p.f == nil {
		str := fmt.Sprintf("Invalid port on read %v %v", p, p.f)
		return 0, os.NewError(str)
	}

	err := resetEvent(p.ro.HEvent)
	if err != nil {
		return 0, err
	}

	err = waitCommEvent(p.fd, p.ro);
	if  err != nil {
		if err == os.Errno(syscall.ERROR_IO_PENDING) {
			_, err = getOverlappedResult(p.fd, p.ro)
			if err != nil {
				fmt.Println(err)
				return 0, err
			}
		} else {
			return 0, err
		}
	}

	err = resetEvent(p.ro.HEvent)
	if err != nil {
		return 0, err
	}

	var n int
	e := syscall.ReadFile(p.fd, buf, &events, p.ro)
	if e != 0 && e != syscall.ERROR_IO_PENDING {
		return 0, os.Errno(e)
	}
	n, err = getOverlappedResult(p.fd, p.ro)
	if err != nil {
		fmt.Println(err)
		return n, err
	}
	return n, nil
}
