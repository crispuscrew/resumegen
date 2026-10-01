package cli

import (
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

// Bound only the first byte; once a producer starts, read its full stream.
// Tests override this bound to avoid waiting two seconds.
var stdinFirstByteWait = 2 * time.Second

func (resolver resolveCtx) stdinHasData() bool {
	type peekResult struct{ err error }
	done := make(chan peekResult, 1)
	// A timed-out read remains parked until the CLI exits or the pipe closes.
	go func() {
		_, err := resolver.reader.Peek(1)
		done <- peekResult{err}
	}()
	select {
	case result := <-done:
		return result.err == nil
	case <-time.After(stdinFirstByteWait):
		return false
	}
}

func isTerminal(reader io.Reader) bool {
	file, found := reader.(*os.File)
	if !found {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}
