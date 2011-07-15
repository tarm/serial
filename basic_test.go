package serial

import (
	"github.com/tarm/goserial"
	"log"
)

func main() {
	s, err := serial.OpenPort("COM5", 115200)
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
