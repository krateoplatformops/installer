package workflow

import (
	"context"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/krateoplatformops/installer/apis/releases/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
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

	for _, el := range res.Set {
		if len(el.Value) > 0 {
			val := el.Value
			if strings.HasPrefix(val, "$") && len(val) > 1 {
				val, _ = r.env.Get(el.Value[1:])
			}
			el.Value = val
		}
	}
	spew.Dump(res)
	// unstr := unstructured.Unstructured{Object: src}

	// gv, err := schema.ParseGroupVersion(res.GetAPIVersion())
	// if err != nil {
	// 	return err
	// }

	// namespace := res.GetNamespace()
	// if len(namespace) == 0 {
	// 	namespace = r.ns
	// }

	// name := res.GetName()

	// return r.dyn.Apply(ctx, res.Object, dynamic.ApplyOptions{
	// 	GVK:       gv.WithKind(res.GetKind()),
	// 	Namespace: namespace,
	// 	Name:      name,
	// })

	return nil
}
