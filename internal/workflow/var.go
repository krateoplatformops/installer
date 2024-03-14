package workflow

import (
	"context"
	"strings"

	"github.com/krateoplatformops/installer/apis/releases/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/kubernetes/dynamic"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type varHandler struct {
	dyn *dynamic.Getter
	env *cache.Cache[string, string]
	ns  string
}

func (r *varHandler) Namespace(ns string) {
	r.ns = ns
}

func (r *varHandler) Do(ctx context.Context, res *v1alpha1.Var) error {
	if len(res.Value) > 0 {
		val := res.Value
		if strings.HasPrefix(val, "$") && len(val) > 1 {
			val, _ = r.env.Get(res.Value[1:])
		}
		r.env.Set(res.Name, val)
	}

	if res.ValueFrom == nil {
		return nil
	}

	gv, err := schema.ParseGroupVersion(res.ValueFrom.APIVersion)
	if err != nil {
		return err
	}

	namespace := res.ValueFrom.Metadata.Namespace
	if len(namespace) == 0 {
		namespace = r.ns
	}

	name := res.ValueFrom.Metadata.Name

	obj, err := r.dyn.Get(ctx, dynamic.GetOptions{
		Name:      name,
		Namespace: namespace,
		GVK:       gv.WithKind(res.ValueFrom.Kind),
	})
	if err != nil {
		return err
	}

	val, err := dynamic.Extract(ctx, obj, res.ValueFrom.Selector)
	if val != nil {
		r.env.Set(res.Name, strval(val))
	}

	return err
}
