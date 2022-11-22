package ssh

// +enum
type ForwardMode string

const (
	ForwardModeForward ForwardMode = "forward"
	ForwardModeReverse ForwardMode = "reverse"
)
