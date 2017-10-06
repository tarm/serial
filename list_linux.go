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

		var name string
		for _, line := range bytes.Split(uevent, []byte{'\n'}) {
			if bytes.HasPrefix(line, []byte("DEVNAME=")) {
				name = "/dev/" + string(line[8:])
				break
			}
		}

		if _, err := os.Lstat(name); err == nil {
			names = append(names, name)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}

	sort.Strings(names)

	return
}
