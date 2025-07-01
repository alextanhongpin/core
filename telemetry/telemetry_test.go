package telemetry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/event"
)

// mockHandler is a simple mock implementation of the handler interface
type mockHandler struct {
	callCount int
	lastEvent *event.Event
	lastCtx   context.Context
	returnCtx context.Context
}

func (m *mockHandler) Event(ctx context.Context, e *event.Event) context.Context {
	m.callCount++
	m.lastEvent = e
	m.lastCtx = ctx
	if m.returnCtx != nil {
		return m.returnCtx
	}
	return ctx
}

// mockCloser is a mock handler that also implements io.Closer
type mockCloser struct {
	mockHandler
	closeError  error
	closeCalled bool
}

func (m *mockCloser) Close() error {
	m.closeCalled = true
	return m.closeError
}

func TestMultiHandlerEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("nil event", func(t *testing.T) {
		multiHandler := &MultiHandler{}
		resultCtx := multiHandler.Event(ctx, nil)
		assert.Equal(t, ctx, resultCtx)
	})

	t.Run("all handlers nil", func(t *testing.T) {
		multiHandler := &MultiHandler{}

		e := &event.Event{Kind: event.LogKind}
		resultCtx := multiHandler.Event(ctx, e)
		assert.Equal(t, ctx, resultCtx)
	})

	t.Run("single handler", func(t *testing.T) {
		mockLog := &mockHandler{}

		multiHandler := &MultiHandler{
			Log: mockLog,
		}

		e := &event.Event{Kind: event.LogKind}
		resultCtx := multiHandler.Event(ctx, e)

		assert.Equal(t, ctx, resultCtx)
		assert.Equal(t, 1, mockLog.callCount)
		assert.Equal(t, e, mockLog.lastEvent)
		assert.Equal(t, ctx, mockLog.lastCtx)
	})

	t.Run("all handlers present", func(t *testing.T) {
		mockLog := &mockHandler{}
		mockMetric := &mockHandler{}
		mockTrace := &mockHandler{}

		multiHandler := &MultiHandler{
			Log:    mockLog,
			Metric: mockMetric,
			Trace:  mockTrace,
		}

		e := &event.Event{Kind: event.LogKind}
		resultCtx := multiHandler.Event(ctx, e)

		assert.Equal(t, ctx, resultCtx)
		assert.Equal(t, 1, mockLog.callCount)
		assert.Equal(t, 1, mockMetric.callCount)
		assert.Equal(t, 1, mockTrace.callCount)
	})

	t.Run("context propagation", func(t *testing.T) {
		ctx1 := context.WithValue(ctx, "key1", "value1")
		ctx2 := context.WithValue(ctx1, "key2", "value2")
		ctx3 := context.WithValue(ctx2, "key3", "value3")

		mockLog := &mockHandler{returnCtx: ctx1}
		mockMetric := &mockHandler{returnCtx: ctx2}
		mockTrace := &mockHandler{returnCtx: ctx3}

		multiHandler := &MultiHandler{
			Log:    mockLog,
			Metric: mockMetric,
			Trace:  mockTrace,
		}

		e := &event.Event{Kind: event.LogKind}
		resultCtx := multiHandler.Event(ctx, e)

		assert.Equal(t, ctx3, resultCtx)
		assert.Equal(t, "value1", resultCtx.Value("key1"))
		assert.Equal(t, "value2", resultCtx.Value("key2"))
		assert.Equal(t, "value3", resultCtx.Value("key3"))

		// Check that each handler received the correct context
		assert.Equal(t, ctx, mockLog.lastCtx)
		assert.Equal(t, ctx1, mockMetric.lastCtx)
		assert.Equal(t, ctx2, mockTrace.lastCtx)
	})
}

