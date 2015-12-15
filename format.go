package serial

import (
	"strconv"
	"time"
)

// ErrParse is the error returned if ParseConfig or ParseSettings have failed.
type ErrParse string

func (e ErrParse) Error() string { return string(e) }

// ParseConfig parses a config using the template:
//  <Name> ":" <Baud> "," <Settings> "," <ReadTimeout>
// Both Settings and ReadTimeout may be omitted, in that
// case the default values 8-N-1 and 0 will be used.
// Settings are parsed with ParseSettings.
// ReadTimeout is parsed using time.ParseDuration().
func ParseConfig(s string) (*Config, error) {
	colon := -1
	var comma []int
	for i, r := range s {
		switch r {
		case ':':
			colon = i
			// disregard commas seen so far
			comma = comma[:0]
		case ',':
			if colon >= 0 {
				comma = append(comma, i)
			}
		}
	}
	if colon == -1 {
		return nil, ErrParse("serial.ParseConfig: missing name or colon")
	}

	// extend comma slice to simplify the code below
	comma = append(comma, len(s))

	// parse Name and Baud
	c := &Config{Name: s[:colon]}
	baud, err := strconv.Atoi(s[colon+1 : comma[0]])
	if err != nil {
		return nil, ErrParse("serial.ParseConfig: error parsing baud")
	}
	c.Baud = int(baud)
	if len(comma) == 1 {
		return c, nil
	}

	// parse Settings
	c.Size, c.Parity, c.StopBits, err = ParseSettings(s[comma[0]+1 : comma[1]])
	if err != nil {
		return nil, err
	}
	if len(comma) == 2 {
		return c, nil
	}

	// parse ReadTimeout
	c.ReadTimeout, err = time.ParseDuration(s[comma[1]+1 : comma[2]])
	if err != nil {
		return nil, ErrParse("serial.ParseConfig: bad duration")
	}
	if len(comma) == 3 {
		return c, nil
	}
	return nil, ErrParse("serial.ParseConfig: garbage after timeout")
}

// String formats a config using the template:
//  <Name> ":" <Baud> "," <Settings> "," <ReadTimeout>
// Settings are formatted using Config.Settings.
// ReadTimeout is formatted using time.Duration.String()
func (c *Config) String() string {
	// Largest Baud is ±IntMax
	// Settings is 3 or 5 chars
	buf := make([]byte, len(c.Name)+1, len(c.Name)+32)
	copy(buf, c.Name)
	buf[len(c.Name)] = ':'
	buf = strconv.AppendInt(buf, int64(c.Baud), 10)
	buf = append(buf, ',')
	buf = appendSettings(buf, c.Size, c.Parity, c.StopBits)
	buf = append(buf, ',')
	return string(buf) + c.ReadTimeout.String()
}

// Settings formats data, parity and stop bit
// settings using FormatSettings.
func (c *Config) Settings() string {
	return FormatSettings(c.Size, c.Parity, c.StopBits)
}

// FormatSettings formats data, parity and stop bits settings
// using the conventional notation <data> <parity> <stop>
// with no separator in between.
func FormatSettings(size byte, par Parity, stop StopBits) string {
	return string(appendSettings(make([]byte, 0, 8), size, par, stop))
}

func appendSettings(p []byte, size byte, par Parity, stop StopBits) []byte {
	// size
	if size == 0 {
		size = DefaultSize
	}
	if size < 9 {
		p = append(p, '0'+size)
	} else {
		p = append(p, '9')
	}

	// parity
	if par == 0 {
		par = ParityNone
	}
	p = append(p, byte(par))

	// stop bit
	if stop == 0 {
		stop = Stop1
	}
	switch stop {
	case Stop1:
		p = append(p, '1')
	case Stop2:
		p = append(p, '2')
	case Stop1Half:
		p = append(p, '1', '.', '5')
	default:
		p = append(p, '9')
	}

	return p
}

// ParseSettings parses data, parity and stop bits settings.
// It accepts either '-' or '/' as separator, or no separator.
func ParseSettings(s string) (size byte, parity Parity, stop StopBits, err error) {
	size, parity, stop = DefaultSize, ParityNone, Stop1
	if s == "" {
		return size, parity, stop, nil
	}

	// parse data bits
	p := 0
	c := s[p]
	p++
	if c < '1' || '8' < c {
		return size, parity, stop, ErrParse("serial.ParseSettings: invalid data size")
	}
	size = c - '0'

	// parse parity
	if p == len(s) {
		return size, parity, stop, ErrParse("serial.ParseSettings: invalid input")
	}
	c = s[p]
	p++
	var sep byte
	if !isalnum(c) {
		if c != '-' && c != '/' {
			return size, parity, stop, ErrParse("serial.ParseSettings: mismatched separator")
		}
		// store and skip separator
		sep = c
		if p == len(s) {
			return size, parity, stop, ErrParse("serial.ParseSettings: invalid input")
		}
		c = s[p]
		p++
	}
	switch c {
	case 'n', 'o', 'e', 'm', 's':
		c -= 'a' - 'A' // convert to uppercase
	case 'N', 'O', 'E', 'M', 'S':
		// pass
	default:
		return size, parity, stop, ErrParse("serial.ParseSettings: invalid parity")
	}
	parity = Parity(c)

	// parse stop bits
	if p == len(s) {
		return size, parity, stop, ErrParse("serial.ParseSettings: invalid input")
	}
	c = s[p]
	p++
	if sep != 0 {
		// compare and skip separator
		if c != sep || p == len(s) {
			return size, parity, stop, ErrParse("serial.ParseSettings: bad separator")
		}
		c = s[p]
		p++
	}
	switch {
	case endnum(s[p:]) && (c == '1' || c == '2'):
		if c == '1' {
			stop = Stop1
		} else {
			stop = Stop2
		}
	case c == '1' && (p+2 <= len(s) && s[p] == '.' && s[p+1] == '5' && endnum(s[p+2:])):
		stop = Stop1Half
		p += 2
	default:
		return size, parity, stop, ErrParse("serial.ParseSettings: invalid stop bits")
	}

	if p != len(s) {
		return size, parity, stop, ErrParse("serial.ParseSettings: garbage after input")
	}
	return size, parity, stop, nil
}

func isalnum(c byte) bool {
	return ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z') || ('0' <= c && c <= '9')
}

func endnum(s string) bool {
	if len(s) == 0 {
		return true
	}
	return s[0] != '.' && !('0' <= s[0] && s[0] <= '9')
}
