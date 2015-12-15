package serial

import (
	"testing"
	"time"
)

func TestParseConfig(t *testing.T) {
	for _, x := range []struct {
		in   string
		name string
		baud int
		siz  byte
		par  Parity
		stop StopBits
		rto  string
		msg  string
	}{
		{"/dev/ttyUSB0:9600,8N1,500ms", "/dev/ttyUSB0", 9600, 8, ParityNone, Stop1, "500ms", ""},
		{"COM1:2400,8N1,1s", "COM1", 2400, 8, ParityNone, Stop1, "1s", ""},
		{"COM1::19200,7-E-1,0", "COM1:", 19200, 7, ParityEven, Stop1, "0", ""},

		// strange port name with comma in it
		{"portname,with,comma:19200,7o2,0", "portname,with,comma", 19200, 7, ParityOdd, Stop2, "0", ""},

		{in: "/dev/ttyUSB0", msg: "serial.ParseConfig: missing name or colon"},
		{in: "/dev/ttyUSB0:0x800,8N1,500ms", msg: "serial.ParseConfig: error parsing baud"},
		{in: "/dev/ttyUSB0:4800,6E2,500", msg: "serial.ParseConfig: bad duration"},
		{in: "/dev/ttyUSB0:4800,8E2,500ms,extra", msg: "serial.ParseConfig: garbage after timeout"},
	} {
		c, err := ParseConfig(x.in)
		if err != nil {
			if x.msg == "" {
				t.Error(x.in, "got error:", err.Error())
			} else if err.Error() != x.msg {
				t.Error(x.in, "got error:", err.Error(), "want:", x.msg)
			}
		} else {
			if x.msg != "" {
				t.Error(x.in, "worked but wanted error:", x.msg)
			} else {
				rto, err := time.ParseDuration(x.rto)
				if err != nil {
					t.Fatal(err)
				}
				xc := Config{
					Name:     x.name,
					Baud:     x.baud,
					Size:     x.siz,
					Parity:   x.par,
					StopBits: x.stop,

					ReadTimeout: rto,
				}
				if *c != xc {
					t.Error(x.in, "mismatch:", c, "!=", xc)
				}
			}
		}
	}
}

func TestFormatSettings(t *testing.T) {
	for _, x := range []struct {
		siz  byte
		par  Parity
		stop StopBits
		want string
	}{
		{8, ParityNone, Stop1, "8N1"},
		{7, ParityEven, Stop1, "7E1"},
		{7, ParityEven, Stop1Half, "7E1.5"},
		{5, ParityOdd, Stop1Half, "5O1.5"},
	} {
		s := FormatSettings(x.siz, x.par, x.stop)
		if s != x.want {
			t.Error("got: ", s, " want: ", x.want)
		}
	}
}

func TestParseSettings(t *testing.T) {
	for _, x := range []struct {
		in string

		siz  byte
		par  Parity
		stop StopBits

		msg string
	}{
		{"8N1", 8, ParityNone, Stop1, ""},
		{"8-N-1", 8, ParityNone, Stop1, ""},
		{"8/N/1", 8, ParityNone, Stop1, ""},
		{"8/n/1", 8, ParityNone, Stop1, ""},
		{"8/N-1", 8, ParityNone, Stop1, "serial.ParseSettings: bad separator"},
		{"8,N,1", 8, ParityNone, Stop1, "serial.ParseSettings: mismatched separator"},
		{"8-N-1-x", 8, ParityNone, Stop1, "serial.ParseSettings: garbage after input"},
		{"7-E-1", 7, ParityEven, Stop1, ""},
		{"7-E-2", 7, ParityEven, Stop2, ""},
		{"7-e-2", 7, ParityEven, Stop2, ""},
		{"7-Z-1", 7, ParityEven, Stop1, "serial.ParseSettings: invalid parity"},
		{"5O1.5", 5, ParityOdd, Stop1Half, ""},
		{"5o1.5", 5, ParityOdd, Stop1Half, ""},
		{"5O1.6", 5, ParityOdd, Stop1Half, "serial.ParseSettings: invalid stop bits"},
		{"5O2.5", 5, ParityOdd, Stop1Half, "serial.ParseSettings: invalid stop bits"},
		{"5O23", 5, ParityOdd, Stop1Half, "serial.ParseSettings: invalid stop bits"},
	} {
		siz, par, stop, err := ParseSettings(x.in)
		if err != nil {
			if x.msg == "" {
				t.Error(x.in, "got error:", err.Error())
			} else if err.Error() != x.msg {
				t.Error(x.in, "got error:", err.Error(), "want:", x.msg)
			}
		} else {
			if x.msg != "" {
				t.Error(x.in, "worked but wanted error:", x.msg)
			}
			if siz != x.siz || par != x.par || stop != x.stop {
				t.Error(x.in, "got:", siz, string([]byte{byte(par)}), stop,
					"want:", x.siz, string([]byte{byte(x.par)}), x.stop)
			}
		}
	}
}
