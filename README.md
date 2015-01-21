GoSerial
========
A simple go package to allow you to read and write from the
serial port as a stream of bytes.

Details
-------
It aims to have the same API on all platforms, including windows.  As
an added bonus, the windows package does not use cgo, so you can cross
compile for windows from another platform.  Unfortunately goinstall
does not currently let you cross compile so you will have to do it
manually:

    GOOS=windows make clean install

Currently there is very little in the way of configurability.  You can
set the baud rate.  Then you can Read(), Write(), or Close() the
connection.  Read() will block until at least one byte is returned.
Write is the same.  There is currently no exposed way to set the
timeouts, though patches are welcome.

Currently all ports are opened with 8 data bits, 1 stop bit, no
parity, no hardware flow control, and no software flow control.  This
works fine for many real devices and many faux serial devices
including usb-to-serial converters and bluetooth serial ports.

You may Read() and Write() simulantiously on the same connection (from
different goroutines).

Usage
-----
```go
package main

import (
        "github.com/tarm/goserial"
        "log"
)

func main() {
        c := &serial.Config{Name: "COM45", Baud: 115200}
        s, err := serial.OpenPort(c)
        if err != nil {
                log.Fatal(err)
        }
        
        n, err := s.Write([]byte("test"))
        if err != nil {
                log.Fatal(err)
        }
        
        buf := make([]byte, 128)
        n, err = s.Read(buf)
        if err != nil {
                log.Fatal(err)
        }
        log.Print("%q", buf[:n])
}
```

NonBlocking Mode
----------------
By default goserial reads in blocking mode. Which means `Read()` will
block until at least one byte is returned. If that's not what you want,
specify a positive ReadTimeout and the Read() will timeout returning 0
bytes if no bytes are read.
Please note that this is the total timeout the read operation will wait
and not the interval timeout between two bytes.

```go
	c := &serial.Config{Name: "COM45", Baud: 115200, ReadTimeout: time.Second * 5}
	
	// In this mode, you will want to suppress error for read
	// as 0 bytes return EOF error on Linux / POSIX
	n, _ = s.Read(buf)
```

Possible Future Work
-------------------- 
- better tests (loopback etc)
