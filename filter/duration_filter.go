package filter

import (
	"context"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"time"
)

// DurationFilter is a SpanProcessor that filters spans that have lifetimes outside of a defined range.
type DurationFilter struct {
	// Next is the next SpanProcessor in the chain.
	Next sdktrace.SpanProcessor

	// Min is the duration under which spans are dropped.
	Min time.Duration
	// Max is the duration over which spans are dropped.
	Max time.Duration
}

func (f DurationFilter) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
	f.Next.OnStart(parent, s)
}
func (f DurationFilter) Shutdown(ctx context.Context) error   { return f.Next.Shutdown(ctx) }
func (f DurationFilter) ForceFlush(ctx context.Context) error { return f.Next.ForceFlush(ctx) }
func (f DurationFilter) OnEnd(s sdktrace.ReadOnlySpan) {
	if f.Min > 0 && s.EndTime().Sub(s.StartTime()) < f.Min {
		// Drop short lived spans.
		return
	}
	if f.Max > 0 && s.EndTime().Sub(s.StartTime()) > f.Max {
		// Drop long lived spans.
		return
	}
	f.Next.OnEnd(s)
}
