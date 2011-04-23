package serial

import (
	"os"
	"io"
	//"fmt"
	"unsafe"
	"syscall"
)

type serialPort struct {
	f *os.File
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
var nWaitCommEvent uint32
var nGetOverlappedResult uint32
var nCreateEvent uint32

func loadDll(lib uint32, name string) uint32 {
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

	nSetCommState = loadDll(k32, "SetCommState")
	nSetCommTimeouts = loadDll(k32, "SetCommTimeouts")
	nSetCommMask = loadDll(k32, "SetCommMask")
	nWaitCommEvent = loadDll(k32, "WaitCommEvent")
	nGetOverlappedResult = loadDll(k32, "GetOverlappedResult")
	nCreateEvent = loadDll(k32, "CreateEventW")
}

func setCommState(h int32, baud int) os.Error {
	var params structDCB
	params.DCBlength = uint32(unsafe.Sizeof(params))

	params.flags[0] |= 0x01 // fBinary

	params.BaudRate = uint32(baud)
	params.ByteSize = 8

	r, _, e := syscall.Syscall(uintptr(nSetCommState), uintptr(h), uintptr(unsafe.Pointer(&params)), 0, 0)
	if r == 0 {
		return os.Errno(e)
	}
	return nil
}

func setCommTimeouts(h int32) os.Error {
	var timeouts structTimeouts
	timeouts.ReadIntervalTimeout = 1<<32 - 1
	timeouts.ReadTotalTimeoutConstant = 0
	r, _, e := syscall.Syscall(uintptr(nSetCommTimeouts), uintptr(h), uintptr(unsafe.Pointer(&timeouts)), 0, 0)
	if r == 0 {
		return os.Errno(e)
	}
	return nil
}

func setCommMask(h int32) os.Error {
	r, _, e := syscall.Syscall(uintptr(nSetCommMask), uintptr(h), EV_RXCHAR, 0, 0)
	if r == 0 {
		return os.Errno(e)
	}
	return nil
}

const FILE_FLAGS_OVERLAPPED = 0x40000000

func OpenPort(name string, baud int) (io.ReadWriteCloser, os.Error) {
	//h, e := CreateFile(StringToUTF16Ptr(path), access, sharemode, sa, createmode, FILE_ATTRIBUTE_NORMAL, 0)

	h, e := syscall.CreateFile(syscall.StringToUTF16Ptr(name),
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		uint32(syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE),
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

	// Turn off buffers
	/*
		if ok, err := C.SetupComm(*handle, 16, 16); ok == C.FALSE {
			f.Close()
			return nil, err
		}
	*/

	if err := setCommTimeouts(h); err != nil {
		f.Close()
		return nil, err
	}
	if err := setCommMask(h); err != nil {
		f.Close()
		return nil, err
	}

	port := serialPort{f}

	return &port, nil
}

func (p *serialPort) Close() os.Error {
	return p.f.Close()
}

func newOverlapped() (*syscall.Overlapped, os.Error) {
	var overlapped syscall.Overlapped
	r, _, e := syscall.Syscall(uintptr(nCreateEvent), 0, 1, 0, 0)
	if r == 0 {
		return nil, os.Errno(e)
	}
	overlapped.HEvent = (*uint8)(unsafe.Pointer(r))
	return &overlapped, nil
}

func getOverlappedResults(h int, overlapped *syscall.Overlapped) (int, os.Error) {
	var n int
	r, _, e := syscall.Syscall(uintptr(nGetOverlappedResult), uintptr(h),
		uintptr(unsafe.Pointer(overlapped)),
		uintptr(unsafe.Pointer(&n)), 1)
	if r == 0 {
		return 0, os.Errno(e)
	}

	return n, nil
}

func (p *serialPort) Write(buf []byte) (int, os.Error) {
	var events uint32

	fd := p.f.Fd()

	overlapped, err := newOverlapped()
	if err != nil {
		return 0, err
	}

	e := syscall.WriteFile(int32(fd), buf, &events, overlapped)
	if e != 0 {
		return 0, os.Errno(e)
	}

	n, err := getOverlappedResults(fd, overlapped)

	return n, err
}

func waitCommEvent(overlapped *syscall.Overlapped) os.Error {
	var events uint32
	r, _, e := syscall.Syscall(uintptr(nWaitCommEvent), uintptr(unsafe.Pointer(&events)), uintptr(unsafe.Pointer(overlapped)), 0, 0)
	if r == 0 {
		return os.Errno(e)
	}
	return nil
}

func (p *serialPort) Read(buf []byte) (int, os.Error) {
	var events uint32

	fd := p.f.Fd()

	overlapped, err := newOverlapped()
	if err != nil {
		return 0, err
	}

loop:
	if err = waitCommEvent(overlapped); err != nil {
		return 0, nil
	}

	_, err = getOverlappedResults(fd, overlapped)
	if err != nil {
		return 0, err
	}

	// There is a small race window here between returning from WaitCommEvent and reading from the file.
	// If we receive data in this window, we will read that data, but the RXFLAG will still be set
	// and next time the WaitCommEvent() will return but we will have already read the data.  That's
	// why we have the stupid goto loop on EOF.
	e := syscall.ReadFile(int32(fd), buf, &events, overlapped)
	if e != 0 {
		return 0, os.Errno(e)
	}

	n, err := getOverlappedResults(fd, overlapped)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		goto loop
	}

	return n, nil
}
