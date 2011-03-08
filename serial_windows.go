package serial

// #include <windows.h>
import "C"
import (
	"os"
	"unsafe"
)

func OpenPort(name string, baud int) (*os.File, os.Error) {
	f, err := os.Open(name, os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	type hp *C.HANDLE
	var handle *C.HANDLE
	fd := f.Fd()
	handle = hp(unsafe.Pointer(&fd))
	var params C.struct__DCB
	params.DCBlength = C.DWORD(unsafe.Sizeof(params))
	if ok, err := C.GetCommState(*handle, &params); ok == C.FALSE {
		f.Close()
		return nil, err
	}

	params.BaudRate = C.DWORD(baud)
	params.ByteSize = 8
	params.StopBits = C.ONESTOPBIT
	params.Parity   = C.NOPARITY

	if ok, err := C.SetCommState(*handle, &params); ok == C.FALSE {
		f.Close()
		return nil, err
	}

	return f, nil
}
