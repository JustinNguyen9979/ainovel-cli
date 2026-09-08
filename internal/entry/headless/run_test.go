package headless

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host"
)

// fakeHost exposes only the channels consumed by consume and drainPending. The
// production Host intentionally keeps its channel senders private, so the test
// installs deterministic in-memory channels without starting an engine.
type fakeHost struct {
	eng    *host.Host
	events chan host.Event
	stream chan string
	done   chan struct{}
}

func newFakeHost(t *testing.T) fakeHost {
	t.Helper()

	f := fakeHost{
		eng:    &host.Host{},
		events: make(chan host.Event, 8),
		stream: make(chan string, 8),
		done:   make(chan struct{}, 1),
	}
	setHostPrivateField(t, f.eng, "events", f.events)
	setHostPrivateField(t, f.eng, "streamCh", f.stream)
	setHostPrivateField(t, f.eng, "done", f.done)
	return f
}

func setHostPrivateField[T any](t *testing.T, eng *host.Host, name string, value chan T) {
	t.Helper()

	field := reflect.ValueOf(eng).Elem().FieldByName(name)
	if !field.IsValid() {
		t.Fatalf("host field %q not found", name)
	}
	writable := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	writable.Set(reflect.ValueOf(value))
}

type signalWriter struct {
	buf     bytes.Buffer
	written chan string
	err     error
}

func (w *signalWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	_, _ = w.buf.Write(p)
	if w.written != nil {
		w.written <- string(p)
	}
	return len(p), nil
}

func (w *signalWriter) String() string { return w.buf.String() }

type fixedErrorWriter struct{ err error }

func (w fixedErrorWriter) Write([]byte) (int, error) { return 0, w.err }

