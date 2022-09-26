package ssh

import (
	"io"
	"os"

	"github.com/shiena/ansicolor"
	"golang.org/x/term"
)

func stdStreams() (stdIn io.ReadCloser, stdOut, stdErr io.Writer) {
	return os.Stdin, ansicolor.NewAnsiColorWriter(os.Stdout), ansicolor.NewAnsiColorWriter(os.Stderr)
}

func makeRawTerminal(fd int) (*term.State, error) {
	return nil, nil
}
