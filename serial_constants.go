package serial

type DataBitsType byte
type StopBitsType byte
type ParityType byte

const (
	DATABITS_8 DataBitsType = iota
	DATABITS_7
	DATABITS_6
	DATABITS_5
)

const (
	STOPBITS_1 StopBitsType = iota
	STOPBITS_15
	STOPBITS_2
)

const (
	PARITY_NONE ParityType = iota
	PARITY_ODD
	PARITY_EVEN
	PARITY_MARK
	PARITY_SPACE
)
