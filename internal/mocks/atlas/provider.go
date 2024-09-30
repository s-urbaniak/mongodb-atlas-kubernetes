package atlas

import (
	"context"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/internal/dryrun"

	"go.mongodb.org/atlas-sdk/v20231115008/admin"
	"go.mongodb.org/atlas/mongodbatlas"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/api"
)

type TestProvider struct {
	ClientFunc      func(secretRef *client.ObjectKey, log *zap.SugaredLogger, dryRun bool) (*mongodbatlas.Client, string, error)
	SdkClientFunc   func(secretRef *client.ObjectKey, log *zap.SugaredLogger, dryRun bool) (*admin.APIClient, string, error)
	IsCloudGovFunc  func() bool
	IsSupportedFunc func() bool
}

func (f *TestProvider) Client(ctx context.Context, secretRef *client.ObjectKey, log *zap.SugaredLogger, dryRun bool, recorder dryrun.Recorder) (*mongodbatlas.Client, string, error) {
	return f.ClientFunc(secretRef, log, dryRun)
}

func (f *TestProvider) SdkClient(ctx context.Context, secretRef *client.ObjectKey, log *zap.SugaredLogger, dryRun bool, recorder dryrun.Recorder) (*admin.APIClient, string, error) {
	return f.SdkClientFunc(secretRef, log, dryRun)
}

func (f *TestProvider) IsCloudGov() bool {
	return f.IsCloudGovFunc()
}

func (f *TestProvider) IsResourceSupported(_ api.AtlasCustomResource) bool {
	return f.IsSupportedFunc()
}
