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
	DCBlength, BaudRate uint32
	flags [4]byte
	wReserved, XonLim, XoffLim uint16
	ByteSize, Parity, StopBits byte
	XonChar, XoffChar, ErrorChar, EofChar, EvtChar byte
	wReserved1 uint16
}

type structTimeouts struct {
	ReadIntervalTimeout uint32
	ReadTotalTimeoutMultiplier uint32
	ReadTotalTimeoutConstant uint32
	WriteTotalTimeoutMultiplier uint32
	WriteTotalTimeoutConstant uint32
}


var setCommState uint32
var setCommTimeouts uint32
var setCommMask uint32
var waitCommEvent uint32
var getOverlappedResult uint32
var createEvent uint32

func init() {
	k32, err := syscall.LoadLibrary("kernel32.dll")
	if err != 0 {
		panic("LoadLibrary "+ syscall.Errstr(err))
	}
	defer syscall.FreeLibrary(k32)

	setCommState, err = syscall.GetProcAddress(k32, "SetCommState")
	if err != 0 {
		panic("GetProcAddress "+ syscall.Errstr(err))
	}
	setCommTimeouts, err = syscall.GetProcAddress(k32, "SetCommTimeouts")
	if err != 0 {
		panic("SetCommTimeouts "+ syscall.Errstr(err))
	}
	setCommMask, err = syscall.GetProcAddress(k32, "SetCommMask")
	if err != 0 {
		panic("SetCommEvent "+ syscall.Errstr(err))
	}
	waitCommEvent, err = syscall.GetProcAddress(k32, "WaitCommEvent")
	if err != 0 {
		panic("WaitCommEvent "+ syscall.Errstr(err))
	}
	getOverlappedResult, err = syscall.GetProcAddress(k32, "GetOverlappedResult")
	if err != 0 {
		panic("GetOverlappedResult "+ syscall.Errstr(err))
	}
	createEvent, err = syscall.GetProcAddress(k32, "CreateEvent")
	if err != 0 {
		panic("CreateEvent "+ syscall.Errstr(err))
	}
}

const FILE_FLAGS_OVERLAPPED = 0x40000000

func OpenPort(name string, baud int) (io.ReadWriteCloser, os.Error) {
	//h, e := CreateFile(StringToUTF16Ptr(path), access, sharemode, sa, createmode, FILE_ATTRIBUTE_NORMAL, 0)

	h, e := syscall.CreateFile(syscall.StringToUTF16Ptr(name), 
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE),
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL|FILE_FLAGS_OVERLAPPED,
		0)

	if e != 0 {
		return nil, &os.PathError{"open", name, os.Errno(e)}
	}

	f := os.NewFile(int(h), name)

	var params structDCB
	params.DCBlength = uint32(unsafe.Sizeof(params))

	params.BaudRate = uint32(baud)
	params.ByteSize = 8

	_, _, e2 := syscall.Syscall(uintptr(setCommState), uintptr(h), uintptr(unsafe.Pointer(&params)), 0, 0)
	if e2 != 0 {
		f.Close()
		return nil, os.Errno(e)
	}

	// Turn off buffers
	/*
	if ok, err := C.SetupComm(*handle, 16, 16); ok == C.FALSE {
		f.Close()
		return nil, err
	}
	 */

	var timeouts structTimeouts
	timeouts.ReadIntervalTimeout = 1<<32 - 1
	timeouts.ReadTotalTimeoutConstant = 0
	_, _, e2 = syscall.Syscall(uintptr(setCommTimeouts), uintptr(h), uintptr(unsafe.Pointer(&timeouts)), 0, 0)
	if e2 != 0 {
		f.Close()
		return nil, os.Errno(e)
	}

	_, _, e2 = syscall.Syscall(uintptr(setCommMask), uintptr(h), EV_RXCHAR, 0, 0)
	if e2 != 0 {
		f.Close()
		return nil, os.Errno(e)
	}

	port := serialPort{f}

	return &port, nil
}

func (p *serialPort) Close() os.Error {
	return p.f.Close()
}

func (p *serialPort) Write(buf []byte) (int, os.Error) {
	var events uint32
	var overlapped syscall.Overlapped
	var n int

	fd := p.f.Fd()

	r, _, e := syscall.Syscall(uintptr(createEvent), 0, 1, 0, 0)
	if e != 0 {
		return 0, os.Errno(e)
	}
	overlapped.HEvent = (*uint8)(unsafe.Pointer(r))

	e1 := syscall.WriteFile(int32(fd), buf, &events, &overlapped)
	if e1 != 0 {
		return 0, os.Errno(e)
	}

	_, _, e = syscall.Syscall(uintptr(getOverlappedResult), uintptr(fd), uintptr(unsafe.Pointer(&overlapped)), uintptr(unsafe.Pointer(&n)), 1)
	if e != 0 {
		return 0, os.Errno(e)
	}

	return p.f.Write(buf)
}

func (p *serialPort) Read(buf []byte) (int, os.Error) {
	var events uint32
	var overlapped syscall.Overlapped
	var n int

	fd := p.f.Fd()

	r, _, e := syscall.Syscall(uintptr(createEvent), 0, 1, 0, 0)
	if e != 0 {
		return 0, os.Errno(e)
	}
	overlapped.HEvent = (*uint8)(unsafe.Pointer(r))

loop:
	_, _, e = syscall.Syscall(uintptr(waitCommEvent), uintptr(unsafe.Pointer(&events)), uintptr(unsafe.Pointer(&overlapped)), 0, 0)
	if e != 0 {
		return 0, os.Errno(e)
	}

	_, _, e = syscall.Syscall(uintptr(getOverlappedResult), uintptr(fd), uintptr(unsafe.Pointer(&overlapped)), uintptr(unsafe.Pointer(&n)), 1)
	if e != 0 {
		return 0, os.Errno(e)
	}

	// There is a small race window here between returning from WaitCommEvent and reading from the file.
	// If we receive data in this window, we will read that data, but the RXFLAG will still be set
	// and next time the WaitCommEvent() will return but we will have already read the data.  That's
	// why we have the stupid goto loop on EOF.
	e1 := syscall.ReadFile(int32(fd), buf, &events, &overlapped)
	if e1 != 0 {
		return 0, os.Errno(e)
	}

	_, _, e = syscall.Syscall(uintptr(getOverlappedResult), uintptr(fd), uintptr(unsafe.Pointer(&overlapped)), uintptr(unsafe.Pointer(&n)), 1)
	if e != 0 {
		return 0, os.Errno(e)
	}
	if n == 0 {
		goto loop
	}

	return n, nil
}
