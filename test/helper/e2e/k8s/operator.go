package k8s

import (
	"context"
	"github.com/go-logr/zapr"
	"go.uber.org/zap/zapcore"
	"os"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimezap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/internal/collection"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/internal/featureflags"
	akov2 "github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/api/v1"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/operator"
)

var (
	setupSignalHandlerOnce sync.Once
	signalCancelledCtx     context.Context
)

func BuildManager(initCfg *Config) (manager.Manager, error) {
	akoScheme := runtime.NewScheme()
	utilruntime.Must(scheme.AddToScheme(akoScheme))
	utilruntime.Must(akov2.AddToScheme(akoScheme))

	rawLogger := ctrlruntimezap.NewRaw(
		ctrlruntimezap.WriteTo(GinkgoWriter),
		func(options *ctrlruntimezap.Options) {
			options.TimeEncoder = func(t time.Time, e zapcore.PrimitiveArrayEncoder) {
				zapcore.TimeEncoderOfLayout(time.RFC3339)(t.UTC(), e)
			}
		},
	)

	ctrl.SetLogger(zapr.NewLogger(rawLogger))
	setupLog := rawLogger.Named("setup").Sugar()

	config := mergeConfiguration(initCfg)
	setupLog.Info("starting with configuration", zap.Any("config", *config))

	// Ensure all concurrent managers configured per test share a single exit signal handler
	setupSignalHandlerOnce.Do(func() {
		signalCancelledCtx = ctrl.SetupSignalHandler()
	})

	return operator.NewBuilder(operator.ManagerProviderFunc(ctrl.NewManager), akoScheme).
		WithConfig(ctrl.GetConfigOrDie()).
		WithNamespaces(collection.Keys(config.WatchedNamespaces)...).
		WithLogger(rawLogger).
		WithMetricAddress(config.MetricsAddr).
		WithProbeAddress(config.ProbeAddr).
		WithLeaderElection(config.EnableLeaderElection).
		WithAtlasDomain(config.AtlasDomain).
		WithAPISecret(config.GlobalAPISecret).
		WithDeletionProtection(config.ObjectDeletionProtection).
		Build(signalCancelledCtx)
}

type Config struct {
	AtlasDomain                 string
	EnableLeaderElection        bool
	MetricsAddr                 string
	WatchedNamespaces           map[string]bool
	ProbeAddr                   string
	GlobalAPISecret             client.ObjectKey
	ObjectDeletionProtection    bool
	SubObjectDeletionProtection bool
	FeatureFlags                *featureflags.FeatureFlags
}

// ParseConfiguration fills the 'OperatorConfig' from the flags passed to the program
func mergeConfiguration(initCfg *Config) *Config {
	config := initCfg
	if config.AtlasDomain == "" {
		config.AtlasDomain = "https://cloud-qa.mongodb.com/"
	}
	if config.MetricsAddr == "" {
		// random port
		config.MetricsAddr = ":0"
	}
	if config.ProbeAddr == "" {
		// random port
		config.ProbeAddr = ":0"
	}

	return config
}

type ManagerStart func(ctx context.Context) error
type ManagerConfig func(config *Config)

func managerDefaults() *Config {
	return &Config{
		AtlasDomain:                 "https://cloud-qa.mongodb.com/",
		EnableLeaderElection:        false,
		MetricsAddr:                 "0",
		WatchedNamespaces:           map[string]bool{},
		ProbeAddr:                   "0",
		GlobalAPISecret:             client.ObjectKey{},
		ObjectDeletionProtection:    false,
		SubObjectDeletionProtection: false,
		FeatureFlags:                featureflags.NewFeatureFlags(os.Environ),
	}
}

func WithAtlasDomain(domain string) ManagerConfig {
	return func(config *Config) {
		config.AtlasDomain = domain
	}
}

func WithNamespaces(namespaces ...string) ManagerConfig {
	return func(config *Config) {
		for _, namespace := range namespaces {
			config.WatchedNamespaces[namespace] = true
		}
	}
}

func WithObjectDeletionProtection(flag bool) ManagerConfig {
	return func(config *Config) {
		config.ObjectDeletionProtection = flag
	}
}

func WithSubObjectDeletionProtection(flag bool) ManagerConfig {
	return func(config *Config) {
		config.SubObjectDeletionProtection = flag
	}
}

func WithGlobalKey(key client.ObjectKey) ManagerConfig {
	return func(config *Config) {
		config.GlobalAPISecret = key
	}
}

func RunManager(withConfigs ...ManagerConfig) (ManagerStart, error) {
	managerConfig := managerDefaults()

	for _, withConfig := range withConfigs {
		withConfig(managerConfig)
	}

	mgr, err := BuildManager(managerConfig)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context) error {
		err = mgr.Start(ctx)
		if err != nil {
			return err
		}

		return nil
	}, nil
}
