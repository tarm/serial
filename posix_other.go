// +build !linux,!darwin

package serial

/*
This is a catchall for posix platforms other than linux and OS X
*/
const (
	portNameFilter = `^tty.*`
)
