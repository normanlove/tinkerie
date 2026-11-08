//go:build !linux && !darwin

package serialport

func open(name string, cfg Config) (*Port, error)  { return nil, ErrUnsupported }
func configure(fd int, cfg Config) error           { return ErrUnsupported }
func setModemBit(fd int, bit int32, on bool) error { return ErrUnsupported }
func flush(fd int) error                           { return ErrUnsupported }
func list() []Info                                 { return nil }

const (
	tiocmDTR = 0x002
	tiocmRTS = 0x004
)
