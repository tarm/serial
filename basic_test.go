package serial

import (
	"testing"
)

func TestConnection(t *testing.T) {
	c0 := &Config{Name: "COM5", Baud: 115200}

	/*
		c1 := new(Config)
		c1.Name = "COM5"
		c1.Baud = 115200
	*/

	s, err := OpenPort(c0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Write([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 128)
	_, err = s.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
}

// BUG(tarmigan): Add loopback test
func TestLoopback(t *testing.T) {

}
