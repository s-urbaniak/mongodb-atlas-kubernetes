package atlas

import (
	"net/http"

	"github.com/mongodb-forks/digest"
	"go.mongodb.org/atlas-sdk/v20231115008/admin"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/internal/dryrun"
)

type DryRunTransport struct {
	Recorder dryrun.Recorder
	Delegate http.RoundTripper
}

func (t *DryRunTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.Method {
	case http.MethodGet:
	case http.MethodConnect:
	case http.MethodTrace:
	case http.MethodHead:
	default:
		t.Recorder.Recordf(req.Method, "Would execute %v %v", req.Method, req.URL.Path)
		return nil, dryrun.ErrDryRun
	}

	return t.Delegate.RoundTrip(req)
}

func NewClient(domain, publicKey, privateKey string, dryRun bool, recorder dryrun.Recorder) (*admin.APIClient, error) {
	var transport http.RoundTripper = digest.NewTransport(publicKey, privateKey)

	if dryRun {
		transport = &DryRunTransport{
			Recorder: recorder,
			Delegate: transport,
		}
	}

	client := &http.Client{
		Transport: transport,
	}

	return admin.NewClient(
		admin.UseBaseURL(domain),
		admin.UseHTTPClient(client),
		admin.UseUserAgent(operatorUserAgent()),
	)
}
