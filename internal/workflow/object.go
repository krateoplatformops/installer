package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/krateoplatformops/installer/apis/releases/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/kubernetes/dynamic"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type objHandler struct {
	rc  *rest.Config
	env *cache.Cache[string, string]
	ns  string
}

func (r *objHandler) Namespace(ns string) {
	r.ns = ns
}

func (r *objHandler) Do(ctx context.Context, res *v1alpha1.Object) error {
	all := make([]string, len(res.Set))
	for i, el := range res.Set {
		if len(el.Value) > 0 {
			val := el.Value
			if strings.HasPrefix(val, "$") && len(val) > 1 {
				val, _ = r.env.Get(el.Value[1:])
			}
			all[i] = fmt.Sprintf("%s=%s", el.Name, val)
		}
	}

	src := map[string]any{
		"apiVersion": res.APIVersion,
		"kind":       res.Kind,
		"metadata": map[string]any{
			"name":      res.Metadata.Name,
			"namespace": res.Metadata.Namespace,
		},
	}

	err := strvals.ParseInto(strings.Join(all, ","), src)
	if err != nil {
		return err
	}

	uns := unstructured.Unstructured{Object: src}

	gv, err := schema.ParseGroupVersion(uns.GetAPIVersion())
	if err != nil {
		return err
	}

	namespace := uns.GetNamespace()
	if len(namespace) == 0 {
		namespace = r.ns
	}

	dyn, err := dynamic.NewApplier(r.rc)
	if err != nil {
		return err
	}

	return dyn.Apply(ctx, uns.Object, dynamic.ApplyOptions{
		GVK:       gv.WithKind(uns.GetKind()),
		Namespace: namespace,
		Name:      uns.GetName(),
	})
}
