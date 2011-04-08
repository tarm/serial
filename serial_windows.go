package serial

// #include <windows.h>
import "C"
import (
	"os"
	"io"
	"fmt"
	"unsafe"
)

type serialPort struct {
	f *os.File
}

const EV_RXCHAR = 0x0001

func OpenPort(name string, baud int) (io.ReadWriteCloser, os.Error) {
	f, err := os.Open(name, os.O_RDWR|os.O_NDELAY, 0666)
	if err != nil {
		return nil, err
	}

	type hp *C.HANDLE
	var handle *C.HANDLE
	fd := f.Fd()
	handle = hp(unsafe.Pointer(&fd))
	var params C.struct__DCB
	params.DCBlength = C.DWORD(unsafe.Sizeof(params))
	
	params.XonLim = 0
	params.XoffLim = 0
	
	params.BaudRate      = C.DWORD(baud)
	params.ByteSize      = 8
	params.StopBits      = C.ONESTOPBIT
	params.Parity        = C.NOPARITY

	//fmt.Printf("%#v %v\n", params, params)

	if ok, err := C.SetCommState(*handle, &params); ok == C.FALSE {
		f.Close()
		return nil, err
	}

	// Turn off buffers
	if ok, err := C.SetupComm(*handle, 16, 16); ok == C.FALSE {
		f.Close()
		return nil, err
	}
	
	var timeouts C.struct__COMMTIMEOUTS
	const DWORDMAX = 1<<32 - 1
	timeouts.ReadIntervalTimeout = DWORDMAX
	timeouts.ReadTotalTimeoutConstant = 0
	if ok, err := C.SetCommTimeouts(*handle, &timeouts); ok == C.FALSE {
		f.Close()
		return nil, err
	}
	//fmt.Printf("%#v\n", timeouts)

	if ok, err := C.SetCommMask(*handle, EV_RXCHAR); ok == C.FALSE {
		f.Close()
		return nil, err
	}

	port := serialPort{f}

	return &port, nil
}

func (p *serialPort) Close() os.Error {
	return p.f.Close()
}

func (p *serialPort) Write(buf []byte) (int, os.Error) {
	return p.f.Write(buf)
}

func (p *serialPort) Read(buf []byte) (int, os.Error) {
	type hp *C.HANDLE
	var handle *C.HANDLE
	fd := p.f.Fd()
	handle = hp(unsafe.Pointer(&fd))

	var events C.DWORD
	var overlapped *C.struct__OVERLAPPED

loop:
	if ok, err := C.WaitCommEvent(*handle, &events, overlapped); ok == C.FALSE {
		fmt.Printf("%v, 0x%04x\n", err, events)
		return 0, err
	}
	// There is a small race window here between returning from WaitCommEvent and reading from the file.
	// If we receive data in this window, we will read that data, but the RXFLAG will still be set
	// and next time the WaitCommEvent() will return but we will have already read the data.  That's
	// why we have the stupid goto loop on EOF.
	n, err := p.f.Read(buf)
	if err == os.EOF && n == 0 {
		//fmt.Printf("%v, %v, %v, 0x%04x\n", err, n, len(buf), events)
		goto loop
	}

	return n, err
}
