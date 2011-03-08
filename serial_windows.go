package serial

// #include <windows.h>
import "C"

func OpenPort(name string, baud int) (*os.File, os.Error) {
	f, err := os.Open(name, os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	handle := f.Fd()
	var params C.DCB
	params.DCBLength = C.sizeof(C.DCB)
	if ok, err := C.GetCommState(handle, &params); !ok {
		return nil, err
	}

	params.BaudRate = baud
	params.ByteSize = 8
	params.StopBits = C.ONESTOPBIT
	params.Parity   = C.NOPARITY


	if ok, err := C.SetCommState(handle, &params); !ok {
		return nil, err
	}
	
	return f, nil
}