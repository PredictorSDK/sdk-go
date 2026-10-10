package internal

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// closedPortURL is a URL whose connections are refused: a port that was
// listening a moment ago and is not.
func closedPortURL(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	return "http://" + address
}

// dialCountingClient counts the connections its transport dials, so a test can
// count attempts that never reach a server.
func dialCountingClient(dials *atomic.Int32, timeout time.Duration) *http.Client {
	dialer := &net.Dialer{}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				dials.Add(1)
				return dialer.DialContext(ctx, network, address)
			},
		},
	}
}

func callGet(ctx context.Context, client *http.Client, target string, maxAttempts uint) (*CallResponse, error) {
	var response *InternalTestResponse
	return NewCaller(&CallerParams{Client: client}).Call(ctx, &CallParams{
		URL:                target,
		Method:             http.MethodGet,
		Request:            &InternalTestRequest{},
		Response:           &response,
		MaxAttempts:        maxAttempts,
		ResponseIsOptional: true,
	})
}

// TestPredictorSDKDefaultsToThreeAttempts: the default is the request and two
// retries, like TypeScript's maxRetries and Python's max_retries.
func TestPredictorSDKDefaultsToThreeAttempts(t *testing.T) {
	t.Parallel()

	assert.Equal(t, uint(3), NewRetrier().attempts)
	assert.Equal(t, uint(3), NewRetrier(buildRetryOptions(0, false)...).attempts, "WithMaxAttempts(0) falls through to the default")
	assert.Equal(t, uint(5), NewRetrier(buildRetryOptions(5, false)...).attempts)
}

// TestPredictorSDKRetriesARefusedConnection: a request that got no response
// because the connection was refused is retried, once per remaining attempt.
func TestPredictorSDKRetriesARefusedConnection(t *testing.T) {
	t.Parallel()

	var dials atomic.Int32
	started := time.Now()
	_, err := callGet(context.Background(), dialCountingClient(&dials, 0), closedPortURL(t), 2)

	require.Error(t, err)
	assert.Equal(t, int32(2), dials.Load(), "two attempts, one retry")
	assert.GreaterOrEqual(t, time.Since(started), 900*time.Millisecond, "the retry waited out a backoff")
	assert.Less(t, time.Since(started), 3*time.Second)
}

// TestPredictorSDKRetriesAConnectionClosedBeforeAResponse: a server that
// closes the connection without answering is retried, and the retry succeeds.
func TestPredictorSDKRetriesAConnectionClosedBeforeAResponse(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			connection, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			_ = connection.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, "{\"id\":\"ok\"}")
	}))
	defer server.Close()

	response, err := callGet(context.Background(), &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}, server.URL, 0)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, int32(2), requests.Load(), "the closed connection, then the retry")
}

// TestPredictorSDKTransportRetriesAreCapped: a connection that keeps failing
// stops at the default of three attempts, waiting between them but not after
// the last.
func TestPredictorSDKTransportRetriesAreCapped(t *testing.T) {
	t.Parallel()

	var dials atomic.Int32
	started := time.Now()
	_, err := callGet(context.Background(), dialCountingClient(&dials, 0), closedPortURL(t), 0)

	require.Error(t, err)
	assert.Equal(t, int32(3), dials.Load(), "the request and two retries")
	assert.GreaterOrEqual(t, time.Since(started), 2700*time.Millisecond, "two backoffs: about 1s and 2s")
	assert.Less(t, time.Since(started), 4500*time.Millisecond, "no wait after the last attempt")
}

// TestPredictorSDKDoesNotRetryATimeout: a request that timed out is returned
// at once, not repeated.
func TestPredictorSDKDoesNotRetryATimeout(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	defer close(release)

	started := time.Now()
	_, err := callGet(context.Background(), &http.Client{Timeout: 300 * time.Millisecond}, server.URL, 3)

	require.Error(t, err)
	var netErr net.Error
	require.True(t, errors.As(err, &netErr) && netErr.Timeout(), "want a timeout, got %v", err)
	assert.Equal(t, int32(1), requests.Load())
	assert.Less(t, time.Since(started), 1500*time.Millisecond, "no backoff, no second request")
}

