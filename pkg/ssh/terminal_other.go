//go:build !windows
// +build !windows

package ssh

import (
	"io"
	"os"

	"golang.org/x/term"
)

func stdStreams() (stdIn io.ReadCloser, stdOut, stdErr io.Writer) {
	return os.Stdin, os.Stdout, os.Stderr
}

func makeRawTerminal(fd int) (*term.State, error) {
	return term.MakeRaw(fd)
}
