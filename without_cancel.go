// Package tracing
// ref https://stackoverflow.com/a/54132324
// https://github.com/datawire/dlib/blob/master/dcontext/without_cancel.go
package tracing

import (
	"context"
	reflectlite "reflect"
	"time"
)

type withoutCancel struct {
	context.Context
}

func (withoutCancel) Deadline() (deadline time.Time, ok bool) { return }
func (withoutCancel) Done() <-chan struct{}                   { return nil }
func (withoutCancel) Err() error                              { return nil }
func (c withoutCancel) String() string                        { return contextName(c.Context) + ".WithoutCancel" }
func (c withoutCancel) Value(key interface{}) interface{} {
	return c.Context.Value(key)
}

// WithoutCancel returns a copy of parent that inherits only values and not
// deadlines/cancellation/errors.  This is useful for implementing non-timed-out
// tasks during cleanup.
func WithoutCancel(parent context.Context) context.Context {
	return withoutCancel{parent}
}

type stringer interface {
	String() string
}

func contextName(c context.Context) string {
	if s, ok := c.(stringer); ok {
		return s.String()
	}
	return reflectlite.TypeOf(c).String()
}