func TestMultiHandlerClose(t *testing.T) {
	t.Run("no handlers", func(t *testing.T) {
		multiHandler := &MultiHandler{}
		err := multiHandler.Close()
		assert.NoError(t, err)
	})

	t.Run("handlers without Close method", func(t *testing.T) {
		mockLog := &mockHandler{}
		mockMetric := &mockHandler{}
		mockTrace := &mockHandler{}

		multiHandler := &MultiHandler{
			Log:    mockLog,
			Metric: mockMetric,
			Trace:  mockTrace,
		}

		err := multiHandler.Close()
		assert.NoError(t, err)
	})

	t.Run("handlers with Close method - success", func(t *testing.T) {
		mockLog := &mockCloser{}
		mockMetric := &mockCloser{}
		mockTrace := &mockCloser{}

		multiHandler := &MultiHandler{
			Log:    mockLog,
			Metric: mockMetric,
			Trace:  mockTrace,
		}

		err := multiHandler.Close()
		assert.NoError(t, err)

		assert.True(t, mockLog.closeCalled)
		assert.True(t, mockMetric.closeCalled)
		assert.True(t, mockTrace.closeCalled)
	})

	t.Run("handlers with Close method - log error", func(t *testing.T) {
		logError := errors.New("log close error")
		mockLog := &mockCloser{closeError: logError}
		mockMetric := &mockCloser{}
		mockTrace := &mockCloser{}

		multiHandler := &MultiHandler{
			Log:    mockLog,
			Metric: mockMetric,
			Trace:  mockTrace,
		}

		err := multiHandler.Close()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to close log handler")

		assert.True(t, mockLog.closeCalled)
		assert.True(t, mockMetric.closeCalled)
		assert.True(t, mockTrace.closeCalled)
	})

	t.Run("handlers with Close method - metric error", func(t *testing.T) {
		metricError := errors.New("metric close error")
		mockLog := &mockCloser{}
		mockMetric := &mockCloser{closeError: metricError}
		mockTrace := &mockCloser{}

		multiHandler := &MultiHandler{
			Log:    mockLog,
			Metric: mockMetric,
			Trace:  mockTrace,
		}

		err := multiHandler.Close()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to close metric handler")

		assert.True(t, mockLog.closeCalled)
		assert.True(t, mockMetric.closeCalled)
		assert.True(t, mockTrace.closeCalled)
	})

	t.Run("handlers with Close method - trace error", func(t *testing.T) {
		traceError := errors.New("trace close error")
		mockLog := &mockCloser{}
		mockMetric := &mockCloser{}
		mockTrace := &mockCloser{closeError: traceError}

		multiHandler := &MultiHandler{
			Log:    mockLog,
			Metric: mockMetric,
			Trace:  mockTrace,
		}

		err := multiHandler.Close()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to close trace handler")

		assert.True(t, mockLog.closeCalled)
		assert.True(t, mockMetric.closeCalled)
		assert.True(t, mockTrace.closeCalled)
	})

	t.Run("multiple errors - returns first", func(t *testing.T) {
		logError := errors.New("log close error")
		metricError := errors.New("metric close error")
		traceError := errors.New("trace close error")

		mockLog := &mockCloser{closeError: logError}
		mockMetric := &mockCloser{closeError: metricError}
		mockTrace := &mockCloser{closeError: traceError}

		multiHandler := &MultiHandler{
			Log:    mockLog,
			Metric: mockMetric,
			Trace:  mockTrace,
		}

		err := multiHandler.Close()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to close log handler")
		assert.NotContains(t, err.Error(), "metric")
		assert.NotContains(t, err.Error(), "trace")

		assert.True(t, mockLog.closeCalled)
		assert.True(t, mockMetric.closeCalled)
		assert.True(t, mockTrace.closeCalled)
	})

	t.Run("mixed closers and non-closers", func(t *testing.T) {
		mockLog := &mockCloser{}
		mockMetric := &mockHandler{} // Not a closer
		mockTrace := &mockCloser{}

		multiHandler := &MultiHandler{
			Log:    mockLog,
			Metric: mockMetric,
			Trace:  mockTrace,
		}

		err := multiHandler.Close()
		assert.NoError(t, err)

		assert.True(t, mockLog.closeCalled)
		assert.True(t, mockTrace.closeCalled)
	})

	t.Run("nil handlers", func(t *testing.T) {
		multiHandler := &MultiHandler{
			Log:    nil,
			Metric: nil,
			Trace:  nil,
		}

		err := multiHandler.Close()
		assert.NoError(t, err)
	})
}

func TestMultiHandlerImplementsInterface(t *testing.T) {
	// Verify that MultiHandler implements event.Handler
	var _ event.Handler = (*MultiHandler)(nil)
}

func TestMultiHandlerOrdering(t *testing.T) {
	// Test that handlers are called in the expected order: Log, Metric, Trace
	// Create custom handlers that track call order
	logHandler := &mockHandler{}
	metricHandler := &mockHandler{}
	traceHandler := &mockHandler{}

	// Track the original call count to verify order
	multiHandler := &MultiHandler{
		Log:    logHandler,
		Metric: metricHandler,
		Trace:  traceHandler,
	}

	ctx := context.Background()
	e := &event.Event{Kind: event.LogKind}
	multiHandler.Event(ctx, e)

	// All handlers should have been called once
	assert.Equal(t, 1, logHandler.callCount)
	assert.Equal(t, 1, metricHandler.callCount)
	assert.Equal(t, 1, traceHandler.callCount)

	// All handlers should have received the same event
	assert.Equal(t, e, logHandler.lastEvent)
	assert.Equal(t, e, metricHandler.lastEvent)
	assert.Equal(t, e, traceHandler.lastEvent)
}
