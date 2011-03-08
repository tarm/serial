package serial

// #include <windows.h>
import "C"
import (
	"os"
	"fmt"
	"unsafe"
)

func OpenPort(name string, baud int) (*os.File, os.Error) {
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
	/*if ok, err := C.GetCommState(*handle, &params); ok == C.FALSE {
		f.Close()
		return nil, err
	}
	*/
	params.BaudRate = C.DWORD(baud)
	params.ByteSize = 8
	params.StopBits = C.ONESTOPBIT
	params.Parity   = C.NOPARITY
	fmt.Println(params)

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
	timeouts.ReadIntervalTimeout = 1
	if ok, err := C.SetCommTimeouts(*handle, &timeouts); ok == C.FALSE {
		f.Close()
		return nil, err
	}
	fmt.Println(timeouts)

	return f, nil
}
