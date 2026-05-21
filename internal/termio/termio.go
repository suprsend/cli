// Package termio coordinates writes to the terminal between a transient
// progress spinner (on stdout) and the structured logger (on stderr).
//
// Without coordination, a log line written while the spinner is animating
// gets appended to the spinner's row — the row never receives a newline
// from the spinner — and the merged row is then orphaned when the spinner
// redraws on the row below.
//
// This package solves that with a single mutex shared by both streams plus
// a counter tracking whether any spinner is currently active. When a log
// line arrives while a spinner is active, the writer prefixes the line
// with the ANSI "carriage return + erase to end of line" sequence, which
// wipes the spinner's current row before the log content lands. The
// spinner's next tick (also under the mutex) redraws cleanly on a fresh
// row below the log line.
package termio

import (
	"io"
	"sync"
	"sync/atomic"
)

// clearLine is the ANSI sequence to return the cursor to column 0 and erase
// to end of line. It is written to whichever stream needs to wipe the
// spinner row before emitting new content.
const clearLine = "\r\033[K"

var (
	mu     sync.Mutex
	active int32
)

// MarkSpinnerStart records that a spinner has started drawing. Pair with a
// single MarkSpinnerStop when the spinner is torn down.
func MarkSpinnerStart() { atomic.AddInt32(&active, 1) }

// MarkSpinnerStop records that a spinner has stopped. Safe to call multiple
// times on the same spinner — extra calls are no-ops, so the counter never
// goes negative.
func MarkSpinnerStop() {
	for {
		n := atomic.LoadInt32(&active)
		if n == 0 {
			return
		}
		if atomic.CompareAndSwapInt32(&active, n, n-1) {
			return
		}
	}
}

// SpinnerWriter wraps w so that every write is serialized under the shared
// mutex. Pass the wrapped writer into the spinner library so its tick
// goroutine cannot interleave with logger writes.
func SpinnerWriter(w io.Writer) io.Writer { return spinnerWriter{w: w} }

// LogWriter wraps w so that every write happens under the shared mutex.
// When a spinner is active, the write is prefixed with a line-clear escape
// so the log line cleanly replaces the spinner's current row.
func LogWriter(w io.Writer) io.Writer { return logWriter{w: w} }

type spinnerWriter struct{ w io.Writer }

func (s spinnerWriter) Write(p []byte) (int, error) {
	mu.Lock()
	defer mu.Unlock()
	return s.w.Write(p)
}

type logWriter struct{ w io.Writer }

func (l logWriter) Write(p []byte) (int, error) {
	mu.Lock()
	defer mu.Unlock()
	if atomic.LoadInt32(&active) > 0 {
		_, _ = l.w.Write([]byte(clearLine))
	}
	return l.w.Write(p)
}
