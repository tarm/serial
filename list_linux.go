package serial

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
)

// ListRX returns the detected serial ports sorted.
func ListRX() (names []string, err error) {
	matches, err := filepath.Glob("/sys/class/tty/*/rx_trig_bytes")
	if err != nil {
		return nil, err
	}

	names = make([]string, 0, len(matches))
	for _, m := range matches {
		uevent, err := ioutil.ReadFile(filepath.Clean(m + "/../uevent"))
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		for _, line := range bytes.Split(uevent, []byte{'\n'}) {
			if bytes.HasPrefix(line, []byte("DEVNAME=")) {
				names = append(names, "/dev/"+string(line[8:]))
				break
			}
		}
	}

	sort.Strings(names)

	return
}
