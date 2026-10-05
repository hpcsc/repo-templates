package exit

type Code int

const (
	OK Code = iota
	Refused
	Usage
	State
)

func (c Code) Status() int {
	switch c {
	case OK:
		return 0
	case Refused:
		return 1
	case Usage:
		return 2
	case State:
		return 3
	}
	return Usage.Status()
}
