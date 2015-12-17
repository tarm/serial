//+build linux

package serial

/*
This heuristic regex filter should catch most serial devices under linux.
The prefixes represent devices of the following types
ttyS:   onboard uarts
ttyUSB: USB<->uart bridges
ttyACM: Abstract Control Model devices (e.g. modems -- see https://www.rfc1149.net/blog/2013/03/05/what-is-the-difference-between-devttyusbx-and-devttyacmx/)
ttyAMA: Don't know what AMA stands for, but seems to be used for Raspberry PI onboard ports at least
*/
const (
	portNameFilter = `^(ttyS|ttyUSB|ttyACM|ttyAMA)\d+`
)
