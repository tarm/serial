package serial

import (
	"path/filepath"
	"sort"
)

// ListRX returns the detected serial ports sorted.
func ListRX() (names []string, err error) {
	matches, err := filepath.Glob("/sys/class/tty/*/rx_trig_bytes")
	if err != nil {
		return nil, err
	}

	names = make([]string, len(matches))
	for i, m := range matches {
		names[i] = "/dev/" + filepath.Base(filepath.Dir(m))
	}

	sort.Strings(names)

	return
}