func waitForWrite(t *testing.T, written <-chan string, want string) {
	t.Helper()
	select {
	case got := <-written:
		if got != want {
			t.Fatalf("write = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("expected write %q", want)
	}
}

func TestWriteEventAndReplayQueue(t *testing.T) {
	var out bytes.Buffer
	writeEvent(&out, host.Event{Category: "INFO", Summary: "hello"})
	writeEvent(&out, host.Event{Time: time.Date(2026, 9, 6, 12, 34, 56, 0, time.UTC), Category: "INFO", Summary: "later"})
	replayQueue([]domain.RuntimeQueueItem{{Category: "QUEUE", Summary: "queued"}}, &out)
	if !strings.Contains(out.String(), "hello") || !strings.Contains(out.String(), "12:34:56") || !strings.Contains(out.String(), "queued") {
		t.Fatalf("event replay output = %q", out.String())
	}
}

func TestConsumeStreamsEventsAndDrainsAfterDone(t *testing.T) {
	f := newFakeHost(t)
	f.stream = make(chan string)
	setHostPrivateField(t, f.eng, "streamCh", f.stream)
	if f.eng.Stream() != f.stream || f.eng.Events() != f.events || f.eng.Done() != f.done {
		t.Fatal("private channel setup did not take effect")
	}
	stdout := &signalWriter{written: make(chan string, 8)}
	stderr := &signalWriter{written: make(chan string, 8)}
	result := make(chan error, 1)
	go func() { result <- consume(f.eng, stdout, stderr, false) }()

	// Empty deltas and a clear sentinel with no current content are ignored;
	// the following delta proves consume remains active after both branches.
	go func() {
		f.stream <- ""
		f.stream <- host.StreamClearSentinel
		f.stream <- "draft"
	}()
	waitForWrite(t, stdout.written, "draft")

	// Once content exists, a clear sentinel terminates that round with a blank
	// line and resets the state before the Done notification arrives.
	f.stream <- host.StreamClearSentinel
	waitForWrite(t, stdout.written, "\n\n")

	f.done <- struct{}{}
	if err := <-result; err != nil {
		t.Fatalf("consume returned error: %v", err)
	}
	if got := stdout.String(); got != "draft\n\n" {
		t.Fatalf("stdout = %q, want %q", got, "draft\n\n")
	}
}

func TestConsumeWritesQueuedEventBeforeDone(t *testing.T) {
	f := newFakeHost(t)
	stderr := &signalWriter{written: make(chan string, 1)}
	f.events <- host.Event{Category: "SYSTEM", Summary: "queued event"}

	result := make(chan error, 1)
	go func() { result <- consume(f.eng, ioDiscard{}, stderr, false) }()
	waitForWrite(t, stderr.written, "[--:--:--] [SYSTEM] queued event\n")

	f.done <- struct{}{}
	if err := <-result; err != nil {
		t.Fatalf("consume returned error: %v", err)
	}
}

func TestConsumeReturnsWhenDoneIsClosed(t *testing.T) {
	f := newFakeHost(t)
	close(f.done)

	if err := consume(f.eng, ioDiscard{}, ioDiscard{}, false); err != nil {
		t.Fatalf("consume with closed Done returned error: %v", err)
	}
}

func TestConsumeReturnsStreamWriterErrors(t *testing.T) {
	want := errors.New("stdout failed")

	t.Run("delta", func(t *testing.T) {
		f := newFakeHost(t)
		f.stream <- "draft"
		if err := consume(f.eng, fixedErrorWriter{err: want}, ioDiscard{}, false); !errors.Is(err, want) {
			t.Fatalf("consume error = %v, want %v", err, want)
		}
	})

	t.Run("clear", func(t *testing.T) {
		f := newFakeHost(t)
		f.stream <- host.StreamClearSentinel
		if err := consume(f.eng, fixedErrorWriter{err: want}, ioDiscard{}, true); !errors.Is(err, want) {
			t.Fatalf("consume error = %v, want %v", err, want)
		}
	})
}

func TestDrainPendingHandlesQueuedEventsAndStreamValues(t *testing.T) {
	f := newFakeHost(t)
	f.events <- host.Event{Category: "SYSTEM", Summary: "pending event"}
	f.stream <- ""
	f.stream <- host.StreamClearSentinel
	f.stream <- "tail"

	var stdout, stderr bytes.Buffer
	if err := drainPending(f.eng, &stdout, &stderr, false); err != nil {
		t.Fatalf("drainPending returned error: %v", err)
	}
	if got := stdout.String(); got != "tail\n" {
		t.Fatalf("stdout = %q, want %q", got, "tail\n")
	}
	if got := stderr.String(); got != "[--:--:--] [SYSTEM] pending event\n" {
		t.Fatalf("stderr = %q, want queued event", got)
	}
}

func TestDrainPendingFlushesClearAndFinalRound(t *testing.T) {
	t.Run("clear existing round", func(t *testing.T) {
		f := newFakeHost(t)
		f.stream <- host.StreamClearSentinel

		var stdout, stderr bytes.Buffer
		if err := drainPending(f.eng, &stdout, &stderr, true); err != nil {
			t.Fatalf("drainPending returned error: %v", err)
		}
		if got := stdout.String(); got != "\n\n" {
			t.Fatalf("stdout = %q, want %q", got, "\n\n")
		}
	})

	t.Run("flushes remaining content", func(t *testing.T) {
		f := newFakeHost(t)

		var stdout, stderr bytes.Buffer
		if err := drainPending(f.eng, &stdout, &stderr, true); err != nil {
			t.Fatalf("drainPending returned error: %v", err)
		}
		if got := stdout.String(); got != "\n" {
			t.Fatalf("stdout = %q, want %q", got, "\n")
		}
	})
}

func TestDrainPendingReturnsWriterErrors(t *testing.T) {
	want := errors.New("stdout failed")

	t.Run("delta", func(t *testing.T) {
		f := newFakeHost(t)
		f.stream <- "tail"
		if err := drainPending(f.eng, fixedErrorWriter{err: want}, ioDiscard{}, false); !errors.Is(err, want) {
			t.Fatalf("drainPending error = %v, want %v", err, want)
		}
	})

	t.Run("clear", func(t *testing.T) {
		f := newFakeHost(t)
		f.stream <- host.StreamClearSentinel
		if err := drainPending(f.eng, fixedErrorWriter{err: want}, ioDiscard{}, true); !errors.Is(err, want) {
			t.Fatalf("drainPending error = %v, want %v", err, want)
		}
	})

	t.Run("final newline", func(t *testing.T) {
		f := newFakeHost(t)
		if err := drainPending(f.eng, fixedErrorWriter{err: want}, ioDiscard{}, true); !errors.Is(err, want) {
			t.Fatalf("drainPending error = %v, want %v", err, want)
		}
	})
}

// ioDiscard avoids coupling these tests to os.DevNull while keeping the
// writer side effects under direct test control.
type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
