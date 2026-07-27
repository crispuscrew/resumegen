package cli

import (
	"bufio"
	"os"
	"testing"
	"time"
)

// --no-input must never block. A non-TTY stdin can still be a pipe held open
// by a supervisor or CI runner that never reaches EOF, which used to hang
// io.ReadAll forever. Only the wait for the FIRST byte is bounded, so a slow
// but real producer is still read in full.
func TestStdinHasData(t *testing.T) {
	orig := stdinFirstByteWait
	stdinFirstByteWait = 150 * time.Millisecond
	defer func() { stdinFirstByteWait = orig }()

	newCtx := func(r *os.File) resolveCtx {
		return resolveCtx{stdin: r, reader: bufio.NewReader(r)}
	}

	t.Run("open but idle pipe reports no data", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = r.Close(); _ = w.Close() }() // w stays open: never EOF
		start := time.Now()
		if newCtx(r).stdinHasData() {
			t.Error("stdinHasData = true for an idle pipe, want false")
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Errorf("blocked %v; the wait is not bounded", elapsed)
		}
	})

	t.Run("closed pipe reports no data", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		_ = w.Close() // immediate EOF
		defer func() { _ = r.Close() }()
		if newCtx(r).stdinHasData() {
			t.Error("stdinHasData = true at EOF, want false")
		}
	})

	t.Run("piped data reports data and is not consumed", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		go func() { _, _ = w.WriteString("hello from the pipe\n"); _ = w.Close() }()
		defer func() { _ = r.Close() }()
		rc := newCtx(r)
		if !rc.stdinHasData() {
			t.Fatal("stdinHasData = false for a pipe carrying data, want true")
		}
		// Peek must not consume: the later read still sees everything.
		got, err := rc.reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if got != "hello from the pipe\n" {
			t.Errorf("read %q after peek, want the full line", got)
		}
	})
}
