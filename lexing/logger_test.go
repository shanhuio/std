package lexing

import (
	"errors"
	"fmt"
	"testing"
)

// fakeLogger records the messages passed to Errorf.
type fakeLogger struct {
	msgs []string
}

func (l *fakeLogger) Errorf(p *Pos, f string, args ...any) {
	l.msgs = append(l.msgs, fmt.Sprintf(f, args...))
}

func TestLogErrorNil(t *testing.T) {
	log := new(fakeLogger)
	if LogError(log, nil) {
		t.Errorf("LogError(nil): got true, want false")
	}
	if len(log.msgs) != 0 {
		t.Errorf("LogError(nil) should not log, got %v", log.msgs)
	}
}

func TestLogErrorNonNil(t *testing.T) {
	log := new(fakeLogger)
	if !LogError(log, errors.New("boom")) {
		t.Errorf("LogError(err): got false, want true")
	}
	if len(log.msgs) != 1 || log.msgs[0] != "boom" {
		t.Errorf("LogError logged %v, want [boom]", log.msgs)
	}
}
