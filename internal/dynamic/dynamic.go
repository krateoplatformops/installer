package dynamic

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/itchyny/gojq"
	"k8s.io/apimachinery/pkg/api/meta"
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

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

type GetOptions struct {
	GVK       schema.GroupVersionKind
	Namespace string
	Name      string
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

func Extract(ctx context.Context, obj *unstructured.Unstructured, filter string) (any, error) {
	query, err := gojq.Parse(filter)
	if err != nil {
		return nil, err
	}

	var rawJson interface{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &rawJson)
	if err != nil {
		return nil, err
	}

	enc := newEncoder(false, 0)

	iter := query.RunWithContext(ctx, rawJson)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, err
		}
		if err := enc.encode(v); err != nil {
			return nil, err
		}
	}

	buf := strings.NewReader(enc.w.String())

	var xxx any
	err = json.NewDecoder(buf).Decode(&xxx)
	return xxx, err
}
