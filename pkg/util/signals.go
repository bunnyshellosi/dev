package util

import (
	"os"
	"syscall"
)

var TerminationSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
}
