/*
Goserial is a simple go package to allow you to read and write from
the serial port as a stream of bytes.

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

Currently ports are opened with 8 data bits, 1 stop bit, no parity, no hardware
flow control, and no software flow control by default.  This works fine for
many real devices and many faux serial devices including usb-to-serial
converters and bluetooth serial ports.

You may Read() and Write() simulantiously on the same connection (from
different goroutines).

Example usage:

  package main

  import (
        "github.com/tarm/goserial"
        "log"
  )

  func main() {
        c := &serial.Config{Name: "COM5", Baud: 115200}
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
*/
package goserial

import (
	"errors"
	"io"
)

var (
	ErrConfigStopBits = errors.New("goserial config: bad number of stop bits")
	ErrConfigSize     = errors.New("goserial config: bad size")
	ErrConfigParity   = errors.New("goserial config: bad parity")
)

type ParityMode byte

const (
	ParityNone = iota
	ParityEven
	ParityOdd
)

// Config contains the information needed to open a serial port.
//
// Currently few options are implemented, but more may be added in the
// future (patches welcome), so it is recommended that you create a
// new config addressing the fields by name rather than by order.
//
// For example:
//
//    c0 := &serial.Config{Name: "COM45", Baud: 115200}
// or
//    c1 := new(serial.Config)
//    c1.Name = "/dev/tty.usbserial"
//    c1.Baud = 115200
//
type Config struct {
	Name string
	Baud int

	Size     int // 0 get translated to 8
	Parity   ParityMode
	StopBits int // 0 (default) and 1 means 1 stop bit, 2 means 2 stop bits

	// RTSFlowControl bool
	// DTRFlowControl bool
	// XONFlowControl bool

	CRLFTranslate bool // Ignored on Windows.
	// TimeoutStuff int
}

func (c *Config) size() (int, error) {
	switch c.Size {
	case 0:
		return 8, nil
	case 5, 6, 7, 8:
		return c.Size, nil
	}
	return 0, ErrConfigSize
}

func (c *Config) stopBits() (int, error) {
	switch c.StopBits {
	case 0:
		return 1, nil
	case 1, 2:
		return c.StopBits, nil
	}
	return 0, ErrConfigStopBits
}

func (c *Config) checkParityMode() error {
	switch c.Parity {
	case ParityNone, ParityEven, ParityOdd:
		return nil
	}
	return ErrConfigParity
}

// OpenPort opens a serial port with the specified configuration
func OpenPort(c *Config) (io.ReadWriteCloser, error) {
	return openPort(c.Name, c)
}

// func Flush()

// func SendBreak()

// func RegisterBreakHandler(func())
