package dryrun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	yaml "sigs.k8s.io/yaml/goyaml.v2"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

type DryRunner interface {
	DryRun(ctx context.Context, object runtime.Object, recorder Recorder) error
}

type DryRunHandler struct {
	projectDryRunner DryRunner
	scheme           *runtime.Scheme
}

func NewDryRunHandler(projectDryRunner DryRunner, scheme *runtime.Scheme) *DryRunHandler {
	return &DryRunHandler{
		projectDryRunner: projectDryRunner,
		scheme:           scheme,
	}
}

func (h *DryRunHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("error reading request: %v\n", err)
		return
	}
	codecs := serializer.NewCodecFactory(h.scheme)
	obj, _, err := codecs.UniversalDeserializer().Decode(data, nil, nil)
	if err != nil {
		fmt.Printf("error deserializing request: %v\n", err)
		return
	}

	recorder := SimpleRecorder{
		mu: sync.RWMutex{},
	}

	// ignore error, it is expected for a dry-run
	_ = h.projectDryRunner.DryRun(context.Background(), obj, &recorder)

	var dryRunResult = struct {
		Object         runtime.Object  `json:"object,omitempty"`
		PlannedActions []PlannedAction `json:"plannedActions,omitempty"`
	}{
		Object:         obj,
		PlannedActions: recorder.PlannedActions(),
	}
	enc := yaml.NewEncoder(w)
	if err := enc.Encode(&dryRunResult); err != nil {
		fmt.Printf("error serializing result: %v\n", err)
		return
	}
}