// TestPredictorSDKDoesNotRetryACancelledContext: a context the caller cancels
// mid-request ends the call at once with that error.
func TestPredictorSDKDoesNotRetryACancelledContext(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)
	started := time.Now()
	_, err := callGet(ctx, server.Client(), server.URL, 3)

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(1), requests.Load())
	assert.Less(t, time.Since(started), 1500*time.Millisecond)
}

// TestPredictorSDKContextEndsAConnectionRetryBackoff: a context that ends
// while the client waits to retry a refused connection returns then, not when
// the backoff would have finished, and sends no second request.
func TestPredictorSDKContextEndsAConnectionRetryBackoff(t *testing.T) {
	t.Parallel()

	var dials atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := callGet(ctx, dialCountingClient(&dials, 0), closedPortURL(t), 3)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, int32(1), dials.Load())
	assert.Less(t, time.Since(started), 900*time.Millisecond, "the wait did not run its full second")
}

// timeoutError is a timeout that also wraps a connection reset. Whatever else
// an error says, a timeout is never retried.
type timeoutError struct{ err error }

func (e timeoutError) Error() string   { return "timeout: " + e.err.Error() }
func (e timeoutError) Unwrap() error   { return e.err }
func (e timeoutError) Timeout() bool   { return true }
func (e timeoutError) Temporary() bool { return false }

// TestPredictorSDKTransportErrorClassification: which errors mean the
// connection failed before any response, and which are returned at once.
func TestPredictorSDKTransportErrorClassification(t *testing.T) {
	t.Parallel()

	get := func(ctx context.Context) *http.Request {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.invalid/", nil)
		require.NoError(t, err)
		return request
	}
	wrap := func(err error) error {
		return &url.Error{Op: "Get", URL: "http://example.invalid/", Err: err}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	for _, tc := range []struct {
		name      string
		ctx       context.Context
		err       error
		wantRetry bool
	}{
		{"refused", context.Background(), wrap(&net.OpError{Op: "dial", Net: "tcp", Err: &os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED}}), true},
		{"reset", context.Background(), wrap(&net.OpError{Op: "read", Net: "tcp", Err: &os.SyscallError{Syscall: "read", Err: syscall.ECONNRESET}}), true},
		{"broken pipe", context.Background(), wrap(&net.OpError{Op: "write", Net: "tcp", Err: &os.SyscallError{Syscall: "write", Err: syscall.EPIPE}}), true},
		{"windows reset", context.Background(), wrap(&net.OpError{Op: "wsarecv", Net: "tcp", Err: &os.SyscallError{Syscall: "wsarecv", Err: wsaeConnReset}}), true},
		{"closed before a response", context.Background(), wrap(io.EOF), true},
		{"closed mid-message", context.Background(), wrap(io.ErrUnexpectedEOF), true},
		{"client timeout", context.Background(), wrap(&net.OpError{Op: "read", Net: "tcp", Err: os.ErrDeadlineExceeded}), false},
		{"dial timeout", context.Background(), wrap(&net.OpError{Op: "dial", Net: "tcp", Err: &os.SyscallError{Syscall: "connect", Err: syscall.ETIMEDOUT}}), false},
		{"timeout wrapping a reset", context.Background(), wrap(timeoutError{err: syscall.ECONNRESET}), false},
		{"deadline exceeded", context.Background(), wrap(context.DeadlineExceeded), false},
		{"cancelled error", context.Background(), wrap(context.Canceled), false},
		{"cancelled wrapping a reset", context.Background(), wrap(fmt.Errorf("%w while %w", context.Canceled, syscall.ECONNRESET)), false},
		{"cancelled context", cancelled, wrap(io.EOF), false},
		{"unresolvable host", context.Background(), wrap(&net.DNSError{Err: "no such host", Name: "example.invalid", IsNotFound: true}), false},
		{"untrusted certificate", context.Background(), wrap(x509.UnknownAuthorityError{}), false},
		{"anything else", context.Background(), errors.New("unsupported protocol scheme"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantRetry, isRetryableTransportError(get(tc.ctx), tc.err))
		})
	}
}
