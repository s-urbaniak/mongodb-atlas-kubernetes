package dryrun

import (
	"net/http"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

const DryRunReason = "DryRun"

var verbMap = map[string]string{
	http.MethodPost:   "create (" + http.MethodPost + ")",
	http.MethodPut:    "update (" + http.MethodPut + ")",
	http.MethodPatch:  "update (" + http.MethodPatch + ")",
	http.MethodDelete: "delete (" + http.MethodDelete + ")",
}

type DryRunTransport struct {
	Object   runtime.Object
	Recorder record.EventRecorder
	Delegate http.RoundTripper
}

func NewDryRunTransport(obj runtime.Object, recorder record.EventRecorder, delegate http.RoundTripper) *DryRunTransport {
	return &DryRunTransport{
		Object:   obj,
		Recorder: recorder,
		Delegate: delegate,
	}
}

func (t *DryRunTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.Method {
	case http.MethodGet:
	case http.MethodConnect:
	case http.MethodTrace:
	case http.MethodHead:
	default:
		verb, ok := verbMap[req.Method]
		if !ok {
			verb = "execute " + req.Method
		}
		t.Recorder.Eventf(t.Object, v1.EventTypeNormal, DryRunReason, "Would %v %v", verb, req.URL.Path)
		return nil, ErrDryRun
	}

	return t.Delegate.RoundTrip(req)
}
