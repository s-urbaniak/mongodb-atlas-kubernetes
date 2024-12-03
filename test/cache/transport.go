package cache

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/internal/dryrun"
)

type DryRunTransport struct {
	Recorder dryrun.Recorder
	Delegate http.RoundTripper
}

type blockingReadCloser struct {
	ctx context.Context
}

func (e *blockingReadCloser) Read([]byte) (int, error) {
	// Watch streaming decoder is supposed to block until it receives data, so let's mimick that behavior.
	// See https://github.com/kubernetes/kubernetes/blob/810e9e212ec5372d16b655f57b9231d8654a2179/staging/src/k8s.io/apimachinery/pkg/runtime/serializer/streaming/streaming.go#L77.
	<-e.ctx.Done()

	// once request context is done, close this reader and tell consumer we're EOF.
	// This causes the watch list to be empty.
	return 0, io.EOF
}

func (e *blockingReadCloser) Close() error {
	return nil
}

func newBlockingWatchResponse(ctx context.Context) *http.Response {
	resp := &http.Response{}
	resp.Header = make(http.Header)
	resp.Header.Set("Content-Type", "application/json")
	resp.StatusCode = http.StatusOK
	resp.Body = &blockingReadCloser{ctx: ctx}
	return resp
}

func (t *DryRunTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.Method {
	case http.MethodConnect:
	case http.MethodTrace:
	case http.MethodHead:
	case http.MethodGet:
		values, err := url.ParseQuery(req.URL.RawQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to parse query parameters: %w", err)
		}

		// If these are watch requests, then block them.
		// We don't want to receive watch streams from API-Server as this is a dry-run.
		if values.Get("watch") == "true" {
			return newBlockingWatchResponse(req.Context()), nil
		}

	default:
		//t.Recorder.Recordf(req.Method, "Would execute %v %v", req.Method, req.URL.Path)
		return nil, dryrun.ErrDryRun
	}

	return t.Delegate.RoundTrip(req)
}

func NewDryRunTransport(delegate http.RoundTripper) *DryRunTransport {
	return &DryRunTransport{
		Delegate: delegate,
	}
}
