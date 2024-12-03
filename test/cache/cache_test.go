package cache

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/kylelemons/godebug/diff"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	clientgo_cache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	akov2 "github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/api/v1"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/api/v1/common"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/api/v1/project"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/indexer"
)

func TestCache(t *testing.T) {
	klog.InitFlags(nil)
	require.NoError(t, flag.Set("v", "6"))
	flag.Parse()

	zlog := zaptest.NewLogger(t,
		zaptest.WrapOptions(
			zap.Development(),
			zap.AddCaller(),
			//zap.AddStacktrace(zapcore.Level(-9)),
		),
		zaptest.Level(zapcore.Level(-9)),
	)
	logrLogger := zapr.NewLogger(zlog)
	ctrl.SetLogger(logrLogger.WithName("ctrl"))
	klog.SetLogger(logrLogger.WithName("klog"))
	akoLogger := zlog.Named("ako")

	akoScheme := runtime.NewScheme()
	utilruntime.Must(akov2.AddToScheme(akoScheme))

	cfg := config.GetConfigOrDie()
	cl, err := client.New(cfg, client.Options{Scheme: akoScheme})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// yaml:
	//
	// atlas cli kubernetes dry-run foo.yaml
	//
	// foo.yaml:
	//
	// kind: AtlasProject
	// metadata:
	//   name: my-project
	//   namespace: ns-encryption-at-rest-aws-thi11lo6uh
	// spec:
	//   name: CHANGED FOR DRY-RUN

	prj := &akov2.AtlasProject{}
	prj.Name = "my-project"
	//prj.Name = "new-project"
	prj.Namespace = "ns-encryption-at-rest-aws-thi11lo6uh"
	//prj.Spec.RegionUsageRestrictions = "INVALID"
	//prj.Spec.Name = "CHANGED FOR DRY-RUN"
	//prj.Spec.ConnectionSecret = &common.ResourceRefNamespaced{
	//	Name:      "CHANGE",
	//	Namespace: "bar",
	//}
	prj.Spec.ProjectIPAccessList = append(prj.Spec.ProjectIPAccessList, project.IPAccessList{
		CIDRBlock: "0.0.0.0/1",
		Comment:   "CHANGED COMMENT FOR DRY-RUN",
	})

	gvk, err := apiutil.GVKForObject(prj, akoScheme)
	if err != nil {
		t.Fatal(err)
	}
	prj.TypeMeta.Kind = gvk.Kind
	prj.TypeMeta.APIVersion = gvk.GroupVersion().String()

	patchOptions := []client.PatchOption{
		client.ForceOwnership,            // Optional: Forces ownership of the fields
		client.FieldOwner("ako-dry-run"), // Name your field manager
		client.DryRunAll,                 // Enable dry-run
	}
	if err := cl.Patch(ctx, prj, client.Apply, patchOptions...); err != nil {
		t.Fatal(err)
	}

	// everything until here is OK

	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		t.Fatalf("could not create HTTP client from config: %v", err)
	}
	tr := NewDryRunTransport(httpClient.Transport)
	httpClient.Transport = tr

	never := time.Duration(0)
	localCache, err := cache.New(cfg, cache.Options{
		Scheme: akoScheme,
		// We don't want cache resyncs happen during dry-running.
		// Setting this to 0 will cause the reflector cause never to send anything, see
		// https://github.com/kubernetes/client-go/blob/37045084c2aa82927b0e5ffc752861430fd7e4ab/tools/cache/reflector.go#L335.
		SyncPeriod: &never,
		HTTPClient: httpClient,
	})

	// Register AKO indexers to act on changes on the underlying cache.
	// All objects present in this cache are going to be reflected in the AKO index topology.
	err = indexer.RegisterAll(ctx, localCache, akoLogger)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := localCache.Start(ctx); err != nil {
			panic(fmt.Sprintf("Failed to start cache: %v", err))
		}
	}()

	if !localCache.WaitForCacheSync(ctx) {
		t.Fatal("Cache failed to sync")
	}

	yes := true
	localClient, err := client.New(cfg, client.Options{
		HTTPClient: httpClient,
		Scheme:     akoScheme,
		Cache: &client.CacheOptions{
			Reader: localCache,
		},
		DryRun: &yes,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = localClient

	prjList := &akov2.AtlasProjectList{}
	err = localClient.List(ctx, prjList)
	if err != nil {
		t.Fatal(err)
	}
	before := MustMarshalYAML(prjList, akoScheme)

	store, err := GetStoreForObject(ctx, akoScheme, localCache, prj)
	if err != nil {
		t.Fatal(err)
	}

	// Updates delegates to Add,
	// see https://github.com/kubernetes/client-go/blob/37045084c2aa82927b0e5ffc752861430fd7e4ab/tools/cache/thread_safe_store.go#L233.
	err = store.Update(prj)
	if err != nil {
		t.Fatal(err)
	}

	err = localClient.List(ctx, prjList)
	if err != nil {
		t.Fatal(err)
	}
	after := MustMarshalYAML(prjList, akoScheme)

	fmt.Println("DIFF:", diff.Diff(before, after))

	df := &akov2.AtlasFederatedAuth{}
	df.Name = "some-federated-auth"
	df.Namespace = "default"
	df.Spec.ConnectionSecretRef = common.ResourceRefNamespaced{Name: "name", Namespace: "bar"}

	store, err = GetStoreForObject(ctx, akoScheme, localCache, df)
	if err != nil {
		t.Fatal(err)
	}

	err = store.Update(df)
	if err != nil {
		t.Fatal(err)
	}

	//for indexerName, _ := range idx.GetIndexers() {
	//	fmt.Println(" |")
	//	fmt.Printf(" \\- index: %q\n", indexerName)
	//	for _, indexedValue := range idx.ListIndexFuncValues(indexerName) {
	//		keys, err := idx.IndexKeys(indexerName, indexedValue)
	//		if err != nil {
	//			t.Fatal(err)
	//			return
	//		}
	//		fmt.Printf(" | \\- %q -> %q\n", indexedValue, keys)
	//	}
	//}

	//time.Sleep(20 * time.Minute)
}

type storeHolder interface {
	GetStore() clientgo_cache.Store
}

func GetStoreForObject(ctx context.Context, scheme *runtime.Scheme, infs cache.Informers, obj runtime.Object) (clientgo_cache.Store, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return nil, fmt.Errorf("error getting GVK: %w", err)
	}

	inf, err := infs.GetInformerForKind(ctx, gvk, cache.BlockUntilSynced(true))
	if err != nil {
		return nil, fmt.Errorf("error getting informer for kind: %v", gvk)
	}

	// We can't work with informers who don't have an underlying cache (store).
	// In dry-run, we must be able to update the cache without watching api server
	// with objects loaded from local state (file).
	//
	// In controller-runtime this defaults to k8s.io/client-go/tools/cache.NewSharedIndexInformer,
	// see https://github.com/kubernetes-sigs/controller-runtime/blob/8e44a4307f70a4d5f4c5aaf0869e2bec7167f4ab/pkg/cache/internal/informers.go#L60.
	sharedInf, ok := inf.(storeHolder)
	if !ok {
		return nil, fmt.Errorf("informer is not having a store: %T", inf)
	}

	return sharedInf.GetStore(), nil
}

func MustMarshalYAML(obj runtime.Object, scheme *runtime.Scheme) string {
	y := json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{Yaml: true, Pretty: true})
	var buf bytes.Buffer
	if err := y.Encode(obj, &buf); err != nil {
		panic(err)
	}
	return buf.String()
}
