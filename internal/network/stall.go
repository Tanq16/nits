package network

import (
	"context"
	"errors"
	"io"
	"time"
)

var ErrStalled = errors.New("connection stalled")

type StallGuard struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	timer  *time.Timer
	idle   time.Duration
}

func NewStallGuard(parent context.Context, idle time.Duration) (context.Context, *StallGuard) {
	ctx, cancel := context.WithCancelCause(parent)
	g := &StallGuard{ctx: ctx, cancel: cancel, idle: idle}
	g.timer = time.AfterFunc(idle, func() { cancel(ErrStalled) })
	return ctx, g
}

func (g *StallGuard) Reader(r io.Reader) io.Reader {
	return stallReader{r: r, g: g}
}

func (g *StallGuard) Stop() {
	g.timer.Stop()
	g.cancel(nil)
}

func (g *StallGuard) Err(err error) error {
	if err != nil && errors.Is(context.Cause(g.ctx), ErrStalled) {
		return ErrStalled
	}
	return err
}

type stallReader struct {
	r io.Reader
	g *StallGuard
}

func (s stallReader) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	if n > 0 || errors.Is(err, io.EOF) {
		s.g.timer.Reset(s.g.idle)
	}
	return n, err
}
