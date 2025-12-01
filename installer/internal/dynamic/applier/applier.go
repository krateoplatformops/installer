package applier

import (
	"context"
	"encoding/json"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/utils/ptr"
)

const (
	InstalledByLabel = "app.kubernetes.io/installed-by"
	InstalledByValue = "krateo"
)

func NewApplier(rc *rest.Config) (*Applier, error) {
	dynamicClient, err := dynamic.NewForConfig(rc)
	if err != nil {
		return nil, err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(rc)
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(
		cacheddiscovery.NewMemCacheClient(discoveryClient),
	)

	return &Applier{
		dynamicClient: dynamicClient,
		mapper:        mapper,
	}, nil
}

type Applier struct {
	dynamicClient *dynamic.DynamicClient
	mapper        *restmapper.DeferredDiscoveryRESTMapper
}

type ApplyOptions struct {
	GVK       schema.GroupVersionKind
	Namespace string
	Name      string
}

func (a *Applier) Apply(ctx context.Context, content map[string]any, opts ApplyOptions) error {
	if len(content) == 0 {
		return nil
	}

	obj := unstructured.Unstructured{}
	obj.SetUnstructuredContent(content)
	obj.SetGroupVersionKind(opts.GVK)
	obj.SetNamespace(opts.Namespace)
	obj.SetName(opts.Name)

	restMapping, err := a.mapper.RESTMapping(opts.GVK.GroupKind(), opts.GVK.Version)
	if err != nil {
		return err
	}

	var ri dynamic.ResourceInterface
	if restMapping.Scope.Name() == meta.RESTScopeNameRoot {
		ri = a.dynamicClient.Resource(restMapping.Resource)
	} else {
		ri = a.dynamicClient.Resource(restMapping.Resource).
			Namespace(opts.Namespace)
	}

	data, err := json.Marshal(&obj)
	if err != nil {
		return err
	}

	// create or Update the object with SSA (types.ApplyPatchType indicates SSA).
	_, err = ri.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: InstalledByValue,
		Force:        ptr.To(true),
	})

	return err
}
