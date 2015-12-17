//+build darwin

package serial

//
// Apparently OSX creates a CU (Call-Up) and a tty device entry for
// each attached serial device, prefixing them with 'cu.' and 'tty.'
// respectively
// Should maybe restrict filter to cu devices?
// (see http://pbxbook.com/other/mac-tty.html)
// Although linux has dispensed with / deprecated this distinction:
// http://tldp.org/HOWTO/Modem-HOWTO-9.html#ss9.8
const (
	portNameFilter = `^(cu|tty)\..*`
)
