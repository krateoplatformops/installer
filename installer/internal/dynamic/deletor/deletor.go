package deletor

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/utils/ptr"
)

type DeleteOptions struct {
	GVK       schema.GroupVersionKind
	Namespace string
	Name      string
}

func NewDeletor(rc *rest.Config) (*Deletor, error) {
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

	return &Deletor{
		dynamicClient: dynamicClient,
		mapper:        mapper,
	}, nil
}

type Deletor struct {
	dynamicClient *dynamic.DynamicClient
	mapper        *restmapper.DeferredDiscoveryRESTMapper
}

func (g *Deletor) Delete(ctx context.Context, opts DeleteOptions) error {
	restMapping, err := g.mapper.RESTMapping(opts.GVK.GroupKind(), opts.GVK.Version)
	if err != nil {
		return err
	}

	var ri dynamic.ResourceInterface
	if restMapping.Scope.Name() == meta.RESTScopeNameRoot {
		ri = g.dynamicClient.Resource(restMapping.Resource)
	} else {
		ri = g.dynamicClient.Resource(restMapping.Resource).
			Namespace(opts.Namespace)
	}

	return ri.Delete(ctx, opts.Name, metav1.DeleteOptions{
		PropagationPolicy: ptr.To(metav1.DeletePropagationForeground),
	})
}
