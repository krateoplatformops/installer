package getter

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type GetOptions struct {
	GVK       schema.GroupVersionKind
	Namespace string
	Name      string
}

func NewGetter(rc *rest.Config) (*Getter, error) {
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

	return &Getter{
		dynamicClient: dynamicClient,
		mapper:        mapper,
	}, nil
}

type Getter struct {
	dynamicClient *dynamic.DynamicClient
	mapper        *restmapper.DeferredDiscoveryRESTMapper
}

func (g *Getter) Get(ctx context.Context, opts GetOptions) (*unstructured.Unstructured, error) {
	restMapping, err := g.mapper.RESTMapping(opts.GVK.GroupKind(), opts.GVK.Version)
	if err != nil {
		return nil, err
	}

	var ri dynamic.ResourceInterface
	if restMapping.Scope.Name() == meta.RESTScopeNameRoot {
		ri = g.dynamicClient.Resource(restMapping.Resource)
	} else {
		ri = g.dynamicClient.Resource(restMapping.Resource).
			Namespace(opts.Namespace)
	}

	return ri.Get(ctx, opts.Name, corev1.GetOptions{})
}
